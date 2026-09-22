import 'package:ai_uml_architect_mobile/voice_command_preview.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';

void main() {
  testWidgets('preview exposes explicit confirm and cancel actions',
      (tester) async {
    var confirmed = false;
    var cancelled = false;
    await tester.pumpWidget(
      MaterialApp(
        home: VoiceCommandPreviewPanel(
          transcript: 'crear una clase Factura',
          description: 'Crear la clase Factura',
          onConfirm: () => confirmed = true,
          onCancel: () => cancelled = true,
        ),
      ),
    );

    expect(find.byKey(const Key('workspace.voice.preview')), findsOneWidget);
    await tester.tap(find.byKey(const Key('workspace.voice.cancel')));
    await tester.tap(find.byKey(const Key('workspace.voice.confirm')));
    expect(cancelled, isTrue);
    expect(confirmed, isTrue);
  });
}
