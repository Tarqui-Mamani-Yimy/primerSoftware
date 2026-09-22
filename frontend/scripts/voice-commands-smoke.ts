import { parseVoiceCommand } from '../src/diagram/voiceCommands';

const cases: Array<[string, unknown]> = [
  // Undo commands
  ['deshacer', { kind: 'undo-voice-command' }],
  ['deshacer comando de voz', { kind: 'undo-voice-command' }],
  ['deshacer último comando de voz', { kind: 'undo-voice-command' }],

  // Attribute and method commands
  ['agregar atributo email de tipo String a Usuario', { kind: 'add-attribute', name: 'email', type: 'String', className: 'Usuario' }],
  ['añadir método validar de retorno Boolean a Usuario', { kind: 'add-method', name: 'validar', returnType: 'Boolean', className: 'Usuario' }],
  ['agregar atributo email a Usuario', null],
  ['agregar método validar de retorno Boolean a Usuario con parámetro id', null],

  // Relationship commands - baseline (VC-03)
  ['relacionar Usuario con Pedido', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido' }],

  // Relationship commands - full VC-06 example from specification
  ['relacionar Usuario con Pedido como composición con cardinalidad origen 1 y cardinalidad destino 0..* con verbo realiza', {
    kind: 'create-relationship',
    sourceName: 'Usuario',
    targetName: 'Pedido',
    type: 'composition',
    sourceMultiplicity: '1',
    targetMultiplicity: '0..*',
    label: 'realiza',
  }],

  // Relationship commands - all supported types in Spanish & English
  ['relacionar Usuario con Pedido como asociación', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'association' }],
  ['relacionar Usuario con Pedido como asociacion', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'association' }],
  ['relacionar Usuario con Pedido como agregación', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'aggregation' }],
  ['relacionar Usuario con Pedido como agregacion', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'aggregation' }],
  ['relacionar Usuario con Pedido como composición', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'composition' }],
  ['relacionar Usuario con Pedido como composicion', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'composition' }],
  ['relacionar Usuario con Pedido como generalización', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'generalization' }],
  ['relacionar Usuario con Pedido como generalizacion', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'generalization' }],
  ['relacionar Usuario con Pedido como realización', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'realization' }],
  ['relacionar Usuario con Pedido como realizacion', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'realization' }],
  ['relacionar Usuario con Pedido como dependencia', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'dependency' }],
  ['relacionar Usuario con Pedido como association', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'association' }],
  ['relacionar Usuario con Pedido como aggregation', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'aggregation' }],
  ['relacionar Usuario con Pedido como composition', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'composition' }],
  ['relacionar Usuario con Pedido como generalization', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'generalization' }],
  ['relacionar Usuario con Pedido como realization', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'realization' }],
  ['relacionar Usuario con Pedido como dependency', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'dependency' }],

  // Relationship commands - source and target cardinalities
  ['relacionar Usuario con Pedido con cardinalidad origen 1', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1' }],
  ['relacionar Usuario con Pedido con cardinalidad destino 0..*', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '0..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen 1 y destino 1..*', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1', targetMultiplicity: '1..*' }],
  ['relacionar Usuario con Pedido con cardinalidad destino * y cardinalidad origen 0..1', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..1', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen 1 con cardinalidad destino *', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1', targetMultiplicity: '*' }],

  // Relationship commands - verb label
  ['relacionar Usuario con Pedido con verbo realiza', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', label: 'realiza' }],
  ['relacionar Usuario con Pedido como agregación con verbo contiene elementos', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', type: 'aggregation', label: 'contiene elementos' }],

  // Relationship commands - multi-word class names
  ['relacionar Cuenta Bancaria con Cliente Preferencial como agregación con cardinalidad origen 1 y cardinalidad destino 1..* con verbo posee', {
    kind: 'create-relationship',
    sourceName: 'Cuenta Bancaria',
    targetName: 'Cliente Preferencial',
    type: 'aggregation',
    sourceMultiplicity: '1',
    targetMultiplicity: '1..*',
    label: 'posee',
  }],

  // Relationship commands - invalid inputs / rejections (must return null)
  ['relacionar Usuario con Pedido como herencia', null],
  ['relacionar Usuario con Pedido como desconocido', null],
  ['relacionar Usuario con Pedido con cardinalidad origen veinte', null],
  ['relacionar Usuario con Pedido con cardinalidad destino 1..2..3', null],
  ['relacionar Usuario con como composición', null],
  ['relacionar con Pedido como composición', null],
  ['relacionar con', null],
  ['relacionar Usuario con Pedido con verbo', null],
  ['relacionar Usuario con Pedido como composición como agregación', null],
  ['relacionar Usuario con Pedido con verbo crea con verbo elimina', null],
  ['relacionar Usuario con Pedido como composición basura al final', null],
  ['relacionar Usuario con Pedido con cardinalidad origen 1 con cardinalidad origen 2', null],

  // Transcript normalization (VOICE-02): smart_format punctuation and quotes
  // must not break parsing.
  ['Relacionar Cliente con Pedido, como composición.', {
    kind: 'create-relationship', sourceName: 'Cliente', targetName: 'Pedido', type: 'composition',
  }],
  ['¿Crear una clase Cliente?', { kind: 'create-class', name: 'Cliente' }],
  ['¡Crear una clase Cliente!', { kind: 'create-class', name: 'Cliente' }],
  ['Crear una clase "Cliente"', { kind: 'create-class', name: 'Cliente' }],
  ['Crear una clase «Cliente»', { kind: 'create-class', name: 'Cliente' }],
  ['agregar atributo email: de tipo String a Usuario', { kind: 'add-attribute', name: 'email', type: 'String', className: 'Usuario' }],

  // Spoken multiplicities (VOICE-02).
  ['relacionar Usuario con Pedido con cardinalidad origen uno', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1' }],
  ['relacionar Usuario con Pedido con cardinalidad origen cero', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0' }],
  ['relacionar Usuario con Pedido con cardinalidad destino muchos', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad destino varios', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad destino asterisco', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad destino estrella', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad destino n', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', targetMultiplicity: '*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen cero a muchos', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen de cero a muchos', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen uno a diez', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1..10' }],
  ['relacionar Usuario con Pedido con cardinalidad origen cero punto punto muchos', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen 1..5', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1..5' }],
  ['relacionar Usuario con Pedido con cardinalidad origen uno o más', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen uno o mas', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1..*' }],
  ['relacionar Usuario con Pedido con cardinalidad origen cero o uno', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..1' }],
  ['relacionar Usuario con Pedido con cardinalidad origen uno y destino cero a muchos', {
    kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1', targetMultiplicity: '0..*',
  }],
  ['relacionar Usuario con Pedido con cardinalidad destino muchos y origen uno', {
    kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '1', targetMultiplicity: '*',
  }],
  ['relacionar Usuario con Pedido con cardinalidad origen cero o uno', { kind: 'create-relationship', sourceName: 'Usuario', targetName: 'Pedido', sourceMultiplicity: '0..1' }],

  // Acceptance transcript (VOICE-02): a real Deepgram smart_format transcript
  // with a comma and spoken multiplicities.
  ['Relacionar Cliente con Pedido, con cardinalidad origen uno y destino cero a muchos.', {
    kind: 'create-relationship', sourceName: 'Cliente', targetName: 'Pedido', sourceMultiplicity: '1', targetMultiplicity: '0..*',
  }],

  // Spanish attribute/method type aliases (VOICE-03).
  ['agregar atributo edad de tipo entero a Cliente', { kind: 'add-attribute', name: 'edad', type: 'Integer', className: 'Cliente' }],
  ['agregar atributo nombre de tipo texto a Cliente', { kind: 'add-attribute', name: 'nombre', type: 'String', className: 'Cliente' }],
  ['agregar atributo nombre de tipo cadena a Cliente', { kind: 'add-attribute', name: 'nombre', type: 'String', className: 'Cliente' }],
  ['agregar atributo saldo de tipo decimal a Cliente', { kind: 'add-attribute', name: 'saldo', type: 'BigDecimal', className: 'Cliente' }],
  ['agregar atributo nacimiento de tipo fecha a Cliente', { kind: 'add-attribute', name: 'nacimiento', type: 'LocalDate', className: 'Cliente' }],
  ['agregar atributo registro de tipo fecha hora a Cliente', { kind: 'add-attribute', name: 'registro', type: 'ZonedDateTime', className: 'Cliente' }],
  ['agregar atributo registro de tipo fecha y hora a Cliente', { kind: 'add-attribute', name: 'registro', type: 'ZonedDateTime', className: 'Cliente' }],
  ['agregar atributo activo de tipo booleano a Cliente', { kind: 'add-attribute', name: 'activo', type: 'Boolean', className: 'Cliente' }],
  ['agregar atributo activo de tipo lógico a Cliente', { kind: 'add-attribute', name: 'activo', type: 'Boolean', className: 'Cliente' }],
  ['agregar atributo activo de tipo logico a Cliente', { kind: 'add-attribute', name: 'activo', type: 'Boolean', className: 'Cliente' }],
  ['agregar atributo codigo de tipo largo a Cliente', { kind: 'add-attribute', name: 'codigo', type: 'Long', className: 'Cliente' }],
  ['agregar atributo peso de tipo flotante a Cliente', { kind: 'add-attribute', name: 'peso', type: 'Float', className: 'Cliente' }],
  ['agregar atributo altura de tipo doble a Cliente', { kind: 'add-attribute', name: 'altura', type: 'Double', className: 'Cliente' }],
  ['agregar atributo externo de tipo uuid a Cliente', { kind: 'add-attribute', name: 'externo', type: 'UUID', className: 'Cliente' }],
  ['agregar atributo edad de tipo ENTERO a Cliente', { kind: 'add-attribute', name: 'edad', type: 'Integer', className: 'Cliente' }],
  ['añadir método calcularEdad de retorno entero a Cliente', { kind: 'add-method', name: 'calcularEdad', returnType: 'Integer', className: 'Cliente' }],
  // Unknown/English types still pass through unchanged.
  ['agregar atributo email de tipo String a Usuario', { kind: 'add-attribute', name: 'email', type: 'String', className: 'Usuario' }],
];

for (const [input, expected] of cases) {
  const actual = parseVoiceCommand(input);
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`Failed for input "${input}":\n  expected: ${JSON.stringify(expected)}\n  actual:   ${JSON.stringify(actual)}`);
  }
}
console.log(`voice parser smoke passed: ${cases.length} cases`);
