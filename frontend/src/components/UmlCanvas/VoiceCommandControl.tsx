import React, { useEffect, useRef, useState } from 'react';
import { ApiError, voiceApi } from '../../api/diagramApi';
import { parseVoiceCommand, VoiceCommand } from '../../diagram/voiceCommands';
import { VoiceCommandHelpPanel } from './VoiceCommandHelpPanel';

interface VoiceCommandControlProps {
  disabled: boolean;
  onConfirmCommand: (command: VoiceCommand) => { ok: boolean; message: string };
}

type RecordingState = 'idle' | 'recording' | 'transcribing';

const supported = () => typeof window !== 'undefined' && 'MediaRecorder' in window && !!navigator.mediaDevices?.getUserMedia;

export const VoiceCommandControl: React.FC<VoiceCommandControlProps> = ({ disabled, onConfirmCommand }) => {
  const [state, setState] = useState<RecordingState>('idle');
  const [transcript, setTranscript] = useState('');
  const [command, setCommand] = useState<VoiceCommand | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [showHelp, setShowHelp] = useState(false);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);

  const releaseStream = () => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  };

  useEffect(() => () => {
    recorderRef.current?.state === 'recording' && recorderRef.current.stop();
    streamRef.current?.getTracks().forEach((track) => track.stop());
  }, []);

  const resetPreview = () => {
    setTranscript('');
    setCommand(null);
    setNotice(null);
  };

  const startRecording = async () => {
    if (disabled) return;
    if (!supported()) {
      setNotice('Tu navegador no admite grabación de audio. Usá un navegador compatible con MediaRecorder.');
      return;
    }
    resetPreview();
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mimeType = MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm' : undefined;
      const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
      streamRef.current = stream;
      recorderRef.current = recorder;
      chunksRef.current = [];
      recorder.ondataavailable = (event) => { if (event.data.size > 0) chunksRef.current.push(event.data); };
      recorder.onerror = () => { releaseStream(); setState('idle'); setNotice('No se pudo grabar el audio. Revisá el micrófono e intentá de nuevo.'); };
      recorder.onstop = async () => {
        releaseStream();
        const blob = new Blob(chunksRef.current, { type: recorder.mimeType || 'audio/webm' });
        if (blob.size === 0) { setState('idle'); setNotice('No se registró audio. Intentá de nuevo.'); return; }
        setState('transcribing');
        try {
          const response = await voiceApi.transcribe(blob);
          setTranscript(response.text);
          const parsed = parseVoiceCommand(response.text);
          setCommand(parsed);
          setNotice(parsed ? 'Revisá y confirmá el comando antes de modificar el diagrama.' : 'No entendí un comando compatible. Probá “crear clase Usuario”, “agregar atributo email de tipo String a Usuario” o “deshacer”.');
        } catch (error) {
          setNotice(error instanceof ApiError ? error.message : 'No se pudo transcribir el audio.');
        } finally {
          setState('idle');
          recorderRef.current = null;
        }
      };
      recorder.start();
      setState('recording');
    } catch {
      releaseStream();
      setNotice('No se pudo acceder al micrófono. Permití el acceso e intentá de nuevo.');
    }
  };

  const stopRecording = () => {
    if (recorderRef.current?.state === 'recording') recorderRef.current.stop();
  };

  const confirm = () => {
    if (!command) return;
    const result = onConfirmCommand(command);
    setNotice(result.message);
    if (result.ok) setCommand(null);
  };

  const commandLabel = command?.kind === 'create-class'
    ? `Crear clase ${command.name}`
    : command?.kind === 'create-relationship'
      ? `Relacionar ${command.sourceName} con ${command.targetName}`
      : command?.kind === 'undo-voice-command'
        ? 'Deshacer'
        : command?.kind === 'add-attribute'
          ? `Agregar atributo ${command.name} de tipo ${command.type} a ${command.className}`
          : command?.kind === 'add-method'
            ? `Agregar método ${command.name} de retorno ${command.returnType} a ${command.className}`
            : '';

  return <div className="border-l border-[#3c4a42] pl-2">
    <div className="flex items-center gap-2">
      <button type="button" disabled={disabled || state === 'transcribing'} onClick={state === 'recording' ? stopRecording : startRecording} className="disabled:cursor-not-allowed disabled:text-[#86948a] hover:text-[#4edea3]" aria-label={state === 'recording' ? 'Detener grabación de voz' : 'Grabar comando de voz'}>
        {state === 'recording' ? 'Detener voz' : state === 'transcribing' ? 'Transcribiendo…' : 'Comando de voz'}
      </button>
      <button type="button" onClick={() => setShowHelp(true)} className="border-l border-[#3c4a42] pl-2 hover:text-[#4edea3]" aria-label="Ver ayuda con los comandos de voz disponibles" title="Ayuda: comandos de voz">
        Ayuda
      </button>
    </div>
    {(transcript || notice) && <div role="status" aria-live="polite" className="absolute right-4 top-14 z-30 w-80 border border-[#4cd7f6] bg-[#1c2028] p-3 text-xs text-[#dfe2ee] shadow-xl">
      {transcript && <p><span className="text-[#bbcabf]">Transcripción:</span> {transcript}</p>}
      {notice && <p className="mt-2 text-[#bbcabf]">{notice}</p>}
      {command && <div className="mt-3 flex gap-2"><button type="button" onClick={confirm} className="border border-[#4edea3] px-2 py-1 text-[#4edea3]">Confirmar{command?.kind === 'undo-voice-command' ? ' deshacer' : `: ${commandLabel}`}</button><button type="button" onClick={resetPreview} className="border border-[#86948a] px-2 py-1">Cancelar</button></div>}
    </div>}
    {showHelp && <VoiceCommandHelpPanel onClose={() => setShowHelp(false)} />}
  </div>;
};
