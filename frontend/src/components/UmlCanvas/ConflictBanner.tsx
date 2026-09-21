import React from 'react';
import { UMLDiagramDocument } from '../../types';
import { es } from '../../i18n/es';

interface ConflictBannerProps {
  conflicting: UMLDiagramDocument;
  localName: string;
  onResolve: (keepMine: boolean) => void;
}

export const ConflictBanner: React.FC<ConflictBannerProps> = ({ conflicting, localName, onResolve }) => {
  const remoteVersion = conflicting.version ?? 0;
  const remoteReview = conflicting.reviewNumber ?? 0;
  const remoteName = conflicting.name;
  return (
    <div role="alert" className="fixed bottom-4 left-1/2 z-40 w-[36rem] -translate-x-1/2 border border-[#ffb4ab] bg-[#1c2028] p-4 font-mono text-xs text-[#dfe2ee] shadow-xl">
      <p className="uppercase font-bold text-[#ffb4ab]">{es.canvas.checkpointConflict}</p>
      <p className="mt-2">
        Tu versión local: <strong className="text-white">{localName}</strong>. Versión remota: <strong className="text-white">{remoteName}</strong> (v{remoteVersion}, r{remoteReview}).
      </p>
      <div className="mt-3 flex items-center justify-end gap-2">
        <button type="button" onClick={() => onResolve(true)} className="px-3 py-1 border border-[#ffb4ab] text-[#ffb4ab] hover:bg-[#ffb4ab] hover:text-[#1c2028] transition-colors uppercase font-bold">
          {es.canvas.checkpointConflictKeepMine}
        </button>
        <button type="button" onClick={() => onResolve(false)} className="px-3 py-1 bg-[#4edea3] text-black uppercase font-bold hover:brightness-110 active:scale-95 transition-all">
          {es.canvas.checkpointConflictReload}
        </button>
      </div>
    </div>
  );
};
