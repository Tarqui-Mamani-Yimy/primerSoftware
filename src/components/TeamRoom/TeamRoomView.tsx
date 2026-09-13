import React, { useState } from 'react';

export const TeamRoomView: React.FC = () => {
  const [messages, setMessages] = useState([
    { id: 1, sender: 'Sarah Chen', role: 'Lead Architect', text: 'Hey team, verified that JPA JOINED strategy works cleanest with Flyway for PostgreSQL 16 partition queries.', time: '14:28' },
    { id: 2, sender: 'Alex Kumar', role: 'Backend Dev', text: 'Checked the indexes: idx_order_customer_id and idx_order_created_at are generated properly in V1__init_schema.sql.', time: '14:31' },
    { id: 3, sender: 'You', role: 'Principal Engineer', text: 'Nice! The offline WebGPU ONNX engine is compiling the AST directly with 0 warnings.', time: '14:35' }
  ]);
  const [inputMsg, setInputMsg] = useState('');
  const [micMuted, setMicMuted] = useState(false);
  const [screenShared, setScreenShared] = useState(false);

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputMsg.trim()) return;
    setMessages([...messages, {
      id: Date.now(),
      sender: 'You',
      role: 'Principal Engineer',
      text: inputMsg.trim(),
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }]);
    setInputMsg('');
  };

  return (
    <div className="flex flex-col w-full min-h-[calc(100vh-4rem)] bg-[#0f131c] font-mono">
      {/* Top Session Status Bar */}
      <div className="p-3 bg-[#181c24] border-b border-[#3c4a42] flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 px-2 py-1 bg-[#10b981]/20 border border-[#4edea3]">
            <span className="w-2 h-2 rounded-full bg-[#4edea3] animate-ping"></span>
            <span className="text-xs text-[#4edea3] font-bold">LIVE WEBRTC ROOM</span>
          </div>
          <span className="text-xs text-[#bbcabf]">Session: E-Commerce Architecture Review</span>
        </div>

        {/* Room Audio Controls */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setMicMuted(!micMuted)}
            className={`flex items-center gap-1.5 px-3 py-1 text-xs border transition-colors ${
              micMuted ? 'bg-[#93000a] text-white border-red-500' : 'bg-[#1c2028] text-[#4edea3] border-[#3c4a42]'
            }`}
          >
            <span className="material-symbols-outlined text-sm">{micMuted ? 'mic_off' : 'mic'}</span>
            <span>{micMuted ? 'Mic Muted' : 'Mic On'}</span>
          </button>

          <button
            onClick={() => setScreenShared(!screenShared)}
            className={`flex items-center gap-1.5 px-3 py-1 text-xs border transition-colors ${
              screenShared ? 'bg-[#03b5d3] text-black border-[#4cd7f6]' : 'bg-[#1c2028] text-[#4cd7f6] border-[#3c4a42]'
            }`}
          >
            <span className="material-symbols-outlined text-sm">screen_share</span>
            <span>{screenShared ? 'Sharing Canvas' : 'Share Canvas'}</span>
          </button>
        </div>
      </div>

      <div className="grid grid-cols-12 flex-1">
        {/* Architects in session */}
        <div className="col-span-12 md:col-span-4 bg-[#181c24] border-r border-[#3c4a42] p-4 space-y-3">
          <div className="text-[10px] uppercase font-bold text-[#bbcabf]">Active Participants (3)</div>

          {[
            { name: 'Sarah Chen', role: 'Lead Architect', initials: 'SC', color: 'bg-[#4cd7f6] text-black', talking: true },
            { name: 'Alex Kumar', role: 'Backend Dev', initials: 'AK', color: 'bg-[#d0bcff] text-black', talking: false },
            { name: 'You (Lead)', role: 'Principal Engineer', initials: 'ME', color: 'bg-[#4edea3] text-black', talking: !micMuted }
          ].map(p => (
            <div key={p.name} className="p-3 bg-[#1c2028] border border-[#3c4a42] flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-9 h-9 rounded-full ${p.color} font-bold flex items-center justify-center text-xs relative`}>
                  {p.initials}
                  {p.talking && (
                    <span className="absolute -top-1 -right-1 w-3 h-3 bg-[#4edea3] rounded-full ring-2 ring-[#1c2028] animate-pulse"></span>
                  )}
                </div>
                <div>
                  <div className="text-xs text-white font-bold">{p.name}</div>
                  <div className="text-[10px] text-[#86948a]">{p.role}</div>
                </div>
              </div>

              {p.talking ? (
                <div className="flex items-center gap-0.5 h-4">
                  <span className="w-1 bg-[#4edea3] h-2 animate-pulse"></span>
                  <span className="w-1 bg-[#4edea3] h-4 animate-pulse"></span>
                  <span className="w-1 bg-[#4edea3] h-3 animate-pulse"></span>
                </div>
              ) : (
                <span className="material-symbols-outlined text-xs text-[#86948a]">mic_off</span>
              )}
            </div>
          ))}

          {/* Quick collaboration tool */}
          <div className="p-3 bg-[#0a0e16] border border-[#3c4a42] space-y-2 mt-6">
            <span className="text-[10px] uppercase text-[#4edea3] font-bold block">Live Scratchpad</span>
            <p className="text-[11px] text-[#bbcabf]">Shared AST state automatically propagates to all collaborator viewports via WebRTC DataChannel.</p>
          </div>
        </div>

        {/* Live Chat & Log */}
        <div className="col-span-12 md:col-span-8 bg-[#0a0e16] flex flex-col justify-between">
          <div className="p-4 space-y-3 overflow-y-auto max-h-[calc(100vh-12rem)]">
            {messages.map(m => (
              <div key={m.id} className="p-2.5 bg-[#1c2028] border border-[#3c4a42] space-y-1">
                <div className="flex items-center justify-between text-[10px]">
                  <span className="text-[#4edea3] font-bold">{m.sender} <span className="text-[#86948a]">({m.role})</span></span>
                  <span className="text-[#86948a]">{m.time}</span>
                </div>
                <p className="text-xs text-[#dfe2ee]">{m.text}</p>
              </div>
            ))}
          </div>

          <form onSubmit={handleSend} className="p-3 bg-[#181c24] border-t border-[#3c4a42] flex items-center gap-2">
            <input
              type="text"
              value={inputMsg}
              onChange={(e) => setInputMsg(e.target.value)}
              placeholder="Escribe un mensaje para el equipo de arquitectura..."
              className="flex-1 bg-[#0a0e16] border border-[#3c4a42] px-3 py-2 text-xs text-white focus:outline-none focus:border-[#4edea3]"
            />
            <button type="submit" className="px-4 py-2 bg-[#4edea3] text-black font-bold text-xs uppercase">
              Enviar
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};
