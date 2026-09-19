import { FormEvent, useEffect, useId, useRef, useState } from 'react';
import { X } from 'lucide-react';
import { es } from '../../i18n/es';

interface CreateProjectModalProps {
  open: boolean;
  submitting: boolean;
  error: string;
  onClose: () => void;
  onSubmit: (input: { name: string; description: string }) => void;
}

export function CreateProjectModal({ open, submitting, error, onClose, onSubmit }: CreateProjectModalProps) {
  const titleId = useId();
  const introId = useId();
  const nameId = useId();
  const nameErrorId = useId();
  const descriptionId = useId();
  const nameInputRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [nameError, setNameError] = useState('');

  useEffect(() => {
    if (!open) return;
    setName('');
    setDescription('');
    setNameError('');
    const frame = requestAnimationFrame(() => nameInputRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || submitting) return;
      event.preventDefault();
      onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [open, submitting, onClose]);

  if (!open) return null;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError(es.projects.createNameRequired);
      nameInputRef.current?.focus();
      return;
    }
    setNameError('');
    onSubmit({ name: trimmedName, description: description.trim() });
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 py-8 backdrop-blur-sm"
      onMouseDown={event => { if (event.target === event.currentTarget && !submitting) onClose(); }}
    >
      <div role="dialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={introId} className="w-full max-w-lg rounded-2xl border border-[#2b3b4c] bg-[#101722] p-6 shadow-2xl shadow-black/50 sm:p-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 id={titleId} className="font-heading text-xl font-bold tracking-tight text-white">{es.projects.createTitle}</h2>
            <p id={introId} className="mt-2 text-sm leading-6 text-[#9aa9ba]">{es.projects.createIntro}</p>
          </div>
          <button type="button" onClick={onClose} disabled={submitting} aria-label={es.projects.createClose} className="rounded-lg border border-[#354352] p-2 text-[#b9c5d3] transition hover:border-[#607185] hover:bg-[#172231] hover:text-white focus:outline-none focus:ring-2 focus:ring-[#70e6b6] disabled:cursor-not-allowed disabled:opacity-60">
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <form className="mt-6 space-y-5" onSubmit={handleSubmit} noValidate>
          <div>
            <label className="mb-2 block text-sm font-medium text-[#dfe7f1]" htmlFor={nameId}>{es.projects.createName}</label>
            <input
              ref={nameInputRef}
              id={nameId}
              value={name}
              onChange={event => { setName(event.target.value); if (nameError) setNameError(''); }}
              maxLength={150}
              required
              aria-invalid={nameError ? true : undefined}
              aria-describedby={nameError ? nameErrorId : undefined}
              placeholder={es.projects.createNamePlaceholder}
              className="w-full rounded-lg border border-[#364352] bg-[#0d141e] px-3 py-3 text-sm text-white outline-none transition placeholder:text-[#667588] focus:border-[#5ce0b2] focus:ring-2 focus:ring-[#5ce0b2]/20"
            />
            {nameError && <p id={nameErrorId} role="alert" className="mt-2 text-xs text-red-200">{nameError}</p>}
          </div>

          <div>
            <label className="mb-2 block text-sm font-medium text-[#dfe7f1]" htmlFor={descriptionId}>{es.projects.createDescription}</label>
            <textarea
              id={descriptionId}
              value={description}
              onChange={event => setDescription(event.target.value)}
              rows={3}
              maxLength={500}
              placeholder={es.projects.createDescriptionPlaceholder}
              className="w-full resize-none rounded-lg border border-[#364352] bg-[#0d141e] px-3 py-3 text-sm text-white outline-none transition placeholder:text-[#667588] focus:border-[#5ce0b2] focus:ring-2 focus:ring-[#5ce0b2]/20"
            />
          </div>

          {error && <p role="alert" className="rounded-lg border border-red-700 bg-red-950/40 p-3 text-xs text-red-200">{error}</p>}

          <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
            <button type="button" onClick={onClose} disabled={submitting} className="inline-flex items-center justify-center rounded-lg border border-[#354352] px-4 py-3 text-sm font-semibold text-[#b9c5d3] transition hover:border-[#607185] hover:bg-[#172231] hover:text-white focus:outline-none focus:ring-2 focus:ring-[#70e6b6] disabled:cursor-not-allowed disabled:opacity-60">
              {es.projects.createCancel}
            </button>
            <button type="submit" disabled={submitting} className="inline-flex items-center justify-center rounded-lg bg-[#1fc88e] px-4 py-3 text-sm font-bold text-[#052d20] transition hover:bg-[#58e0b0] focus:outline-none focus:ring-2 focus:ring-[#8cf3cf] focus:ring-offset-2 focus:ring-offset-[#101722] disabled:cursor-not-allowed disabled:opacity-60">
              {submitting ? es.projects.createSubmitting : es.projects.createSubmit}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
