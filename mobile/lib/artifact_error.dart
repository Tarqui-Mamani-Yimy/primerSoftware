import 'api.dart';
import 'strings.dart';

/// Returns the message shown when backend generation fails: the backend's own
/// explanation (e.g. a diagram validation error) when it sent one, otherwise
/// the generic failure text.
String artifactFailureMessage(Object error) {
  if (error is ApiException && error.message.trim().isNotEmpty) {
    return error.message.trim();
  }
  return AppStrings.backendFailed;
}
