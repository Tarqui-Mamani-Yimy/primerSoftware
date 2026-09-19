import React, { useEffect, useState } from 'react';
import { RelationshipType, Stereotype, UMLClassNode, UMLRelationship } from '../../types';
import { DiagramSummary, DiagramVersion } from '../../api/diagramApi';
import { Inspector } from './Inspector';
import { RelationshipInspector } from './RelationshipInspector';
import { Toolbox } from './Toolbox';
import { clipEdgeToNodeBorders, pointAlongEdge, validateRelationshipCreation } from '../../diagram/relationshipHelpers';
import { es } from '../../i18n/es';

interface CanvasViewProps {
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
  selectedClassId: string;
  selectedRelationshipId: string;
  onSelectClass: (id: string) => void;
  onSelectRelationship: (id: string) => void;
  onUpdateClass: (cls: UMLClassNode) => void;
  onUpdateRelationship: (relationship: UMLRelationship) => void;
  onDeleteRelationship: (id: string) => void;
  onFinishNodeDrag: () => void;
  onPersistChange: () => void;
  onAddClass: (type: Stereotype) => void;
  onAddRelationship: (relationship: UMLRelationship) => void;
  onDeleteClass: (id: string) => void;
  onSwitchToBackend: () => void;
  diagramId?: string;
  diagramName: string;
  diagrams: DiagramSummary[];
  versions: DiagramVersion[];
  persistenceStatus: 'idle' | 'saving' | 'saved' | 'error';
  isDocumentLoading: boolean;
  onOpenDiagram: (id: string) => void;
  onCreateDiagram: () => void;
  onRenameDiagram: (name: string) => void;
  onLoadVersions: () => void;
  onRestoreVersion: (versionNumber: number) => void;
  onCreateCheckpoint: () => void;
  persistenceLabel: string;
}

type Point = { x: number; y: number };
const MIN_ZOOM = 0.35;
const MAX_ZOOM = 2.5;
const CANVAS_SIZE = 2600;
const nodeWidth = (umlClass: UMLClassNode) => umlClass.width ?? 250;
const nodeHeight = (umlClass: UMLClassNode) => Math.max(130, 92 + (umlClass.attributes.length + umlClass.methods.length) * 18);
const relationshipDash: Partial<Record<RelationshipType, string>> = { realization: '6 4', dependency: '6 4' };

const markerFor = (type: RelationshipType) => {
  if (type === 'aggregation') return 'url(#aggregation)';
  if (type === 'composition') return 'url(#composition)';
  if (type === 'generalization' || type === 'realization') return 'url(#triangle)';
  return 'url(#arrow)';
};

