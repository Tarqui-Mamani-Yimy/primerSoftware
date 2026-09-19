import { UMLDiagramDocument, UMLRelationship } from '../types';

export const DIAGRAM_SCHEMA_VERSION = 1 as const;

export function createDiagramDocument(
  name: string,
  classes: UMLDiagramDocument['classes'],
  relationships: UMLRelationship[],
  id?: string,
  version: number = 0,
): UMLDiagramDocument {
  return { schemaVersion: DIAGRAM_SCHEMA_VERSION, id, version, name, classes, relationships };
}

/** Returns contract violations rather than coupling callers to a UI renderer. */
export function validateDiagramDocument(document: UMLDiagramDocument): string[] {
  const ids = new Set<string>();
  const errors: string[] = [];
  document.classes.forEach((umlClass) => {
    if (!umlClass.id || ids.has(umlClass.id)) errors.push(`Class id must be unique: ${umlClass.id}`);
    ids.add(umlClass.id);
    if (!umlClass.name.trim()) errors.push(`Class ${umlClass.id} must have a name`);
  });
  document.relationships.forEach((relationship) => {
    if (!ids.has(relationship.sourceId) || !ids.has(relationship.targetId)) {
      errors.push(`Relationship ${relationship.id} references a missing class`);
    }
  });
  return errors;
}
