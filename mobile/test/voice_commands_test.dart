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
      parseVoiceCommand('relacionar A con B con cardinalidad origen veinte'),
      isNull,
    );
    expect(
      parseVoiceCommand('relacionar A con B con verbo etiqueta con verbo otra'),
      isNull,
    );
    expect(
      parseVoiceCommand('relacionar A con B con cardinalidad origen 1+'),
      isNull,
    );
  });

  test('normalizes smart_format punctuation before parsing (VOICE-02)', () {
    expect(
      parseVoiceCommand('Relacionar Cliente con Pedido, como composición.')
          ?.relationshipType,
      VoiceRelationshipType.composition,
    );
    expect(
      parseVoiceCommand('¿Crear una clase Cliente?')?.name,
      'Cliente',
    );
    expect(
      parseVoiceCommand('¡Crear una clase Cliente!')?.name,
      'Cliente',
    );
    expect(
      parseVoiceCommand('Crear una clase "Cliente"')?.name,
      'Cliente',
    );
    expect(
      parseVoiceCommand('Crear una clase «Cliente»')?.name,
      'Cliente',
    );
    expect(
      parseVoiceCommand('agregar atributo email: de tipo String a Usuario')
          ?.type,
      'String',
    );
  });

  test('parses spoken Spanish multiplicities (VOICE-02)', () {
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad origen uno')
          ?.sourceMultiplicity,
      '1',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad origen cero')
          ?.sourceMultiplicity,
      '0',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad destino muchos')
          ?.targetMultiplicity,
      '*',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad destino varios')
          ?.targetMultiplicity,
      '*',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad destino asterisco')
          ?.targetMultiplicity,
      '*',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad destino estrella')
          ?.targetMultiplicity,
      '*',
    );
    expect(
      parseVoiceCommand('relacionar Usuario con Pedido con cardinalidad destino n')
          ?.targetMultiplicity,
      '*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen cero a muchos',
      )?.sourceMultiplicity,
      '0..*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen de cero a muchos',
      )?.sourceMultiplicity,
      '0..*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen uno a diez',
      )?.sourceMultiplicity,
      '1..10',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen cero punto punto muchos',
      )?.sourceMultiplicity,
      '0..*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen uno o más',
      )?.sourceMultiplicity,
      '1..*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen uno o mas',
      )?.sourceMultiplicity,
      '1..*',
    );
    expect(
      parseVoiceCommand(
        'relacionar Usuario con Pedido con cardinalidad origen cero o uno',
      )?.sourceMultiplicity,
      '0..1',
    );
  });

  test('parses the origen/destino y separator without colliding with spoken multiplicities (VOICE-02)', () {
    final a = parseVoiceCommand(
      'relacionar Usuario con Pedido con cardinalidad origen uno y destino cero a muchos',
    );
    expect(a?.sourceMultiplicity, '1');
    expect(a?.targetMultiplicity, '0..*');

    final b = parseVoiceCommand(
      'relacionar Usuario con Pedido con cardinalidad destino muchos y origen uno',
    );
    expect(b?.sourceMultiplicity, '1');
    expect(b?.targetMultiplicity, '*');

    final c = parseVoiceCommand(
      'relacionar Usuario con Pedido con cardinalidad origen cero o uno',
    );
    expect(c?.sourceMultiplicity, '0..1');
  });

  test('parses the acceptance transcript from a real Deepgram smart_format output (VOICE-02)', () {
    final command = parseVoiceCommand(
      'Relacionar Cliente con Pedido, con cardinalidad origen uno y destino cero a muchos.',
    );
    expect(command?.sourceName, 'Cliente');
    expect(command?.targetName, 'Pedido');
    expect(command?.sourceMultiplicity, '1');
    expect(command?.targetMultiplicity, '0..*');
  });

  test('maps Spanish attribute/method type vocabulary (VOICE-03)', () {
    expect(
      parseVoiceCommand('agregar atributo edad de tipo entero a Cliente')
          ?.type,
      'Integer',
    );
    expect(
      parseVoiceCommand('agregar atributo edad de tipo ENTERO a Cliente')
          ?.type,
      'Integer',
    );
    expect(
      parseVoiceCommand('agregar atributo nombre de tipo texto a Cliente')
          ?.type,
      'String',
    );
    expect(
      parseVoiceCommand('agregar atributo nombre de tipo cadena a Cliente')
          ?.type,
      'String',
    );
    expect(
      parseVoiceCommand('agregar atributo saldo de tipo decimal a Cliente')
          ?.type,
      'BigDecimal',
    );
    expect(
      parseVoiceCommand('agregar atributo nacimiento de tipo fecha a Cliente')
          ?.type,
      'LocalDate',
    );
    expect(
      parseVoiceCommand(
        'agregar atributo registro de tipo fecha hora a Cliente',
      )?.type,
      'ZonedDateTime',
    );
    expect(
      parseVoiceCommand(
        'agregar atributo registro de tipo fecha y hora a Cliente',
      )?.type,
      'ZonedDateTime',
    );
    expect(
      parseVoiceCommand('agregar atributo activo de tipo booleano a Cliente')
          ?.type,
      'Boolean',
    );
    expect(
      parseVoiceCommand('agregar atributo activo de tipo lógico a Cliente')
          ?.type,
      'Boolean',
    );
    expect(
      parseVoiceCommand('agregar atributo activo de tipo logico a Cliente')
          ?.type,
      'Boolean',
    );
    expect(
      parseVoiceCommand('agregar atributo codigo de tipo largo a Cliente')
          ?.type,
      'Long',
    );
    expect(
      parseVoiceCommand('agregar atributo peso de tipo flotante a Cliente')
          ?.type,
      'Float',
    );
    expect(
      parseVoiceCommand('agregar atributo altura de tipo doble a Cliente')
          ?.type,
      'Double',
    );
    expect(
      parseVoiceCommand('agregar atributo externo de tipo uuid a Cliente')
          ?.type,
      'UUID',
    );
    expect(
      parseVoiceCommand(
        'añadir método calcularEdad de retorno entero a Cliente',
      )?.returnType,
      'Integer',
    );
    // Unknown/English types still pass through unchanged.
    expect(
      parseVoiceCommand(
        'agregar atributo email de tipo String a Usuario',
      )?.type,
      'String',
    );
  });
}
