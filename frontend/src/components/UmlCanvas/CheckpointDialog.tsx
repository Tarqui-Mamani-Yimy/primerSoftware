import React, { useState } from 'react';
import { es } from '../../i18n/es';

interface CheckpointDialogProps {
  busy: boolean;
  onCancel: () => void;
  onSubmit: (message: string) => void;
}

export const CheckpointDialog: React.FC<CheckpointDialogProps> = ({ busy, onCancel, onSubmit }) => {
  const [message, setMessage] = useState('');
  return (
    <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-[28rem] border border-[#3c4a42] bg-[#181c24] p-5 font-mono shadow-xl">
        <div className="flex items-center justify-between border-b border-[#3c4a42] pb-2">
          <span className="text-sm uppercase font-bold text-[#4edea3]">{es.canvas.createCheckpointTitle}</span>
          <button type="button" disabled={busy} onClick={onCancel} aria-label={es.canvas.close} className="text-[#bbcabf] hover:text-white">×</button>
        </div>
        <p className="mt-4 text-xs text-[#dfe2ee]">{es.canvas.createCheckpointPrompt}</p>
        <label htmlFor="checkpoint-message" className="mt-4 block text-[10px] uppercase text-[#bbcabf]">{es.canvas.createCheckpointMessageLabel}</label>
        <input
          id="checkpoint-message"
          type="text"
          value={message}
          disabled={busy}
          placeholder={es.canvas.createCheckpointMessagePlaceholder}
          onChange={(event) => setMessage(event.target.value)}
          className="mt-2 w-full bg-[#0a0e16] border border-[#3c4a42] px-2 py-1 text-sm text-[#dfe2ee] focus:outline-none focus:border-[#4edea3]"
        />
        <div className="mt-6 flex items-center justify-end gap-2">
          <button type="button" disabled={busy} onClick={onCancel} className="px-3 py-1 text-xs text-[#bbcabf] hover:text-white">{es.canvas.cancel}</button>
          <button
            type="button"
            disabled={busy}
            onClick={() => {
              const trimmed = message.trim();
              if (!trimmed) {
                // Mirror the canonical UI prompt: the empty message gets sent
                // verbatim with a short append so authorship is never lost.
                onSubmit(`${es.canvas.createCheckpointEmpty} | (${new Date().toISOString().slice(0, 16)})`);
                return;
              }
              onSubmit(trimmed);
            }}
            className="px-3 py-1 bg-[#4edea3] text-black text-xs uppercase font-bold hover:brightness-110 active:scale-95 transition-all"
          >
            {busy ? es.canvas.createCheckpointBusy : es.canvas.createCheckpointSubmit}
          </button>
        </div>
      </div>
    </div>
  );
};
