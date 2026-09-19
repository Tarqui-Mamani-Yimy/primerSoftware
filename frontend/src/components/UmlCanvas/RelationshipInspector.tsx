import React, { useEffect, useState } from 'react';
import { RelationshipType, UMLClassNode, UMLRelationship } from '../../types';
import { isValidMultiplicity } from '../../diagram/relationshipHelpers';
import { es } from '../../i18n/es';

interface RelationshipInspectorProps {
  selectedRelationship: UMLRelationship;
  classes: UMLClassNode[];
  onUpdateRelationship: (updated: UMLRelationship) => string | undefined;
  onDeleteRelationship: (id: string) => void;
  onClose: () => void;
}

const relationshipTypes: RelationshipType[] = ['association', 'aggregation', 'composition', 'generalization', 'realization', 'dependency'];

export const RelationshipInspector: React.FC<RelationshipInspectorProps> = ({
  selectedRelationship,
  classes,
  onUpdateRelationship,
  onDeleteRelationship,
  onClose,
}) => {
  const [draft, setDraft] = useState(selectedRelationship);
  const [errors, setErrors] = useState<{ sourceMultiplicity?: string; targetMultiplicity?: string; relationship?: string }>({});

  useEffect(() => {
    setDraft(selectedRelationship);
    setErrors({});
  }, [selectedRelationship]);

  const sourceName = classes.find((item) => item.id === draft.sourceId)?.name ?? draft.sourceId;
  const targetName = classes.find((item) => item.id === draft.targetId)?.name ?? draft.targetId;
  const updateDraft = <K extends keyof UMLRelationship>(field: K, value: UMLRelationship[K]) => {
    setDraft((current) => ({ ...current, [field]: value }));
    setErrors((current) => ({ ...current, relationship: undefined }));
  };

  const handleMultiplicityChange = (field: 'sourceMultiplicity' | 'targetMultiplicity', value: string) => {
    updateDraft(field, value);
    setErrors((current) => ({ ...current, [field]: isValidMultiplicity(value) ? undefined : es.canvas.invalidMultiplicity }));
  };

  const saveRelationship = () => {
    if (!isValidMultiplicity(draft.sourceMultiplicity) || !isValidMultiplicity(draft.targetMultiplicity)) {
      setErrors((current) => ({ ...current, relationship: es.canvas.fixMultiplicity }));
      return;
    }
    const reason = onUpdateRelationship(draft);
    if (reason) setErrors((current) => ({ ...current, relationship: reason }));
  };

  return (
    <div className="w-80 flex-shrink-0 bg-[#181c24] flex flex-col z-20 shadow-xl border-l border-[#3c4a42] overflow-hidden">
      <div className="p-3 bg-[#1c2028] border-b border-[#3c4a42] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#4cd7f6] text-sm">alt_route</span>
          <span className="font-heading text-sm uppercase tracking-wider text-[#dfe2ee] font-bold">{es.canvas.relationshipInspector}</span>
        </div>
        <button type="button" onClick={onClose} className="text-[#bbcabf] hover:text-[#dfe2ee] transition-colors" aria-label={es.canvas.closeInspector}>
          <span className="material-symbols-outlined text-sm">close</span>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-4 font-mono">
        <div className="space-y-2 bg-[#1c2028] p-2.5 border border-[#3c4a42]">
          <span className="text-[10px] uppercase font-bold text-[#bbcabf]">{es.canvas.connection}</span>
          <p className="text-xs text-[#dfe2ee]">{sourceName} <span className="text-[#86948a]">→</span> {targetName}</p>
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-type">{es.canvas.relationshipType}</label>
          <select id="rel-type" value={draft.type} onChange={(event) => updateDraft('type', event.target.value as RelationshipType)} className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none">
            {relationshipTypes.map((type) => <option key={type} value={type}>{type}</option>)}
          </select>
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-source">{es.canvas.sourceClass}</label>
          <select id="rel-source" value={draft.sourceId} onChange={(event) => updateDraft('sourceId', event.target.value)} className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none">
            {classes.map((umlClass) => <option key={umlClass.id} value={umlClass.id}>{umlClass.name}</option>)}
          </select>
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-target">{es.canvas.targetClass}</label>
          <select id="rel-target" value={draft.targetId} onChange={(event) => updateDraft('targetId', event.target.value)} className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none">
            {classes.map((umlClass) => <option key={umlClass.id} value={umlClass.id}>{umlClass.name}</option>)}
          </select>
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-label">{es.canvas.label}</label>
          <input id="rel-label" className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none" type="text" value={draft.label ?? ''} onChange={(event) => updateDraft('label', event.target.value)} placeholder="e.g. places" />
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-source-mult">{es.canvas.sourceMultiplicity} (cerca de {sourceName})</label>
          <input id="rel-source-mult" className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none" type="text" value={draft.sourceMultiplicity ?? ''} onChange={(event) => handleMultiplicityChange('sourceMultiplicity', event.target.value)} placeholder="e.g. 1" />
          {errors.sourceMultiplicity && <p className="mt-1 text-[10px] text-[#ffb4ab]">{errors.sourceMultiplicity}</p>}
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-target-mult">{es.canvas.targetMultiplicity} (cerca de {targetName})</label>
          <input id="rel-target-mult" className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none" type="text" value={draft.targetMultiplicity ?? ''} onChange={(event) => handleMultiplicityChange('targetMultiplicity', event.target.value)} placeholder="e.g. 1..*" />
          {errors.targetMultiplicity && <p className="mt-1 text-[10px] text-[#ffb4ab]">{errors.targetMultiplicity}</p>}
        </div>

        {errors.relationship && <p role="alert" className="text-[10px] text-[#ffb4ab]">{errors.relationship}</p>}
        <button type="button" onClick={saveRelationship} className="w-full py-1.5 bg-[#4edea3] hover:bg-[#63ebb3] text-[#0a0e16] border border-[#4edea3] text-xs uppercase text-center font-bold transition-colors">{es.canvas.saveRelationship}</button>
        <button type="button" onClick={() => onDeleteRelationship(selectedRelationship.id)} className="w-full py-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#ffb4ab] border border-[#ffb4ab] text-xs uppercase text-center font-bold transition-colors">{es.canvas.deleteRelationship}</button>
      </div>
    </div>
  );
};
