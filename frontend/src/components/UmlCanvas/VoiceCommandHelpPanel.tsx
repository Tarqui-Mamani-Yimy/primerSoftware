import React, { useEffect, useId, useRef } from 'react';
import {
  voiceCommandHelpGroups,
  voiceCommandHelpNotes,
} from '../../diagram/voiceCommandHelp';

interface VoiceCommandHelpPanelProps {
  onClose: () => void;
}

/** Modal listing every voice command the diagram accepts, grouped by intent. */
export const VoiceCommandHelpPanel: React.FC<VoiceCommandHelpPanelProps> = ({ onClose }) => {
  const titleId = useId();
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const frame = requestAnimationFrame(() => closeButtonRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 py-8"
      onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="max-h-[85vh] w-full max-w-2xl overflow-y-auto border border-[#3c4a42] bg-[#181c24] p-5 font-mono shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-[#3c4a42] pb-2">
          <span id={titleId} className="text-sm uppercase font-bold text-[#4edea3]">Comandos de voz</span>
          <button ref={closeButtonRef} type="button" onClick={onClose} aria-label="Cerrar ayuda de comandos de voz" className="text-[#bbcabf] hover:text-white">×</button>
        </div>

        <div className="mt-4 space-y-5 text-xs text-[#dfe2ee]">
          {voiceCommandHelpGroups.map((group) => (
            <section key={group.id}>
              <h3 className="text-[13px] font-bold text-[#4cd7f6]">{group.title}</h3>
              <p className="mt-1 text-[#bbcabf]">{group.description}</p>
              <ul className="mt-2 space-y-1.5">
                {group.examples.map((example) => (
                  <li key={example.phrase} className="border-l-2 border-[#3c4a42] pl-2">
                    <span className="text-[#4edea3]">“{example.phrase}”</span>
                    <span className="block text-[#86948a]">{example.note}</span>
                  </li>
                ))}
              </ul>
              {group.vocabulary && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {group.vocabulary.map((entry) => (
                    <span key={`${group.id}-${entry.phrase}`} className="border border-[#3c4a42] px-1.5 py-0.5 text-[10px] text-[#bbcabf]">
                      {entry.phrase} <span className="text-[#4edea3]">→</span> {entry.value}
                    </span>
                  ))}
                </div>
              )}
            </section>
          ))}

          <section>
            <h3 className="text-[13px] font-bold text-[#4cd7f6]">Notas</h3>
            <ul className="mt-2 list-disc space-y-1 pl-4 text-[#bbcabf]">
              {voiceCommandHelpNotes.map((note) => <li key={note}>{note}</li>)}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
};
