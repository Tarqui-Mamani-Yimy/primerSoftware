import { RelationshipType } from '../types';
import { VoiceCommand } from './voiceCommands';

/**
 * The subset of a parsed VoiceCommand that one help example promises to
 * produce. Field names are disambiguated from VoiceCommand's own shape
 * (`relationshipType` vs. `attributeType`) because "type" means a different
 * thing on a relationship command than on an attribute/method command.
 */
export interface VoiceCommandHelpExpectation {
  kind: VoiceCommand['kind'];
  relationshipType?: RelationshipType;
  sourceMultiplicity?: string;
  targetMultiplicity?: string;
  label?: string;
  attributeType?: string;
  returnType?: string;
}

export interface VoiceCommandHelpExample {
  /** The exact phrase a user can say or type, as the parser expects it. */
  phrase: string;
  /** One-line, user-facing explanation of what this phrase does. */
  note: string;
  /** What parseVoiceCommand must return for this phrase; verified by a smoke test. */
  expect: VoiceCommandHelpExpectation;
}

/** One word/phrase in a small spoken vocabulary mapped to its parsed value. */
export interface VoiceCommandVocabularyEntry {
  phrase: string;
  value: string;
}

export interface VoiceCommandHelpGroup {
  id: string;
  title: string;
  description: string;
  examples: VoiceCommandHelpExample[];
  /** Optional reference table (relationship types, multiplicities, or attribute types). */
  vocabulary?: VoiceCommandVocabularyEntry[];
}

// The six relationship types parseRelationshipType() accepts in Spanish.
export const relationshipTypeVocabulary: VoiceCommandVocabularyEntry[] = [
  { phrase: 'asociación', value: 'association' satisfies RelationshipType },
  { phrase: 'agregación', value: 'aggregation' satisfies RelationshipType },
  { phrase: 'composición', value: 'composition' satisfies RelationshipType },
  { phrase: 'generalización', value: 'generalization' satisfies RelationshipType },
  { phrase: 'realización', value: 'realization' satisfies RelationshipType },
  { phrase: 'dependencia', value: 'dependency' satisfies RelationshipType },
];

// The spoken multiplicity vocabulary cleanMultiplicity()/spokenBound() accept.
export const multiplicityVocabulary: VoiceCommandVocabularyEntry[] = [
  { phrase: 'cero', value: '0' },
  { phrase: 'uno', value: '1' },
  { phrase: 'dos', value: '2' },
  { phrase: 'tres', value: '3' },
  { phrase: 'cuatro', value: '4' },
  { phrase: 'cinco', value: '5' },
  { phrase: 'seis', value: '6' },
  { phrase: 'siete', value: '7' },
  { phrase: 'ocho', value: '8' },
  { phrase: 'nueve', value: '9' },
  { phrase: 'diez', value: '10' },
  { phrase: 'muchos', value: '*' },
  { phrase: 'varios', value: '*' },
  { phrase: 'asterisco', value: '*' },
  { phrase: 'estrella', value: '*' },
  { phrase: 'n', value: '*' },
  { phrase: 'cero a muchos', value: '0..*' },
  { phrase: 'cero punto punto muchos', value: '0..*' },
  { phrase: 'uno o más', value: '1..*' },
  { phrase: 'cero o uno', value: '0..1' },
];

// The Spanish attribute/return-type vocabulary mapSpanishType() accepts.
export const attributeTypeVocabulary: VoiceCommandVocabularyEntry[] = [
  { phrase: 'entero', value: 'Integer' },
  { phrase: 'texto', value: 'String' },
  { phrase: 'cadena', value: 'String' },
  { phrase: 'decimal', value: 'BigDecimal' },
  { phrase: 'fecha', value: 'LocalDate' },
  { phrase: 'fecha y hora', value: 'ZonedDateTime' },
  { phrase: 'booleano', value: 'Boolean' },
  { phrase: 'lógico', value: 'Boolean' },
  { phrase: 'largo', value: 'Long' },
  { phrase: 'flotante', value: 'Float' },
  { phrase: 'doble', value: 'Double' },
  { phrase: 'uuid', value: 'UUID' },
];

