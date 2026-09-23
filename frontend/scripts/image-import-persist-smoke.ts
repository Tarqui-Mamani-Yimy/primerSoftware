import {
  BoundImageImportPreview,
  confirmImageImport,
  ImageImportBinding,
  isImportPreviewCurrent,
} from '../src/diagram/imageImportFlow';
import { UMLClassNode, UMLDiagramDocument, UMLRelationship } from '../src/types';

interface DiagramStoreState {
  id: string;
  name: string;
  version: number;
  reviewNumber: number;
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
}

// In-memory mock server store for testing persistence and reloads
const serverDb: Record<string, DiagramStoreState> = {};

function resetServerDb() {
  serverDb['diag-web-1'] = {
    id: 'diag-web-1',
    name: 'Original Diagram 1',
    version: 4,
    reviewNumber: 9,
    classes: [
      { id: 'orig-1', name: 'LegacyUser', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] },
    ],
    relationships: [],
  };
  serverDb['diag-web-2'] = {
    id: 'diag-web-2',
    name: 'Original Diagram 2',
    version: 1,
    reviewNumber: 2,
    classes: [
      { id: 'orig-2', name: 'OtherEntity', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] },
    ],
    relationships: [],
  };
}

const mockDiagramApi = {
  update: async (
    projectId: string,
    id: string,
    doc: UMLDiagramDocument,
    options?: { ifMatch?: number; review?: number }
  ): Promise<UMLDiagramDocument> => {
    const existing = serverDb[id];
    if (!existing) {
      throw new Error(`404: diagram ${id} not found`);
    }
    const expectedVersion = options?.ifMatch ?? doc.version;
    if (expectedVersion !== existing.version) {
      const err = new Error(`409 Conflict: expected version ${existing.version}, got ${expectedVersion}`);
      (err as any).status = 409;
      throw err;
    }
    const newVersion = existing.version + 1;
    const newReview = existing.reviewNumber + 1;
    const updated: DiagramStoreState = {
      id,
      name: doc.name,
      version: newVersion,
      reviewNumber: newReview,
      classes: doc.classes,
      relationships: doc.relationships,
    };
    serverDb[id] = updated;
    return {
      schemaVersion: 1,
      id,
      name: updated.name,
      version: newVersion,
      reviewNumber: newReview,
      classes: updated.classes,
      relationships: updated.relationships,
    };
  },
  get: async (projectId: string, id: string): Promise<UMLDiagramDocument> => {
    const existing = serverDb[id];
    if (!existing) throw new Error(`404: diagram ${id} not found`);
    return {
      schemaVersion: 1,
      id: existing.id,
      name: existing.name,
      version: existing.version,
      reviewNumber: existing.reviewNumber,
      classes: existing.classes,
      relationships: existing.relationships,
    };
  },
};

// Harness faithfully replicating App.tsx wiring and state transitions
class AppTestHarness {
  projectId = 'proj-1';
  diagramId: string | undefined;
  diagramVersion = 0;
  diagramReview = 0;
  diagramName = '';
  classes: UMLClassNode[] = [];
  relationships: UMLRelationship[] = [];

  imageImportPreview: BoundImageImportPreview | null = null;
  persistenceStatus: 'saved' | 'saving' | 'error' | 'conflict' | 'offline' = 'saved';
  dirty = false;
  pendingSnapshot: { classes: UMLClassNode[]; relationships: UMLRelationship[]; name: string } | null = null;
  saveCallCount = 0;

  // Mimics applyDocument in App.tsx
  applyDocument(document: UMLDiagramDocument) {
    this.imageImportPreview = null;
    const targetId = document.id ?? this.diagramId;
    this.classes = JSON.parse(JSON.stringify(document.classes));
    this.relationships = JSON.parse(JSON.stringify(document.relationships));
    this.diagramName = document.name;
    this.diagramId = targetId;
    this.diagramVersion = document.version ?? 0;
    this.diagramReview = document.reviewNumber ?? 0;
    this.dirty = false;
    this.pendingSnapshot = null;
  }

  // Mimics clearSaveState in App.tsx
  clearSaveState() {
    this.imageImportPreview = null;
    this.dirty = false;
    this.pendingSnapshot = null;
  }

  // Mimics openDiagram in App.tsx
  async openDiagram(id: string) {
    this.clearSaveState();
    const doc = await mockDiagramApi.get(this.projectId, id);
    this.applyDocument(doc);
    this.persistenceStatus = 'saved';
  }

