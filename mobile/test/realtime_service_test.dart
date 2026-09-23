import 'dart:async';

import 'package:ai_uml_architect_mobile/api.dart';
import 'package:ai_uml_architect_mobile/realtime_service.dart';
import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:stream_channel/stream_channel.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// `Stream` has no built-in `whereType`, unlike `Iterable`; this fills the
/// gap for the realtime event stream used throughout the tests below.
extension _RealtimeEventStreamX on Stream<RealtimeEvent> {
  Stream<T> ofType<T extends RealtimeEvent>() =>
      where((event) => event is T).cast<T>();
}

/// Minimal fake [WebSocketChannel] driven entirely by the test, so no real
/// socket or real time is ever involved. [StreamChannelMixin] supplies the
/// default stream-channel helper methods (transform, pipe, cast, ...); only
/// the members [WebSocketChannel] adds on top are implemented here.
class FakeWebSocketChannel with StreamChannelMixin<dynamic> implements WebSocketChannel {
  FakeWebSocketChannel({Future<void>? readyFuture})
      : ready = readyFuture ?? Future<void>.value();

  final StreamController<dynamic> _incoming = StreamController<dynamic>.broadcast();
  final List<dynamic> sent = [];
  bool closed = false;

  @override
  final Future<void> ready;

  @override
  String? protocol;

  @override
  int? closeCode;

  @override
  String? closeReason;

  @override
  Stream get stream => _incoming.stream;

  @override
  WebSocketSink get sink => _FakeSink(this);

  void emit(String raw) => _incoming.add(raw);
  void emitError(Object error) => _incoming.addError(error);
  void close_() => _incoming.close();
}

class _FakeSink implements WebSocketSink {
  _FakeSink(this._channel);
  final FakeWebSocketChannel _channel;

  @override
  void add(dynamic event) => _channel.sent.add(event);

  @override
  void addError(Object error, [StackTrace? stackTrace]) {}

  @override
  Future addStream(Stream stream) async {}

  @override
  Future close([int? closeCode, String? closeReason]) async {
    _channel.closed = true;
    _channel.closeCode = closeCode;
    _channel.closeReason = closeReason;
    _channel.close_();
  }

  @override
  Future get done => Future<void>.value();
}

const snapshotMessage = '''
{"type":"snapshot","payload":{"document":{"id":"d1","name":"Diagram","classes":[],"relationships":[]},"members":[],"reviewNumber":1,"version":1}}
''';

