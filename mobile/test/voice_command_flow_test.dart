import 'package:ai_uml_architect_mobile/models.dart';
import 'package:ai_uml_architect_mobile/voice_command_flow.dart';
import 'package:ai_uml_architect_mobile/voice_commands.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('preview creates a detached class and confirmation candidate', () {
    final original = _document();
    final command = parseVoiceCommand('crear una clase Factura')!;
    final preview = previewVoiceCommand(original, command);

    expect(preview, isNotNull);
    expect(
      preview!.document.classes.map((item) => item.name),
      contains('Factura'),
    );
    expect(original.classes, hasLength(1));
  });

  test('preview rejects unknown or ambiguous targets without mutation', () {
    final original = _document(
      classes: [
        UmlClass(id: 'user-1', name: 'User'),
        UmlClass(id: 'user-2', name: 'User'),
      ],
    );
    final command = parseVoiceCommand(
      'agregar atributo correo de tipo String a User',
    )!;

    expect(previewVoiceCommand(original, command), isNull);
    expect(original.classes, hasLength(2));
  });

  test('preview supports relationship commands and preserves input', () {
    final original = _document(
      classes: [
        UmlClass(id: 'user', name: 'User'),
        UmlClass(id: 'order', name: 'Order'),
      ],
    );
    final command = parseVoiceCommand(
      'relacionar User con Order como association '
      'con cardinalidad origen 1 y destino 0..*',
    )!;
    final before = original.toJson();
    final preview = previewVoiceCommand(original, command);

    expect(preview, isNotNull);
    expect(preview!.document.relationships.single.type, 'association');
    expect(original.toJson(), equals(before));
  });

  test('manual or remote invalidation blocks a pending regular preview', () {
    final preview = previewVoiceCommand(
      _document(),
      parseVoiceCommand('crear una clase Factura')!,
    );

    expect(canConfirmVoicePreview(preview, hasConflict: false), isTrue);
    final invalidated = invalidateVoicePreview(preview);
    expect(invalidated, isNull);
    expect(canConfirmVoicePreview(invalidated, hasConflict: false), isFalse);
    expect(canConfirmVoicePreview(preview, hasConflict: true), isFalse);
  });

  test('manual or remote invalidation blocks a pending undo preview', () {
    final snapshot = _document(
      classes: [
        UmlClass(id: 'user', name: 'User'),
        UmlClass(id: 'invoice', name: 'Invoice'),
      ],
    );
    final undo = VoiceMutationPreview(
      command: parseVoiceCommand('deshacer')!,
      document: snapshot,
    );

    expect(canConfirmVoicePreview(undo, hasConflict: false), isTrue);
    expect(invalidateVoicePreview(undo), isNull);
    expect(canConfirmVoicePreview(undo, hasConflict: true), isFalse);
  });
}

UmlDocument _document({List<UmlClass>? classes}) => UmlDocument(
      id: 'diagram-1',
      name: 'Orders',
      classes: classes ?? [UmlClass(id: 'user', name: 'User')],
    );