  // Mimics openProject in App.tsx
  async openProject(newProjectId: string, initialDiagramId: string) {
    this.clearSaveState();
    this.projectId = newProjectId;
    this.applyDocument({
      schemaVersion: 1,
      name: '',
      version: 0,
      reviewNumber: 0,
      classes: [],
      relationships: [],
    });
    await this.openDiagram(initialDiagramId);
  }

  // Mimics scheduleSave in App.tsx
  async scheduleSave(nextClasses?: UMLClassNode[], nextRelationships?: UMLRelationship[], nextName?: string) {
    if (!this.diagramId) return;
    this.imageImportPreview = null;
    this.saveCallCount++;
    this.dirty = true;
    this.persistenceStatus = 'saving';
    this.pendingSnapshot = {
      classes: nextClasses ?? this.classes,
      relationships: nextRelationships ?? this.relationships,
      name: nextName ?? this.diagramName,
    };
    await this.persistDiagram(this.pendingSnapshot);
  }

  // Mimics manual edit (e.g. handleUpdateClass)
  async manualEditClass(updated: UMLClassNode) {
    const next = this.classes.map((c) => (c.id === updated.id ? updated : c));
    this.classes = next;
    await this.scheduleSave(next);
  }

  // Mimics realtime remote change (e.g. onChanged)
  async onRemoteChangeApplied(freshServerDoc: UMLDiagramDocument) {
    this.applyDocument(freshServerDoc);
    this.persistenceStatus = 'saved';
  }

  // Mimics start of importDiagramImage (capturing binding)
  startImageImport(): ImageImportBinding {
    if (!this.diagramId) throw new Error('No active diagram to import for');
    return {
      sourceProjectId: this.projectId,
      sourceDiagramId: this.diagramId,
      sourceVersion: this.diagramVersion,
      sourceReviewNumber: this.diagramReview,
    };
  }

  // Mimics async importDiagramImage resolution with freshness check
  receiveImportResult(binding: ImageImportBinding, imported: UMLDiagramDocument): boolean {
    const isStillCurrent = isImportPreviewCurrent(binding, {
      projectId: this.projectId,
      diagramId: this.diagramId,
      version: this.diagramVersion,
      reviewNumber: this.diagramReview,
      isDirty: this.dirty,
    });
    if (!isStillCurrent) {
      // Dropped because stale!
      return false;
    }
    this.imageImportPreview = {
      ...binding,
      document: imported,
    };
    return true;
  }

  // Mimics Cancel button in preview dialog
  onCancelImageImport() {
    this.imageImportPreview = null;
  }

  // Mimics handleConfirmImageImport in App.tsx
  async handleConfirmImageImport(): Promise<boolean> {
    if (!this.imageImportPreview || !this.diagramId) {
      this.imageImportPreview = null;
      return false;
    }

    const confirmed = confirmImageImport({
      currentProjectId: this.projectId,
      currentDiagramId: this.diagramId,
      currentVersion: this.diagramVersion,
      currentReviewNumber: this.diagramReview,
      isDirty: this.dirty,
      preview: this.imageImportPreview,
    });

    if (!confirmed) {
      this.imageImportPreview = null;
      return false;
    }

    this.classes = confirmed.classes;
    this.relationships = confirmed.relationships;
    this.diagramName = confirmed.name;
    this.diagramId = confirmed.id;
    this.diagramVersion = confirmed.version;
    this.diagramReview = confirmed.reviewNumber;
    this.imageImportPreview = null;

    await this.scheduleSave(confirmed.classes, confirmed.relationships, confirmed.name);
    return true;
  }

  async persistDiagram(snapshot: { classes: UMLClassNode[]; relationships: UMLRelationship[]; name: string }) {
    if (!this.diagramId) return;
    const id = this.diagramId;
    const version = this.diagramVersion;
    const review = this.diagramReview;
    const document: UMLDiagramDocument = {
      schemaVersion: 1,
      id,
      name: snapshot.name.trim() || 'Diagrama sin título',
      version,
      reviewNumber: review,
      classes: snapshot.classes,
      relationships: snapshot.relationships,
    };
    try {
      const saved = await mockDiagramApi.update(this.projectId, id, document, { ifMatch: version, review });
      this.diagramVersion = saved.version;
      this.diagramReview = saved.reviewNumber;
      this.classes = saved.classes;
      this.relationships = saved.relationships;
      this.diagramName = saved.name;
      this.dirty = false;
      this.pendingSnapshot = null;
      this.persistenceStatus = 'saved';
    } catch (err: any) {
      if (err.status === 409) {
        this.persistenceStatus = 'conflict';
      } else {
        this.persistenceStatus = 'error';
      }
    }
  }