export const voiceCommandHelpGroups: VoiceCommandHelpGroup[] = [
  {
    id: 'create-class',
    title: 'Crear una clase',
    description: 'Crea una clase nueva en el diagrama con el nombre indicado.',
    examples: [
      {
        phrase: 'crear clase Pedido',
        note: 'Crea la clase "Pedido".',
        expect: { kind: 'create-class' },
      },
      {
        phrase: 'crear una clase Detalle Pedido',
        note: 'El artículo "una" es opcional y el nombre puede tener varias palabras.',
        expect: { kind: 'create-class' },
      },
    ],
  },
  {
    id: 'add-attribute',
    title: 'Agregar un atributo',
    description: 'Agrega un atributo con nombre y tipo a una clase existente.',
    examples: [
      {
        phrase: 'agregar atributo total de tipo decimal a Pedido',
        note: 'Agrega el atributo "total" (decimal) a la clase "Pedido".',
        expect: { kind: 'add-attribute', attributeType: 'BigDecimal' },
      },
      {
        phrase: 'añadir atributo total de tipo decimal a Pedido',
        note: '"añadir" funciona igual que "agregar"; los acentos son opcionales.',
        expect: { kind: 'add-attribute', attributeType: 'BigDecimal' },
      },
    ],
  },
  {
    id: 'add-method',
    title: 'Agregar un método',
    description: 'Agrega un método con nombre y tipo de retorno a una clase existente.',
    examples: [
      {
        phrase: 'agregar método calcularTotal de retorno decimal a Pedido',
        note: 'Agrega el método "calcularTotal" con retorno decimal a "Pedido".',
        expect: { kind: 'add-method', returnType: 'BigDecimal' },
      },
    ],
  },
  {
    id: 'relate-classes',
    title: 'Relacionar dos clases',
    description:
      'Crea una relación entre dos clases existentes. Si Cliente y Pedido ya tienen una relación, el comando la actualiza en lugar de crear una nueva.',
    examples: [
      {
        phrase: 'relacionar Cliente con Pedido',
        note: 'Relación básica, sin tipo, cardinalidades ni etiqueta.',
        expect: { kind: 'create-relationship' },
      },
      {
        phrase: 'relacionar Cliente con Pedido como composición',
        note: 'Fija el tipo de relación con "como <tipo>".',
        expect: { kind: 'create-relationship', relationshipType: 'composition' },
      },
      {
        phrase: 'relacionar Cliente con Pedido con cardinalidad origen uno y destino cero a muchos',
        note: 'Fija las cardinalidades de origen y destino con "con cardinalidad".',
        expect: { kind: 'create-relationship', sourceMultiplicity: '1', targetMultiplicity: '0..*' },
      },
      {
        phrase: 'relacionar Cliente con Pedido con verbo realiza',
        note: 'Agrega una etiqueta a la relación con "con verbo".',
        expect: { kind: 'create-relationship', label: 'realiza' },
      },
      {
        phrase:
          'relacionar Cliente con Pedido como composición con cardinalidad origen uno y destino cero a muchos con verbo realiza',
        note: 'Combina tipo, cardinalidades y etiqueta en un solo comando.',
        expect: {
          kind: 'create-relationship',
          relationshipType: 'composition',
          sourceMultiplicity: '1',
          targetMultiplicity: '0..*',
          label: 'realiza',
        },
      },
    ],
    vocabulary: relationshipTypeVocabulary,
  },
  {
    id: 'multiplicities',
    title: 'Multiplicidades habladas',
    description:
      'Las cardinalidades ("con cardinalidad origen/destino …") aceptan números en palabras y las formas "X a Y", "X punto punto Y", "uno o más" y "cero o uno".',
    examples: [
      {
        phrase: 'relacionar Cliente con Pedido con cardinalidad origen cero a muchos',
        note: 'Cardinalidad de origen 0..*.',
        expect: { kind: 'create-relationship', sourceMultiplicity: '0..*' },
      },
      {
        phrase: 'relacionar Cliente con Pedido con cardinalidad destino uno o más',
        note: 'Cardinalidad de destino 1..*.',
        expect: { kind: 'create-relationship', targetMultiplicity: '1..*' },
      },
    ],
    vocabulary: multiplicityVocabulary,
  },
  {
    id: 'types',
    title: 'Tipos en español',
    description: 'El tipo de un atributo o el retorno de un método aceptan este vocabulario en español.',
    examples: [
      {
        phrase: 'agregar atributo edad de tipo entero a Cliente',
        note: 'El tipo "entero" se mapea a Integer.',
        expect: { kind: 'add-attribute', attributeType: 'Integer' },
      },
      {
        phrase: 'agregar atributo saldo de tipo decimal a Cliente',
        note: 'El tipo "decimal" se mapea a BigDecimal.',
        expect: { kind: 'add-attribute', attributeType: 'BigDecimal' },
      },
    ],
    vocabulary: attributeTypeVocabulary,
  },
  {
    id: 'undo',
    title: 'Deshacer',
    description: 'Restaura el diagrama al estado previo al último comando de voz confirmado.',
    examples: [
      {
        phrase: 'deshacer',
        note: 'Deshace el último comando de voz confirmado.',
        expect: { kind: 'undo-voice-command' },
      },
      {
        phrase: 'deshacer comando de voz',
        note: 'Forma equivalente a "deshacer".',
        expect: { kind: 'undo-voice-command' },
      },
      {
        phrase: 'deshacer último comando de voz',
        note: 'Forma equivalente a "deshacer".',
        expect: { kind: 'undo-voice-command' },
      },
    ],
  },
];

export const voiceCommandHelpNotes: string[] = [
  'Cada comando muestra una vista previa y debe confirmarse antes de aplicarse al diagrama.',
  'Los parámetros de los métodos no se pueden dictar por voz; agregalos manualmente después de confirmar el comando.',
  'Las comas y otros signos de puntuación de la transcripción se ignoran al interpretar el comando.',
];
