import { createDiagramXmiExport } from '../src/diagram/xmiExport';
import { UMLClassNode, UMLRelationship } from '../src/types';

const assertTrue = (condition: boolean, message: string) => {
  if (!condition) throw new Error(`Assertion failed: ${message}`);
};

const makeClass = (overrides: Partial<UMLClassNode> & { id: string; name: string }): UMLClassNode => ({
  stereotype: '«Entity»',
  package: '',
  tableBinding: '',
  x: 0,
  y: 0,
  attributes: [],
  methods: [],
  ...overrides,
});

// --- Fixture model -----------------------------------------------------
// Pedido (whole) --composition--> Linea (part)
// Pedido (whole) --aggregation--> Etiqueta (part)
// Pedido --association 1..*--> Cliente
const pedido = makeClass({
  id: 'pedido',
  name: 'Pedido',
  attributes: [
    { id: 'a1', name: 'total', type: 'BigDecimal', visibility: '+' },
    { id: 'a2', name: 'notas', type: 'String', visibility: '-' },
    { id: 'a3', name: 'referencia', type: 'String', visibility: '#' },
  ],
  methods: [
    { id: 'm1', name: 'confirmar', returnType: 'void', visibility: '+' },
  ],
});
const linea = makeClass({ id: 'linea', name: 'Linea' });
const etiqueta = makeClass({ id: 'etiqueta', name: 'Etiqueta' });
const cliente = makeClass({ id: 'cliente', name: 'Cliente' });
const impresora = makeClass({ id: 'impresora', name: 'Impresora', stereotype: '«Interface»' });
const impresoraTermica = makeClass({
  id: 'impresora-termica',
  name: 'ImpresoraTermica',
  attributes: [{ id: 'a4', name: 'puerto', type: 'String', visibility: '+' }],
});
const estadoPedido = makeClass({
  id: 'estado-pedido',
  name: 'EstadoPedido',
  stereotype: '«Enum»',
  attributes: [
    { id: 'e1', name: 'PENDIENTE', type: '', visibility: '+' },
    { id: 'e2', name: 'PAGADO', type: '', visibility: '+' },
  ],
});
const escaped = makeClass({
  id: 'escaped',
  name: 'A<B & "C" \'D\'',
  attributes: [{ id: 'a5', name: 'x<y', type: '', visibility: '+' }],
});

const relationships: UMLRelationship[] = [
  { id: 'r-comp', sourceId: 'pedido', targetId: 'linea', type: 'composition', sourceMultiplicity: '1', targetMultiplicity: '*' },
  { id: 'r-agg', sourceId: 'pedido', targetId: 'etiqueta', type: 'aggregation', sourceMultiplicity: '1', targetMultiplicity: '0..1' },
  { id: 'r-assoc', sourceId: 'pedido', targetId: 'cliente', type: 'association', sourceMultiplicity: '1..*', targetMultiplicity: '1' },
  { id: 'r-assoc-invalid', sourceId: 'pedido', targetId: 'cliente', type: 'association', sourceMultiplicity: '3..1' },
  { id: 'r-real', sourceId: 'impresora-termica', targetId: 'impresora', type: 'realization' },
  { id: 'r-gen', sourceId: 'impresora-termica', targetId: 'impresora', type: 'generalization' },
];

const classes: UMLClassNode[] = [pedido, linea, etiqueta, cliente, impresora, impresoraTermica, estadoPedido, escaped];

const result = createDiagramXmiExport('Modelo de Pedido', classes, relationships);
const xmi = result.xmi;

// --- 1. Visibility mapping ----------------------------------------------
assertTrue(/visibility="public"/.test(xmi), 'expected visibility="public" for +');
assertTrue(/visibility="private"/.test(xmi), 'expected visibility="private" for -');
assertTrue(/visibility="protected"/.test(xmi), 'expected visibility="protected" for #');
assertTrue(!/visibility="\+"/.test(xmi), 'raw + visibility literal must not appear');
assertTrue(!/visibility="-"/.test(xmi), 'raw - visibility literal must not appear');
assertTrue(!/visibility="#"/.test(xmi), 'raw # visibility literal must not appear');

// --- 2. Aggregation / composition end placement --------------------------
const classifierIdOf = (name: string) => {
  const match = xmi.match(new RegExp(`<packagedElement xmi:type="uml:(?:Class|Interface|Enumeration)" xmi:id="([^"]+)" name="${name}"`));
  assertTrue(!!match, `classifier id not found for ${name}`);
  return match![1];
};
const lineaId = classifierIdOf('Linea');
const pedidoId = classifierIdOf('Pedido');
const etiquetaId = classifierIdOf('Etiqueta');

// Composition: Linea end (part) has aggregation="composite"; Pedido end (whole) has none, within the composition association.
const associationBlocks = [...xmi.matchAll(/<packagedElement xmi:type="uml:Association"[^>]*>((?:<ownedEnd[^]*?<\/ownedEnd>){2})<\/packagedElement>/g)];
assertTrue(associationBlocks.length === 4, `expected 4 association elements, found ${associationBlocks.length}`);

const findAssociationEndsByTypeIds = (typeIdA: string, typeIdB: string) => {
  const block = associationBlocks.find((entry) => entry[1].includes(`type="${typeIdA}"`) && entry[1].includes(`type="${typeIdB}"`));
  assertTrue(!!block, `no association found between ${typeIdA} and ${typeIdB}`);
  const ends = [...block![1].matchAll(/<ownedEnd[^>]*>/g)].map((entry) => entry[0]);
  return ends;
};

