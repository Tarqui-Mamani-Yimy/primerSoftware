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

class DiagramRealtimeService {
  DiagramRealtimeService({required this.api, required this.projectId, required this.diagramId});
  final ApiClient api;
  final String projectId;
  final String diagramId;
  final _events = StreamController<RealtimeEvent>.broadcast();
  WebSocketChannel? _channel;
  Timer? _heartbeat;
  StreamSubscription<dynamic>? _subscription;

  Stream<RealtimeEvent> get events => _events.stream;

  Future<void> connect() async {
    await disconnect();
    final ticket = await api.realtimeTicket(projectId, diagramId);
    final uri = Uri.parse('${api.websocketBaseUrl}/projects/$projectId/diagrams/$diagramId/ws?ticket=${Uri.encodeQueryComponent(ticket)}');
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    _subscription = channel.stream.listen(_handleMessage, onError: (Object error) {
      _events.add(RealtimeErrorEvent('Conexión realtime interrumpida: $error'));
    }, onDone: () {
      _heartbeat?.cancel();
      _heartbeat = null;
    });
    _heartbeat = Timer.periodic(const Duration(seconds: 20), (_) {
      channel.sink.add(jsonEncode({'type': 'presence.heartbeat', 'payload': {}}));
    });
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

  Future<void> disconnect() async {
    _heartbeat?.cancel();
    _heartbeat = null;
    await _subscription?.cancel();
    _subscription = null;
    await _channel?.sink.close();
    _channel = null;
  }

  Future<void> dispose() async {
    await disconnect();
    await _events.close();
  }
}