void main() {
  late ApiClient api;
  late List<String> ticketCalls;
  late List<Uri> connectCalls;
  late List<FakeWebSocketChannel> issuedChannels;
  late List<Object?> pendingTicketFailures;

  setUp(() {
    api = ApiClient(baseUrl: 'https://example.test/api/v1');
    ticketCalls = [];
    connectCalls = [];
    issuedChannels = [];
    pendingTicketFailures = [];
  });

  Future<String> ticketSource(String projectId, String diagramId) async {
    ticketCalls.add('$projectId/$diagramId');
    if (pendingTicketFailures.isNotEmpty) {
      final failure = pendingTicketFailures.removeAt(0);
      if (failure != null) throw failure;
    }
    return 'ticket-${ticketCalls.length}';
  }

  WebSocketChannel channelFactory(Uri uri) {
    connectCalls.add(uri);
    final channel = FakeWebSocketChannel();
    issuedChannels.add(channel);
    return channel;
  }

  DiagramRealtimeService buildService() => DiagramRealtimeService(
        api: api,
        projectId: 'p1',
        diagramId: 'd1',
        channelFactory: channelFactory,
        ticketSource: ticketSource,
      );

  test('connects, receives a snapshot and emits a connected status', () {
    fakeAsync((async) {
      final service = buildService();
      final events = <RealtimeEvent>[];
      service.events.listen(events.add);

      unawaited(service.connect());
      async.flushMicrotasks();

      expect(connectCalls, hasLength(1));
      expect(ticketCalls, hasLength(1));
      expect(
          events.whereType<RealtimeConnectionChanged>().first.state,
          RealtimeConnectionState.connected);

      issuedChannels.single.emit(snapshotMessage);
      async.flushMicrotasks();

      expect(events.whereType<RealtimeSnapshot>(), hasLength(1));

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('reconnects automatically after the socket closes, with growing capped backoff', () {
    fakeAsync((async) {
      final service = buildService();
      final states = <RealtimeConnectionChanged>[];
      service.events.ofType<RealtimeConnectionChanged>().listen(states.add);

      unawaited(service.connect());
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.connected);

      // Simulate consecutive drops and verify the backoff sequence grows and
      // then caps, without ever waiting real time.
      final expectedDelays = [1, 2, 4, 8, 16, 30, 30];
      for (final expectedSeconds in expectedDelays) {
        issuedChannels.last.close_();
        async.flushMicrotasks();

        final reconnecting = states.last;
        expect(reconnecting.state, RealtimeConnectionState.reconnecting);
        expect(reconnecting.retryIn, Duration(seconds: expectedSeconds));

        async.elapse(Duration(seconds: expectedSeconds));
        async.flushMicrotasks();

        expect(states.last.state, RealtimeConnectionState.connected);
      }

      expect(connectCalls.length, expectedDelays.length + 1);
      expect(ticketCalls.length, expectedDelays.length + 1);

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('backoff resets to the shortest delay only once a snapshot confirms sync, not merely on socket open', () {
    fakeAsync((async) {
      final service = buildService();
      final states = <RealtimeConnectionChanged>[];
      service.events.ofType<RealtimeConnectionChanged>().listen(states.add);

      unawaited(service.connect());
      async.flushMicrotasks();
      // The real server always sends a snapshot right after connecting; do
      // the same here so the first connection counts as a confirmed sync.
      issuedChannels.last.emit(snapshotMessage);
      async.flushMicrotasks();

      // First drop: backoff starts at the shortest delay.
      issuedChannels.last.close_();
      async.flushMicrotasks();
      expect(states.last.retryIn, const Duration(seconds: 1));
      async.elapse(const Duration(seconds: 1));
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.connected);

      // The socket reopened but drops again before ever receiving a
      // snapshot: the backoff must keep growing, not reset just because the
      // socket briefly opened (guards against hammering a flapping server).
      issuedChannels.last.close_();
      async.flushMicrotasks();
      expect(states.last.retryIn, const Duration(seconds: 2));
      async.elapse(const Duration(seconds: 2));
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.connected);

      // This time a snapshot confirms the reconnection actually synced.
      issuedChannels.last.emit(snapshotMessage);
      async.flushMicrotasks();

      // A further drop should restart backoff from the shortest delay.
      issuedChannels.last.close_();
      async.flushMicrotasks();
      expect(states.last.retryIn, const Duration(seconds: 1));

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('each reconnection attempt requests a fresh single-use ticket', () {
    fakeAsync((async) {
      final service = buildService();
      unawaited(service.connect());
      async.flushMicrotasks();
      expect(ticketCalls, ['p1/d1']);

      issuedChannels.last.close_();
      async.flushMicrotasks();
      async.elapse(const Duration(seconds: 1));
      async.flushMicrotasks();

      expect(ticketCalls, ['p1/d1', 'p1/d1']);
      expect(connectCalls, hasLength(2));

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('no reconnection happens after an explicit disconnect', () {
    fakeAsync((async) {
      final service = buildService();
      final states = <RealtimeConnectionChanged>[];
      service.events.ofType<RealtimeConnectionChanged>().listen(states.add);

      unawaited(service.connect());
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.connected);

      unawaited(service.disconnect());
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.disconnected);

      final callsBefore = connectCalls.length;

      // Even if the underlying socket reports done after disconnect, no
      // reconnection attempt should be scheduled.
      issuedChannels.last.close_();
      async.elapse(const Duration(seconds: 60));
      async.flushMicrotasks();

      expect(connectCalls.length, callsBefore);
      expect(states.last.state, RealtimeConnectionState.disconnected);

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('a failing ticket request is retried with backoff', () {
    fakeAsync((async) {
      pendingTicketFailures.add(Exception('network down'));
      final service = buildService();
      final states = <RealtimeConnectionChanged>[];
      service.events.ofType<RealtimeConnectionChanged>().listen(states.add);
      final errors = <RealtimeErrorEvent>[];
      service.events.ofType<RealtimeErrorEvent>().listen(errors.add);

      unawaited(service.connect());
      async.flushMicrotasks();

      expect(connectCalls, isEmpty);
      expect(errors, hasLength(1));
      expect(states.last.state, RealtimeConnectionState.reconnecting);
      expect(states.last.retryIn, const Duration(seconds: 1));

      async.elapse(const Duration(seconds: 1));
      async.flushMicrotasks();

      expect(connectCalls, hasLength(1));
      expect(states.last.state, RealtimeConnectionState.connected);

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('status events are emitted in order: connected, reconnecting, connected', () {
    fakeAsync((async) {
      final service = buildService();
      final states = <RealtimeConnectionState>[];
      service.events.ofType<RealtimeConnectionChanged>().listen((e) => states.add(e.state));

      unawaited(service.connect());
      async.flushMicrotasks();

      issuedChannels.last.close_();
      async.flushMicrotasks();
      async.elapse(const Duration(seconds: 1));
      async.flushMicrotasks();

      expect(states, [
        RealtimeConnectionState.connected,
        RealtimeConnectionState.reconnecting,
        RealtimeConnectionState.connected,
      ]);

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('reconnectNow triggers an immediate attempt bypassing the pending backoff delay', () {
    fakeAsync((async) {
      final service = buildService();
      final states = <RealtimeConnectionChanged>[];
      service.events.ofType<RealtimeConnectionChanged>().listen(states.add);

      unawaited(service.connect());
      async.flushMicrotasks();

      issuedChannels.last.close_();
      async.flushMicrotasks();
      expect(states.last.state, RealtimeConnectionState.reconnecting);

      unawaited(service.reconnectNow());
      async.flushMicrotasks();

      expect(states.last.state, RealtimeConnectionState.connected);
      expect(connectCalls, hasLength(2));

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });

  test('reconnectNow is a no-op while already connected', () {
    fakeAsync((async) {
      final service = buildService();
      unawaited(service.connect());
      async.flushMicrotasks();
      expect(connectCalls, hasLength(1));

      unawaited(service.reconnectNow());
      async.flushMicrotasks();

      expect(connectCalls, hasLength(1));

      unawaited(service.dispose());
      async.flushMicrotasks();
    });
  });
}
