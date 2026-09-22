import 'dart:convert';
import 'dart:typed_data';

import 'package:ai_uml_architect_mobile/api.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class MemoryTokenStore implements TokenStore {
  String? token;
  @override
  Future<String?> read() async => token;
  @override
  Future<void> write(String value) async => token = value;
  @override
  Future<void> clear() async => token = null;
}

void main() {
  test('login uses documented path and bearer token is sent to projects',
      () async {
    final store = MemoryTokenStore();
    final requests = <http.Request>[];
    final client = MockClient((request) async {
      requests.add(request);
      if (request.url.path.endsWith('/auth/login')) {
        return http.Response(jsonEncode({'accessToken': 'token-1'}), 200);
      }
      return http.Response(
          jsonEncode([
            {'id': 'p1', 'name': 'Demo'}
          ]),
          200);
    });
    final api = ApiClient(
        client: client,
        tokenStore: store,
        baseUrl: 'https://example.test/api/v1');
    await api.login('a@example.test', 'secret');
    final projects = await api.projects();
    expect(projects.single.name, 'Demo');
    expect(requests[0].url.path, '/api/v1/auth/login');
    expect(requests[1].url.path, '/api/v1/projects');
    expect(requests[1].headers['authorization'], isNotEmpty);
  });

  test('image import sends authenticated multipart with explicit MIME',
      () async {
    final store = MemoryTokenStore()..token = 'token-1';
    http.BaseRequest? captured;
    List<int>? body;
    final client = MockClient((request) async {
      captured = request;
      body = request.bodyBytes;
      return http.Response(
          jsonEncode({
            'id': 'diagram-1',
            'name': 'Imported',
            'classes': [],
            'relationships': [],
          }),
          200);
    });
    final api = ApiClient(
      client: client,
      tokenStore: store,
      baseUrl: 'https://example.test/api/v1',
    );

    final result = await api.importDiagramImage(
      'project-1',
      'diagram-1',
      image: Uint8List.fromList([1, 2, 3]),
      mimeType: 'image/png',
    );

    expect(result.name, 'Imported');
    expect(captured!.headers['authorization'], 'Bearer token-1');
    expect(captured!.headers['content-type'],
        startsWith('multipart/form-data; boundary='));
    expect(utf8.decode(body!),
        contains('content-disposition: form-data; name="image"'));
    expect(utf8.decode(body!), contains('content-type: image/png'));
  });
}
