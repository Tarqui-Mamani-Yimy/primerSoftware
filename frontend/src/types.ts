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
 */
export interface UMLDiagramDocument {
  schemaVersion: 1;
  id?: string;
  name: string;
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
}

export type JpaStrategy = 'JOINED' | 'SINGLE' | 'TABLE_PER';

export interface CodeFile {
  id: string;
  path: string;
  filename: string;
  badge?: string;
  badgeType?: 'primary' | 'secondary' | 'tertiary' | 'outline';
  language: 'java' | 'sql' | 'yaml' | 'xml';
  content: string;
  linesCount?: number;
}
