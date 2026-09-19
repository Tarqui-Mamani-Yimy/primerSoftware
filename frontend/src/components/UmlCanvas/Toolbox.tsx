import React, { useState } from 'react';
import { RelationshipType, UMLClassNode, Stereotype } from '../../types';
import { es } from '../../i18n/es';

interface ToolboxProps {
  classes: UMLClassNode[];
  selectedClassId: string;
  onSelectClass: (id: string) => void;
  onAddClass: (type: Stereotype) => void;
  onSelectRelationshipType: (type: RelationshipType | null) => void;
  activeRelationshipType: RelationshipType | null;
  zoomLevel: number;
}

export const Toolbox: React.FC<ToolboxProps> = ({
  classes,
  selectedClassId,
  onSelectClass,
  onAddClass,
  onSelectRelationshipType,
  activeRelationshipType,
  zoomLevel
}) => {
  const [filterText, setFilterText] = useState('');

  const filteredClasses = classes.filter(c => 
    c.name.toLowerCase().includes(filterText.toLowerCase()) ||
    c.tableBinding.toLowerCase().includes(filterText.toLowerCase())
  );

  return (
    <div className="w-72 flex-shrink-0 bg-[#181c24] flex flex-col z-20 shadow-xl border-r border-[#3c4a42]">
      {/* Toolbox Header */}
      <div className="p-3 flex items-center justify-between bg-[#1c2028] border-b border-[#3c4a42]">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#4cd7f6] text-sm">widgets</span>
          <span className="font-heading text-sm uppercase tracking-wider text-[#dfe2ee] font-bold">{es.canvas.toolbox}</span>
        </div>
        <span className="text-[10px] px-1.5 py-0.5 bg-[#31353e] text-[#4edea3] font-bold font-mono">UML 2.5</span>
      </div>

      {/* Quick Filter */}
      <div className="p-2 bg-[#0a0e16] border-b border-[#3c4a42]">
        <div className="relative flex items-center bg-[#1c2028] px-2 py-1 border border-[#3c4a42]">
          <span className="material-symbols-outlined text-xs text-[#bbcabf] mr-1.5">search</span>
          <input
            className="w-full bg-transparent text-[#dfe2ee] text-xs font-mono focus:outline-none placeholder:text-[#86948a]"
            placeholder={es.canvas.filter}
            aria-label={es.canvas.filter}
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
          />
          <span className="text-[#86948a] font-mono text-[10px]">/</span>
        </div>
      </div>

      {/* Scrollable Palette Sections */}
      <div className="flex-1 overflow-y-auto p-2.5 space-y-4">
        {/* Classifier Elements */}
        <div>
          <div className="px-1 py-0.5 flex items-center justify-between text-[#bbcabf]">
            <span className="text-[10px] uppercase tracking-widest font-mono font-bold">{es.canvas.classifiers}</span>
            <span className="text-[#4edea3] font-mono text-[10px]">[5]</span>
          </div>

          <div className="grid grid-cols-2 gap-1.5 mt-1.5 font-mono text-xs">
            <button
              onClick={() => onAddClass('«Entity»')}
              className="flex items-center gap-2 p-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] text-left transition-colors group border border-[#3c4a42]"
              title="Agregar una nueva clase"
            >
              <span className="w-4 h-4 bg-[#4edea3]/20 text-[#4edea3] flex items-center justify-center font-bold text-[10px]">C</span>
              <span className="group-hover:text-[#4edea3]">{es.canvas.class}</span>
            </button>

            <button
              onClick={() => onAddClass('«Abstract»')}
              className="flex items-center gap-2 p-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] text-left transition-colors group border border-[#3c4a42]"
              title="Agregar una nueva clase abstracta"
            >
              <span className="w-4 h-4 bg-[#4cd7f6]/20 text-[#4cd7f6] flex items-center justify-center font-bold italic text-[10px]">A</span>
              <span className="group-hover:text-[#4cd7f6]">{es.canvas.abstract}</span>
            </button>

            <button
              onClick={() => onAddClass('«Interface»')}
              className="flex items-center gap-2 p-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] text-left transition-colors group border border-[#3c4a42]"
              title="Agregar una interfaz"
            >
              <span className="w-4 h-4 bg-[#d0bcff]/20 text-[#d0bcff] flex items-center justify-center font-bold text-[10px]">I</span>
              <span className="group-hover:text-[#d0bcff]">{es.canvas.interface}</span>
            </button>

            <button
              onClick={() => onAddClass('«Enum»')}
              className="flex items-center gap-2 p-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] text-left transition-colors group border border-[#3c4a42]"
              title="Agregar una enumeración"
            >
              <span className="w-4 h-4 bg-[#353942] text-[#bbcabf] flex items-center justify-center font-bold text-[10px]">E</span>
              <span className="group-hover:text-[#dfe2ee]">{es.canvas.enum}</span>
            </button>

            <button
              onClick={() => onAddClass('«ValueObject»')}
              className="col-span-2 flex items-center gap-2 p-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] text-left transition-colors group border border-[#3c4a42]"
              title="Agregar un registro u objeto de valor"
            >
              <span className="w-4 h-4 bg-[#10b981]/20 text-[#4edea3] flex items-center justify-center font-bold text-[10px]">R</span>
              <span className="group-hover:text-[#4edea3]">{es.canvas.valueObject}</span>
            </button>
          </div>
        </div>

        {/* Relationship Connectors */}
        <div>
          <div className="px-1 py-0.5 flex items-center justify-between text-[#bbcabf]">
            <span className="text-[10px] uppercase tracking-widest font-mono font-bold">{es.canvas.relationships}</span>
            <span className="material-symbols-outlined text-xs text-[#bbcabf]">alt_route</span>
          </div>

          <div className="space-y-1 mt-1.5 font-mono text-xs">
            {[
              { id: 'association', label: es.canvas.association, icon: 'trending_flat', sym: '——>', color: 'text-[#4cd7f6]' },
              { id: 'aggregation', label: es.canvas.aggregation, icon: 'diamond', sym: '♦——', color: 'text-[#bbcabf]' },
              { id: 'composition', label: es.canvas.composition, icon: 'diamond', fill: true, sym: '♦——', color: 'text-[#4edea3]' },
              { id: 'generalization', label: es.canvas.generalization, icon: 'arrow_upward', sym: '——△', color: 'text-[#d0bcff]' },
              { id: 'realization', label: es.canvas.realization, icon: 'arrow_split', sym: '- - -△', color: 'text-[#d0bcff]' },
              { id: 'dependency', label: es.canvas.dependency, icon: 'commit', sym: '- - ->', color: 'text-[#86948a]' }
            ].map(rel => (
              <div
                key={rel.id}
                onClick={() => {
                  const next = activeRelationshipType === rel.id ? null : rel.id as RelationshipType;
                  onSelectRelationshipType(next);
                }}
                className={`flex items-center justify-between p-1.5 border transition-colors cursor-pointer group ${
                  activeRelationshipType === rel.id
                    ? 'bg-[#262a33] border-[#4edea3]'
                    : 'bg-[#1c2028] border-[#3c4a42] hover:bg-[#262a33]'
                }`}
              >
                <div className="flex items-center gap-2">
                  <span className={`material-symbols-outlined text-sm ${rel.color} ${rel.fill ? 'fill-1' : ''}`}>
                    {rel.icon}
                  </span>
                  <span className={`text-[#dfe2ee] group-hover:${rel.color}`}>{rel.label}</span>
                </div>
                <span className="text-[#86948a] text-[10px] font-mono">{rel.sym}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Package Explorer */}
        <div>
          <div className="px-1 py-0.5 flex items-center justify-between text-[#bbcabf]">
            <span className="text-[10px] uppercase tracking-widest font-mono font-bold">{es.canvas.packageExplorer}</span>
            <span className="text-[#4edea3] font-mono text-[10px]">{classes.length} nodes</span>
          </div>

          <div className="mt-1.5 bg-[#0a0e16] p-2 space-y-1.5 font-mono text-xs border border-[#3c4a42]">
            <div className="flex items-center gap-1.5 text-[#dfe2ee] font-bold">
              <span className="material-symbols-outlined text-xs text-[#4cd7f6]">folder</span>
              <span>com.nexus.orders</span>
            </div>

            <div className="pl-3 space-y-1">
              {filteredClasses.map(c => {
                const isSelected = selectedClassId === c.id;
                return (
                  <div
                    key={c.id}
                    onClick={() => onSelectClass(c.id)}
                    className={`flex items-center justify-between px-2 py-1 cursor-pointer transition-colors ${
                      isSelected
                        ? 'bg-[#1c2028] text-[#4edea3] font-bold border-l border-[#4edea3]'
                        : 'text-[#dfe2ee] hover:text-[#4cd7f6] hover:bg-[#1c2028]/60'
                    }`}
                  >
                    <div className="flex items-center gap-1.5 truncate">
                      <span className={`w-1.5 h-1.5 ${c.isAbstract ? 'bg-[#d0bcff]' : 'bg-[#4edea3]'}`}></span>
                      <span className="truncate">{c.name} {c.isAbstract ? '(Abs)' : '(Entity)'}</span>
                    </div>
                    <span className="text-[9px] text-[#86948a]">{c.badge || 'JPA'}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {/* Canvas Minimap Viewport */}
      <div className="p-2.5 bg-[#1c2028] border-t border-[#3c4a42]">
        <div className="flex items-center justify-between text-[#bbcabf] mb-1.5 font-mono">
          <span className="text-[10px] uppercase tracking-wider font-bold">{es.canvas.canvasViewport}</span>
          <span className="text-[#4edea3] text-[10px] font-bold">{Math.round(zoomLevel * 100)}% | 0,0</span>
        </div>
        <div className="w-full h-16 bg-[#0a0e16] relative overflow-hidden border border-[#3c4a42]">
          {/* Mini grid dots */}
          <div className="absolute inset-0 opacity-20 bg-[radial-gradient(#4edea3_1px,transparent_1px)] [background-size:6px_6px]"></div>
          {/* Mini class representations */}
          {classes.map(c => (
            <div
              key={c.id}
              style={{
                left: `${(c.x / 1100) * 80 + 5}%`,
                top: `${(c.y / 700) * 70 + 5}%`,
                width: selectedClassId === c.id ? '22px' : '16px',
                height: selectedClassId === c.id ? '16px' : '12px'
              }}
              className={`absolute transition-all cursor-pointer ${
                selectedClassId === c.id ? 'bg-[#4edea3] ring-1 ring-white' : 'bg-[#4cd7f6]/60'
              }`}
              onClick={() => onSelectClass(c.id)}
            />
          ))}
          {/* Viewport frame */}
          <div className="absolute inset-1 border border-[#4edea3]/40 pointer-events-none"></div>
        </div>
      </div>
    </div>
  );
};
