import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'api.dart';
import 'models.dart';

sealed class RealtimeEvent {
  const RealtimeEvent();
}

class RealtimeSnapshot extends RealtimeEvent {
  const RealtimeSnapshot(this.document, this.members, this.reviewNumber, this.version);
  final UmlDocument document;
  final List<RealtimeMember> members;
  final int reviewNumber;
  final int version;
}

class RealtimeMember {
  const RealtimeMember({required this.userId, required this.displayName, required this.lastSeen});
  final String userId;
  final String displayName;
  final String lastSeen;
}

class RealtimePresenceChanged extends RealtimeEvent {
  const RealtimePresenceChanged(this.member, this.joined);
  final RealtimeMember member;
  final bool joined;
}

class RealtimeDiagramChanged extends RealtimeEvent {
  const RealtimeDiagramChanged(this.reviewNumber, this.version, this.actorId, this.kind);
  final int reviewNumber;
  final int version;
  final String actorId;
  final String kind;
}

class RealtimeErrorEvent extends RealtimeEvent {
  const RealtimeErrorEvent(this.message);
  final String message;
}

/// High-level connection lifecycle, surfaced so the UI can show reconnection
/// progress instead of silently going stale after a network drop.
enum RealtimeConnectionState { connected, reconnecting, disconnected }

class RealtimeConnectionChanged extends RealtimeEvent {
  const RealtimeConnectionChanged(this.state, {this.attempt = 0, this.retryIn});
  final RealtimeConnectionState state;

  /// 1-based reconnection attempt number. Only meaningful when [state] is
  /// [RealtimeConnectionState.reconnecting].
  final int attempt;

  /// Delay before the next reconnection attempt fires. Only meaningful when
  /// [state] is [RealtimeConnectionState.reconnecting].
  final Duration? retryIn;
}

/// Backoff delays (seconds) applied between reconnection attempts. The last
/// value repeats for every attempt beyond the list length, capping the wait.
const List<int> _reconnectBackoffSeconds = [1, 2, 4, 8, 16, 30];

class DiagramRealtimeService {
  DiagramRealtimeService({
    required this.api,
    required this.projectId,
    required this.diagramId,
    WebSocketChannel Function(Uri uri)? channelFactory,
    Future<String> Function(String projectId, String diagramId)? ticketSource,
  })  : _channelFactory = channelFactory ?? WebSocketChannel.connect,
        _ticketSource = ticketSource ?? api.realtimeTicket;

  final ApiClient api;
  final String projectId;
  final String diagramId;
  final WebSocketChannel Function(Uri uri) _channelFactory;
  final Future<String> Function(String projectId, String diagramId) _ticketSource;

  final _events = StreamController<RealtimeEvent>.broadcast();
  WebSocketChannel? _channel;
  Timer? _heartbeat;
  Timer? _reconnectTimer;
  StreamSubscription<dynamic>? _subscription;

  int _attempt = 0;
  bool _connecting = false;
  bool _manualDisconnect = false;
  bool _disposed = false;
  // Guards a single connection's onError/onDone pair against scheduling two
  // reconnection attempts for what is really one drop.
  bool _closeHandled = false;
  RealtimeConnectionState _state = RealtimeConnectionState.disconnected;

  Stream<RealtimeEvent> get events => _events.stream;
  RealtimeConnectionState get connectionState => _state;

  /// Connects (or reconnects from scratch) and arms automatic reconnection
  /// for subsequent drops. Never throws: connection failures are reported
  /// through [events] and retried with backoff.
  Future<void> connect() async {
    _manualDisconnect = false;
    _attempt = 0;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    await _attemptConnect();
  }

  /// Requests an immediate reconnection attempt, bypassing any pending
  /// backoff delay. Intended for foreground/lifecycle triggers. No-op when
  /// already connected, already connecting, or after an explicit disconnect.
  Future<void> reconnectNow() async {
    if (_manualDisconnect || _disposed || _connecting) return;
    if (_state == RealtimeConnectionState.connected) return;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    await _attemptConnect();
  }

