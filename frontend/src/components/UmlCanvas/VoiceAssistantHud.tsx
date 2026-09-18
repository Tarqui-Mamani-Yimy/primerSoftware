import React, { useState, useEffect } from 'react';

interface VoiceAssistantHudProps {
  onApplyAction: (actionText: string) => void;
}

export const VoiceAssistantHud: React.FC<VoiceAssistantHudProps> = ({ onApplyAction }) => {
  const [isListening, setIsListening] = useState(true);
  const [currentPromptIndex, setCurrentPromptIndex] = useState(0);
  const [appliedFeedback, setAppliedFeedback] = useState(false);

  const prompts = [
    {
      heard: '“Relacionar Order con Payment mediante agregación 1 a 0..1”',
      action: 'AddAggregationEdge(src: Order, dst: Payment, card: [1, 0..1])',
      confidence: '98.6%'
    },
    {
      heard: '“Añadir atributo trackingNumber de tipo String a Order”',
      action: 'AddField(target: Order, name: "trackingNumber", type: "String")',
      confidence: '99.1%'
    },
    {
      heard: '“Cambiar estrategia de herencia JPA a SINGLE_TABLE”',
      action: 'SetInheritanceStrategy(strategy: "SINGLE")',
      confidence: '97.8%'
    },
    {
      heard: '“Generar endpoints de búsqueda paginada para Customer”',
      action: 'GenerateRepositoryQueries(entity: Customer, pagination: true)',
      confidence: '99.4%'
    }
  ];

  const current = prompts[currentPromptIndex];

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Spacebar when not in input
      if (e.code === 'Space' && (e.target as HTMLElement).tagName !== 'INPUT') {
        e.preventDefault();
        setIsListening(prev => !prev);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleApply = () => {
    setAppliedFeedback(true);
    onApplyAction(current.action);
    setTimeout(() => {
      setAppliedFeedback(false);
      setCurrentPromptIndex((prev) => (prev + 1) % prompts.length);
    }, 1200);
  };

  const handleNextSimulation = () => {
    setCurrentPromptIndex((prev) => (prev + 1) % prompts.length);
  };

  return (
    <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-30 max-w-2xl w-full px-4 pointer-events-auto">
      <div className="bg-[#0a0e16]/95 backdrop-blur-xl border border-[#4edea3]/40 shadow-2xl p-3">
        {/* Header */}
        <div className="flex items-center justify-between pb-2 border-b border-[#3c4a42]">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setIsListening(!isListening)}
              className="relative flex items-center justify-center w-5 h-5 bg-[#4edea3]/20 text-[#4edea3] hover:scale-110 transition-transform"
              title="Toggle Voice Capture (Space)"
            >
              <span className={`material-symbols-outlined text-xs ${isListening ? 'animate-pulse' : 'text-[#86948a]'}`}>
                {isListening ? 'mic' : 'mic_off'}
              </span>
              {isListening && <span className="absolute -top-1 -right-1 w-1.5 h-1.5 bg-[#4edea3]"></span>}
            </button>
            <span className="font-heading text-sm text-[#dfe2ee] font-bold">Local AI Voice Assistant</span>
            <span className="text-[10px] px-1.5 py-0.5 bg-[#1c2028] text-[#4edea3] font-bold border border-[#3c4a42] font-mono">
              Offline: Whisper.wasm
            </span>
            <span className="text-[10px] px-1.5 py-0.5 bg-[#1c2028] text-[#4cd7f6] font-bold font-mono">
              WebGPU 8.4ms
            </span>
          </div>

          <div className="flex items-center gap-1.5 font-mono">
            <span className="text-[10px] text-[#bbcabf] uppercase">Confidence</span>
            <span className="text-xs text-[#4edea3] font-bold">{current.confidence}</span>
          </div>
        </div>

        {/* Real-time speech recognition feed & SVG wave */}
        <div className="py-2.5 flex items-center justify-between gap-4 font-mono">
          <div className="flex-1 min-w-0">
            <div className="flex items-baseline gap-1.5">
              <span className="text-[#4cd7f6] text-xs font-bold">&gt;&gt;</span>
              <p className="text-xs text-[#dfe2ee] truncate">
                <span className="text-[#bbcabf] font-bold">Escuchando:</span>{' '}
                <span className="text-[#dfe2ee]">{current.heard}</span>
              </p>
            </div>
            <div className="flex items-center gap-1.5 mt-1">
              <span className="material-symbols-outlined text-xs text-[#4edea3]">auto_fix_high</span>
              <span className="text-[10px] text-[#4edea3] uppercase truncate font-bold">
                Synthesized Action: {current.action}
              </span>
            </div>
          </div>

          {/* Dynamic Audio Spectrum Waveform Visualization */}
          <div 
            onClick={handleNextSimulation}
            className="flex items-center gap-1 h-6 px-1.5 bg-[#1c2028] border border-[#3c4a42] cursor-pointer hover:border-[#4edea3]"
            title="Click to cycle simulated speech phrase"
          >
            <div className={`w-1 bg-[#4edea3] h-3 ${isListening ? 'animate-pulse' : 'opacity-40'}`}></div>
            <div className={`w-1 bg-[#4edea3] h-5 ${isListening ? 'animate-pulse' : 'opacity-40'}`} style={{ animationDelay: '100ms' }}></div>
            <div className={`w-1 bg-[#4cd7f6] h-2 ${isListening ? 'animate-pulse' : 'opacity-40'}`} style={{ animationDelay: '200ms' }}></div>
            <div className={`w-1 bg-[#4edea3] h-6 ${isListening ? 'animate-pulse' : 'opacity-40'}`} style={{ animationDelay: '150ms' }}></div>
            <div className={`w-1 bg-[#4cd7f6] h-4 ${isListening ? 'animate-pulse' : 'opacity-40'}`} style={{ animationDelay: '300ms' }}></div>
            <div className={`w-1 bg-[#4edea3] h-3 ${isListening ? 'animate-pulse' : 'opacity-40'}`} style={{ animationDelay: '250ms' }}></div>
            <div className="w-1 bg-[#86948a] h-1"></div>
          </div>
        </div>

        {/* Instant Action Preview Button Row */}
        <div className="flex items-center justify-between pt-2 border-t border-[#3c4a42] font-mono">
          <span className="text-[#86948a] text-[11px]">Press [Space] to speak • Say “Commit to DDL”</span>
          <div className="flex items-center gap-2">
            <button
              onClick={handleNextSimulation}
              className="px-2.5 py-1 bg-[#1c2028] hover:bg-[#262a33] text-[#bbcabf] hover:text-[#dfe2ee] text-xs uppercase transition-colors"
            >
              Skip
            </button>
            <button
              onClick={handleApply}
              className="px-3.5 py-1 bg-[#4edea3] hover:brightness-110 text-black text-xs uppercase font-bold flex items-center gap-1.5 transition-all shadow-[0_0_12px_rgba(78,222,163,0.3)] active:scale-95"
            >
              {appliedFeedback ? (
                <>
                  <span className="material-symbols-outlined text-xs font-bold text-black">done_all</span>
                  <span>Executed!</span>
                </>
              ) : (
                <>
                  <span className="material-symbols-outlined text-xs font-bold">check</span>
                  <span>Apply Instant Action</span>
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
