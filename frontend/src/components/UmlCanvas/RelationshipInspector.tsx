import React, { useState } from 'react';
import { UMLRelationship } from '../../types';
import { isValidMultiplicity } from '../../diagram/relationshipHelpers';

interface RelationshipInspectorProps {
  selectedRelationship: UMLRelationship;
  sourceName: string;
  targetName: string;
  onUpdateRelationship: (updated: UMLRelationship) => void;
  onDeleteRelationship: (id: string) => void;
  onClose: () => void;
}

export const RelationshipInspector: React.FC<RelationshipInspectorProps> = ({
  selectedRelationship,
  sourceName,
  targetName,
  onUpdateRelationship,
  onDeleteRelationship,
  onClose,
}) => {
  const [errors, setErrors] = useState<{ sourceMultiplicity?: string; targetMultiplicity?: string }>({});

  const handleMultiplicityChange = (field: 'sourceMultiplicity' | 'targetMultiplicity', value: string) => {
    if (!isValidMultiplicity(value)) {
      setErrors((current) => ({ ...current, [field]: 'Use e.g. 1, *, 0..1, 1..*.' }));
      return;
    }
    setErrors((current) => ({ ...current, [field]: undefined }));
    onUpdateRelationship({ ...selectedRelationship, [field]: value });
  };

  return (
    <div className="w-80 flex-shrink-0 bg-[#181c24] flex flex-col z-20 shadow-xl border-l border-[#3c4a42] overflow-hidden">
      <div className="p-3 bg-[#1c2028] border-b border-[#3c4a42] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#4cd7f6] text-sm">alt_route</span>
          <span className="font-heading text-sm uppercase tracking-wider text-[#dfe2ee] font-bold">Relationship</span>
        </div>
        <button type="button" onClick={onClose} className="text-[#bbcabf] hover:text-[#dfe2ee] transition-colors" aria-label="Close relationship inspector">
          <span className="material-symbols-outlined text-sm">close</span>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-4 font-mono">
        <div className="space-y-2 bg-[#1c2028] p-2.5 border border-[#3c4a42]">
          <span className="text-[10px] uppercase font-bold text-[#bbcabf]">Connection</span>
          <p className="text-xs text-[#dfe2ee]">
            {sourceName} <span className="text-[#86948a]">→</span> {targetName}
          </p>
          <p className="text-[10px] text-[#86948a]">Type: {selectedRelationship.type}</p>
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-label">Label (central keyword)</label>
          <input
            id="rel-label"
            className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
            type="text"
            value={selectedRelationship.label ?? ''}
            onChange={(event) => onUpdateRelationship({ ...selectedRelationship, label: event.target.value })}
            placeholder="e.g. places"
          />
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-source-mult">Source multiplicity (near {sourceName})</label>
          <input
            id="rel-source-mult"
            className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
            type="text"
            value={selectedRelationship.sourceMultiplicity ?? ''}
            onChange={(event) => handleMultiplicityChange('sourceMultiplicity', event.target.value)}
            placeholder="e.g. 1"
          />
          {errors.sourceMultiplicity && <p className="mt-1 text-[10px] text-[#ffb4ab]">{errors.sourceMultiplicity}</p>}
        </div>

        <div>
          <label className="text-[9px] uppercase text-[#bbcabf] block mb-1" htmlFor="rel-target-mult">Target multiplicity (near {targetName})</label>
          <input
            id="rel-target-mult"
            className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
            type="text"
            value={selectedRelationship.targetMultiplicity ?? ''}
            onChange={(event) => handleMultiplicityChange('targetMultiplicity', event.target.value)}
            placeholder="e.g. 1..*"
          />
          {errors.targetMultiplicity && <p className="mt-1 text-[10px] text-[#ffb4ab]">{errors.targetMultiplicity}</p>}
        </div>

        <button
          type="button"
          onClick={() => onDeleteRelationship(selectedRelationship.id)}
          className="w-full py-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#ffb4ab] border border-[#ffb4ab] text-xs uppercase text-center font-bold transition-colors"
        >
          Delete relationship
        </button>
      </div>
    </div>
  );
};
