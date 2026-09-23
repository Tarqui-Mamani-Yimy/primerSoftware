import 'package:flutter/material.dart';

import 'strings.dart';
import 'voice_commands.dart';

/// The subset of a parsed VoiceCommand that one help example promises to
/// produce. Field names are disambiguated from VoiceCommand's own shape
/// (`relationshipType` vs. `attributeType`) because "type" means a different
/// thing on a relationship command than on an attribute/method command.
class VoiceCommandHelpExpectation {
  const VoiceCommandHelpExpectation({
    required this.kind,
    this.relationshipType,
    this.sourceMultiplicity,
    this.targetMultiplicity,
    this.label,
    this.attributeType,
    this.returnType,
  });

  final VoiceCommandKind kind;
  final VoiceRelationshipType? relationshipType;
  final String? sourceMultiplicity;
  final String? targetMultiplicity;
  final String? label;
  final String? attributeType;
  final String? returnType;
}

class VoiceCommandHelpExample {
  const VoiceCommandHelpExample({
    required this.phrase,
    required this.note,
    required this.expect,
  });

  /// The exact phrase a user can say or type, as the parser expects it.
  final String phrase;

  /// One-line, user-facing explanation of what this phrase does.
  final String note;

  /// What parseVoiceCommand must return for this phrase; verified by tests.
  final VoiceCommandHelpExpectation expect;
}

/// One word/phrase in a small spoken vocabulary mapped to its parsed value.
class VoiceCommandVocabularyEntry {
  const VoiceCommandVocabularyEntry(
      {required this.phrase, required this.value});

  final String phrase;
  final String value;
}

class VoiceCommandHelpGroup {
  const VoiceCommandHelpGroup({
    required this.id,
    required this.title,
    required this.description,
    required this.examples,
    this.vocabulary,
  });

  final String id;
  final String title;
  final String description;
  final List<VoiceCommandHelpExample> examples;

  /// Optional reference table (relationship types, multiplicities, or attribute types).
  final List<VoiceCommandVocabularyEntry>? vocabulary;
}

// The six relationship types _parseRelationshipType() accepts in Spanish.
const relationshipTypeVocabulary = <VoiceCommandVocabularyEntry>[
  VoiceCommandVocabularyEntry(phrase: 'asociación', value: 'association'),
  VoiceCommandVocabularyEntry(phrase: 'agregación', value: 'aggregation'),
  VoiceCommandVocabularyEntry(phrase: 'composición', value: 'composition'),
  VoiceCommandVocabularyEntry(
      phrase: 'generalización', value: 'generalization'),
  VoiceCommandVocabularyEntry(phrase: 'realización', value: 'realization'),
  VoiceCommandVocabularyEntry(phrase: 'dependencia', value: 'dependency'),
];

// The spoken multiplicity vocabulary _cleanMultiplicity()/_spokenBound() accept.
const multiplicityVocabulary = <VoiceCommandVocabularyEntry>[
  VoiceCommandVocabularyEntry(phrase: 'cero', value: '0'),
  VoiceCommandVocabularyEntry(phrase: 'uno', value: '1'),
  VoiceCommandVocabularyEntry(phrase: 'dos', value: '2'),
  VoiceCommandVocabularyEntry(phrase: 'tres', value: '3'),
  VoiceCommandVocabularyEntry(phrase: 'cuatro', value: '4'),
  VoiceCommandVocabularyEntry(phrase: 'cinco', value: '5'),
  VoiceCommandVocabularyEntry(phrase: 'seis', value: '6'),
  VoiceCommandVocabularyEntry(phrase: 'siete', value: '7'),
  VoiceCommandVocabularyEntry(phrase: 'ocho', value: '8'),
  VoiceCommandVocabularyEntry(phrase: 'nueve', value: '9'),
  VoiceCommandVocabularyEntry(phrase: 'diez', value: '10'),
  VoiceCommandVocabularyEntry(phrase: 'muchos', value: '*'),
  VoiceCommandVocabularyEntry(phrase: 'varios', value: '*'),
  VoiceCommandVocabularyEntry(phrase: 'asterisco', value: '*'),
  VoiceCommandVocabularyEntry(phrase: 'estrella', value: '*'),
  VoiceCommandVocabularyEntry(phrase: 'n', value: '*'),
  VoiceCommandVocabularyEntry(phrase: 'cero a muchos', value: '0..*'),
  VoiceCommandVocabularyEntry(phrase: 'cero punto punto muchos', value: '0..*'),
  VoiceCommandVocabularyEntry(phrase: 'uno o más', value: '1..*'),
  VoiceCommandVocabularyEntry(phrase: 'cero o uno', value: '0..1'),
];

