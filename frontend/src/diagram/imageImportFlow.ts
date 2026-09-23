import { UMLClassNode, UMLDiagramDocument, UMLRelationship } from '../types';

export interface ImageImportBinding {
  sourceProjectId: string;
  sourceDiagramId: string;
  sourceVersion: number;
  sourceReviewNumber: number;
}

export interface BoundImageImportPreview extends ImageImportBinding {
  document: UMLDiagramDocument;
}

export interface ConfirmImageImportParams {
  currentProjectId: string;
  currentDiagramId: string;
  currentVersion: number;
  currentReviewNumber: number;
  isDirty: boolean;
  preview: BoundImageImportPreview;
}

export interface ConfirmedDiagramState {
  id: string;
  version: number;
  reviewNumber: number;
  name: string;
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
}

/**
 * Validates if an in-flight or completed image import preview is still current
 * with respect to the active project, diagram ID, version, review number, and dirty state.
 */
export function isImportPreviewCurrent(
  binding: ImageImportBinding,
  current: { projectId?: string; diagramId?: string; version: number; reviewNumber: number; isDirty: boolean }
): boolean {
  return (
    !current.isDirty &&
    Boolean(current.projectId) &&
    current.projectId === binding.sourceProjectId &&
    Boolean(current.diagramId) &&
    current.diagramId === binding.sourceDiagramId &&
    current.version === binding.sourceVersion &&
    current.reviewNumber === binding.sourceReviewNumber
  );
}

/**
 * Confirms the image import if and only if the preview matches the current active diagram
 * identity and revision. Rejects stale confirmations by returning null.
 *
 * Missing or external generated IDs on the imported document are deliberately
 * ignored so the user replaces only the intended diagram they are currently viewing.
 */
export function confirmImageImport(params: ConfirmImageImportParams): ConfirmedDiagramState | null {
  const { currentProjectId, currentDiagramId, currentVersion, currentReviewNumber, isDirty, preview } = params;
  if (
    !isImportPreviewCurrent(preview, {
      projectId: currentProjectId,
      diagramId: currentDiagramId,
      version: currentVersion,
      reviewNumber: currentReviewNumber,
      isDirty,
    })
  ) {
    return null;
  }

  const name = preview.document.name?.trim() || 'Diagrama sin título';
  return {
    id: currentDiagramId,
    version: currentVersion,
    reviewNumber: currentReviewNumber,
    name,
    classes: preview.document.classes,
    relationships: preview.document.relationships,
  };
}