export const CanvasView: React.FC<CanvasViewProps> = (props) => {
  const viewportRef = React.useRef<HTMLElement>(null);
  const dragRef = React.useRef<{ id: string; offset: Point } | null>(null);
  const panRef = React.useRef<{ start: Point; origin: Point } | null>(null);
  const [relationshipType, setRelationshipType] = useState<RelationshipType | null>(null);
  const [pendingSourceId, setPendingSourceId] = useState<string | null>(null);
  const [statusHint, setStatusHint] = useState<string | null>(null);
  const [showDocuments, setShowDocuments] = useState(false);
  const [showVersions, setShowVersions] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState<Point>({ x: 80, y: 80 });
  const selected = props.classes.find((item) => item.id === props.selectedClassId);
  const selectedRelationship = props.relationships.find((item) => item.id === props.selectedRelationshipId);

  const cancelConnectMode = () => {
    setPendingSourceId(null);
    setRelationshipType(null);
    setStatusHint(null);
  };

  useEffect(() => {
    const cancelWithEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && relationshipType) cancelConnectMode();
    };
    window.addEventListener('keydown', cancelWithEscape);
    return () => window.removeEventListener('keydown', cancelWithEscape);
  }, [relationshipType]);

  const selectRelationshipType = (type: RelationshipType | null) => {
    setPendingSourceId(null);
    setStatusHint(null);
    setRelationshipType(type);
  };

  const selectNode = (id: string) => {
    if (relationshipType) {
      const clicked = props.classes.find((umlClass) => umlClass.id === id);
      if (!pendingSourceId) {
        if (clicked && relationshipType === 'realization' && (clicked.stereotype === '«Interface»' || clicked.stereotype === '«Enum»')) {
          setStatusHint('La realización necesita una clase origen que no sea interfaz.');
          return;
        }
        setPendingSourceId(id);
        props.onSelectRelationship('');
        props.onSelectClass(id);
        setStatusHint(`Source selected: ${clicked?.name ?? id} — now click the target class (Esc to cancel).`);
        return;
      }
      const sourceId = pendingSourceId;
      const source = props.classes.find((umlClass) => umlClass.id === sourceId);
      const target = props.classes.find((umlClass) => umlClass.id === id);
      const validation = validateRelationshipCreation(sourceId, id, relationshipType, props.relationships, source, target);
      if (!validation.ok) {
        setStatusHint(validation.reason ?? 'That connection is not allowed.');
        return;
      }
      const created: UMLRelationship = { id: `relationship_${crypto.randomUUID()}`, sourceId, targetId: id, type: relationshipType };
      props.onAddRelationship(created);
      props.onSelectRelationship(created.id);
      setPendingSourceId(null);
      setRelationshipType(null);
      setStatusHint(null);
      return;
    }
    props.onSelectRelationship('');
    props.onSelectClass(id);
  };

  const selectEdge = (id: string) => {
    props.onSelectClass('');
    props.onSelectRelationship(id);
  };

  const modelPoint = (event: React.PointerEvent): Point | null => {
    const bounds = viewportRef.current?.getBoundingClientRect();
    if (!bounds) return null;
    return { x: (event.clientX - bounds.left - pan.x) / zoom, y: (event.clientY - bounds.top - pan.y) / zoom };
  };

  const handleNodePointerDown = (event: React.PointerEvent<HTMLButtonElement>, umlClass: UMLClassNode) => {
    if (relationshipType || event.button !== 0) return;
    const point = modelPoint(event);
    if (!point) return;
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = { id: umlClass.id, offset: { x: point.x - umlClass.x, y: point.y - umlClass.y } };
    props.onSelectClass(umlClass.id);
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLElement>) => {
    if (dragRef.current) {
      const point = modelPoint(event);
      if (!point) return;
      const drag = dragRef.current;
      const current = props.classes.find((umlClass) => umlClass.id === drag.id);
      if (current) props.onUpdateClass({ ...current, x: Math.max(0, point.x - drag.offset.x), y: Math.max(0, point.y - drag.offset.y) });
      return;
    }
    if (panRef.current) setPan({ x: panRef.current.origin.x + event.clientX - panRef.current.start.x, y: panRef.current.origin.y + event.clientY - panRef.current.start.y });
  };

  const endPointerAction = () => {
    const wasDraggingNode = Boolean(dragRef.current);
    dragRef.current = null;
    panRef.current = null;
    if (wasDraggingNode) props.onFinishNodeDrag();
  };

  const handleCanvasPointerDown = (event: React.PointerEvent<HTMLElement>) => {
    if (event.button !== 0 || relationshipType) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    panRef.current = { start: { x: event.clientX, y: event.clientY }, origin: pan };
    props.onSelectClass('');
    props.onSelectRelationship('');
  };

  const handleWheel = (event: React.WheelEvent<HTMLElement>) => {
    event.preventDefault();
    const bounds = viewportRef.current?.getBoundingClientRect();
    if (!bounds) return;
    const nextZoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom * (event.deltaY > 0 ? 0.9 : 1.1)));
    const cursor = { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
    const model = { x: (cursor.x - pan.x) / zoom, y: (cursor.y - pan.y) / zoom };
    setPan({ x: cursor.x - model.x * nextZoom, y: cursor.y - model.y * nextZoom });
    setZoom(nextZoom);
  };

  const changeZoom = (delta: number) => setZoom((current) => Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Number((current + delta).toFixed(2)))));
  const resetViewport = () => { setZoom(1); setPan({ x: 80, y: 80 }); };

  return <div className="relative flex h-[calc(100vh-4rem)] min-w-0 bg-[#0a0e16]">
    <Toolbox classes={props.classes} selectedClassId={props.selectedClassId} onSelectClass={(id) => { props.onSelectRelationship(''); props.onSelectClass(id); }} onAddClass={props.onAddClass} onSelectRelationshipType={selectRelationshipType} activeRelationshipType={relationshipType} zoomLevel={zoom} />
    <section ref={viewportRef} onWheel={handleWheel} onPointerDown={handleCanvasPointerDown} onPointerMove={handlePointerMove} onPointerUp={endPointerAction} onPointerCancel={endPointerAction} className="relative h-full min-w-0 flex-1 overflow-hidden touch-none cursor-grab active:cursor-grabbing" aria-label={es.canvas.interactiveCanvas}>
      <div className="absolute left-4 top-4 z-20 flex items-center gap-2 border border-[#3c4a42] bg-[#262a33] p-2 font-mono text-xs shadow-lg">
        <div className="relative border-r border-[#3c4a42] pr-2">
          <button type="button" onClick={() => setShowDocuments((current) => !current)} className="max-w-44 truncate text-[#dfe2ee] hover:text-[#4edea3]" title="Open or create a diagram">
            {props.diagramName || es.canvas.noDiagram} ▾
          </button>
          {showDocuments && <div className="absolute left-0 top-7 w-64 border border-[#3c4a42] bg-[#1c2028] p-1 shadow-xl">
            <button type="button" onClick={() => { props.onCreateDiagram(); setShowDocuments(false); }} className="w-full border-b border-[#3c4a42] px-2 py-2 text-left text-[#4edea3] hover:bg-[#262a33]">{es.canvas.newDiagram}</button>
            {props.diagrams.map((diagram) => <button key={diagram.id} type="button" onClick={() => { props.onOpenDiagram(diagram.id); setShowDocuments(false); }} className={`w-full px-2 py-2 text-left hover:bg-[#262a33] ${diagram.id === props.diagramId ? 'text-[#4edea3]' : 'text-[#dfe2ee]'}`}>
              <span className="block truncate">{diagram.name}</span><span className="text-[10px] text-[#86948a]">{new Date(diagram.updatedAt).toLocaleString()}</span>
            </button>)}
          </div>}
        </div>
        {props.diagramId && <input aria-label="Diagram name" value={props.diagramName} onChange={(event) => props.onRenameDiagram(event.target.value)} className="w-32 bg-transparent text-[#bbcabf] outline-none focus:text-white" placeholder="Diagram name" />}
        <span className={props.persistenceStatus === 'error' ? 'text-[#ffb4ab]' : props.persistenceStatus === 'saved' ? 'text-[#4edea3]' : 'text-[#bbcabf]'}>{props.isDocumentLoading ? es.canvas.loading : props.persistenceLabel}</span>
        {props.diagramId && <button type="button" onClick={props.onCreateCheckpoint} className="border-l border-[#3c4a42] pl-2 hover:text-[#4edea3]" aria-label={es.canvas.createCheckpoint} title={es.canvas.createCheckpoint}>{es.canvas.createCheckpoint}</button>}
        {props.diagramId && <div className="relative border-l border-[#3c4a42] pl-2">
          <button type="button" onClick={() => { setShowVersions((current) => !current); if (!showVersions) props.onLoadVersions(); }} className="hover:text-[#4edea3]" aria-label={es.canvas.history}>{es.canvas.history}</button>
          {showVersions && <div className="absolute left-0 top-7 w-56 border border-[#3c4a42] bg-[#1c2028] p-1 shadow-xl">
            {props.versions.length === 0 ? <p className="p-2 text-[#86948a]">{es.canvas.noVersions}</p> : props.versions.map((version) => <button key={version.id} type="button" onClick={() => { props.onRestoreVersion(version.versionNumber); setShowVersions(false); }} className="w-full px-2 py-2 text-left text-[#dfe2ee] hover:bg-[#262a33]">
              {es.canvas.version(version.versionNumber)}
              <span className="ml-2 text-[10px] text-[#86948a]">{new Date(version.createdAt).toLocaleString()}</span>
              <span className="ml-2 text-[10px] text-[#4edea3]">{version.createdBy ? version.createdBy.slice(0, 12) : 'anónimo'}</span>
              {version.message && <span className="ml-2 text-[10px] text-[#bbcabf]">{version.message}</span>
            </button>)}
          </div>}
        </div>}
        <button type="button" className="px-1 text-lg hover:text-[#4edea3]" onClick={() => changeZoom(-.1)} aria-label="Zoom out">−</button>
        <button type="button" className="min-w-12 hover:text-[#4edea3]" onClick={resetViewport} title="Reset viewport">{Math.round(zoom * 100)}%</button>
        <button type="button" className="px-1 text-lg hover:text-[#4edea3]" onClick={() => changeZoom(.1)} aria-label="Zoom in">+</button>
        <span className="border-l border-[#3c4a42] pl-2 text-[#4edea3]">{relationshipType ? pendingSourceId ? `Conectar: seleccioná el destino (${relationshipType})` : `Conectar: seleccioná el origen (${relationshipType})` : es.canvas.dragPanZoom}</span>
        {relationshipType && <button type="button" className="border-l border-[#3c4a42] pl-2 text-[#ffb4ab] hover:text-white" onClick={cancelConnectMode}>{es.canvas.cancel}</button>}
      </div>
      {statusHint && <div role="status" className="absolute left-4 top-16 z-20 max-w-md border border-[#4cd7f6] bg-[#1c2028] px-3 py-2 font-mono text-xs text-[#dfe2ee] shadow-lg">{statusHint}</div>}
      <div className="absolute inset-0 opacity-30 [background-image:radial-gradient(#4edea3_1px,transparent_1px)] [background-size:24px_24px] pointer-events-none" />
      <div className="relative" style={{ width: CANVAS_SIZE, height: CANVAS_SIZE, transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`, transformOrigin: '0 0' }}>
        <svg className="pointer-events-none absolute inset-0 overflow-visible" width={CANVAS_SIZE} height={CANVAS_SIZE} aria-hidden="true">
          <defs>
            <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto"><path d="M 0 0 L 10 5 L 0 10 z" fill="#4cd7f6" /></marker>
            <marker id="triangle" markerWidth="12" markerHeight="12" refX="10" refY="6" orient="auto"><path d="M 0 0 L 12 6 L 0 12 z" fill="#1c2028" stroke="#d0bcff" strokeWidth="1.5" /></marker>
            <marker id="aggregation" markerWidth="13" markerHeight="13" refX="2" refY="6" orient="auto"><path d="M 7 0 L 13 6 L 7 12 L 1 6 z" fill="#1c2028" stroke="#bbcabf" strokeWidth="1.5" /></marker>
            <marker id="composition" markerWidth="13" markerHeight="13" refX="2" refY="6" orient="auto"><path d="M 7 0 L 13 6 L 7 12 L 1 6 z" fill="#4edea3" stroke="#4edea3" strokeWidth="1.5" /></marker>
          </defs>
          {props.relationships.map((relationship) => {
            const source = props.classes.find((item) => item.id === relationship.sourceId);
            const target = props.classes.find((item) => item.id === relationship.targetId);
            if (!source || !target) return null;
            const color = relationship.type === 'composition' ? '#4edea3' : relationship.type === 'generalization' || relationship.type === 'realization' ? '#d0bcff' : '#4cd7f6';
            const edge = clipEdgeToNodeBorders(
              { x: source.x, y: source.y, width: nodeWidth(source), height: nodeHeight(source) },
              { x: target.x, y: target.y, width: nodeWidth(target), height: nodeHeight(target) },
            );
            const isSelected = relationship.id === props.selectedRelationshipId;
            const mid = pointAlongEdge(edge, 0.5);
            const nearSource = pointAlongEdge(edge, 0.12);
            const nearTarget = pointAlongEdge(edge, 0.88);
            const labelWidth = relationship.label ? relationship.label.length * 7 + 18 : 0;
            return (
              <g key={relationship.id} style={{ pointerEvents: 'auto' }} onClick={(event) => { event.stopPropagation(); selectEdge(relationship.id); }}>
                <line x1={edge.x1} y1={edge.y1} x2={edge.x2} y2={edge.y2} stroke="transparent" strokeWidth="16" />
                <line
                  x1={edge.x1} y1={edge.y1} x2={edge.x2} y2={edge.y2}
                  stroke={isSelected ? '#ffffff' : color} strokeWidth={isSelected ? 3 : 2}
                  strokeDasharray={relationshipDash[relationship.type]}
                  markerStart={relationship.type === 'aggregation' || relationship.type === 'composition' ? markerFor(relationship.type) : undefined}
                  markerEnd={relationship.type === 'aggregation' || relationship.type === 'composition' ? undefined : markerFor(relationship.type)}
                />
                {relationship.label && (
                  <g>
                    <rect x={mid.x - labelWidth / 2} y={mid.y - 11} width={labelWidth} height={20} rx={10} fill="#0a0e16" stroke={color} strokeWidth={1} />
                    <text x={mid.x} y={mid.y + 4} fill={color} fontSize="12" textAnchor="middle">{relationship.label}</text>
                  </g>
                )}
                {relationship.sourceMultiplicity && (
                  <text x={nearSource.x + 8} y={nearSource.y - 8} fill={color} fontSize="12" stroke="#0a0e16" strokeWidth={3} paintOrder="stroke">{relationship.sourceMultiplicity}</text>
                )}
                {relationship.targetMultiplicity && (
                  <text x={nearTarget.x + 8} y={nearTarget.y - 8} fill={color} fontSize="12" stroke="#0a0e16" strokeWidth={3} paintOrder="stroke">{relationship.targetMultiplicity}</text>
                )}
              </g>
            );
          })}
        </svg>
        {props.classes.map((umlClass) => <button key={umlClass.id} type="button" onPointerDown={(event) => handleNodePointerDown(event, umlClass)} onClick={(event) => { event.stopPropagation(); selectNode(umlClass.id); }} className={`absolute z-10 overflow-hidden border bg-[#1c2028] text-left font-mono text-[11px] text-[#dfe2ee] shadow-lg transition-colors cursor-move ${umlClass.id === props.selectedClassId ? 'border-2 border-[#4edea3]' : pendingSourceId === umlClass.id ? 'border-2 border-[#4cd7f6]' : 'border-[#3c4a42] hover:border-[#4cd7f6]'}`} style={{ left: umlClass.x, top: umlClass.y, width: nodeWidth(umlClass), minHeight: nodeHeight(umlClass) }}>
          <div className="border-b border-[#3c4a42] px-3 py-2 text-center text-[#4edea3]">{umlClass.stereotype}<br /><strong>{umlClass.name}</strong></div>
          <div className="border-b border-[#3c4a42] px-3 py-2">{umlClass.attributes.map((attribute) => <div key={attribute.id}>{attribute.visibility} {attribute.name}: {attribute.type}</div>)}</div>
          <div className="px-3 py-2">{umlClass.methods.map((method) => <div key={method.id}>{method.visibility} {method.name}: {method.returnType}</div>)}</div>
        </button>)}
      </div>
    </section>
    {selectedRelationship && !selected ? (
      <RelationshipInspector
        selectedRelationship={selectedRelationship}
        classes={props.classes}
        onUpdateRelationship={(updated) => {
          const source = props.classes.find((umlClass) => umlClass.id === updated.sourceId);
          const target = props.classes.find((umlClass) => umlClass.id === updated.targetId);
          const validation = validateRelationshipCreation(
            updated.sourceId,
            updated.targetId,
            updated.type,
            props.relationships.filter((relationship) => relationship.id !== updated.id),
            source,
            target,
          );
          if (!validation.ok) return validation.reason;
          props.onUpdateRelationship(updated);
          props.onPersistChange();
          return undefined;
        }}
        onDeleteRelationship={props.onDeleteRelationship}
        onClose={() => props.onSelectRelationship('')}
      />
    ) : selected ? (
      <Inspector selectedClass={selected} onUpdateClass={(umlClass) => { props.onUpdateClass(umlClass); props.onPersistChange(); }} onClose={() => props.onSelectClass('')} onGenerateCode={props.onSwitchToBackend} />
    ) : null}
  </div>;
};
