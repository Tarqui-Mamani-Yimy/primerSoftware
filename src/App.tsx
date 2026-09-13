import { useState } from 'react';
import { ActiveView, UMLClassNode, Stereotype, JpaStrategy } from './types';
import { INITIAL_CLASSES, INITIAL_RELATIONSHIPS } from './data/initialModel';
import { generateAllCodeFiles } from './data/codeGenerator';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { CanvasView } from './components/UmlCanvas/CanvasView';
import { BackendGeneratorView } from './components/BackendGenerator/BackendGeneratorView';
import { GitDiffView } from './components/GitDiff/GitDiffView';
import { TeamRoomView } from './components/TeamRoom/TeamRoomView';
import { AiConsoleView } from './components/AiConsole/AiConsoleView';
import { FloatingAssistant } from './components/FloatingAssistant/FloatingAssistant';
import JSZip from 'jszip';

export default function App() {
  const [activeView, setActiveView] = useState<ActiveView>('uml-canvas');
  const [classes, setClasses] = useState<UMLClassNode[]>(INITIAL_CLASSES);
  const [relationships, setRelationships] = useState(INITIAL_RELATIONSHIPS);
  const [selectedClassId, setSelectedClassId] = useState<string>('order');
  const [strategy, setStrategy] = useState<JpaStrategy>('JOINED');

  const handleUpdateClass = (updated: UMLClassNode) => {
    setClasses(prev => prev.map(c => (c.id === updated.id ? updated : c)));
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
    setClasses(prev => [...prev, newClass]);
    setSelectedClassId(newId);
  };

  const handleDeleteClass = (id: string) => {
    if (classes.length <= 1) return;
    setClasses(prev => prev.filter(c => c.id !== id));
    setRelationships(prev => prev.filter(r => r.sourceId !== id && r.targetId !== id));
    if (selectedClassId === id) {
      const remaining = classes.filter(c => c.id !== id);
      if (remaining.length > 0) setSelectedClassId(remaining[0].id);
    }
  };

  const handleExport = async (type: 'xmi' | 'plantuml' | 'sql' | 'zip') => {
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
      const xmi = `<?xml version="1.0" encoding="UTF-8"?>\n<xmi:XMI xmi:version="2.1" xmlns:uml="http://schema.omg.org/spec/UML/2.1">\n  <uml:Model name="ECommerceDomain">\n  </uml:Model>\n</xmi:XMI>`;
      const blob = new Blob([xmi], { type: 'text/xml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'domain_model.xmi';
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  return (
    <div className="min-h-screen bg-[#0f131c] text-[#dfe2ee] selection:bg-[#10b981] selection:text-[#00422b]">
      {/* Fixed Top Header */}
      <Header
        activeView={activeView}
        onSelectView={setActiveView}
        onExport={handleExport}
      />

      {/* Fixed Left Sidebar */}
      <Sidebar
        activeView={activeView}
        onSelectView={setActiveView}
        telemetryLatency="ONNX 14ms"
      />

      {/* Main Viewport Container */}
      <div className="pl-64 pt-16 min-h-screen">
        {activeView === 'uml-canvas' && (
          <CanvasView
            classes={classes}
            relationships={relationships}
            selectedClassId={selectedClassId}
            onSelectClass={setSelectedClassId}
            onUpdateClass={handleUpdateClass}
            onAddClass={handleAddClass}
            onDeleteClass={handleDeleteClass}
            onSwitchToBackend={() => setActiveView('backend-db-generator')}
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

        {activeView === 'git-diff-versions' && (
          <GitDiffView />
        )}

        {activeView === 'team-room-live' && (
          <TeamRoomView />
        )}

        {activeView === 'ai-architect-console' && (
          <AiConsoleView />
        )}
      </div>

      {/* Floating Observer Assistant Bubble */}
      <FloatingAssistant
        activeView={activeView}
        selectedClassId={selectedClassId}
        classes={classes}
      />
    </div>
  );
}
