import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ActiveView, UMLClassNode, Stereotype, JpaStrategy, UMLRelationship } from './types';
import { createDiagramDocument } from './diagram/document';
import { downloadDiagramPng, downloadDiagramSvg } from './diagram/visualExport';
import { downloadDiagramXmi } from './diagram/xmiExport';
import { ApiError, authApi, diagramApi, projectApi, AssignedProject, CreateProjectInput, DiagramSummary, DiagramVersion, UMLDiagramDocument } from './api/diagramApi';
import { generateAllCodeFiles } from './data/codeGenerator';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { CanvasView } from './components/UmlCanvas/CanvasView';
import { BackendGeneratorView } from './components/BackendGenerator/BackendGeneratorView';
import { LoginScreen } from './components/UserAccess/LoginScreen';
import { ProjectDashboard } from './components/UserAccess/ProjectDashboard';
import { CheckpointDialog } from './components/UmlCanvas/CheckpointDialog';
import { ConflictBanner } from './components/UmlCanvas/ConflictBanner';
import JSZip from 'jszip';
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
  const [strategy, setStrategy] = useState<JpaStrategy>('JOINED');
  const [diagramId, setDiagramId] = useState<string | undefined>();
  const [diagramName, setDiagramName] = useState('');
  const [diagramVersion, setDiagramVersion] = useState(0);
  const [diagrams, setDiagrams] = useState<DiagramSummary[]>([]);
  const [versions, setVersions] = useState<DiagramVersion[]>([]);
  const [persistenceStatus, setPersistenceStatus] = useState<PersistenceState>('offline');
  const [isDocumentLoading, setIsDocumentLoading] = useState(false);
  const [checkpointOpen, setCheckpointOpen] = useState(false);
  const [conflictSnapshot, setConflictSnapshot] = useState<UMLDiagramDocument | null>(null);
  const classesRef = useRef(classes);
  const relationshipsRef = useRef(relationships);
  const diagramNameRef = useRef(diagramName);
  const diagramIdRef = useRef(diagramId);
  const diagramVersionRef = useRef(diagramVersion);
  const activeProjectRef = useRef(activeProject);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>();
  const saveSequenceRef = useRef(0);
  const inflightRef = useRef(false);
  const dirtyRef = useRef(false);
  // pendingSnapshot holds the in-memory state waiting to be persisted. It is
  // the source of truth for "what to PUT on this flush", regardless of when
  // the timer was scheduled.
  const pendingSnapshotRef = useRef<{ classes: UMLClassNode[]; relationships: UMLRelationship[]; name: string } | null>(null);

  useEffect(() => { classesRef.current = classes; }, [classes]);
  useEffect(() => { relationshipsRef.current = relationships; }, [relationships]);
  useEffect(() => { diagramNameRef.current = diagramName; }, [diagramName]);
  useEffect(() => { diagramIdRef.current = diagramId; }, [diagramId]);
  useEffect(() => { diagramVersionRef.current = diagramVersion; }, [diagramVersion]);
  useEffect(() => { activeProjectRef.current = activeProject; }, [activeProject]);

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
    setClasses(document.classes);
    setRelationships(document.relationships);
    setDiagramName(document.name);
    setDiagramId(document.id);
    setDiagramVersion(document.version ?? 0);
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
    if (!project || !id) return;
    if (inflightRef.current) return;
    inflightRef.current = true;
    const sequence = ++saveSequenceRef.current;
    setPersistenceStatus('saving');
    const document = createDiagramDocument(snapshot.name.trim() || 'Diagrama sin título', snapshot.classes, snapshot.relationships, id, version);
    const baseline = diagramVersionRef.current;
    try {
      const saved = await diagramApi.update(project.id, id, document);
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id) {
        classesRef.current = saved.classes;
        relationshipsRef.current = saved.relationships;
        diagramNameRef.current = saved.name;
        diagramIdRef.current = saved.id;
        diagramVersionRef.current = saved.version ?? version;
        setClasses(saved.classes);
        setRelationships(saved.relationships);
        setDiagramName(saved.name);
        setDiagramId(saved.id);
        setDiagramVersion(saved.version ?? version);
        dirtyRef.current = false;
        pendingSnapshotRef.current = null;
        setPersistenceStatus('saved');
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
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id && baseline === diagramVersionRef.current) {
        try {
          const saved = await diagramApi.update(project.id, id, document);
          if (sequence === saveSequenceRef.current) {
            diagramVersionRef.current = saved.version ?? version;
            setDiagramVersion(saved.version ?? version);
            dirtyRef.current = false;
            pendingSnapshotRef.current = null;
            setPersistenceStatus('saved');
            void refreshDiagrams(project.id).catch(() => undefined);
          }
        } catch {
          if (sequence === saveSequenceRef.current) setPersistenceStatus('error');
        }
      } else if (sequence === saveSequenceRef.current) {
        setPersistenceStatus('error');
      }
    } finally {
      inflightRef.current = false;
    }
  }, []);

  const scheduleSave = useCallback((nextClasses?: UMLClassNode[], nextRelationships?: UMLRelationship[], nextName?: string) => {
    if (!activeProjectRef.current || !diagramIdRef.current) return;
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
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = undefined;
    }
    saveSequenceRef.current += 1;
    inflightRef.current = false;
    dirtyRef.current = false;
    pendingSnapshotRef.current = null;
  }, []);

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

  const openDiagram = async (project: AssignedProject, id: string) => {
    clearSaveState();
    setIsDocumentLoading(true);
    setPersistenceStatus('offline');
    setConflictSnapshot(null);
    try {
      applyDocument(await diagramApi.get(project.id, id));
      setVersions([]);
      setPersistenceStatus('saved');
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
    } catch {
      setPersistenceStatus('error');
    } finally {
      setIsDocumentLoading(false);
    }
  };

  const handleUpdateClass = (updated: UMLClassNode) => {
    const next = classesRef.current.map((umlClass) => umlClass.id === updated.id ? updated : umlClass);
    classesRef.current = next;
    setClasses(next);
    scheduleSave(next);
  };

  const handleFinishNodeDrag = () => scheduleSave();

  const handleSelectClass = (id: string) => {
    if (id) setSelectedRelationshipId('');
    setSelectedClassId(id);
  };

  const handleSelectRelationship = (id: string) => {
    if (id) setSelectedClassId('');
    setSelectedRelationshipId(id);
  };

  const handleUpdateRelationship = (updated: UMLRelationship) => {
    const next = relationshipsRef.current.map((relationship) => relationship.id === updated.id ? updated : relationship);
    relationshipsRef.current = next;
    setRelationships(next);
    scheduleSave(classesRef.current, next);
  };

  const handleDeleteRelationship = (id: string) => {
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
    const next = [...relationshipsRef.current, relationship];
    relationshipsRef.current = next;
    setRelationships(next);
    scheduleSave(classesRef.current, next);
  };

  const handleAddClass = (stereotype: Stereotype) => {
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

  const handleAddAssociationClass = () => {
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
    if (classes.length <= 1) return;
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
      const remaining = classesRef.current.filter(c => c.id !== id);
      if (remaining.length > 0) setSelectedClassId(remaining[0].id);
    }
    if (selectedRelationshipId && nextRelationships.every(r => r.id !== selectedRelationshipId)) setSelectedRelationshipId('');
  };

  const handleRenameDiagram = (name: string) => {
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
      );
      const version = await diagramApi.checkpoint(project.id, id, doc, message || undefined);
      diagramVersionRef.current = version.document.version;
      setDiagramVersion(version.document.version);
      dirtyRef.current = false;
      pendingSnapshotRef.current = null;
      setPersistenceStatus('saved');
      await Promise.all([refreshDiagrams(project.id), loadVersions()]);
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setPersistenceStatus('conflict');
        setConflictSnapshot(error.payload?.current ?? null);
      } else {
        setPersistenceStatus('error');
      }
    }
  }, [loadVersions]);

  const resolveConflict = useCallback(async (keepMine: boolean) => {
    if (!conflictSnapshot) return;
    if (keepMine) {
      // Accept the cost: surface the server version as the new baseline so
      // autosave resumes, and let the user know their edit will overwrite it
      // (or they can copy changes manually and recompute their version).
      diagramVersionRef.current = conflictSnapshot.version ?? diagramVersionRef.current;
      setDiagramVersion(diagramVersionRef.current);
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

  const handleExport = async (type: 'svg' | 'png' | 'xmi' | 'plantuml' | 'sql' | 'zip') => {
    if (type === 'svg') {
      downloadDiagramSvg(diagramName, classes, relationships);
      return;
    }

    if (type === 'png') {
      await downloadDiagramPng(diagramName, classes, relationships);
      return;
    }

    const codeFiles = generateAllCodeFiles(classes, strategy);

    if (type === 'zip') {
      const zip = new JSZip();
      codeFiles.forEach(file => {
        zip.file(file.path, file.content);
      });
      zip.file('README.md', `# Spring Boot E-Commerce Core\nGenerated with AI UML v2.4\n`);
      const blob = await zip.generateAsync({ type: 'blob' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'spring-boot-ecommerce-core.zip';
      a.click();
      URL.revokeObjectURL(url);
    } else if (type === 'sql') {
      const sqlFile = codeFiles.find(f => f.id === 'sql-ddl') || codeFiles[1];
      const blob = new Blob([sqlFile.content], { type: 'text/sql' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'V1__init_schema.sql';
      a.click();
      URL.revokeObjectURL(url);
    } else if (type === 'plantuml') {
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

  if (screen === 'login') {
    return <LoginScreen onContinue={async (email, password) => { const login = await authApi.login(email, password); authApi.setToken(login.accessToken); setUserName(login.displayName); setProjects(await projectApi.list()); setScreen('projects'); }} />;
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
              onPersistChange={() => scheduleSave()}
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
              isDocumentLoading={isDocumentLoading}
              onOpenDiagram={(id) => { if (activeProject) void openDiagram(activeProject, id); }}
              onCreateDiagram={() => { void createDiagram(); }}
              onRenameDiagram={handleRenameDiagram}
              onLoadVersions={() => { void loadVersions(); }}
              onRestoreVersion={(versionNumber) => { void restoreVersion(versionNumber); }}
              onCreateCheckpoint={() => setCheckpointOpen(true)}
            />
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
            classes={classes}
            strategy={strategy}
            onStrategyChange={setStrategy}
            onDownloadZip={() => handleExport('zip')}
          />
        )}
      </div>
    </div>
  );
}
