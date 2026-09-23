import { validateRelationshipCreation } from '../src/diagram/relationshipHelpers';
import { UMLClassNode, UMLRelationship } from '../src/types';
import { parseVoiceCommand, VoiceCommand } from '../src/diagram/voiceCommands';

// Mock diagram state
let classes: UMLClassNode[] = [
  {
    id: 'c1',
    name: 'Usuario',
    stereotype: '«Entity»',
    package: 'pkg',
    tableBinding: 't_user',
    x: 0,
    y: 0,
    attributes: [],
    methods: [],
  },
  {
    id: 'c2',
    name: 'Pedido',
    stereotype: '«Entity»',
    package: 'pkg',
    tableBinding: 't_order',
    x: 100,
    y: 100,
    attributes: [],
    methods: [],
  },
  {
    id: 'c3',
    name: 'IAutenticable',
    stereotype: '«Interface»',
    package: 'pkg',
    tableBinding: 't_auth',
    x: 200,
    y: 200,
    attributes: [],
    methods: [],
  },
];

let relationships: UMLRelationship[] = [];
let lastVoiceSnapshot: { classes: UMLClassNode[]; relationships: UMLRelationship[] } | null = null;

const rememberVoiceState = () => {
  lastVoiceSnapshot = {
    classes: JSON.parse(JSON.stringify(classes)),
    relationships: JSON.parse(JSON.stringify(relationships)),
  };
};

// Read through a function (instead of the bare `relationships[0]` expression)
// so TypeScript's control-flow narrowing from an earlier literal comparison
// (e.g. `relationships[0].type !== 'composition'`) doesn't leak into later,
// unrelated comparisons after the array has been reassigned by a call to
// `handleRelationshipCommand`/`handleUndo`.
const currentRelationship = (): UMLRelationship => relationships[0];

const matching = (name: string) =>
  classes.filter((c) => c.name.localeCompare(name, 'es', { sensitivity: 'accent' }) === 0);

const parseRelationshipTranscript = (transcript: string): Extract<VoiceCommand, { kind: 'create-relationship' }> => {
  const command = parseVoiceCommand(transcript);
  if (!command || command.kind !== 'create-relationship') {
    throw new Error(`Expected relationship command from transcript: ${transcript}`);
  }
  return command;
};

function handleRelationshipCommand(command: Extract<VoiceCommand, { kind: 'create-relationship' }>): { ok: boolean; message: string } {
  const sources = matching(command.sourceName);
  const targets = matching(command.targetName);
  if (sources.length !== 1 || targets.length !== 1) {
    return { ok: false, message: 'La relación requiere exactamente una clase origen y una clase destino existentes.' };
  }
  const source = sources[0];
  const target = targets[0];
  const existingMatches = relationships.filter((r) => r.sourceId === source.id && r.targetId === target.id);
  if (existingMatches.length > 1) {
    return { ok: false, message: 'Existe más de una relación entre esas clases; la actualización es ambigua.' };
  }

  if (existingMatches.length === 1) {
    const existing = existingMatches[0];
    const relationshipType = command.type ?? existing.type;
    const otherRelationships = relationships.filter((r) => r.id !== existing.id);
    const validation = validateRelationshipCreation(source.id, target.id, relationshipType, otherRelationships, source, target);
    if (!validation.ok) return { ok: false, message: validation.reason ?? 'No se pudo actualizar la relación.' };
    rememberVoiceState();
    const updated: UMLRelationship = {
      ...existing,
      type: relationshipType,
      sourceMultiplicity: command.sourceMultiplicity !== undefined ? command.sourceMultiplicity : existing.sourceMultiplicity,
      targetMultiplicity: command.targetMultiplicity !== undefined ? command.targetMultiplicity : existing.targetMultiplicity,
      label: command.label !== undefined ? command.label : existing.label,
    };
    relationships = relationships.map((r) => (r.id === updated.id ? updated : r));
    return { ok: true, message: `Relación actualizada: ${command.sourceName} con ${command.targetName}.` };
  }

  const relationshipType = command.type ?? 'association';
  const validation = validateRelationshipCreation(source.id, target.id, relationshipType, relationships, source, target);
  if (!validation.ok) return { ok: false, message: validation.reason ?? 'No se pudo crear la relación.' };
  rememberVoiceState();
  const relationship: UMLRelationship = {
    id: `rel_${Date.now()}`,
    sourceId: source.id,
    targetId: target.id,
    type: relationshipType,
    ...(command.sourceMultiplicity !== undefined ? { sourceMultiplicity: command.sourceMultiplicity } : {}),
    ...(command.targetMultiplicity !== undefined ? { targetMultiplicity: command.targetMultiplicity } : {}),
    ...(command.label !== undefined ? { label: command.label } : {}),
  };
  relationships = [...relationships, relationship];
  return { ok: true, message: `Relación creada: ${command.sourceName} con ${command.targetName}.` };
}

function handleUndo(): { ok: boolean; message: string } {
  if (!lastVoiceSnapshot) return { ok: false, message: 'No hay un último cambio de voz para deshacer en este diagrama.' };
  classes = JSON.parse(JSON.stringify(lastVoiceSnapshot.classes));
  relationships = JSON.parse(JSON.stringify(lastVoiceSnapshot.relationships));
  lastVoiceSnapshot = null;
  return { ok: true, message: 'Último cambio realizado por voz deshecho.' };
}

