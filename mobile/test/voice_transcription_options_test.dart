import 'package:ai_uml_architect_mobile/voice_transcription_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('transcribes voice commands as Spanish without translating', () {
    const options = VoiceTranscriptionService.transcribeOptions;
    expect(options.language, 'es');
    expect(options.translate, isFalse);
  });
}
