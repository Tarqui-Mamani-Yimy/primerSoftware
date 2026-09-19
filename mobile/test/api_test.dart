import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:ai_uml_architect_mobile/api.dart';

class MemoryTokenStore implements TokenStore {
  String? token;
  @override Future<String?> read() async => token;
  @override Future<void> write(String value) async => token = value;
  @override Future<void> clear() async => token = null;
}

void main() {
  test('login uses documented path and bearer token is sent to projects', () async {
    final store = MemoryTokenStore();
    final requests = <http.Request>[];
    final client = MockClient((request) async {
      requests.add(request);
      if (request.url.path.endsWith('/auth/login')) {
        return http.Response(jsonEncode({'accessToken': 'token-1'}), 200);
      }
      return http.Response(jsonEncode([{'id': 'p1', 'name': 'Demo'}]), 200);
    });
    final api = ApiClient(client: client, tokenStore: store, baseUrl: 'https://example.test/api/v1');
    await api.login('a@example.test', 'secret');
    final projects = await api.projects();
    expect(projects.single.name, 'Demo');
    expect(requests[0].url.path, '/api/v1/auth/login');
    expect(requests[1].url.path, '/api/v1/projects');
    expect(requests[1].headers['authorization'], 'Bearer token-1');
  });
}
