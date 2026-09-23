import 'package:ai_uml_architect_mobile/voice_command_help.dart';
import 'package:ai_uml_architect_mobile/voice_commands.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

/// Verifies one example phrase parses to the exact command shape the help
/// content promises, so the help panel cannot drift from the real grammar.
void checkExample(String phrase, VoiceCommandHelpExpectation expect) {
  final actual = parseVoiceCommand(phrase);
  if (actual == null) {
    fail('Expected a parsed command for "$phrase" but got null');
  }
  if (actual.kind != expect.kind) {
    fail(
        'Expected kind "${expect.kind}" for "$phrase" but got "${actual.kind}"');
  }
  if (expect.relationshipType != null &&
      actual.relationshipType != expect.relationshipType) {
    fail(
        'Expected relationship type "${expect.relationshipType}" for "$phrase"');
  }
  if (expect.sourceMultiplicity != null &&
      actual.sourceMultiplicity != expect.sourceMultiplicity) {
    fail(
        'Expected sourceMultiplicity "${expect.sourceMultiplicity}" for "$phrase"');
  }
  if (expect.targetMultiplicity != null &&
      actual.targetMultiplicity != expect.targetMultiplicity) {
    fail(
        'Expected targetMultiplicity "${expect.targetMultiplicity}" for "$phrase"');
  }
  if (expect.label != null && actual.label != expect.label) {
    fail('Expected label "${expect.label}" for "$phrase"');
  }
  if (expect.attributeType != null && actual.type != expect.attributeType) {
    fail('Expected attribute type "${expect.attributeType}" for "$phrase"');
  }
  if (expect.returnType != null && actual.returnType != expect.returnType) {
    fail('Expected return type "${expect.returnType}" for "$phrase"');
  }
}

void main() {
  test('every help example parses to the shape it promises', () {
    var exampleCount = 0;
    for (final group in voiceCommandHelpGroups) {
      expect(group.examples.length, greaterThanOrEqualTo(1));
      expect(group.examples.length, lessThanOrEqualTo(5));
      for (final example in group.examples) {
        checkExample(example.phrase, example.expect);
        exampleCount++;
      }
    }
    expect(exampleCount, greaterThan(0));
    expect(voiceCommandHelpNotes, isNotEmpty);
  });

  test('every relationship type word maps to the parser value', () {
    for (final entry in relationshipTypeVocabulary) {
      final command = parseVoiceCommand(
          'relacionar Cliente con Pedido como ${entry.phrase}');
      expect(command, isNotNull, reason: entry.phrase);
      expect(command!.kind, VoiceCommandKind.createRelationship);
      expect(command.relationshipType.toString().split('.').last, entry.value,
          reason: entry.phrase);
    }
  });

  test('every multiplicity phrase maps to the parser value', () {
    for (final entry in multiplicityVocabulary) {
      final command = parseVoiceCommand(
        'relacionar Cliente con Pedido con cardinalidad origen ${entry.phrase}',
      );
      expect(command, isNotNull, reason: entry.phrase);
      expect(command!.sourceMultiplicity, entry.value, reason: entry.phrase);
    }
  });

  test('every attribute type word maps to the parser value', () {
    for (final entry in attributeTypeVocabulary) {
      final command = parseVoiceCommand(
        'agregar atributo campo de tipo ${entry.phrase} a Clase',
      );
      expect(command, isNotNull, reason: entry.phrase);
      expect(command!.kind, VoiceCommandKind.addAttribute);
      expect(command.type, entry.value, reason: entry.phrase);
    }
  });

  testWidgets('tapping the help button shows the dialog with the group titles',
      (tester) async {
    await tester.pumpWidget(const MaterialApp(
      home: Scaffold(body: Center(child: VoiceCommandHelpButton())),
    ));

    expect(find.text('Ayuda'), findsOneWidget);
    await tester.tap(find.byKey(const Key('voice-command-help.button')));
    await tester.pumpAndSettle();

    for (final group in voiceCommandHelpGroups) {
      expect(find.text(group.title), findsOneWidget);
    }
  });
}
