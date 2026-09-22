import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_uml_architect_mobile/api.dart';
import 'package:ai_uml_architect_mobile/artifact_error.dart';
import 'package:ai_uml_architect_mobile/strings.dart';

void main() {
  group('artifactFailureMessage', () {
    test('surfaces the backend validation message', () {
      const error =
          ApiException(400, 'Relationship r1 has invalid target multiplicity');
      expect(artifactFailureMessage(error),
          'Relationship r1 has invalid target multiplicity');
    });

    test('falls back to the generic message for a blank backend message', () {
      const error = ApiException(500, '   ');
      expect(artifactFailureMessage(error), AppStrings.backendFailed);
    });

    test('falls back to the generic message for non-API errors', () {
      expect(artifactFailureMessage(const SocketException('offline')),
          AppStrings.backendFailed);
      expect(
          artifactFailureMessage(StateError('boom')), AppStrings.backendFailed);
    });
  });
}