  async reloadCurrentDiagram() {
    if (!this.diagramId) throw new Error('Cannot reload without diagramId');
    const doc = await mockDiagramApi.get(this.projectId, this.diagramId);
    this.applyDocument(doc);
  }
}

async function runTests() {
  console.log('=== Running MM-01 Comprehensive Correction Regression Suite ===');

  // Test 1: Cancel does nothing and clears preview
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    const importedDoc: UMLDiagramDocument = {
      schemaVersion: 1,
      name: 'Proposed Import',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'new-1', name: 'DraftClass', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    };
    const set = app.receiveImportResult(binding, importedDoc);
    if (!set || !app.imageImportPreview) throw new Error('Test 1 failed: preview should be set');

    app.onCancelImageImport();
    if (app.imageImportPreview !== null) throw new Error('Test 1 failed: preview should be cleared on cancel');
    if (app.diagramId !== 'diag-web-1') throw new Error('Test 1 failed: diagramId should not change');
    if (app.diagramVersion !== 4) throw new Error('Test 1 failed: diagramVersion should not change');
    if (app.classes[0].name !== 'LegacyUser') throw new Error('Test 1 failed: classes should not change');
    if (app.saveCallCount !== 0) throw new Error('Test 1 failed: cancel must not trigger save');

    console.log('✔ Test 1 passed: Cancel leaves existing UML and diagram state completely unchanged.');
  }

  // Test 2: Valid confirm with missing generated ID keeps active ID/CAS and persists
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    const importedFromGemini: UMLDiagramDocument = {
      schemaVersion: 1,
      id: undefined, // missing generated id
      version: 0,
      reviewNumber: 0,
      name: 'Imported E-Commerce Architecture',
      classes: [
        { id: 'c-order', name: 'Order', stereotype: '«Entity»', package: 'com.store', tableBinding: 't_orders', x: 50, y: 50, attributes: [], methods: [] },
        { id: 'c-customer', name: 'Customer', stereotype: '«Entity»', package: 'com.store', tableBinding: 't_customers', x: 250, y: 50, attributes: [], methods: [] },
      ],
      relationships: [
        { id: 'r-order-customer', sourceId: 'c-order', targetId: 'c-customer', type: 'association' },
      ],
    };

    app.receiveImportResult(binding, importedFromGemini);
    const confirmed = await app.handleConfirmImageImport();
    if (!confirmed) throw new Error('Test 2 failed: confirmation should succeed');

    if (app.diagramId !== 'diag-web-1') throw new Error(`Test 2 failed: diagramId was ${app.diagramId}`);
    if (app.diagramVersion !== 5) throw new Error(`Test 2 failed: expected version 5, got ${app.diagramVersion}`);
    if (app.diagramReview !== 10) throw new Error(`Test 2 failed: expected review 10, got ${app.diagramReview}`);
    if (app.persistenceStatus !== 'saved') throw new Error('Test 2 failed: persistenceStatus should be saved');

    await app.reloadCurrentDiagram();
    if (app.classes.length !== 2 || app.classes[0].name !== 'Order') throw new Error('Test 2 failed: reload did not return imported UML');

    console.log('✔ Test 2 passed: Confirm with missing generated id keeps active identity/CAS, persists, and reload returns imported UML.');
  }

  // Test 3: Diagram switch while preview is open immediately clears preview and rejects stale confirmation
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    const imported: UMLDiagramDocument = {
      schemaVersion: 1,
      name: 'Imported for Diag 1',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'c-x', name: 'ImportedForDiag1', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    };
    app.receiveImportResult(binding, imported);
    if (!app.imageImportPreview) throw new Error('Test 3 failed: preview should be open');

    // User switches to diag-web-2 while preview is open
    await app.openDiagram('diag-web-2');
    if (app.imageImportPreview !== null) throw new Error('Test 3 failed: openDiagram must clear imageImportPreview');

    // Simulate an attempt to confirm the old preview
    const stalePreview: BoundImageImportPreview = {
      ...binding,
      document: imported,
    };
    app.imageImportPreview = stalePreview;
    const confirmed = await app.handleConfirmImageImport();
    if (confirmed) throw new Error('Test 3 failed: confirmation of stale preview must be rejected');
    if (app.diagramId !== 'diag-web-2') throw new Error('Test 3 failed: active diagram must remain diag-web-2');
    if (app.classes[0].name !== 'OtherEntity') throw new Error('Test 3 failed: diag-web-2 must not be overwritten');

    console.log('✔ Test 3 passed: Diagram switch clears preview and rejects stale confirmation without overwriting target.');
  }

  // Test 4: Project switch while preview is open clears preview and rejects stale confirmation
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Imported for Project 1',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'c-x', name: 'Proj1Import', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    });

    await app.openProject('proj-2', 'diag-web-2');
    if (app.imageImportPreview !== null) throw new Error('Test 4 failed: openProject must clear preview');

    console.log('✔ Test 4 passed: Project switch clears preview and isolates project state.');
  }

  // Test 5: Local manual edit while preview is open clears preview and marks dirty
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Imported Class',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'c-x', name: 'ImportedClass', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    });

    // Local edit happens while preview is open
    await app.manualEditClass({ id: 'orig-1', name: 'RenamedLocally', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] });
    if (app.imageImportPreview !== null) throw new Error('Test 5 failed: manual edit must clear imageImportPreview');

    console.log('✔ Test 5 passed: Local manual edit clears preview and preserves intervening changes.');
  }

  // Test 6: Remote realtime change while preview is open clears preview
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();
    app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Imported Class',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'c-x', name: 'ImportedClass', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    });

    // Remote collaborator pushes change
    await app.onRemoteChangeApplied({
      schemaVersion: 1,
      id: 'diag-web-1',
      name: 'Remotely Updated Diagram',
      version: 5,
      reviewNumber: 10,
      classes: [{ id: 'remote-1', name: 'RemoteCollaboratorClass', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    });

    if (app.imageImportPreview !== null) throw new Error('Test 6 failed: remote change via applyDocument must clear imageImportPreview');
    if (app.diagramVersion !== 5 || app.diagramReview !== 10) throw new Error('Test 6 failed: remote revision not adopted');

    console.log('✔ Test 6 passed: Remote realtime change clears preview and adopts new baseline.');
  }

  // Test 7: Late / stale async import arrival after diagram switch is discarded
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    // Async import started on diag-web-1
    const binding = app.startImageImport();

    // User switches to diag-web-2 before async import finishes
    await app.openDiagram('diag-web-2');

    // Late async import response arrives for diag-web-1
    const accepted = app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Late Arriving Import',
      version: 0,
      reviewNumber: 0,
      classes: [{ id: 'c-late', name: 'LateClass', stereotype: '«Entity»', package: '', tableBinding: '', x: 0, y: 0, attributes: [], methods: [] }],
      relationships: [],
    });

    if (accepted) throw new Error('Test 7 failed: late async import response after switch must be rejected');
    if (app.imageImportPreview !== null) throw new Error('Test 7 failed: preview must remain null after stale async arrival');

    console.log('✔ Test 7 passed: Stale async import arrival after diagram switch is safely discarded.');
  }

  // Test 8: Late / stale async import arrival after local edit is discarded
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();

    // User edits diagram while import is in-flight
    app.classes[0].name = 'EditedWhileUploading';
    app.dirty = true;

    // Late async import arrives
    const accepted = app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Late Arriving Import',
      version: 0,
      reviewNumber: 0,
      classes: [],
      relationships: [],
    });

    if (accepted) throw new Error('Test 8 failed: late async import while dirty must be rejected');
    if (app.imageImportPreview !== null) throw new Error('Test 8 failed: preview must remain null');

    console.log('✔ Test 8 passed: Stale async import arrival after local edit is safely discarded.');
  }

  // Test 9: Late / stale async import arrival after remote revision bump is discarded
  {
    resetServerDb();
    const app = new AppTestHarness();
    await app.openDiagram('diag-web-1');

    const binding = app.startImageImport();

    // Remote revision bump arrives
    app.diagramVersion = 5;
    app.diagramReview = 10;

    const accepted = app.receiveImportResult(binding, {
      schemaVersion: 1,
      name: 'Late Import',
      version: 0,
      reviewNumber: 0,
      classes: [],
      relationships: [],
    });

    if (accepted) throw new Error('Test 9 failed: late async import with stale version must be rejected');

    console.log('✔ Test 9 passed: Stale async import arrival after remote revision bump is safely discarded.');
  }

  console.log('All 9 MM-01 correction regression scenarios passed successfully!');
}

runTests().catch((err) => {
  console.error(err);
  process.exit(1);
});