  Future<void> _attemptConnect() async {
    if (_disposed || _connecting) return;
    _connecting = true;
    await _teardownChannel();
    _closeHandled = false;
    try {
      // Tickets are single-use, so every attempt (including retries) must
      // request a fresh one.
      final ticket = await _ticketSource(projectId, diagramId);
      final uri = Uri.parse('${api.websocketBaseUrl}/projects/$projectId/diagrams/$diagramId/ws?ticket=${Uri.encodeQueryComponent(ticket)}');
      final channel = _channelFactory(uri);
      _channel = channel;
      await channel.ready;
      if (_disposed) {
        await channel.sink.close();
        return;
      }
      _subscription = channel.stream.listen(_handleMessage, onError: (Object error) {
        _handleClosed(RealtimeErrorEvent('Conexión realtime interrumpida: $error'));
      }, onDone: () {
        _handleClosed(null);
      });
      _heartbeat = Timer.periodic(const Duration(seconds: 20), (_) {
        channel.sink.add(jsonEncode({'type': 'presence.heartbeat', 'payload': {}}));
      });
      _state = RealtimeConnectionState.connected;
      _events.add(const RealtimeConnectionChanged(RealtimeConnectionState.connected));
      // Backoff intentionally resets only once a snapshot actually arrives
      // (see _handleMessage), not merely because the socket opened: a server
      // that accepts the handshake and immediately closes again should keep
      // backing off instead of hammering it every second.
    } catch (error) {
      _events.add(RealtimeErrorEvent('Conexión realtime interrumpida: $error'));
      _scheduleReconnect();
    } finally {
      _connecting = false;
    }
  }

  void _handleClosed(RealtimeErrorEvent? errorEvent) {
    if (_closeHandled) return;
    _closeHandled = true;
    _heartbeat?.cancel();
    _heartbeat = null;
    unawaited(_subscription?.cancel());
    _subscription = null;
    if (errorEvent != null) _events.add(errorEvent);
    if (_manualDisconnect || _disposed) return;
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_manualDisconnect || _disposed) return;
    _reconnectTimer?.cancel();
    final delay = _backoffFor(_attempt);
    _attempt++;
    _state = RealtimeConnectionState.reconnecting;
    _events.add(RealtimeConnectionChanged(RealtimeConnectionState.reconnecting, attempt: _attempt, retryIn: delay));
    _reconnectTimer = Timer(delay, () {
      _reconnectTimer = null;
      unawaited(_attemptConnect());
    });
  }

  Duration _backoffFor(int failureCount) {
    final index = failureCount.clamp(0, _reconnectBackoffSeconds.length - 1);
    return Duration(seconds: _reconnectBackoffSeconds[index]);
  }

  void _handleMessage(dynamic raw) {
    try {
      final envelope = jsonDecode(raw as String) as Map<String, dynamic>;
      final type = envelope['type']?.toString();
      final payload = (envelope['payload'] as Map?)?.cast<String, dynamic>() ?? const <String, dynamic>{};
      if (type == 'snapshot') {
        final members = ((payload['members'] as List?) ?? const []).map((item) {
          final member = item as Map<String, dynamic>;
          return RealtimeMember(userId: member['userId']?.toString() ?? '', displayName: member['displayName']?.toString() ?? '', lastSeen: member['lastSeen']?.toString() ?? '');
        }).toList();
        // A snapshot confirms a fully successful handshake, so any pending
        // backoff from earlier flapping resets.
        _attempt = 0;
        _events.add(RealtimeSnapshot(UmlDocument.fromJson(payload['document'] as Map<String, dynamic>), members, (payload['reviewNumber'] as num?)?.toInt() ?? 0, (payload['version'] as num?)?.toInt() ?? 0));
      } else if (type == 'presence.join' || type == 'presence.leave') {
        final member = (payload['member'] as Map).cast<String, dynamic>();
        _events.add(RealtimePresenceChanged(RealtimeMember(userId: member['userId']?.toString() ?? '', displayName: member['displayName']?.toString() ?? '', lastSeen: member['lastSeen']?.toString() ?? ''), type == 'presence.join'));
      } else if (type == 'diagram.changed') {
        _events.add(RealtimeDiagramChanged((payload['reviewNumber'] as num?)?.toInt() ?? 0, (payload['version'] as num?)?.toInt() ?? 0, payload['actorId']?.toString() ?? '', payload['kind']?.toString() ?? 'autosave'));
      } else if (type == 'error') {
        _events.add(RealtimeErrorEvent((payload['message'] ?? 'Error realtime').toString()));
      }
    } catch (error) {
      _events.add(RealtimeErrorEvent('Mensaje realtime inválido: $error'));
    }
  }

  Future<void> _teardownChannel() async {
    _heartbeat?.cancel();
    _heartbeat = null;
    // Cancellation of a broadcast subscription is fire-and-forget: we don't
    // need it to complete before proceeding (the reference is dropped right
    // away), and waiting for it needlessly ties connection teardown to
    // stream-internals timing.
    unawaited(_subscription?.cancel());
    _subscription = null;
    await _channel?.sink.close();
    _channel = null;
  }

  /// Disconnects explicitly and stops all future automatic reconnection.
  Future<void> disconnect() async {
    _manualDisconnect = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _attempt = 0;
    await _teardownChannel();
    _state = RealtimeConnectionState.disconnected;
    if (!_disposed) {
      _events.add(const RealtimeConnectionChanged(RealtimeConnectionState.disconnected));
    }
  }

  Future<void> dispose() async {
    _disposed = true;
    _manualDisconnect = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    await _teardownChannel();
    await _events.close();
  }
}
