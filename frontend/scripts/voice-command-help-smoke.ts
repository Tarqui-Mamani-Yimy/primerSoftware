import { parseVoiceCommand, VoiceCommand } from '../src/diagram/voiceCommands';
import {
  voiceCommandHelpGroups,
  voiceCommandHelpNotes,
  relationshipTypeVocabulary,
  multiplicityVocabulary,
  attributeTypeVocabulary,
  VoiceCommandHelpExpectation,
} from '../src/diagram/voiceCommandHelp';

function fail(message: string): never {
  throw new Error(message);
}

/** Verifies one example phrase parses to the exact command shape the help content promises. */
function checkExample(phrase: string, expect: VoiceCommandHelpExpectation) {
  const actual: VoiceCommand | null = parseVoiceCommand(phrase);
  if (!actual) fail(`Expected a parsed command for "${phrase}" but got null`);
  if (actual.kind !== expect.kind) {
    fail(`Expected kind "${expect.kind}" for "${phrase}" but got "${actual.kind}"`);
  }
  if (expect.relationshipType !== undefined) {
    if (actual.kind !== 'create-relationship' || actual.type !== expect.relationshipType) {
      fail(`Expected relationship type "${expect.relationshipType}" for "${phrase}"`);
    }
  }
  if (expect.sourceMultiplicity !== undefined) {
    if (actual.kind !== 'create-relationship' || actual.sourceMultiplicity !== expect.sourceMultiplicity) {
      fail(`Expected sourceMultiplicity "${expect.sourceMultiplicity}" for "${phrase}"`);
    }
  }
  if (expect.targetMultiplicity !== undefined) {
    if (actual.kind !== 'create-relationship' || actual.targetMultiplicity !== expect.targetMultiplicity) {
      fail(`Expected targetMultiplicity "${expect.targetMultiplicity}" for "${phrase}"`);
    }
  }
  if (expect.label !== undefined) {
    if (actual.kind !== 'create-relationship' || actual.label !== expect.label) {
      fail(`Expected label "${expect.label}" for "${phrase}"`);
    }
  }
  if (expect.attributeType !== undefined) {
    if (actual.kind !== 'add-attribute' || actual.type !== expect.attributeType) {
      fail(`Expected attribute type "${expect.attributeType}" for "${phrase}"`);
    }
  }
  if (expect.returnType !== undefined) {
    if (actual.kind !== 'add-method' || actual.returnType !== expect.returnType) {
      fail(`Expected return type "${expect.returnType}" for "${phrase}"`);
    }
  }
}

let exampleCount = 0;
for (const group of voiceCommandHelpGroups) {
  if (group.examples.length < 1 || group.examples.length > 5) {
    fail(`Group "${group.id}" must list between 1 and 5 example phrases, has ${group.examples.length}`);
  }
  for (const example of group.examples) {
    checkExample(example.phrase, example.expect);
    exampleCount += 1;
  }
}

if (voiceCommandHelpNotes.length === 0) fail('voiceCommandHelpNotes must not be empty');

for (const entry of relationshipTypeVocabulary) {
  const command = parseVoiceCommand(`relacionar Cliente con Pedido como ${entry.phrase}`);
  if (!command || command.kind !== 'create-relationship' || command.type !== entry.value) {
    fail(`Relationship type word "${entry.phrase}" did not map to "${entry.value}"`);
  }
}

for (const entry of multiplicityVocabulary) {
  const command = parseVoiceCommand(`relacionar Cliente con Pedido con cardinalidad origen ${entry.phrase}`);
  if (!command || command.kind !== 'create-relationship' || command.sourceMultiplicity !== entry.value) {
    fail(`Multiplicity phrase "${entry.phrase}" did not map to "${entry.value}"`);
  }
}

for (const entry of attributeTypeVocabulary) {
  const command = parseVoiceCommand(`agregar atributo campo de tipo ${entry.phrase} a Clase`);
  if (!command || command.kind !== 'add-attribute' || command.type !== entry.value) {
    fail(`Type word "${entry.phrase}" did not map to "${entry.value}"`);
  }
}

console.log(
  `voice command help smoke passed: ${exampleCount} examples, ${relationshipTypeVocabulary.length} relationship types, ${multiplicityVocabulary.length} multiplicities, ${attributeTypeVocabulary.length} types`,
);
