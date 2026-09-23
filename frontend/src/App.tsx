import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ActiveView, UMLClassNode, Stereotype, UMLRelationship } from './types';
import { createDiagramDocument } from './diagram/document';
import { VoiceCommand } from './diagram/voiceCommands';
import { validateRelationshipCreation } from './diagram/relationshipHelpers';
import { downloadDiagramPng, downloadDiagramSvg } from './diagram/visualExport';
import { downloadDiagramXmi } from './diagram/xmiExport';
import { BoundImageImportPreview, confirmImageImport, ImageImportBinding, isImportPreviewCurrent } from './diagram/imageImportFlow';
import { ApiError, artifactApi, authApi, diagramApi, imageImportApi, projectApi, realtimeApi, AssignedProject, CreateProjectInput, DiagramSummary, DiagramVersion, UMLDiagramDocument } from './api/diagramApi';
import { RealtimeClient, buildRealtimeWsUrl, PresenceMember } from './api/realtime';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { CanvasView } from './components/UmlCanvas/CanvasView';
import { BackendGeneratorView } from './components/BackendGenerator/BackendGeneratorView';
import { LoginScreen } from './components/UserAccess/LoginScreen';
import { ProjectDashboard } from './components/UserAccess/ProjectDashboard';
import { CheckpointDialog } from './components/UmlCanvas/CheckpointDialog';
import { ConflictBanner } from './components/UmlCanvas/ConflictBanner';
import { es } from './i18n/es';

type AppScreen = 'login' | 'projects' | 'workspace';
type PersistenceState = 'offline' | 'saving' | 'saved' | 'error' | 'conflict';

