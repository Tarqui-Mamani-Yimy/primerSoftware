export type ActiveView = 
  | 'uml-canvas' 
  | 'backend-db-generator';

export type Visibility = '+' | '-' | '#';

export interface UMLAttribute {
  id: string;
  name: string;
  type: string;
  visibility: Visibility;
  annotations?: string[];
  isPk?: boolean;
  notes?: string;
}

export interface UMLMethod {
  id: string;
  name: string;
  returnType: string;
  visibility: Visibility;
  isAbstract?: boolean;
}

export type Stereotype = 
  | '«Entity»' 
  | '«Entity, AggregateRoot»' 
  | '«Entity, Part»' 
  | '«Abstract»' 
  | '«Interface»' 
  | '«Enum»' 
  | '«ValueObject»';

export interface UMLClassNode {
  id: string;
  name: string;
  stereotype: Stereotype;
  package: string;
  tableBinding: string;
  badge?: string;
  x: number;
  y: number;
  width?: number;
  attributes: UMLAttribute[];
  methods: UMLMethod[];
  statusText?: string;
  isAbstract?: boolean;
  /** Association-class extension: marks the class as an association class. */
  isAssociationClass?: boolean;
  /** Association-class extension: id of the relationship the class is attached to. */
  attachedRelationshipId?: string;
}

export type RelationshipType = 
  | 'association' 
  | 'aggregation' 
  | 'composition' 
  | 'generalization' 
  | 'realization' 
  | 'dependency';

export interface UMLRelationship {
  id: string;
  sourceId: string;
  targetId: string;
  type: RelationshipType;
  sourceMultiplicity?: string;
  targetMultiplicity?: string;
  label?: string;
}

/**
 * Portable document exchanged with the diagram API. Keep this independent of
 * JointJS: the renderer is an implementation detail, not the source of truth.
 *
 * `version` is the optimistic-concurrency baseline echoed by the server after
 * every GET/PUT and after every explicit checkpoint creation. Clients include
 * it on their next PUT and POST /checkpoints call (and as the `If-Match`
 * header) so a stale write surfaces as 409 with the server's current
 * document, instead of overwriting another collaborator's checkpoint.
 */
export interface UMLDiagramDocument {
  schemaVersion: 1;
  id?: string;
  version: number;
  name: string;
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
}

export interface DiagramCheckpointAuthor {
  createdBy: string;
  message?: string | null;
  createdAt: string;
}
