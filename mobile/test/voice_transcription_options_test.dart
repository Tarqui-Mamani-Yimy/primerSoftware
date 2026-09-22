import 'package:ai_uml_architect_mobile/voice_transcription_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('transcribes voice commands as Spanish without translating', () {
    const options = VoiceTranscriptionService.transcribeOptions;
    expect(options.language, 'es');
    expect(options.translate, isFalse);
  });

  test('primes Whisper with the voice command vocabulary', () {
    final prompt = VoiceTranscriptionService.transcribeOptions.initialPrompt;
    expect(prompt, isNotNull);
    for (final word in [
      'crear clase',
      'relacionar',
      'cardinalidad',
      'origen',
      'destino',
      'método',
      'atributo',
      'composición',
    ]) {
      expect(prompt, contains(word));
    }
  });
}
