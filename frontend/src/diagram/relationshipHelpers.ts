import { RelationshipType, UMLClassNode, UMLRelationship } from '../types';

export const MULTIPLICITY_PATTERN = /^\s*(\d+|\*)\s*(\.\.\s*(\d+|\*))?\s*$/;

/** Empty value means "no multiplicity" and is always accepted. */
export const isValidMultiplicity = (value?: string): boolean => {
  if (value === undefined || value.trim() === '') return true;
  return MULTIPLICITY_PATTERN.test(value);
};

export interface RelationshipValidation {
  ok: boolean;
  reason?: string;
}

/**
 * Single validation entry point shared by the canvas interaction.
 * Order matters: self-association is reported before duplicates.
 */
export const validateRelationshipCreation = (
  sourceId: string,
  targetId: string,
  type: RelationshipType,
  existing: UMLRelationship[],
  source?: UMLClassNode,
  target?: UMLClassNode,
): RelationshipValidation => {
  if (sourceId === targetId) {
    return { ok: false, reason: 'Self-association is not allowed: pick a different target class.' };
  }
  if (existing.some((r) => r.sourceId === sourceId && r.targetId === targetId && r.type === type)) {
    return { ok: false, reason: 'That relationship already exists between these classes.' };
  }
  if (source?.isAssociationClass || target?.isAssociationClass) {
    return { ok: false, reason: 'An association class cannot be a relationship endpoint: pick a regular class.' };
  }
  if (type === 'realization') {
    const sourceOk = source ? source.stereotype !== '«Interface»' && source.stereotype !== '«Enum»' : true;
    const targetOk = target ? target.stereotype === '«Interface»' : true;
    if (!sourceOk || !targetOk) {
      return { ok: false, reason: 'Realization requires a non-interface source and an interface target.' };
    }
  }
  if (!source || !target) {
    return { ok: false, reason: 'Both ends of the relationship must reference existing classes.' };
  }
  return { ok: true };
};

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface EdgeEndpoints {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

/**
 * Intersects a ray starting at `origin` going in `direction` with a rectangle border.
 * Returns null when the origin is outside the rect or the direction is degenerate.
 */
const intersectRayWithRect = (origin: { x: number; y: number }, direction: { x: number; y: number }, rect: Rect) => {
  if (direction.x === 0 && direction.y === 0) return null;
  let tMin = -Infinity;
  let tMax = Infinity;
  if (direction.x !== 0) {
    const t1 = (rect.x - origin.x) / direction.x;
    const t2 = (rect.x + rect.width - origin.x) / direction.x;
    tMin = Math.max(tMin, Math.min(t1, t2));
    tMax = Math.min(tMax, Math.max(t1, t2));
  } else if (origin.x < rect.x || origin.x > rect.x + rect.width) {
    return null;
  }
  if (direction.y !== 0) {
    const t1 = (rect.y - origin.y) / direction.y;
    const t2 = (rect.y + rect.height - origin.y) / direction.y;
    tMin = Math.max(tMin, Math.min(t1, t2));
    tMax = Math.min(tMax, Math.max(t1, t2));
  } else if (origin.y < rect.y || origin.y > rect.y + rect.height) {
    return null;
  }
  // Origin is the node center (inside the rect), so the exit point is tMax.
  if (tMax < 0 || tMin > tMax) return null;
  return { x: origin.x + direction.x * tMax, y: origin.y + direction.y * tMax };
};

/**
 * Clips a center-to-center edge to the borders of the source/target rectangles
 * (StarUML-style). Falls back to center points when geometry is degenerate
 * (e.g. overlapping nodes).
 */
export const clipEdgeToNodeBorders = (source: Rect, target: Rect): EdgeEndpoints => {
  const sourceCenter = { x: source.x + source.width / 2, y: source.y + source.height / 2 };
  const targetCenter = { x: target.x + target.width / 2, y: target.y + target.height / 2 };
  const direction = { x: targetCenter.x - sourceCenter.x, y: targetCenter.y - sourceCenter.y };
  if (direction.x === 0 && direction.y === 0) return { x1: sourceCenter.x, y1: sourceCenter.y, x2: targetCenter.x, y2: targetCenter.y };
  const clippedSource = intersectRayWithRect(sourceCenter, direction, source);
  const clippedTarget = intersectRayWithRect(targetCenter, { x: -direction.x, y: -direction.y }, target);
  return {
    x1: clippedSource?.x ?? sourceCenter.x,
    y1: clippedSource?.y ?? sourceCenter.y,
    x2: clippedTarget?.x ?? targetCenter.x,
    y2: clippedTarget?.y ?? targetCenter.y,
  };
};

/** Linear interpolation between edge endpoints; t=0 is source, t=1 is target. */
export const pointAlongEdge = (edge: EdgeEndpoints, t: number) => ({
  x: edge.x1 + (edge.x2 - edge.x1) * t,
  y: edge.y1 + (edge.y2 - edge.y1) * t,
});