const compositionEnds = findAssociationEndsByTypeIds(pedidoId, lineaId);
const compositionPedidoEnd = compositionEnds.find((end) => end.includes(`type="${pedidoId}"`))!;
const compositionLineaEnd = compositionEnds.find((end) => end.includes(`type="${lineaId}"`))!;
assertTrue(compositionLineaEnd.includes('aggregation="composite"'), 'part end (Linea) must carry aggregation="composite"');
assertTrue(!compositionPedidoEnd.includes('aggregation='), 'whole end (Pedido) must not carry aggregation');

const aggregationEnds = findAssociationEndsByTypeIds(pedidoId, etiquetaId);
const aggregationPedidoEnd = aggregationEnds.find((end) => end.includes(`type="${pedidoId}"`))!;
const aggregationEtiquetaEnd = aggregationEnds.find((end) => end.includes(`type="${etiquetaId}"`))!;
assertTrue(aggregationEtiquetaEnd.includes('aggregation="shared"'), 'part end (Etiqueta) must carry aggregation="shared"');
assertTrue(!aggregationPedidoEnd.includes('aggregation='), 'whole end (Pedido) must not carry aggregation');

// --- 3. Plain association has no aggregation -----------------------------
const clienteId = classifierIdOf('Cliente');
const plainEnds = findAssociationEndsByTypeIds(pedidoId, clienteId);
plainEnds.forEach((end) => assertTrue(!end.includes('aggregation='), 'plain association end must not carry aggregation'));

// --- 4. Multiplicities ----------------------------------------------------
assertTrue(/lowerValue xmi:type="uml:LiteralInteger" value="0"/.test(xmi) && /upperValue xmi:type="uml:LiteralUnlimitedNatural" value="\*"/.test(xmi), "expected '*' to map to lower 0 / upper *");
assertTrue(/lowerValue xmi:type="uml:LiteralInteger" value="1"\/><upperValue xmi:type="uml:LiteralUnlimitedNatural" value="\*"/.test(xmi), "expected '1..*' to map to lower 1 / upper *");
assertTrue(/lowerValue xmi:type="uml:LiteralInteger" value="0"\/><upperValue xmi:type="uml:LiteralUnlimitedNatural" value="1"/.test(xmi), "expected '0..1' to be preserved");
// Invalid multiplicity '3..1' must produce no lowerValue/upperValue for that end.
const invalidAssocBlocks = associationBlocks.filter((entry) => entry[1].includes(`type="${pedidoId}"`) && entry[1].includes(`type="${clienteId}"`));
assertTrue(invalidAssocBlocks.length === 2, 'expected two Pedido-Cliente associations (valid + invalid multiplicity)');
const hasEmptyEnd = invalidAssocBlocks.some((entry) => /<ownedEnd[^>]*><\/ownedEnd>/.test(entry[1]));
assertTrue(hasEmptyEnd, "expected the '3..1' multiplicity association to have an ownedEnd with no lower/upperValue");

// --- 5. Enumeration --------------------------------------------------------
const enumMatch = xmi.match(/<packagedElement xmi:type="uml:Enumeration"[^>]*name="EstadoPedido"[^>]*>([^]*?)<\/packagedElement>/);
assertTrue(!!enumMatch, 'expected uml:Enumeration packagedElement for EstadoPedido');
const enumBody = enumMatch![1];
assertTrue(!enumBody.includes('ownedAttribute'), 'enumeration must not contain ownedAttribute');
assertTrue(/<ownedLiteral xmi:type="uml:EnumerationLiteral" xmi:id="[^"]+" name="PENDIENTE"\/>/.test(enumBody), 'expected ownedLiteral for PENDIENTE');
assertTrue(/<ownedLiteral xmi:type="uml:EnumerationLiteral" xmi:id="[^"]+" name="PAGADO"\/>/.test(enumBody), 'expected ownedLiteral for PAGADO');

// --- 6. Interface realization ---------------------------------------------
const realizationMatch = xmi.match(/<interfaceRealization xmi:id="[^"]+" client="([^"]+)" supplier="([^"]+)" contract="([^"]+)"\/>/);
assertTrue(!!realizationMatch, 'expected interfaceRealization with client, supplier and contract');
assertTrue(realizationMatch![3] === realizationMatch![2], 'contract must equal supplier (the target interface)');

// --- 7. Generalization owned by specific classifier ------------------------
const impresoraTermicaId = classifierIdOf('ImpresoraTermica');
const impresoraTermicaBlock = xmi.match(new RegExp(`<packagedElement xmi:type="uml:Class" xmi:id="${impresoraTermicaId}"[^]*?<\\/packagedElement>`));
assertTrue(!!impresoraTermicaBlock, 'expected ImpresoraTermica packagedElement');
assertTrue(/<generalization xmi:id="[^"]+" general="[^"]+"\/>/.test(impresoraTermicaBlock![0]), 'expected generalization nested under ImpresoraTermica');

// --- 8. XML escaping --------------------------------------------------------
assertTrue(xmi.includes('name="A&lt;B &amp; &quot;C&quot; &apos;D&apos;"'), 'expected class name to be XML-escaped');
assertTrue(xmi.includes('name="x&lt;y"'), 'expected attribute name to be XML-escaped');

// --- 9. Stable ids across two exports, and valid XML ID characters ----------
const second = createDiagramXmiExport('Modelo de Pedido', classes, relationships);
assertTrue(second.xmi === xmi, 'exporting the same input twice must produce identical XMI (stable ids)');
const ids = [...xmi.matchAll(/xmi:id="([^"]+)"/g)].map((entry) => entry[1]);
assertTrue(ids.length > 0, 'expected at least one xmi:id in the output');
ids.forEach((id) => assertTrue(/^[A-Za-z_][A-Za-z0-9_.-]*$/.test(id), `xmi:id "${id}" is not a valid XML ID`));

console.log(`xmi export smoke passed: ${ids.length} ids checked`);