// Test 1: Create relationship with composition, multiplicities, and label
let res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'Pedido',
  type: 'composition',
  sourceMultiplicity: '1',
  targetMultiplicity: '0..*',
  label: 'realiza',
});
if (!res.ok || relationships.length !== 1) throw new Error('Test 1 failed: should create relationship');
if (relationships[0].type !== 'composition') throw new Error('Test 1 failed: type should be composition');
if (relationships[0].sourceMultiplicity !== '1' || relationships[0].targetMultiplicity !== '0..*') throw new Error('Test 1 failed: multiplicities');
if (relationships[0].label !== 'realiza') throw new Error('Test 1 failed: label');

// Test 2: Update existing unambiguous relationship
res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'Pedido',
  type: 'aggregation',
  sourceMultiplicity: '1..*',
  targetMultiplicity: '*',
  label: 'contiene',
});
if (!res.ok || relationships.length !== 1) throw new Error('Test 2 failed: should update existing');
if (currentRelationship().type !== 'aggregation') throw new Error('Test 2 failed: type should be updated to aggregation');
if (currentRelationship().sourceMultiplicity !== '1..*' || currentRelationship().targetMultiplicity !== '*') throw new Error('Test 2 failed: multiplicities updated');
if (currentRelationship().label !== 'contiene') throw new Error('Test 2 failed: label updated');

// Test 3: Undo restores state before update
let undoRes = handleUndo();
if (!undoRes.ok || relationships[0].type !== 'composition' || relationships[0].label !== 'realiza') {
  throw new Error('Test 3 failed: undo should restore composition');
}

// Test 4: Actual transcripts that omit `como <tipo>` preserve every existing type.
const preservationFixtures: Array<{ type: UMLRelationship['type']; targetId: string; targetName: string }> = [
  { type: 'composition', targetId: 'c2', targetName: 'Pedido' },
  { type: 'aggregation', targetId: 'c2', targetName: 'Pedido' },
  { type: 'generalization', targetId: 'c2', targetName: 'Pedido' },
  { type: 'dependency', targetId: 'c2', targetName: 'Pedido' },
  { type: 'realization', targetId: 'c3', targetName: 'IAutenticable' },
];
for (const fixture of preservationFixtures) {
  relationships = [{ id: `existing_${fixture.type}`, sourceId: 'c1', targetId: fixture.targetId, type: fixture.type }];
  const transcript = `relacionar Usuario con ${fixture.targetName} con cardinalidad destino 1..* con verbo conserva`;
  const command = parseRelationshipTranscript(transcript);
  if (command.type !== undefined) throw new Error(`Test 4 failed: omitted type must stay absent for ${fixture.type}`);
  res = handleRelationshipCommand(command);
  if (!res.ok || relationships[0].type !== fixture.type) {
    throw new Error(`Test 4 failed: omitted type must preserve ${fixture.type}`);
  }
  if (relationships[0].targetMultiplicity !== '1..*' || relationships[0].label !== 'conserva') {
    throw new Error(`Test 4 failed: cardinality and verb should update ${fixture.type}`);
  }
}

// Test 5: Undo restores the composition after a transcript-driven partial update.
relationships = [{ id: 'existing_composition', sourceId: 'c1', targetId: 'c2', type: 'composition', label: 'realiza' }];
res = handleRelationshipCommand(parseRelationshipTranscript('relacionar Usuario con Pedido con cardinalidad destino 1..* con verbo gestiona'));
if (!res.ok || relationships[0].type !== 'composition') throw new Error('Test 5 failed: partial update should preserve composition');
undoRes = handleUndo();
if (!undoRes.ok || relationships[0].type !== 'composition' || relationships[0].label !== 'realiza') {
  throw new Error('Test 5 failed: undo should restore the prior composition');
}

// Test 6: Only one voice snapshot is held.
undoRes = handleUndo();
if (undoRes.ok) throw new Error('Test 6 failed: second undo should fail');

// Test 7: Rejection when target class is missing
res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'Inexistente',
  type: 'association',
});
if (res.ok) throw new Error('Test 7 failed: missing target must be rejected');

// Test 8: Realization validation (requires interface target)
res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'Pedido',
  type: 'realization',
});
if (res.ok) throw new Error('Test 8 failed: realization to non-interface must be rejected');

// Test 9: Realization to interface target succeeds
res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'IAutenticable',
  type: 'realization',
});
if (!res.ok) throw new Error('Test 9 failed: realization to interface should succeed');

// Test 10: Ambiguity rejection when multiple relationships exist between same pair
relationships.push({
  id: 'rel_extra',
  sourceId: 'c1',
  targetId: 'c2',
  type: 'dependency',
});
// Now there are 2 relationships between Usuario (c1) and Pedido (c2)
res = handleRelationshipCommand({
  kind: 'create-relationship',
  sourceName: 'Usuario',
  targetName: 'Pedido',
  type: 'association',
});
if (res.ok || !res.message.includes('ambigua')) {
  throw new Error('Test 10 failed: multiple relationships between pair must reject as ambiguous');
}

console.log('voice relationship execution smoke passed all 10 scenarios!');
