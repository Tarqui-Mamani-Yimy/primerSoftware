import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

import 'models.dart';

abstract class TokenStore {
  Future<String?> read();
  Future<void> write(String token);
  Future<void> clear();
}

class SecureTokenStore implements TokenStore {
  const SecureTokenStore([this.storage = const FlutterSecureStorage()]);
  final FlutterSecureStorage storage;

  @override
  Future<String?> read() => storage.read(key: 'access_token');
  @override
  Future<void> write(String token) => storage.write(key: 'access_token', value: token);
  @override
  Future<void> clear() => storage.delete(key: 'access_token');
}

class ApiException implements Exception {
  const ApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;
  @override
  String toString() => 'ApiException($statusCode): $message';
}

class ApiClient {
  ApiClient({
    http.Client? client,
    TokenStore? tokenStore,
    String? baseUrl,
  })  : client = client ?? http.Client(),
        tokenStore = tokenStore ?? const SecureTokenStore(),
        baseUrl = _normalizeBaseUrl(baseUrl ?? const String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://10.0.2.2:8080/api/v1',
        ));

  final http.Client client;
  final TokenStore tokenStore;
  final String baseUrl;

  static String _normalizeBaseUrl(String value) => value.endsWith('/')
      ? value.substring(0, value.length - 1)
      : value;

  Future<dynamic> _request(String method, String path, {Object? body}) async {
    final token = await tokenStore.read();
    final headers = <String, String>{'Content-Type': 'application/json'};
    if (token != null && token.isNotEmpty) headers['Authorization'] = 'Bearer $token';
    final uri = Uri.parse('$baseUrl$path');
    final request = http.Request(method, uri)..headers.addAll(headers);
    if (body != null) request.body = jsonEncode(body);
    final response = await client.send(request);
    final text = await response.stream.bytesToString();
    dynamic decoded;
    if (text.isNotEmpty) decoded = jsonDecode(text);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message = decoded is Map ? decoded['message']?.toString() : null;
      throw ApiException(response.statusCode, message ?? 'Request failed (${response.statusCode})');
    }
    return decoded;
  }

  Future<void> login(String email, String password) async {
    final result = await _request('POST', '/auth/login', body: {'email': email, 'password': password});
    final token = result is Map ? (result['accessToken'] ?? result['token'])?.toString() : null;
    if (token == null || token.isEmpty) throw const ApiException(200, 'Login response did not include an access token');
    await tokenStore.write(token);
  }

  Future<List<Project>> projects() async => ((await _request('GET', '/projects')) as List)
      .map((item) => Project.fromJson(item as Map<String, dynamic>))
      .toList();

  Future<List<DiagramSummary>> diagrams(String projectId) async =>
      ((await _request('GET', '/projects/$projectId/diagrams')) as List)
          .map((item) => DiagramSummary.fromJson(item as Map<String, dynamic>))
          .toList();

  Future<UmlDocument> createDiagram(String projectId, UmlDocument document) async =>
      UmlDocument.fromJson(await _request('POST', '/projects/$projectId/diagrams', body: document.toJson()) as Map<String, dynamic>);

  Future<UmlDocument> diagram(String projectId, String diagramId) async =>
      UmlDocument.fromJson(await _request('GET', '/projects/$projectId/diagrams/$diagramId') as Map<String, dynamic>);

  Future<UmlDocument> updateDiagram(String projectId, String diagramId, UmlDocument document) async =>
      UmlDocument.fromJson(await _request('PUT', '/projects/$projectId/diagrams/$diagramId', body: document.toJson()) as Map<String, dynamic>);

  Future<List<DiagramVersion>> versions(String projectId, String diagramId) async =>
      ((await _request('GET', '/projects/$projectId/diagrams/$diagramId/versions')) as List)
          .map((item) => DiagramVersion.fromJson(item as Map<String, dynamic>))
          .toList();

  Future<UmlDocument> restore(String projectId, String diagramId, int version) async =>
      UmlDocument.fromJson(await _request('POST', '/projects/$projectId/diagrams/$diagramId/versions/$version/restore') as Map<String, dynamic>);
}