export default function App() {
  const [screen, setScreen] = useState<AppScreen>('login');
  const [userName, setUserName] = useState('');
  const [activeProject, setActiveProject] = useState<AssignedProject | undefined>();
  const [projects, setProjects] = useState<AssignedProject[]>([]);
  const [activeView, setActiveView] = useState<ActiveView>('uml-canvas');
  const [classes, setClasses] = useState<UMLClassNode[]>([]);
  const [relationships, setRelationships] = useState<UMLRelationship[]>([]);
  const [selectedClassId, setSelectedClassId] = useState<string>('');
  const [selectedRelationshipId, setSelectedRelationshipId] = useState<string>('');
  const [diagramId, setDiagramId] = useState<string | undefined>();
  const [diagramName, setDiagramName] = useState('');
  const [diagramVersion, setDiagramVersion] = useState(0);
  const [diagramReview, setDiagramReview] = useState(0);
  const [diagrams, setDiagrams] = useState<DiagramSummary[]>([]);
  const [versions, setVersions] = useState<DiagramVersion[]>([]);
  const [persistenceStatus, setPersistenceStatus] = useState<PersistenceState>('offline');
  const [isDocumentLoading, setIsDocumentLoading] = useState(false);
  const [checkpointOpen, setCheckpointOpen] = useState(false);
  const [conflictSnapshot, setConflictSnapshot] = useState<UMLDiagramDocument | null>(null);
  const [userId, setUserId] = useState('');
  const [presenceMembers, setPresenceMembers] = useState<PresenceMember[]>([]);
  const [realtimeState, setRealtimeState] = useState<'idle' | 'connecting' | 'live' | 'unavailable'>('idle');
  const [remoteNotice, setRemoteNotice] = useState<string | null>(null);
  const [imageImportPreview, setImageImportPreview] = useState<BoundImageImportPreview | null>(null);
  const classesRef = useRef(classes);
  const relationshipsRef = useRef(relationships);
  const diagramNameRef = useRef(diagramName);
  const diagramIdRef = useRef(diagramId);
  const diagramVersionRef = useRef(diagramVersion);
  const diagramReviewRef = useRef(diagramReview);
  const userIdRef = useRef(userId);
  const realtimeClientRef = useRef<RealtimeClient | null>(null);
  const realtimeGenRef = useRef(0);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>();
  const activeProjectRef = useRef(activeProject);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>();
  const saveSequenceRef = useRef(0);
  const inflightRef = useRef(false);
  const dirtyRef = useRef(false);
  // pendingSnapshot holds the in-memory state waiting to be persisted. It is
  // the source of truth for "what to PUT on this flush", regardless of when
  // the timer was scheduled.
  const pendingSnapshotRef = useRef<{ classes: UMLClassNode[]; relationships: UMLRelationship[]; name: string } | null>(null);
  // Voice undo is deliberately isolated from normal, remote, and manual edits.
  const lastVoiceSnapshotRef = useRef<{ diagramId: string; classes: UMLClassNode[]; relationships: UMLRelationship[] } | null>(null);

  useEffect(() => { classesRef.current = classes; }, [classes]);
  useEffect(() => { relationshipsRef.current = relationships; }, [relationships]);
  useEffect(() => { diagramNameRef.current = diagramName; }, [diagramName]);
  useEffect(() => { diagramIdRef.current = diagramId; }, [diagramId]);
  useEffect(() => { diagramVersionRef.current = diagramVersion; }, [diagramVersion]);
  useEffect(() => { diagramReviewRef.current = diagramReview; }, [diagramReview]);
  useEffect(() => { userIdRef.current = userId; }, [userId]);
  useEffect(() => { activeProjectRef.current = activeProject; }, [activeProject]);

  const invalidateVoiceUndo = () => { lastVoiceSnapshotRef.current = null; };

  const refreshDiagrams = async (projectId: string) => {
    const nextDiagrams = await diagramApi.list(projectId);
    setDiagrams(nextDiagrams);
    return nextDiagrams;
  };

  const applyDocument = useCallback((document: UMLDiagramDocument) => {
    classesRef.current = document.classes;
    relationshipsRef.current = document.relationships;
    diagramNameRef.current = document.name;
    diagramIdRef.current = document.id;
    diagramVersionRef.current = document.version ?? 0;
    diagramReviewRef.current = document.reviewNumber ?? 0;
    setClasses(document.classes);
    setRelationships(document.relationships);
    setDiagramName(document.name);
    setDiagramId(document.id);
    setDiagramVersion(document.version ?? 0);
    setDiagramReview(document.reviewNumber ?? 0);
    setSelectedClassId(document.classes[0]?.id ?? '');
    setSelectedRelationshipId('');
    dirtyRef.current = false;
    pendingSnapshotRef.current = null;
  }, []);

  // persistDiagram is the single place that talks to the network for autosave.
  // It takes the latest snapshot from the refs (not from state, which can be
  // stale inside effect flushes) and surfaces every failure through the
  // canonical PersistenceState so the UI banner has exactly one source.
  const persistDiagram = useCallback(async (snapshot: { classes: UMLClassNode[]; relationships: UMLRelationship[]; name: string }) => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    const version = diagramVersionRef.current;
    const review = diagramReviewRef.current;
    if (!project || !id) return;
    if (inflightRef.current) return;
    inflightRef.current = true;
    const sequence = ++saveSequenceRef.current;
    setPersistenceStatus('saving');
    const document = createDiagramDocument(snapshot.name.trim() || 'Diagrama sin título', snapshot.classes, snapshot.relationships, id, version, review);
    const baseline = diagramVersionRef.current;
    const reviewBaseline = diagramReviewRef.current;
    try {
      const saved = await diagramApi.update(project.id, id, document);
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id) {
        // Adopt the fresh baselines first: the server already bumped review.
        diagramVersionRef.current = saved.version ?? version;
        diagramReviewRef.current = saved.reviewNumber ?? review;
        setDiagramVersion(saved.version ?? version);
        setDiagramReview(saved.reviewNumber ?? review);
        if (pendingSnapshotRef.current === snapshot) {
          // Nothing newer arrived during the PUT: adopt content, clear dirty.
          classesRef.current = saved.classes;
          relationshipsRef.current = saved.relationships;
          diagramNameRef.current = saved.name;
          diagramIdRef.current = saved.id;
          setClasses(saved.classes);
          setRelationships(saved.relationships);
          setDiagramName(saved.name);
          setDiagramId(saved.id);
          dirtyRef.current = false;
          pendingSnapshotRef.current = null;
          setPersistenceStatus('saved');
        } else {
          // Edits landed mid-PUT: keep them and re-flush on the new baseline
          // instead of discarding them as saved.
          setPersistenceStatus('saving');
          scheduleSave();
        }
        // One safe retry on a transient network failure, never on a 409
        // (which is a true concurrency conflict that the user must resolve).
      } else {
        setPersistenceStatus('saved');
      }
      void refreshDiagrams(project.id).catch(() => undefined);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        if (sequence === saveSequenceRef.current) {
          setPersistenceStatus('conflict');
          setConflictSnapshot(error.payload?.current ?? null);
        }
        return;
      }
      if (error instanceof ApiError && error.status === 401) {
        if (sequence === saveSequenceRef.current) setPersistenceStatus('error');
        return;
      }
      // One bounded retry on transient failures: networks flake, the timer
      // was already cleared, and the document is still dirty. Any other
      // persistent failure surfaces as a banner the user can clear by hand.
      // The retry only runs while the failed snapshot is still the pending
      // one; newer edits already scheduled their own flush.
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id && baseline === diagramVersionRef.current && reviewBaseline === diagramReviewRef.current && pendingSnapshotRef.current === snapshot) {
        try {
          const saved = await diagramApi.update(project.id, id, document);
          if (sequence === saveSequenceRef.current) {
            diagramVersionRef.current = saved.version ?? version;
            diagramReviewRef.current = saved.reviewNumber ?? review;
            setDiagramVersion(saved.version ?? version);
            setDiagramReview(saved.reviewNumber ?? review);
            if (pendingSnapshotRef.current === snapshot) {
              dirtyRef.current = false;
              pendingSnapshotRef.current = null;
              setPersistenceStatus('saved');
            } else {
              setPersistenceStatus('saving');
              scheduleSave();
            }
            void refreshDiagrams(project.id).catch(() => undefined);
          }
        } catch (retryError) {
          if (sequence !== saveSequenceRef.current) return;
          // A 409 on retry is still a real conflict: keep the server's
          // current document for resolution instead of a generic error.
          if (retryError instanceof ApiError && retryError.status === 409) {
            setPersistenceStatus('conflict');
            setConflictSnapshot(retryError.payload?.current ?? null);
          } else if (
            pendingSnapshotRef.current &&
            pendingSnapshotRef.current !== snapshot &&
            activeProjectRef.current?.id === project.id &&
            diagramIdRef.current === id
          ) {
            // Newer edits arrived during the failed PUTs and their timer may
            // already have been consumed by the in-flight guard: re-arm the
            // flush for the NEW pending snapshot instead of stranding it
            // dirty. The re-armed flush follows the normal rules, so a
            // repeated failure with the same pending snapshot still lands
            // on 'error' instead of looping.
            scheduleSave();
          } else {
            setPersistenceStatus('error');
          }
        }
      } else if (sequence === saveSequenceRef.current) {
        if (
          !pendingSnapshotRef.current ||
          activeProjectRef.current?.id !== project.id ||
          diagramIdRef.current !== id
        ) {
          setPersistenceStatus('error');
        } else {
          // A newer snapshot is queued but its timer may already have been
          // consumed by the in-flight guard above: re-arm the flush
          // explicitly so the edit is never stranded dirty. Same loop
          // bound as above — one extra attempt per new snapshot.
          scheduleSave();
        }
      }
    } finally {
      inflightRef.current = false;
    }
  }, []);

  const scheduleSave = useCallback((nextClasses?: UMLClassNode[], nextRelationships?: UMLRelationship[], nextName?: string) => {
    if (!activeProjectRef.current || !diagramIdRef.current) return;
    setImageImportPreview(null);
    setRemoteNotice(null);
    const snapshot = {
      classes: nextClasses ?? classesRef.current,
      relationships: nextRelationships ?? relationshipsRef.current,
      name: nextName ?? diagramNameRef.current,
    };
    pendingSnapshotRef.current = snapshot;
    dirtyRef.current = true;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    setPersistenceStatus('saving');
    saveTimerRef.current = setTimeout(() => { void persistDiagram(snapshot); }, 500);
  }, [persistDiagram]);

  const flushNow = useCallback(async () => {
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = undefined;
    }
    if (!inflightRef.current && dirtyRef.current && pendingSnapshotRef.current) {
      await persistDiagram(pendingSnapshotRef.current);
    }
  }, [persistDiagram]);

  // Open / switch / unmount: drop the timer, advance the sequence so any
  // in-flight PUT benignly no-ops, and let dirty state die with the diagram.
  const clearSaveState = useCallback(() => {
    setImageImportPreview(null);
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = undefined;
    }
    saveSequenceRef.current += 1;
    inflightRef.current = false;
    dirtyRef.current = false;
    pendingSnapshotRef.current = null;
    realtimeGenRef.current += 1;
    reconnectAttemptsRef.current = 0;
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = undefined;
    }
    realtimeClientRef.current?.disconnect();
    realtimeClientRef.current = null;
    setPresenceMembers([]);
    setRealtimeState('idle');
    setRemoteNotice(null);
  }, []);

  // Authenticated realtime: fetch a one-shot ticket over REST, then open the
  // diagram WebSocket. Presence is best-effort — a ticket/socket failure
  // marks presence unavailable but never blocks editing or autosave.
  // Every setup run owns a generation token: slow ticket fetches and late
  // socket callbacks from a previous diagram/project are dropped instead
  // of being applied to the wrong document.
  const setupRealtime = useCallback(async (project: AssignedProject, id: string) => {
    // NOTE: reconnectAttemptsRef is deliberately NOT reset here. The cap of
    // 5 must be real across reconnect loops; it resets only on a stable
    // connect (snapshot received) or on explicit open/change (clearSaveState).
    const generation = ++realtimeGenRef.current;
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = undefined;
    }
    realtimeClientRef.current?.disconnect();
    realtimeClientRef.current = null;
    setPresenceMembers([]);
    setRealtimeState('connecting');
    let ticket: string;
    try {
      const issued = await realtimeApi.ticket(project.id, id);
      ticket = issued.ticket;
    } catch {
      if (realtimeGenRef.current === generation) setRealtimeState('unavailable');
      return;
    }
    if (realtimeGenRef.current !== generation) return;
    const stillCurrent = () =>
      realtimeGenRef.current === generation &&
      activeProjectRef.current?.id === project.id &&
      diagramIdRef.current === id;
    const client = new RealtimeClient(buildRealtimeWsUrl(project.id, id, ticket), {
      onSnapshot: (snapshot) => {
        if (!stillCurrent()) return;
        // Stable connect proof: a snapshot for the current document resets
        // the reconnect budget.
        reconnectAttemptsRef.current = 0;
        setPresenceMembers(snapshot.members ?? []);
        // Adopt the snapshot baseline when it is newer and the editor has
        // no unsaved work; a dirty editor keeps its baseline and resolves
        // through the normal 409/conflict path.
        if ((snapshot.reviewNumber ?? 0) > diagramReviewRef.current && !dirtyRef.current) {
          applyDocument(snapshot.document);
        }
      },
      onJoin: (delta) => {
        if (!stillCurrent()) return;
        setPresenceMembers((current) => {
          if (current.some((m) => m.userId === delta.member.userId)) {
            return current.map((m) => (m.userId === delta.member.userId ? delta.member : m));
          }
          return [...current, delta.member];
        });
      },
      onLeave: (delta) => {
        if (!stillCurrent()) return;
        setPresenceMembers((current) => current.filter((m) => m.userId !== delta.member.userId));
      },
      onChanged: (event) => {
        // The hub already skips the originator; ignore own echoes defensively.
        if (!event || event.actorId === userIdRef.current) return;
        if (!stillCurrent()) return;
        const currentProject = activeProjectRef.current;
        const currentId = diagramIdRef.current;
        if (!currentProject || !currentId) return;
        if (dirtyRef.current) {
          // Dirty editor: surface the server document as a conflict instead
          // of overwriting unsaved work.
          void diagramApi.get(currentProject.id, currentId).then((fresh) => {
            if (!stillCurrent()) return;
            setPersistenceStatus('conflict');
            setConflictSnapshot(fresh);
            setRemoteNotice(es.canvas.remoteChangeConflict);
          }).catch(() => undefined);
          return;
        }
        void diagramApi.get(currentProject.id, currentId).then((fresh) => {
          if (!stillCurrent() || dirtyRef.current) return;
          applyDocument(fresh);
          setPersistenceStatus('saved');
          setRemoteNotice(es.canvas.remoteChangeApplied);
        }).catch(() => undefined);
      },
      onClose: () => {
        if (realtimeClientRef.current !== client) return;
        realtimeClientRef.current = null;
        if (!stillCurrent()) {
          setRealtimeState('idle');
          return;
        }
        // Unexpected close (the ticket is one-shot, so reconnect with a
        // fresh ticket). Bounded: after 5 failed attempts stay unavailable
        // until the next diagram open or explicit save flow.
        if (reconnectAttemptsRef.current >= 5) {
          setRealtimeState('unavailable');
          return;
        }
        reconnectAttemptsRef.current += 1;
        setRealtimeState('connecting');
        reconnectTimerRef.current = setTimeout(() => {
          if (stillCurrent()) void setupRealtime(project, id);
        }, 2000);
      },
    });
    realtimeClientRef.current = client;
    client.connect();
    setRealtimeState('live');
  }, [applyDocument]);

  // beforeunload / pagehide / visibilitychange are the three hooks React
  // unmount misses: a hard refresh or backgrounded tab kills the timer, so
  // we flush synchronously via navigator.sendBeacon when sendBeacon would
  // work, and otherwise await our async flush best-effort.
  useEffect(() => {
    const tryFlushBeacon = () => {
      if (!dirtyRef.current || !pendingSnapshotRef.current) return;
      const project = activeProjectRef.current;
      const id = diagramIdRef.current;
      const version = diagramVersionRef.current;
      const review = diagramReviewRef.current;
      if (!project || !id || typeof navigator === 'undefined' || typeof navigator.sendBeacon !== 'function') {
        // Fall back to a best-effort async flush. Browsers will not await
        // this on real unload, but on visibility-change the next paint can.
        void flushNow();
        return;
      }
      const payload = JSON.stringify({
        schemaVersion: 1 as const,
        id,
        version,
        reviewNumber: review,
        name: pendingSnapshotRef.current.name.trim() || 'Diagrama sin título',
        classes: pendingSnapshotRef.current.classes,
        relationships: pendingSnapshotRef.current.relationships,
      });
      try {
        navigator.sendBeacon(`${import.meta.env.VITE_API_BASE_URL ?? ''}/projects/${project.id}/diagrams/${id}`, new Blob([payload], { type: 'application/json' }));
        dirtyRef.current = false;
      } catch {
        void flushNow();
      }
    };
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!dirtyRef.current) return;
      tryFlushBeacon();
      event.preventDefault();
      event.returnValue = es.canvas.dragPanZoom;
    };
    const onVisibility = () => {
      if (document.visibilityState === 'hidden') tryFlushBeacon();
    };
    window.addEventListener('beforeunload', onBeforeUnload);
    document.addEventListener('visibilitychange', onVisibility);
    return () => { window.removeEventListener('beforeunload', onBeforeUnload); document.removeEventListener('visibilitychange', onVisibility); };
  }, [flushNow]);

  useEffect(() => () => { void flushNow(); }, [flushNow]);

  // Diagram-level realtime socket dies with the App shell: leaving to the
  // project list already disconnects via clearSaveState; this covers reload
  // and full unmount.
  useEffect(() => () => {
    realtimeClientRef.current?.disconnect();
    realtimeClientRef.current = null;
  }, []);

  const openDiagram = async (project: AssignedProject, id: string) => {
    clearSaveState();
    setIsDocumentLoading(true);
    setPersistenceStatus('offline');
    setConflictSnapshot(null);
    try {
      applyDocument(await diagramApi.get(project.id, id));
      setVersions([]);
      setPersistenceStatus('saved');
      void setupRealtime(project, id);
    } catch {
      setPersistenceStatus('error');
    } finally {
      setIsDocumentLoading(false);
    }
  };

  const openProject = async (project: AssignedProject) => {
    clearSaveState();
    setActiveProject(project);
    activeProjectRef.current = project;
    applyDocument(createDiagramDocument('', [], [], undefined, 0));
    setDiagrams([]);
    setVersions([]);
    setIsDocumentLoading(true);
    setScreen('workspace');
    try {
      const projectDiagrams = await refreshDiagrams(project.id);
      if (projectDiagrams[0]) await openDiagram(project, projectDiagrams[0].id);
      else setPersistenceStatus('offline');
    } catch {
      setPersistenceStatus('error');
    } finally {
      setIsDocumentLoading(false);
    }
  };

  const createProject = async (input: CreateProjectInput) => {
    const created = await projectApi.create(input);
    setProjects(current => [...current, created]);
    return created;
  };

  const joinProject = async (accessCode: string) => {
    const joined = await projectApi.join(accessCode);
    setProjects(current => (current.some(project => project.id === joined.id) ? current : [...current, joined]));
    return joined;
  };

  const createDiagram = async () => {
    const project = activeProjectRef.current;
    if (!project) return;
    clearSaveState();
    setIsDocumentLoading(true);
    setPersistenceStatus('saving');
    try {
      const saved = await diagramApi.create(project.id, createDiagramDocument('Diagrama sin título', [], [], undefined, 0));
      applyDocument(saved);
      setPersistenceStatus('saved');
      await refreshDiagrams(project.id);
      if (saved.id) void setupRealtime(project, saved.id);
    } catch {
      setPersistenceStatus('error');
    } finally {
      setIsDocumentLoading(false);
    }
  };

  const handleUpdateClass = (updated: UMLClassNode) => {
    invalidateVoiceUndo();
    const next = classesRef.current.map((umlClass) => umlClass.id === updated.id ? updated : umlClass);
    classesRef.current = next;
    setClasses(next);
    scheduleSave(next);
  };

  const handleFinishNodeDrag = () => { invalidateVoiceUndo(); scheduleSave(); };

  const handleSelectClass = (id: string) => {
    if (id) setSelectedRelationshipId('');
    setSelectedClassId(id);
  };

  const handleSelectRelationship = (id: string) => {
    if (id) setSelectedClassId('');
    setSelectedRelationshipId(id);
  };

  const handleUpdateRelationship = (updated: UMLRelationship) => {
    invalidateVoiceUndo();
    const next = relationshipsRef.current.map((relationship) => relationship.id === updated.id ? updated : relationship);
    relationshipsRef.current = next;
    setRelationships(next);
    scheduleSave(classesRef.current, next);
  };

  const handleDeleteRelationship = (id: string) => {
    invalidateVoiceUndo();
    const next = relationshipsRef.current.filter((relationship) => relationship.id !== id);
    relationshipsRef.current = next;
    setRelationships(next);
    if (selectedRelationshipId === id) setSelectedRelationshipId('');
    // A deleted relationship invalidates any association class attached to it,
    // so clear the stale reference before saving (the server rejects it).
    let clearedAttached = false;
    const nextClasses = classesRef.current.map((umlClass) => {
      if (umlClass.attachedRelationshipId === id) {
        clearedAttached = true;
        return { ...umlClass, attachedRelationshipId: undefined };
      }
      return umlClass;
    });
    if (clearedAttached) {
      classesRef.current = nextClasses;
      setClasses(nextClasses);
    }
    scheduleSave(classesRef.current, next);
  };

  const handleAddRelationship = (relationship: UMLRelationship) => {
    invalidateVoiceUndo();
    const next = [...relationshipsRef.current, relationship];
    relationshipsRef.current = next;
    setRelationships(next);
    scheduleSave(classesRef.current, next);
  };

  const handleAddClass = (stereotype: Stereotype) => {
    invalidateVoiceUndo();
    const newId = `entity_${Date.now()}`;
    const newName = stereotype === '«Enum»' ? 'OrderStatus' : `Entity${classes.length + 1}`;
    const newClass: UMLClassNode = {
      id: newId,
      name: newName,
      stereotype,
      package: 'com.nexus.orders',
      tableBinding: `t_${newName.toLowerCase()}`,
      x: 150 + (classes.length % 3) * 60,
      y: 200 + (classes.length % 3) * 50,
      width: 220,
      attributes: [
        { id: `id_${Date.now()}`, name: 'id', type: 'UUID', visibility: '+', isPk: true, annotations: ['@Id'] },
        { id: `name_${Date.now()}`, name: 'name', type: 'String', visibility: '+', annotations: ['@NotNull'] }
      ],
      methods: [
        { id: `meth_${Date.now()}`, name: 'validate()', returnType: 'boolean', visibility: '+' }
      ]
    };
    const next = [...classesRef.current, newClass];
    classesRef.current = next;
    setClasses(next);
    scheduleSave(next, relationshipsRef.current);
    setSelectedRelationshipId('');
    setSelectedClassId(newId);
  };

  const handleVoiceCommand = (command: VoiceCommand): { ok: boolean; message: string } => {
    const diagram = diagramIdRef.current;
    if (!diagram) return { ok: false, message: 'Seleccioná un diagrama antes de aplicar un comando de voz.' };

    const rememberVoiceState = () => {
      lastVoiceSnapshotRef.current = {
        diagramId: diagram,
        classes: JSON.parse(JSON.stringify(classesRef.current)) as UMLClassNode[],
        relationships: JSON.parse(JSON.stringify(relationshipsRef.current)) as UMLRelationship[],
      };
    };
    const matching = (name: string) => classesRef.current.filter((umlClass) => umlClass.name.localeCompare(name, 'es', { sensitivity: 'accent' }) === 0);
    const resolveMemberTarget = (name: string) => {
      const matches = matching(name);
      if (matches.length !== 1) return { error: `La clase ${name} debe existir una sola vez.` };
      if (matches[0].stereotype === '«Enum»') return { error: `No se pueden agregar miembros por voz a la enumeración ${name}.` };
      return { target: matches[0] };
    };

    if (command.kind === 'undo-voice-command') {
      const snapshot = lastVoiceSnapshotRef.current;
      if (!snapshot || snapshot.diagramId !== diagram) return { ok: false, message: 'No hay un último cambio de voz para deshacer en este diagrama.' };
      const restoredClasses = JSON.parse(JSON.stringify(snapshot.classes)) as UMLClassNode[];
      const restoredRelationships = JSON.parse(JSON.stringify(snapshot.relationships)) as UMLRelationship[];
      lastVoiceSnapshotRef.current = null;
      classesRef.current = restoredClasses;
      relationshipsRef.current = restoredRelationships;
      setClasses(restoredClasses);
      setRelationships(restoredRelationships);
      setSelectedClassId('');
      setSelectedRelationshipId('');
      scheduleSave(restoredClasses, restoredRelationships);
      return { ok: true, message: 'Último cambio realizado por voz deshecho.' };
    }

    if (command.kind === 'add-attribute' || command.kind === 'add-method') {
      const resolved = resolveMemberTarget(command.className);
      if ('error' in resolved) return { ok: false, message: resolved.error };
      const target = resolved.target;
      const duplicate = command.kind === 'add-attribute'
        ? target.attributes.some((attribute) => attribute.name.localeCompare(command.name, 'es', { sensitivity: 'accent' }) === 0)
        : target.methods.some((method) => method.name.localeCompare(command.name, 'es', { sensitivity: 'accent' }) === 0);
      if (duplicate) return { ok: false, message: `${command.kind === 'add-attribute' ? 'El atributo' : 'El método'} ${command.name} ya existe en ${target.name}.` };
      rememberVoiceState();
      const createdAt = Date.now();
      const next = classesRef.current.map((umlClass) => {
        if (umlClass.id !== target.id) return umlClass;
        return command.kind === 'add-attribute'
          ? { ...umlClass, attributes: [...umlClass.attributes, { id: `attr_${createdAt}`, name: command.name, type: command.type, visibility: '+' as const }] }
          : { ...umlClass, methods: [...umlClass.methods, { id: `method_${createdAt}`, name: command.name, returnType: command.returnType, visibility: '+' as const }] };
      });
      classesRef.current = next;
      setClasses(next);
      setSelectedRelationshipId('');
      setSelectedClassId(target.id);
      scheduleSave(next, relationshipsRef.current);
      return { ok: true, message: command.kind === 'add-attribute' ? `Atributo ${command.name} agregado a ${target.name}.` : `Método ${command.name} agregado a ${target.name}.` };
    }

    if (command.kind === 'create-class') {
      const duplicate = matching(command.name).length > 0;
      if (duplicate) return { ok: false, message: `Ya existe una clase llamada ${command.name}.` };
      rememberVoiceState();
      const createdAt = Date.now();
      const id = `entity_${createdAt}`;
      const newClass: UMLClassNode = {
        id, name: command.name, stereotype: '«Entity»', package: 'com.nexus.orders',
        tableBinding: `t_${command.name.toLowerCase().replace(/[^a-z0-9]+/g, '_')}`,
        x: 150 + (classesRef.current.length % 3) * 60, y: 200 + (classesRef.current.length % 3) * 50, width: 220,
        attributes: [{ id: `id_${createdAt}`, name: 'id', type: 'UUID', visibility: '+', isPk: true, annotations: ['@Id'] }], methods: [],
      };
      const next = [...classesRef.current, newClass];
      classesRef.current = next;
      setClasses(next);
      setSelectedRelationshipId('');
      setSelectedClassId(id);
      scheduleSave(next, relationshipsRef.current);
      return { ok: true, message: `Clase ${command.name} creada.` };
    }

    if (command.kind === 'create-relationship') {
      const sources = matching(command.sourceName);
      const targets = matching(command.targetName);
      if (sources.length !== 1 || targets.length !== 1) {
        return { ok: false, message: 'La relación requiere exactamente una clase origen y una clase destino existentes.' };
      }
      const source = sources[0];
      const target = targets[0];
      const existingMatches = relationshipsRef.current.filter((r) => r.sourceId === source.id && r.targetId === target.id);
      if (existingMatches.length > 1) {
        return { ok: false, message: 'Existe más de una relación entre esas clases; la actualización es ambigua.' };
      }

      if (existingMatches.length === 1) {
        const existing = existingMatches[0];
        const relationshipType = command.type ?? existing.type;
        const otherRelationships = relationshipsRef.current.filter((r) => r.id !== existing.id);
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
        const next = relationshipsRef.current.map((r) => (r.id === updated.id ? updated : r));
        relationshipsRef.current = next;
        setRelationships(next);
        setSelectedClassId('');
        setSelectedRelationshipId(updated.id);
        scheduleSave(classesRef.current, next);
        return { ok: true, message: `Relación actualizada: ${command.sourceName} con ${command.targetName}.` };
      }

      const relationshipType = command.type ?? 'association';
      const validation = validateRelationshipCreation(source.id, target.id, relationshipType, relationshipsRef.current, source, target);
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
      const next = [...relationshipsRef.current, relationship];
      relationshipsRef.current = next;
      setRelationships(next);
      setSelectedClassId('');
      setSelectedRelationshipId(relationship.id);
      scheduleSave(classesRef.current, next);
      return { ok: true, message: `Relación creada: ${command.sourceName} con ${command.targetName}.` };
    }

    return { ok: false, message: 'Comando no reconocido.' };
  };

  const handleAddAssociationClass = () => {
    invalidateVoiceUndo();
    const newId = `association_${Date.now()}`;
    const newClass: UMLClassNode = {
      id: newId,
      name: `AssocClass${classes.length + 1}`,
      stereotype: '«Entity»',
      package: 'com.nexus.orders',
      tableBinding: `t_assoc_class${classes.length + 1}`,
      x: 150 + (classes.length % 3) * 60,
      y: 200 + (classes.length % 3) * 50,
      width: 220,
      attributes: [
        { id: `id_${Date.now()}`, name: 'id', type: 'UUID', visibility: '+', isPk: true, annotations: ['@Id'] },
        { id: `name_${Date.now()}`, name: 'name', type: 'String', visibility: '+', annotations: ['@NotNull'] }
      ],
      methods: [
        { id: `meth_${Date.now()}`, name: 'validate()', returnType: 'boolean', visibility: '+' }
      ],
      isAssociationClass: true
    };
    const next = [...classesRef.current, newClass];
    classesRef.current = next;
    setClasses(next);
    scheduleSave(next, relationshipsRef.current);
    setSelectedRelationshipId('');
    setSelectedClassId(newId);
  };

  const handleDeleteClass = (id: string) => {
    invalidateVoiceUndo();
    const nextClasses = classesRef.current.filter(c => c.id !== id);
    const nextRelationships = relationshipsRef.current.filter(r => r.sourceId !== id && r.targetId !== id);
    const remainingRelIds = new Set(nextRelationships.map(r => r.id));
    const nextClassesFixed = nextClasses.map((umlClass) =>
      umlClass.attachedRelationshipId && !remainingRelIds.has(umlClass.attachedRelationshipId)
        ? { ...umlClass, attachedRelationshipId: undefined }
        : umlClass,
    );
    classesRef.current = nextClassesFixed;
    relationshipsRef.current = nextRelationships;
    setClasses(nextClassesFixed);
    setRelationships(nextRelationships);
    scheduleSave(nextClassesFixed, nextRelationships);
    if (selectedClassId === id) {
      setSelectedClassId('');
    }
    setSelectedRelationshipId('');
  };

  const handleRenameDiagram = (name: string) => {
    invalidateVoiceUndo();
    diagramNameRef.current = name;
    setDiagramName(name);
    scheduleSave(classesRef.current, relationshipsRef.current, name);
  };

  const loadVersions = useCallback(async () => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    try { setVersions(await diagramApi.versions(project.id, id)); } catch { setPersistenceStatus('error'); }
  }, []);

  const restoreVersion = async (versionNumber: number) => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    clearSaveState();
    setPersistenceStatus('saving');
    try {
      applyDocument(await diagramApi.restore(project.id, id, versionNumber));
      setPersistenceStatus('saved');
      await Promise.all([refreshDiagrams(project.id), loadVersions()]);
      // Restored content is a new baseline: reconnect with a fresh ticket
      // so presence and change events track the restored document.
      void setupRealtime(project, id);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setPersistenceStatus('conflict');
        setConflictSnapshot(error.payload?.current ?? null);
      } else {
        setPersistenceStatus('error');
      }
    }
  };

  const createCheckpoint = useCallback(async (message: string) => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    if (saveTimerRef.current) { clearTimeout(saveTimerRef.current); saveTimerRef.current = undefined; }
    saveSequenceRef.current += 1;
    setPersistenceStatus('saving');
    try {
      const doc = createDiagramDocument(
        diagramNameRef.current.trim() || 'Diagrama sin título',
        classesRef.current,
        relationshipsRef.current,
        id,
        diagramVersionRef.current,
        diagramReviewRef.current,
      );
      const version = await diagramApi.checkpoint(project.id, id, doc, message || undefined);
      diagramVersionRef.current = version.document.version;
      diagramReviewRef.current = version.document.reviewNumber ?? version.reviewNumber ?? diagramReviewRef.current;
      setDiagramVersion(version.document.version);
      setDiagramReview(diagramReviewRef.current);
      dirtyRef.current = false;
      pendingSnapshotRef.current = null;
      setPersistenceStatus('saved');
      await Promise.all([refreshDiagrams(project.id), loadVersions()]);
      // A checkpoint bumps the baseline: reconnect with a fresh ticket so
      // presence and change events track the checkpointed document.
      void setupRealtime(project, id);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setPersistenceStatus('conflict');
        setConflictSnapshot(error.payload?.current ?? null);
      } else {
        setPersistenceStatus('error');
      }
    }
  }, [loadVersions, setupRealtime]);

  const resolveConflict = useCallback(async (keepMine: boolean) => {
    if (!conflictSnapshot) return;
    if (keepMine) {
      // Accept the cost: surface the server baselines as the new baseline so
      // autosave resumes, and let the user know their edit will overwrite it
      // (or they can copy changes manually and recompute their version).
      diagramVersionRef.current = conflictSnapshot.version ?? diagramVersionRef.current;
      diagramReviewRef.current = conflictSnapshot.reviewNumber ?? diagramReviewRef.current;
      setDiagramVersion(diagramVersionRef.current);
      setDiagramReview(diagramReviewRef.current);
      setConflictSnapshot(null);
      setPersistenceStatus('saved');
      scheduleSave();
    } else {
      // Discard local edits: re-apply the server document verbatim.
      applyDocument({ ...conflictSnapshot });
      setConflictSnapshot(null);
      setPersistenceStatus('saved');
      await loadVersions();
    }
  }, [conflictSnapshot, applyDocument, loadVersions, scheduleSave]);

  // Build the current portable document and ask the server for a real JHipster
  // backend zip, then trigger its download with the exact server filename.
  const generateBackendArtifact = async () => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    const document = createDiagramDocument(
      diagramNameRef.current.trim() || 'Diagrama sin título',
      classesRef.current,
      relationshipsRef.current,
      id,
      diagramVersionRef.current,
      diagramReviewRef.current,
    );
    const response = await artifactApi.generate(project.id, id, document);
    const url = window.URL.createObjectURL(response.blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = response.filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
  };

  const handleExport = async (type: 'svg' | 'png' | 'xmi' | 'plantuml' | 'zip') => {
    if (type === 'svg') {
      downloadDiagramSvg(diagramName, classes, relationships);
      return;
    }

    if (type === 'png') {
      await downloadDiagramPng(diagramName, classes, relationships);
      return;
    }

    if (type === 'zip') {
      // Menu shortcut: generate with server defaults. The dedicated
      // BackendGeneratorView surfaces detailed configuration and results.
      try {
        await generateBackendArtifact();
      } catch (error) {
        console.error('Backend artifact generation failed', error);
      }
      return;
    }

    if (type === 'plantuml') {
      let puml = '@startuml\n';
      classes.forEach(c => {
        puml += `class ${c.name} {\n`;
        c.attributes.forEach(a => {
          puml += `  ${a.visibility} ${a.name} : ${a.type}\n`;
        });
        c.methods.forEach(m => {
          puml += `  ${m.visibility} ${m.name} : ${m.returnType}\n`;
        });
        puml += '}\n';
      });
      puml += '@enduml\n';
      const blob = new Blob([puml], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'domain_model.puml';
      a.click();
      URL.revokeObjectURL(url);
    } else if (type === 'xmi') {
      downloadDiagramXmi(diagramName, classes, relationships);
    }
  };

  const persistenceLabel = useMemo(() => {
    switch (persistenceStatus) {
      case 'saving': return es.canvas.saving;
      case 'saved': return es.canvas.saved;
      case 'error': return es.canvas.saveFailed;
      case 'conflict': return es.canvas.persistenceConflict;
      case 'offline': return es.canvas.persistenceOffline;
      default: return '';
    }
  }, [persistenceStatus]);

  const presenceLabel = useMemo(() => {
    if (!diagramId) return undefined;
    switch (realtimeState) {
      case 'live': return es.canvas.presenceOnline(presenceMembers.length);
      case 'connecting': return es.canvas.presenceConnecting;
      default: return es.canvas.presenceUnavailable;
    }
  }, [diagramId, realtimeState, presenceMembers.length]);

   const importDiagramImage = useCallback(async (file: File) => {
     const project = activeProjectRef.current;
     const id = diagramIdRef.current;
     if (!project || !id) return;
     const binding: ImageImportBinding = {
       sourceProjectId: project.id,
       sourceDiagramId: id,
       sourceVersion: diagramVersionRef.current,
       sourceReviewNumber: diagramReviewRef.current,
     };
     try {
       const imported = await imageImportApi.import(project.id, id, file);
       const isStillCurrent = isImportPreviewCurrent(binding, {
         projectId: activeProjectRef.current?.id,
         diagramId: diagramIdRef.current,
         version: diagramVersionRef.current,
         reviewNumber: diagramReviewRef.current,
         isDirty: dirtyRef.current,
       });
       if (!isStillCurrent) return;

       setImageImportPreview({
         ...binding,
         document: imported,
       });
     } catch {
       // Network or upload error
     }
   }, []);

   const handleConfirmImageImport = useCallback(() => {
     if (!imageImportPreview || !diagramIdRef.current || !activeProjectRef.current) {
       setImageImportPreview(null);
       return;
     }
     const currentProjectId = activeProjectRef.current.id;
     const currentDiagramId = diagramIdRef.current;
     const currentVersion = diagramVersionRef.current;
     const currentReviewNumber = diagramReviewRef.current;

     const confirmed = confirmImageImport({
       currentProjectId,
       currentDiagramId,
       currentVersion,
       currentReviewNumber,
       isDirty: dirtyRef.current,
       preview: imageImportPreview,
     });

     if (!confirmed) {
       setImageImportPreview(null);
       return;
     }

     invalidateVoiceUndo();
     classesRef.current = confirmed.classes;
     relationshipsRef.current = confirmed.relationships;
     diagramNameRef.current = confirmed.name;
     diagramIdRef.current = confirmed.id;
     diagramVersionRef.current = confirmed.version;
     diagramReviewRef.current = confirmed.reviewNumber;

     setClasses(confirmed.classes);
     setRelationships(confirmed.relationships);
     setDiagramName(confirmed.name);
     setDiagramId(confirmed.id);
     setDiagramVersion(confirmed.version);
     setDiagramReview(confirmed.reviewNumber);
     setSelectedClassId(confirmed.classes[0]?.id ?? '');
     setSelectedRelationshipId('');
     setImageImportPreview(null);

     scheduleSave(confirmed.classes, confirmed.relationships, confirmed.name);
   }, [imageImportPreview, scheduleSave]);

   if (screen === 'login') {
     return <LoginScreen onContinue={async (email, password) => { const login = await authApi.login(email, password); authApi.setToken(login.accessToken); setUserName(login.displayName); setUserId(login.userId); setProjects(await projectApi.list()); setScreen('projects'); }} />;
   }

   if (screen === 'projects') {
     return <ProjectDashboard userName={userName} projects={projects} onSignOut={() => { authApi.setToken(); setScreen('login'); }} onOpenProject={(project) => { void openProject(project); }} onCreateProject={createProject} onJoinProject={joinProject} />;
   }

   return (
    <div className="min-h-screen bg-[#0f131c] text-[#dfe2ee] selection:bg-[#10b981] selection:text-[#00422b]">
      {/* Fixed Top Header */}
      <Header
        activeView={activeView}
        onSelectView={setActiveView}
        onExport={handleExport}
        projectName={activeProject?.name}
        onBackToProjects={() => { clearSaveState(); setScreen('projects'); }}
      />

      {/* Fixed Left Sidebar */}
      <Sidebar
        activeView={activeView}
        onSelectView={setActiveView}
      />

      {/* Main Viewport Container */}
      <div className="pl-64 pt-16 min-h-screen">
        {activeView === 'uml-canvas' && (
          <>
            <CanvasView
              classes={classes}
              relationships={relationships}
              selectedClassId={selectedClassId}
              selectedRelationshipId={selectedRelationshipId}
              onSelectClass={handleSelectClass}
              onSelectRelationship={handleSelectRelationship}
              onUpdateRelationship={handleUpdateRelationship}
              onDeleteRelationship={handleDeleteRelationship}
              onUpdateClass={handleUpdateClass}
              onFinishNodeDrag={handleFinishNodeDrag}
              onPersistChange={() => { invalidateVoiceUndo(); scheduleSave(); }}
              onAddClass={handleAddClass}
              onAddAssociationClass={handleAddAssociationClass}
              onAddRelationship={handleAddRelationship}
              onDeleteClass={handleDeleteClass}
              onSwitchToBackend={() => setActiveView('backend-db-generator')}
              diagramId={diagramId}
              diagramName={diagramName}
              diagrams={diagrams}
              versions={versions}
              persistenceStatus={persistenceStatus === 'conflict' ? 'error' : (persistenceStatus === 'offline' ? 'idle' : persistenceStatus)}
              persistenceLabel={persistenceLabel}
              presenceMembers={presenceMembers}
              presenceLabel={presenceLabel}
              isDocumentLoading={isDocumentLoading}
              onOpenDiagram={(id) => { if (activeProject) void openDiagram(activeProject, id); }}
              onCreateDiagram={() => { void createDiagram(); }}
              onRenameDiagram={handleRenameDiagram}
              onLoadVersions={() => { void loadVersions(); }}
              onRestoreVersion={(versionNumber) => { void restoreVersion(versionNumber); }}
              onCreateCheckpoint={() => setCheckpointOpen(true)}
              onImportImage={importDiagramImage}
              onVoiceCommand={handleVoiceCommand}
            />
            {imageImportPreview && (
              <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
                <div className="w-[32rem] border border-[#4cd7f6] bg-[#1c2028] p-5 font-mono text-sm text-[#dfe2ee] shadow-2xl">
                  <h2 className="text-lg text-[#4cd7f6]">Imported UML preview</h2>
                  <p className="mt-2">{imageImportPreview.document.name}: {imageImportPreview.document.classes.length} classes, {imageImportPreview.document.relationships.length} relationships.</p>
                  <p className="mt-2 text-xs text-[#bbcabf]">Review the detected diagram before replacing the current one.</p>
                  <div className="mt-5 flex justify-end gap-3"><button type="button" className="border border-[#86948a] px-3 py-2" onClick={() => setImageImportPreview(null)}>Cancel</button><button type="button" className="border border-[#4de3a3] px-3 py-2 text-[#4de3a3]" onClick={handleConfirmImageImport}>Replace current diagram</button></div>
                </div>
              </div>
            )}
            {remoteNotice && !conflictSnapshot && (
              <div role="status" className="fixed bottom-4 left-1/2 z-40 w-[36rem] -translate-x-1/2 border border-[#4cd7f6] bg-[#1c2028] p-3 font-mono text-xs text-[#dfe2ee] shadow-xl">
                <p>{remoteNotice}</p>
                <div className="mt-2 flex items-center justify-end">
                  <button type="button" onClick={() => setRemoteNotice(null)} className="px-3 py-1 border border-[#4cd7f6] text-[#4cd7f6] hover:bg-[#4cd7f6] hover:text-[#1c2028] transition-colors uppercase font-bold">
                    {es.canvas.close}
                  </button>
                </div>
              </div>
            )}
            {conflictSnapshot && (
              <ConflictBanner
                conflicting={conflictSnapshot}
                localName={diagramName}
                onResolve={(keepMine) => { void resolveConflict(keepMine); }}
              />
            )}
            {checkpointOpen && (
              <CheckpointDialog
                busy={persistenceStatus === 'saving'}
                onCancel={() => setCheckpointOpen(false)}
                onSubmit={(message) => { setCheckpointOpen(false); void createCheckpoint(message); }}
              />
            )}
          </>
        )}

        {activeView === 'backend-db-generator' && (
          <BackendGeneratorView
            projectId={activeProject?.id}
            diagramId={diagramId}
            document={createDiagramDocument(
              diagramName.trim() || 'Diagrama sin título',
              classes,
              relationships,
              diagramId,
              diagramVersion,
              diagramReview,
            )}
          />
        )}
      </div>
    </div>
  );
}