// The Spanish attribute/return-type vocabulary _mapSpanishType() accepts.
const attributeTypeVocabulary = <VoiceCommandVocabularyEntry>[
  VoiceCommandVocabularyEntry(phrase: 'entero', value: 'Integer'),
  VoiceCommandVocabularyEntry(phrase: 'texto', value: 'String'),
  VoiceCommandVocabularyEntry(phrase: 'cadena', value: 'String'),
  VoiceCommandVocabularyEntry(phrase: 'decimal', value: 'BigDecimal'),
  VoiceCommandVocabularyEntry(phrase: 'fecha', value: 'LocalDate'),
  VoiceCommandVocabularyEntry(phrase: 'fecha y hora', value: 'ZonedDateTime'),
  VoiceCommandVocabularyEntry(phrase: 'booleano', value: 'Boolean'),
  VoiceCommandVocabularyEntry(phrase: 'lógico', value: 'Boolean'),
  VoiceCommandVocabularyEntry(phrase: 'largo', value: 'Long'),
  VoiceCommandVocabularyEntry(phrase: 'flotante', value: 'Float'),
  VoiceCommandVocabularyEntry(phrase: 'doble', value: 'Double'),
  VoiceCommandVocabularyEntry(phrase: 'uuid', value: 'UUID'),
];

const voiceCommandHelpGroups = <VoiceCommandHelpGroup>[
  VoiceCommandHelpGroup(
    id: 'create-class',
    title: 'Crear una clase',
    description: 'Crea una clase nueva en el diagrama con el nombre indicado.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'crear clase Pedido',
        note: 'Crea la clase "Pedido".',
        expect: VoiceCommandHelpExpectation(kind: VoiceCommandKind.createClass),
      ),
      VoiceCommandHelpExample(
        phrase: 'crear una clase Detalle Pedido',
        note:
            'El artículo "una" es opcional y el nombre puede tener varias palabras.',
        expect: VoiceCommandHelpExpectation(kind: VoiceCommandKind.createClass),
      ),
    ],
  ),
  VoiceCommandHelpGroup(
    id: 'add-attribute',
    title: 'Agregar un atributo',
    description: 'Agrega un atributo con nombre y tipo a una clase existente.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'agregar atributo total de tipo decimal a Pedido',
        note: 'Agrega el atributo "total" (decimal) a la clase "Pedido".',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.addAttribute,
          attributeType: 'BigDecimal',
        ),
      ),
      VoiceCommandHelpExample(
        phrase: 'añadir atributo total de tipo decimal a Pedido',
        note:
            '"añadir" funciona igual que "agregar"; los acentos son opcionales.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.addAttribute,
          attributeType: 'BigDecimal',
        ),
      ),
    ],
  ),
  VoiceCommandHelpGroup(
    id: 'add-method',
    title: 'Agregar un método',
    description:
        'Agrega un método con nombre y tipo de retorno a una clase existente.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'agregar método calcularTotal de retorno decimal a Pedido',
        note:
            'Agrega el método "calcularTotal" con retorno decimal a "Pedido".',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.addMethod,
          returnType: 'BigDecimal',
        ),
      ),
    ],
  ),
  VoiceCommandHelpGroup(
    id: 'relate-classes',
    title: 'Relacionar dos clases',
    description:
        'Crea una relación entre dos clases existentes. Si Cliente y Pedido ya tienen una relación, el comando la actualiza en lugar de crear una nueva.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'relacionar Cliente con Pedido',
        note: 'Relación básica, sin tipo, cardinalidades ni etiqueta.',
        expect: VoiceCommandHelpExpectation(
            kind: VoiceCommandKind.createRelationship),
      ),
      VoiceCommandHelpExample(
        phrase: 'relacionar Cliente con Pedido como composición',
        note: 'Fija el tipo de relación con "como <tipo>".',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          relationshipType: VoiceRelationshipType.composition,
        ),
      ),
      VoiceCommandHelpExample(
        phrase:
            'relacionar Cliente con Pedido con cardinalidad origen uno y destino cero a muchos',
        note:
            'Fija las cardinalidades de origen y destino con "con cardinalidad".',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          sourceMultiplicity: '1',
          targetMultiplicity: '0..*',
        ),
      ),
      VoiceCommandHelpExample(
        phrase: 'relacionar Cliente con Pedido con verbo realiza',
        note: 'Agrega una etiqueta a la relación con "con verbo".',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          label: 'realiza',
        ),
      ),
      VoiceCommandHelpExample(
        phrase:
            'relacionar Cliente con Pedido como composición con cardinalidad origen uno y destino cero a muchos con verbo realiza',
        note: 'Combina tipo, cardinalidades y etiqueta en un solo comando.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          relationshipType: VoiceRelationshipType.composition,
          sourceMultiplicity: '1',
          targetMultiplicity: '0..*',
          label: 'realiza',
        ),
      ),
    ],
    vocabulary: relationshipTypeVocabulary,
  ),
  VoiceCommandHelpGroup(
    id: 'multiplicities',
    title: 'Multiplicidades habladas',
    description:
        'Las cardinalidades ("con cardinalidad origen/destino …") aceptan números en palabras y las formas "X a Y", "X punto punto Y", "uno o más" y "cero o uno".',
    examples: [
      VoiceCommandHelpExample(
        phrase:
            'relacionar Cliente con Pedido con cardinalidad origen cero a muchos',
        note: 'Cardinalidad de origen 0..*.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          sourceMultiplicity: '0..*',
        ),
      ),
      VoiceCommandHelpExample(
        phrase:
            'relacionar Cliente con Pedido con cardinalidad destino uno o más',
        note: 'Cardinalidad de destino 1..*.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.createRelationship,
          targetMultiplicity: '1..*',
        ),
      ),
    ],
    vocabulary: multiplicityVocabulary,
  ),
  VoiceCommandHelpGroup(
    id: 'types',
    title: 'Tipos en español',
    description:
        'El tipo de un atributo o el retorno de un método aceptan este vocabulario en español.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'agregar atributo edad de tipo entero a Cliente',
        note: 'El tipo "entero" se mapea a Integer.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.addAttribute,
          attributeType: 'Integer',
        ),
      ),
      VoiceCommandHelpExample(
        phrase: 'agregar atributo saldo de tipo decimal a Cliente',
        note: 'El tipo "decimal" se mapea a BigDecimal.',
        expect: VoiceCommandHelpExpectation(
          kind: VoiceCommandKind.addAttribute,
          attributeType: 'BigDecimal',
        ),
      ),
    ],
    vocabulary: attributeTypeVocabulary,
  ),
  VoiceCommandHelpGroup(
    id: 'undo',
    title: 'Deshacer',
    description:
        'Restaura el diagrama al estado previo al último comando de voz confirmado.',
    examples: [
      VoiceCommandHelpExample(
        phrase: 'deshacer',
        note: 'Deshace el último comando de voz confirmado.',
        expect: VoiceCommandHelpExpectation(
            kind: VoiceCommandKind.undoVoiceCommand),
      ),
      VoiceCommandHelpExample(
        phrase: 'deshacer comando de voz',
        note: 'Forma equivalente a "deshacer".',
        expect: VoiceCommandHelpExpectation(
            kind: VoiceCommandKind.undoVoiceCommand),
      ),
      VoiceCommandHelpExample(
        phrase: 'deshacer último comando de voz',
        note: 'Forma equivalente a "deshacer".',
        expect: VoiceCommandHelpExpectation(
            kind: VoiceCommandKind.undoVoiceCommand),
      ),
    ],
  ),
];

