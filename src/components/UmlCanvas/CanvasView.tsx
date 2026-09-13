import React, { useState, useRef, useEffect } from 'react';
import { UMLClassNode, UMLRelationship, Stereotype } from '../../types';
import { Toolbox } from './Toolbox';
import { Inspector } from './Inspector';
import { VoiceAssistantHud } from './VoiceAssistantHud';

interface CanvasViewProps {
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
  selectedClassId: string;
  onSelectClass: (id: string) => void;
  onUpdateClass: (cls: UMLClassNode) => void;
  onAddClass: (type: Stereotype) => void;
  onDeleteClass: (id: string) => void;
  onSwitchToBackend: () => void;
}

export const CanvasView: React.FC<CanvasViewProps> = ({
  classes,
  relationships,
  selectedClassId,
  onSelectClass,
  onUpdateClass,
  onAddClass,
  onDeleteClass,
  onSwitchToBackend
}) => {
  const [zoom, setZoom] = useState(1);
  const [isPanning, setIsPanning] = useState(false);
  const [panOffset, setPanOffset] = useState({ x: 0, y: 0 });
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });
  const [activeTool, setActiveTool] = useState<'pointer' | 'pan' | 'laser'>('pointer');
  
  // Dragging class node
  const [draggedNodeId, setDraggedNodeId] = useState<string | null>(null);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0, origX: 0, origY: 0 });

  // Laser pointer lines
  const [laserPoints, setLaserPoints] = useState<Array<{ x: number; y: number }>>([]);
  const [showLaser, setShowLaser] = useState(false);

  // Collaborator cursor animation
  const [collabPos, setCollabPos] = useState({ x: 580, y: 140 });

  // Whiteboard sketch import dialog
  const [showSketchModal, setShowSketchModal] = useState(false);

  // Validation notification
  const [validationSuccess, setValidationSuccess] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);

  const selectedClass = classes.find(c => c.id === selectedClassId) || classes[0];

  // Collaborator subtle natural wandering
  useEffect(() => {
    const interval = setInterval(() => {
      setCollabPos(prev => ({
        x: Math.min(750, Math.max(480, prev.x + (Math.random() * 20 - 10))),
        y: Math.min(220, Math.max(100, prev.y + (Math.random() * 20 - 10)))
      }));
    }, 2500);
    return () => clearInterval(interval);
  }, []);

  // Node Drag handlers
  const handleNodeMouseDown = (e: React.MouseEvent, node: UMLClassNode) => {
    if (activeTool === 'pan') return;
    e.stopPropagation();
    onSelectClass(node.id);
    setDraggedNodeId(node.id);
    setDragStart({
      x: e.clientX,
      y: e.clientY,
      origX: node.x,
      origY: node.y
    });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (draggedNodeId) {
      const dx = (e.clientX - dragStart.x) / zoom;
      const dy = (e.clientY - dragStart.y) / zoom;
      const targetNode = classes.find(c => c.id === draggedNodeId);
      if (targetNode) {
        onUpdateClass({
          ...targetNode,
          x: Math.max(10, Math.round(dragStart.origX + dx)),
          y: Math.max(10, Math.round(dragStart.origY + dy))
        });
      }
    } else if (isPanning) {
      setPanOffset({
        x: panOffset.x + (e.clientX - panStart.x),
        y: panOffset.y + (e.clientY - panStart.y)
      });
      setPanStart({ x: e.clientX, y: e.clientY });
    } else if (showLaser && activeTool === 'laser') {
      const rect = containerRef.current?.getBoundingClientRect();
      if (rect) {
        setLaserPoints(prev => [...prev.slice(-25), { x: e.clientX - rect.left, y: e.clientY - rect.top }]);
      }
    }
  };

  const handleMouseUp = () => {
    setDraggedNodeId(null);
    setIsPanning(false);
  };

  const handleCanvasMouseDown = (e: React.MouseEvent) => {
    if (activeTool === 'pan' || e.button === 1) {
      setIsPanning(true);
      setPanStart({ x: e.clientX, y: e.clientY });
    } else if (activeTool === 'laser') {
      setShowLaser(true);
    }
  };

  const handleResetLayout = () => {
    // Orthogonal auto layout
    const positions = [
      { id: 'customer', x: 40, y: 100 },
      { id: 'order', x: 360, y: 64 },
      { id: 'order-item', x: 760, y: 100 },
      { id: 'payment', x: 410, y: 380 },
      { id: 'cc-payment', x: 280, y: 560 },
      { id: 'stripe-payment', x: 540, y: 560 }
    ];
    positions.forEach(p => {
      const node = classes.find(c => c.id === p.id);
      if (node) {
        onUpdateClass({ ...node, x: p.x, y: p.y });
      }
    });
    setPanOffset({ x: 0, y: 0 });
    setZoom(1);
  };

  const handleValidateSchema = () => {
    setValidationSuccess(true);
    setTimeout(() => setValidationSuccess(false), 2400);
  };

  // Helper to compute node center coordinates for connector SVG paths
  const getNodeCenter = (nodeId: string) => {
    const node = classes.find(c => c.id === nodeId);
    if (!node) return { x: 0, y: 0, width: 220, height: 180 };
    const width = node.width || (node.id === 'order' ? 290 : 220);
    const height = 160 + (node.attributes.length * 18) + (node.methods.length * 18);
    return {
      x: node.x,
      y: node.y,
      width,
      height,
      centerX: node.x + width / 2,
      centerY: node.y + height / 2,
      right: node.x + width,
      bottom: node.y + height
    };
  };

  return (
    <div className="flex flex-col w-full h-full">
      <div className="relative w-full h-[calc(100vh-6rem)] overflow-hidden flex select-none bg-[#0a0e16]">
        {/* LEFT TOOLBOX DOCK */}
        <Toolbox
          classes={classes}
          selectedClassId={selectedClassId}
          onSelectClass={onSelectClass}
          onAddClass={onAddClass}
          zoomLevel={zoom}
        />

        {/* CENTER INFINITE CANVAS */}
        <div
          ref={containerRef}
          onMouseDown={handleCanvasMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          className="flex-1 relative overflow-hidden bg-[#0a0e16] cursor-default"
          style={{ cursor: activeTool === 'pan' ? 'grab' : activeTool === 'laser' ? 'crosshair' : 'default' }}
        >
          {/* Background Dot Matrix Grid */}
          <div
            className="absolute inset-0 bg-[radial-gradient(#1c2028_1.5px,transparent_1.5px)] [background-size:24px_24px] pointer-events-none"
            style={{
              backgroundPosition: `${panOffset.x}px ${panOffset.y}px`
            }}
          />

          {/* Canvas Control Overlay HUD (Top-Left inside Center) */}
          <div className="absolute top-4 left-4 z-30 flex items-center gap-1 bg-[#262a33]/90 backdrop-blur-md p-1 border border-[#3c4a42] shadow-xl font-mono">
            <button
              onClick={() => setActiveTool('pointer')}
              className={`p-1.5 transition-colors ${activeTool === 'pointer' ? 'bg-[#1c2028] text-[#4edea3]' : 'text-[#dfe2ee] hover:text-[#4edea3]'}`}
              title="Select Pointer"
            >
              <span className="material-symbols-outlined text-sm">near_me</span>
            </button>
            <button
              onClick={() => setActiveTool('pan')}
              className={`p-1.5 transition-colors ${activeTool === 'pan' ? 'bg-[#1c2028] text-[#4edea3]' : 'text-[#dfe2ee] hover:text-[#4edea3]'}`}
              title="Pan Canvas (Hold & Drag)"
            >
              <span className="material-symbols-outlined text-sm">pan_tool</span>
            </button>

            <div className="w-px h-4 bg-[#3c4a42] mx-1"></div>

            <button
              onClick={() => setZoom(z => Math.min(2, +(z + 0.1).toFixed(1)))}
              className="p-1.5 text-[#dfe2ee] hover:text-[#4edea3] transition-colors"
              title="Zoom In"
            >
              <span className="material-symbols-outlined text-sm">zoom_in</span>
            </button>
            <span className="text-xs px-1 text-[#4edea3] font-bold min-w-10 text-center">
              {Math.round(zoom * 100)}%
            </span>
            <button
              onClick={() => setZoom(z => Math.max(0.5, +(z - 0.1).toFixed(1)))}
              className="p-1.5 text-[#dfe2ee] hover:text-[#4edea3] transition-colors"
              title="Zoom Out"
            >
              <span className="material-symbols-outlined text-sm">zoom_out</span>
            </button>

            <div className="w-px h-4 bg-[#3c4a42] mx-1"></div>

            <button
              onClick={handleResetLayout}
              className="p-1.5 text-[#dfe2ee] hover:text-[#4edea3] transition-colors flex items-center gap-1 text-[11px]"
              title="Auto Layout Orthogonal"
            >
              <span className="material-symbols-outlined text-sm">schema</span>
              <span className="hidden sm:inline">Layout</span>
            </button>

            <button
              onClick={handleValidateSchema}
              className="p-1.5 text-[#dfe2ee] hover:text-[#4cd7f6] transition-colors flex items-center gap-1 text-[11px]"
              title="Validate Schema"
            >
              <span className="material-symbols-outlined text-sm">verified</span>
              <span className="hidden sm:inline">Validate</span>
            </button>
          </div>

          {/* Validation Feedback Banner */}
          {validationSuccess && (
            <div className="absolute top-4 left-1/2 -translate-x-1/2 z-40 px-3 py-1.5 bg-[#10b981] text-[#00422b] font-mono text-xs font-bold shadow-2xl flex items-center gap-2 border border-[#4edea3]">
              <span className="material-symbols-outlined text-sm">check_circle</span>
              <span>UML Schema Validated: 0 Circular Deps • JPA 3.2 Compatible</span>
            </div>
          )}

          {/* Canvas Scalable Stage */}
          <div
            className="w-full h-full transform-gpu origin-top-left"
            style={{
              transform: `translate(${panOffset.x}px, ${panOffset.y}px) scale(${zoom})`,
              width: '2400px',
              height: '1800px'
            }}
          >
            {/* SVG RELATIONSHIPS LAYER */}
            <svg className="absolute inset-0 w-full h-full pointer-events-none z-10">
              <defs>
                {/* Triangle marker for Generalization */}
                <marker id="generalization-marker" markerHeight="14" markerWidth="14" orient="auto" refX="13" refY="7">
                  <polygon fill="#1c2028" points="1 1, 13 7, 1 13" stroke="#d0bcff" strokeWidth="1.5" />
                </marker>
                {/* Filled Diamond for Composition */}
                <marker id="composition-diamond" markerHeight="12" markerWidth="16" orient="auto" refX="1" refY="6">
                  <polygon fill="#4edea3" points="1 6, 8 1, 15 6, 8 11" stroke="#4edea3" strokeWidth="1.5" />
                </marker>
                {/* Open Arrow for Association */}
                <marker id="assoc-arrow" markerHeight="10" markerWidth="10" orient="auto" refX="9" refY="5">
                  <polyline fill="none" points="1 1, 9 5, 1 9" stroke="#4cd7f6" strokeWidth="1.5" />
                </marker>
              </defs>

              {relationships.map(rel => {
                const src = getNodeCenter(rel.sourceId);
                const dst = getNodeCenter(rel.targetId);

                // Compute orthogonal or straight line coords
                if (rel.id === 'rel-cust-order') {
                  const startX = src.right;
                  const startY = src.y + 70;
                  const endX = dst.x;
                  const endY = dst.y + 70;
                  const midX = (startX + endX) / 2;

                  return (
                    <g key={rel.id} className="relationship-group">
                      <path
                        d={`M ${startX} ${startY} L ${endX} ${endY}`}
                        fill="none"
                        markerEnd="url(#assoc-arrow)"
                        stroke="#4cd7f6"
                        strokeWidth="1.5"
                      />
                      <text className="font-mono text-[10px]" fill="#4cd7f6" fontWeight="bold" x={startX + 12} y={startY - 6}>
                        {rel.sourceMultiplicity || '1'}
                      </text>
                      <text className="font-mono text-[10px]" fill="#4cd7f6" fontWeight="bold" x={endX - 25} y={endY - 6}>
                        {rel.targetMultiplicity || '1..*'}
                      </text>
                      <text className="font-mono text-[9px]" fill="#bbcabf" x={midX - 18} y={startY + 16}>
                        {rel.label || 'places >'}
                      </text>
                    </g>
                  );
                }

                if (rel.id === 'rel-order-item') {
                  const startX = src.right;
                  const startY = src.y + 70;
                  const endX = dst.x;
                  const endY = dst.y + 70;
                  const midX = (startX + endX) / 2;

                  return (
                    <g key={rel.id} className="relationship-group">
                      <path
                        d={`M ${startX} ${startY} L ${endX} ${endY}`}
                        fill="none"
                        markerStart="url(#composition-diamond)"
                        stroke="#4edea3"
                        strokeWidth="1.5"
                      />
                      <text className="font-mono text-[10px]" fill="#4edea3" fontWeight="bold" x={startX + 18} y={startY - 6}>
                        {rel.sourceMultiplicity || '1'}
                      </text>
                      <text className="font-mono text-[10px]" fill="#4edea3" fontWeight="bold" x={endX - 25} y={endY - 6}>
                        {rel.targetMultiplicity || '1..*'}
                      </text>
                      <text className="font-mono text-[9px]" fill="#4edea3" x={midX - 16} y={startY + 16}>
                        {rel.label || 'contains'}
                      </text>
                    </g>
                  );
                }

                if (rel.id === 'rel-order-payment') {
                  const startX = src.centerX;
                  const startY = src.bottom;
                  const endX = dst.centerX;
                  const endY = dst.y;

                  return (
                    <g key={rel.id} className="relationship-group">
                      <path
                        d={`M ${startX} ${startY} L ${endX} ${endY}`}
                        fill="none"
                        stroke="#86948a"
                        strokeWidth="1.5"
                      />
                      {/* Hollow Aggregation diamond polygon at Order bottom */}
                      <polygon
                        fill="#0a0e16"
                        points={`${startX},${startY} ${startX - 5},${startY + 10} ${startX},${startY + 20} ${startX + 5},${startY + 10}`}
                        stroke="#86948a"
                        strokeWidth="1.5"
                      />
                      <text className="font-mono text-[10px]" fill="#bbcabf" x={startX + 10} y={startY + 30}>
                        {rel.sourceMultiplicity || '1'}
                      </text>
                      <text className="font-mono text-[10px]" fill="#bbcabf" x={endX + 10} y={endY - 10}>
                        {rel.targetMultiplicity || '0..1'}
                      </text>
                      <text className="font-mono text-[9px]" fill="#86948a" x={startX - 65} y={(startY + endY) / 2}>
                        {rel.label || 'settled_by'}
                      </text>
                    </g>
                  );
                }

                if (rel.type === 'generalization') {
                  const startX = src.centerX;
                  const startY = src.y;
                  const endX = dst.centerX + (rel.sourceId.includes('cc') ? -30 : 30);
                  const endY = dst.bottom;
                  const midY = (startY + endY) / 2;

                  return (
                    <g key={rel.id} className="relationship-group">
                      <path
                        d={`M ${startX} ${startY} L ${startX} ${midY} L ${endX} ${midY} L ${endX} ${endY}`}
                        fill="none"
                        markerEnd="url(#generalization-marker)"
                        stroke="#d0bcff"
                        strokeWidth="1.5"
                      />
                    </g>
                  );
                }

                // Fallback direct connector
                return (
                  <g key={rel.id}>
                    <line
                      x1={src.centerX}
                      y1={src.centerY}
                      x2={dst.centerX}
                      y2={dst.centerY}
                      stroke="#86948a"
                      strokeDasharray="4 2"
                      strokeWidth="1.5"
                    />
                  </g>
                );
              })}

              {/* Laser Pointer Trail */}
              {showLaser && laserPoints.length > 1 && (
                <polyline
                  points={laserPoints.map(p => `${p.x},${p.y}`).join(' ')}
                  fill="none"
                  stroke="#ff5555"
                  strokeWidth="3"
                  strokeLinecap="round"
                  filter="drop-shadow(0 0 6px #ff0000)"
                />
              )}
            </svg>

            {/* INTERACTIVE UML CLASS NODES */}
            {classes.map(cls => {
              const isSelected = selectedClassId === cls.id;
              const isOrder = cls.id === 'order';

              return (
                <div
                  key={cls.id}
                  onMouseDown={(e) => handleNodeMouseDown(e, cls)}
                  style={{
                    left: `${cls.x}px`,
                    top: `${cls.y}px`,
                    width: `${cls.width || (isOrder ? 290 : 230)}px`,
                    boxShadow: isSelected
                      ? '0 0 0 1.5px #4edea3, 0 0 24px rgba(78, 222, 163, 0.2)'
                      : '0 0 0 1px #3c4a42, 0 8px 16px rgba(0,0,0,0.5)'
                  }}
                  className={`absolute bg-[#1c2028] transition-shadow select-none cursor-move z-20 ${
                    isSelected ? 'ring-1 ring-[#4edea3]' : 'hover:ring-1 hover:ring-[#86948a]'
                  }`}
                >
                  {/* Stereotype & Name Header */}
                  <div className={`p-2 border-b border-[#3c4a42] text-center relative ${
                    isOrder ? 'bg-[#10b981]/20' : 'bg-[#262a33]'
                  }`}>
                    {isOrder && (
                      <span className="absolute top-1.5 right-2 w-2 h-2 bg-[#4edea3]"></span>
                    )}

                    <span className={`text-[9px] block uppercase tracking-widest font-mono font-bold ${
                      cls.isAbstract ? 'text-[#d0bcff]' : isOrder ? 'text-[#4edea3]' : 'text-[#4cd7f6]'
                    }`}>
                      {cls.stereotype}
                    </span>

                    <div className="flex items-center justify-center gap-1.5 mt-0.5">
                      <span className={`font-heading font-bold text-sm text-[#dfe2ee] ${cls.isAbstract ? 'italic' : ''}`}>
                        {cls.name}
                      </span>
                      {cls.badge && (
                        <span className="font-mono text-[9px] text-[#4edea3] bg-[#0a0e16] px-1 py-0.5 border border-[#3c4a42]">
                          {cls.badge}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Attributes Compartment */}
                  <div className="p-2 space-y-1 font-mono text-xs border-b border-[#3c4a42] bg-[#0a0e16]/80">
                    {cls.attributes.map(attr => (
                      <div key={attr.id} className="flex items-center justify-between">
                        <div className="flex items-center truncate">
                          <span className={`w-3.5 font-bold ${
                            attr.visibility === '+' ? 'text-[#4cd7f6]' : attr.visibility === '-' ? 'text-[#bbcabf]' : 'text-[#86948a]'
                          }`}>
                            {attr.visibility}
                          </span>
                          <span className="text-[#dfe2ee] truncate">{attr.name}:</span>
                          <span className={`ml-1 truncate ${
                            attr.type === 'UUID' ? 'text-[#4cd7f6]' : attr.type === 'OrderStatus' ? 'text-[#d0bcff] font-semibold' : 'text-[#dfe2ee]'
                          }`}>
                            {attr.type}
                          </span>
                        </div>
                        {attr.isPk && (
                          <span className="text-[9px] text-[#86948a] font-mono ml-1">PK</span>
                        )}
                      </div>
                    ))}
                  </div>

                  {/* Methods Compartment */}
                  <div className="p-2 space-y-1 font-mono text-xs bg-[#1c2028]/60">
                    {cls.methods.map(meth => (
                      <div key={meth.id} className="flex items-center truncate">
                        <span className="w-3.5 text-[#4edea3] font-bold">{meth.visibility}</span>
                        <span className={`truncate text-[#dfe2ee] ${meth.isAbstract ? 'italic text-[#d0bcff]' : ''}`}>
                          {meth.name}:
                        </span>
                        <span className="ml-1 text-[#4cd7f6] text-[11px] truncate">{meth.returnType}</span>
                      </div>
                    ))}
                  </div>

                  {/* Bottom Quick Status Bar for Order node */}
                  {isOrder && (
                    <div className="px-2 py-1 bg-[#181c24] flex items-center justify-between text-[10px] text-[#bbcabf] font-mono border-t border-[#3c4a42]">
                      <span className="text-[#4edea3]">JPA 3.1 • Active</span>
                      <div className="flex items-center gap-1.5">
                        <button 
                          onClick={(e) => { e.stopPropagation(); onSelectClass(cls.id); }}
                          className="hover:text-[#4edea3]" 
                          title="Add Attribute"
                        >
                          <span className="material-symbols-outlined text-xs">add_circle</span>
                        </button>
                        <button 
                          onClick={(e) => { e.stopPropagation(); onSwitchToBackend(); }}
                          className="hover:text-[#4cd7f6]" 
                          title="View Generated Code"
                        >
                          <span className="material-symbols-outlined text-xs">code</span>
                        </button>
                        <button 
                          onClick={(e) => { e.stopPropagation(); onDeleteClass(cls.id); }}
                          className="hover:text-[#ffb4ab]" 
                          title="Delete Node"
                        >
                          <span className="material-symbols-outlined text-xs">delete</span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}

            {/* Collaborator Cursor (Sarah) */}
            <div
              className="absolute pointer-events-none flex flex-col items-start z-40 transition-all duration-700 ease-out"
              style={{ left: `${collabPos.x}px`, top: `${collabPos.y}px` }}
            >
              <svg className="w-4 h-4 text-[#4cd7f6] drop-shadow" fill="currentColor" viewBox="0 0 24 24">
                <path d="M3 3l7 18 3-7 7-3L3 3z" />
              </svg>
              <div className="px-1.5 py-0.5 bg-[#4cd7f6] text-[#001f26] font-mono text-[9px] font-bold shadow-md whitespace-nowrap">
                Sarah • Inspecting Order.pk
              </div>
            </div>
          </div>

          {/* FLOATING HUD AT BOTTOM-CENTER: Local AI Voice Assistant */}
          <VoiceAssistantHud
            onApplyAction={(action) => {
              if (action.includes('trackingNumber')) {
                const order = classes.find(c => c.id === 'order');
                if (order) {
                  onUpdateClass({
                    ...order,
                    attributes: [
                      ...order.attributes,
                      { id: `attr_${Date.now()}`, name: 'trackingNumber', type: 'String', visibility: '+', annotations: ['@Column(length=64)'] }
                    ]
                  });
                }
              }
            }}
          />

          {/* RIGHT FLOATING COLLABORATION WIDGET (WebRTC Mini-Dock) */}
          <div className="absolute top-4 right-4 z-30 flex flex-col items-end gap-1.5 pointer-events-auto">
            <div className="flex items-center gap-1.5 bg-[#262a33]/90 backdrop-blur-md p-1 border border-[#3c4a42] shadow-xl">
              {/* Active Cursors Avatar Group */}
              <div className="flex items-center -space-x-1 pl-1">
                <div className="w-6 h-6 rounded-full bg-[#4cd7f6] text-[#001f26] font-bold text-[10px] flex items-center justify-center ring-1 ring-[#0f131c]" title="Sarah (Lead Architect)">
                  SC
                </div>
                <div className="w-6 h-6 rounded-full bg-[#d0bcff] text-[#3c0091] font-bold text-[10px] flex items-center justify-center ring-1 ring-[#0f131c]" title="Alex (Backend Dev)">
                  AK
                </div>
                <div className="w-6 h-6 rounded-full bg-[#4edea3] text-[#003824] font-bold text-[10px] flex items-center justify-center ring-1 ring-[#0f131c]" title="You (Lead)">
                  ME
                </div>
              </div>

              <div className="w-px h-4 bg-[#3c4a42] mx-1"></div>

              {/* Tool: Laser Pointer */}
              <button
                onClick={() => {
                  setActiveTool(activeTool === 'laser' ? 'pointer' : 'laser');
                  setShowLaser(!showLaser);
                  if (showLaser) setLaserPoints([]);
                }}
                className={`flex items-center gap-1 px-1.5 py-1 text-xs font-mono transition-colors ${
                  activeTool === 'laser' ? 'bg-[#ff5555]/20 text-[#ffb4ab] border border-[#ff5555]' : 'bg-[#1c2028] hover:bg-[#31353e] text-[#dfe2ee]'
                }`}
                title="Laser Pointer Tool (Press L)"
              >
                <span className="material-symbols-outlined text-xs text-[#ffb4ab]">adjust</span>
                <span className="hidden sm:inline">Laser</span>
              </button>

              {/* Quick Button: OCR Pizarra / Import Sketch */}
              <button
                onClick={() => setShowSketchModal(true)}
                className="flex items-center gap-1.5 px-2.5 py-1 bg-[#b090ff]/20 hover:bg-[#b090ff]/30 text-[#d0bcff] border border-[#b090ff]/40 transition-colors font-mono text-xs font-bold"
                title="Import whiteboard sketch via Computer Vision"
              >
                <span className="material-symbols-outlined text-sm">photo_camera</span>
                <span className="uppercase">Import Sketch</span>
              </button>
            </div>
          </div>
        </div>

        {/* RIGHT INSPECTOR DOCK */}
        {selectedClass && (
          <Inspector
            selectedClass={selectedClass}
            onUpdateClass={onUpdateClass}
            onClose={() => {}}
            onGenerateCode={onSwitchToBackend}
          />
        )}
      </div>

      {/* Whiteboard Sketch OCR Modal */}
      {showSketchModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#181c24] border border-[#3c4a42] max-w-md w-full p-4 shadow-2xl font-mono">
            <div className="flex items-center justify-between pb-2 border-b border-[#3c4a42]">
              <div className="flex items-center gap-2 text-[#4edea3] font-bold">
                <span className="material-symbols-outlined text-sm">photo_camera</span>
                <span>Whiteboard Sketch OCR Engine</span>
              </div>
              <button onClick={() => setShowSketchModal(false)} className="text-[#86948a] hover:text-white">
                <span className="material-symbols-outlined text-sm">close</span>
              </button>
            </div>

            <div className="py-4 space-y-3 text-xs text-[#bbcabf]">
              <div className="p-4 border-2 border-dashed border-[#3c4a42] hover:border-[#4edea3] text-center cursor-pointer bg-[#0a0e16]">
                <span className="material-symbols-outlined text-3xl text-[#4cd7f6] mb-1">upload_file</span>
                <p className="text-white font-bold">Arrastra una foto de la pizarra o haz clic para subir</p>
                <p className="text-[10px] text-[#86948a] mt-1">Soporta PNG, JPEG, SVG con detección automática de diagramas UML</p>
              </div>

              <div className="bg-[#1c2028] p-2 text-[11px] space-y-1">
                <span className="text-[#4edea3] font-bold">Modelo Offline ONNX YOLOv8-UML:</span>
                <p>Detecta rectángulos con compartimentos de atributos y métodos, flechas con puntas de herencia y agregación.</p>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-[#3c4a42]">
              <button
                onClick={() => setShowSketchModal(false)}
                className="px-3 py-1 bg-[#1c2028] text-xs text-[#bbcabf] hover:text-white"
              >
                Cerrar
              </button>
              <button
                onClick={() => {
                  setShowSketchModal(false);
                  onAddClass('«Entity»');
                }}
                className="px-3 py-1 bg-[#4edea3] text-black text-xs font-bold"
              >
                Procesar Muestra
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
