import 'package:ai_uml_architect_mobile/voice_commands.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses class, attribute, method, and undo commands', () {
    expect(parseVoiceCommand('Crear una clase Usuario.')?.name, 'Usuario');
    final attribute = parseVoiceCommand(
      'añadir atributo correo de tipo String a Usuario',
    );
    expect(attribute?.kind, VoiceCommandKind.addAttribute);
    expect(attribute?.type, 'String');
    expect(attribute?.className, 'Usuario');

    final method = parseVoiceCommand(
      'agregar método validar de retorno boolean a Usuario',
    );
    expect(method?.kind, VoiceCommandKind.addMethod);
    expect(method?.returnType, 'boolean');
    expect(
      parseVoiceCommand('deshacer')?.kind,
      VoiceCommandKind.undoVoiceCommand,
    );
  });

  test('parses relationship type, cardinalities, and label', () {
    final command = parseVoiceCommand(
      'relacionar Usuario con Pedido como composición '
      'con cardinalidad origen 1 y destino 0..* con verbo contiene',
    );

    expect(command?.kind, VoiceCommandKind.createRelationship);
    expect(command?.sourceName, 'Usuario');
    expect(command?.targetName, 'Pedido');
    expect(command?.relationshipType, VoiceRelationshipType.composition);
    expect(command?.sourceMultiplicity, '1');
    expect(command?.targetMultiplicity, '0..*');
    expect(command?.label, 'contiene');
  });

  test('accepts reversed cardinality order and accent variants', () {
    final command = parseVoiceCommand(
      'relacionar Niño con Escuela como generalizacion '
      'con cardinalidad destino 1..* y cardinalidad origen 0..1',
    );

    expect(command?.relationshipType, VoiceRelationshipType.generalization);
    expect(command?.sourceMultiplicity, '0..1');
    expect(command?.targetMultiplicity, '1..*');
  });

  test('rejects ambiguous, unsupported, duplicated, and invalid grammar', () {
    expect(parseVoiceCommand('crear una clase'), isNull);
    expect(
      parseVoiceCommand(
        'agregar atributo id de tipo String a Usuario parámetros',
      ),
      isNull,
    );
    expect(parseVoiceCommand('relacionar A con B como desconocida'), isNull);
    expect(
      parseVoiceCommand('relacionar A con B con cardinalidad origen muchos'),
      isNull,
    );
    expect(
      parseVoiceCommand('relacionar A con B con verbo etiqueta con verbo otra'),
      isNull,
    );
  });
}