const voiceCommandHelpNotes = <String>[
  'Cada comando muestra una vista previa y debe confirmarse antes de aplicarse al diagrama.',
  'Los parámetros de los métodos no se pueden dictar por voz; agregalos manualmente después de confirmar el comando.',
  'Las comas y otros signos de puntuación de la transcripción se ignoran al interpretar el comando.',
];

/// Icon button that opens the voice command help sheet. Kept as its own
/// widget so it can be placed next to the mic control and tested in
/// isolation from the rest of the workspace screen.
class VoiceCommandHelpButton extends StatelessWidget {
  const VoiceCommandHelpButton({super.key});

  @override
  Widget build(BuildContext context) {
    return TextButton.icon(
      key: const Key('voice-command-help.button'),
      onPressed: () => showVoiceCommandHelp(context),
      icon: const Icon(Icons.help_outline),
      label: const Text(AppStrings.voiceCommandHelpButton),
    );
  }
}

/// Shows a scrollable bottom sheet listing every voice command the app
/// accepts, grouped by intent, derived from [voiceCommandHelpGroups].
Future<void> showVoiceCommandHelp(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    builder: (context) => DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.85,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      builder: (context, scrollController) => SafeArea(
        child: SingleChildScrollView(
          controller: scrollController,
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(AppStrings.voiceCommandHelpTitle,
                  style: Theme.of(context).textTheme.titleLarge),
              for (final group in voiceCommandHelpGroups) ...[
                const SizedBox(height: 16),
                Text(group.title,
                    style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 4),
                Text(group.description),
                const SizedBox(height: 4),
                for (final example in group.examples)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: RichText(
                      text: TextSpan(
                        style: DefaultTextStyle.of(context).style,
                        children: [
                          TextSpan(
                            text: '“${example.phrase}” ',
                            style: const TextStyle(fontWeight: FontWeight.bold),
                          ),
                          TextSpan(text: example.note),
                        ],
                      ),
                    ),
                  ),
                if (group.vocabulary != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Wrap(
                      spacing: 8,
                      runSpacing: 4,
                      children: [
                        for (final entry in group.vocabulary!)
                          Chip(label: Text('${entry.phrase} → ${entry.value}')),
                      ],
                    ),
                  ),
              ],
              const SizedBox(height: 16),
              Text(AppStrings.voiceCommandHelpNotesTitle,
                  style: Theme.of(context).textTheme.titleMedium),
              for (final note in voiceCommandHelpNotes)
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text('• $note'),
                ),
            ],
          ),
        ),
      ),
    ),
  );
}
