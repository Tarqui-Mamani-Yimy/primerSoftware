import { useEffect, useRef, useState } from 'react';
import { ActiveView, UMLClassNode, Stereotype, JpaStrategy, UMLRelationship } from './types';
import { createDiagramDocument } from './diagram/document';
import { downloadDiagramPng, downloadDiagramSvg } from './diagram/visualExport';
import { downloadDiagramXmi } from './diagram/xmiExport';
import { authApi, diagramApi, projectApi, AssignedProject, CreateProjectInput, DiagramSummary, DiagramVersion } from './api/diagramApi';
import { generateAllCodeFiles } from './data/codeGenerator';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { CanvasView } from './components/UmlCanvas/CanvasView';
import { BackendGeneratorView } from './components/BackendGenerator/BackendGeneratorView';
import { LoginScreen } from './components/UserAccess/LoginScreen';
import { ProjectDashboard } from './components/UserAccess/ProjectDashboard';
import JSZip from 'jszip';

type AppScreen = 'login' | 'projects' | 'workspace';

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
  const [diagrams, setDiagrams] = useState<DiagramSummary[]>([]);
  const [versions, setVersions] = useState<DiagramVersion[]>([]);
  const [persistenceStatus, setPersistenceStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [isDocumentLoading, setIsDocumentLoading] = useState(false);
  const classesRef = useRef(classes);
  const relationshipsRef = useRef(relationships);
  const diagramNameRef = useRef(diagramName);
  const diagramIdRef = useRef(diagramId);
  const activeProjectRef = useRef(activeProject);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>();
  const saveSequenceRef = useRef(0);

  useEffect(() => { classesRef.current = classes; }, [classes]);
  useEffect(() => { relationshipsRef.current = relationships; }, [relationships]);
  useEffect(() => { diagramNameRef.current = diagramName; }, [diagramName]);
  useEffect(() => { diagramIdRef.current = diagramId; }, [diagramId]);
  useEffect(() => { activeProjectRef.current = activeProject; }, [activeProject]);
  useEffect(() => () => { if (saveTimerRef.current) clearTimeout(saveTimerRef.current); }, []);

  const refreshDiagrams = async (projectId: string) => {
    const nextDiagrams = await diagramApi.list(projectId);
    setDiagrams(nextDiagrams);
    return nextDiagrams;
  };

  const applyDocument = (document: ReturnType<typeof createDiagramDocument>) => {
    classesRef.current = document.classes;
    relationshipsRef.current = document.relationships;
    diagramNameRef.current = document.name;
    diagramIdRef.current = document.id;
    setClasses(document.classes);
    setRelationships(document.relationships);
    setDiagramName(document.name);
    setDiagramId(document.id);
    setSelectedClassId(document.classes[0]?.id ?? '');
    setSelectedRelationshipId('');
  };

  const persistDiagram = async (nextClasses: UMLClassNode[], nextRelationships: UMLRelationship[], nextName: string) => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    const sequence = ++saveSequenceRef.current;
    try {
      const saved = await diagramApi.update(project.id, id, createDiagramDocument(nextName.trim() || 'Diagrama sin título', nextClasses, nextRelationships, id));
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id) {
        applyDocument(saved);
        setPersistenceStatus('saved');
        void refreshDiagrams(project.id).catch(() => undefined);
      }
    } catch {
      if (sequence === saveSequenceRef.current && activeProjectRef.current?.id === project.id && diagramIdRef.current === id) setPersistenceStatus('error');
    }
  };

  const scheduleSave = (nextClasses = classesRef.current, nextRelationships = relationshipsRef.current, nextName = diagramNameRef.current) => {
    if (!activeProjectRef.current || !diagramIdRef.current) return;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    setPersistenceStatus('saving');
    saveTimerRef.current = setTimeout(() => { void persistDiagram(nextClasses, nextRelationships, nextName); }, 500);
  };

  const openDiagram = async (project: AssignedProject, id: string) => {
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveSequenceRef.current += 1;
    setIsDocumentLoading(true);
    setPersistenceStatus('idle');
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
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveSequenceRef.current += 1;
    setActiveProject(project);
    activeProjectRef.current = project;
    applyDocument(createDiagramDocument('', [], []));
    setDiagrams([]);
    setVersions([]);
    setIsDocumentLoading(true);
    setScreen('workspace');
    try {
      const projectDiagrams = await refreshDiagrams(project.id);
      if (projectDiagrams[0]) await openDiagram(project, projectDiagrams[0].id);
      else setPersistenceStatus('idle');
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
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveSequenceRef.current += 1;
    setIsDocumentLoading(true);
    setPersistenceStatus('saving');
    try {
      const saved = await diagramApi.create(project.id, createDiagramDocument('Diagrama sin título', [], []));
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

  const handleDeleteClass = (id: string) => {
    if (classes.length <= 1) return;
    const nextClasses = classesRef.current.filter(c => c.id !== id);
    const nextRelationships = relationshipsRef.current.filter(r => r.sourceId !== id && r.targetId !== id);
    classesRef.current = nextClasses;
    relationshipsRef.current = nextRelationships;
    setClasses(nextClasses);
    setRelationships(nextRelationships);
    scheduleSave(nextClasses, nextRelationships);
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

  const loadVersions = async () => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    try { setVersions(await diagramApi.versions(project.id, id)); } catch { setPersistenceStatus('error'); }
  };

  const restoreVersion = async (versionNumber: number) => {
    const project = activeProjectRef.current;
    const id = diagramIdRef.current;
    if (!project || !id) return;
    setPersistenceStatus('saving');
    try {
      applyDocument(await diagramApi.restore(project.id, id, versionNumber));
      setPersistenceStatus('saved');
      await Promise.all([refreshDiagrams(project.id), loadVersions()]);
    } catch { setPersistenceStatus('error'); }
  };

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
        onBackToProjects={() => setScreen('projects')}
      />

      {/* Fixed Left Sidebar */}
      <Sidebar
        activeView={activeView}
        onSelectView={setActiveView}
      />

      {/* Main Viewport Container */}
      <div className="pl-64 pt-16 min-h-screen">
        {activeView === 'uml-canvas' && (
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
            onAddRelationship={handleAddRelationship}
            onDeleteClass={handleDeleteClass}
            onSwitchToBackend={() => setActiveView('backend-db-generator')}
            diagramId={diagramId}
            diagramName={diagramName}
            diagrams={diagrams}
            versions={versions}
            persistenceStatus={persistenceStatus}
            isDocumentLoading={isDocumentLoading}
            onOpenDiagram={(id) => { if (activeProject) void openDiagram(activeProject, id); }}
            onCreateDiagram={() => { void createDiagram(); }}
            onRenameDiagram={handleRenameDiagram}
            onLoadVersions={() => { void loadVersions(); }}
            onRestoreVersion={(versionNumber) => { void restoreVersion(versionNumber); }}
          />
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
