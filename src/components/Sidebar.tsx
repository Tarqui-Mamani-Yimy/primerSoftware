import React from 'react';
import { ActiveView } from '../types';

interface SidebarProps {
  activeView: ActiveView;
  onSelectView: (view: ActiveView) => void;
  telemetryLatency?: string;
}

export const Sidebar: React.FC<SidebarProps> = ({ 
  activeView, 
  onSelectView, 
  telemetryLatency = 'ONNX 14ms' 
}) => {
  const navItems = [
    {
      id: 'uml-canvas' as ActiveView,
      label: 'UML Class Diagram',
      icon: 'account_tree',
      badge: null
    },
    {
      id: 'backend-db-generator' as ActiveView,
      label: 'Entity ERD & DDL',
      icon: 'database',
      badge: null
    },
    {
      id: 'git-diff-versions' as ActiveView,
      label: 'Branch & Schema Diff',
      icon: 'history_toggle_off',
      badge: null
    },
    {
      id: 'team-room-live' as ActiveView,
      label: 'Collaborative Session',
      icon: 'groups',
      badge: null
    },
    {
      id: 'ai-architect-console' as ActiveView,
      label: 'AI Architect Console',
      icon: 'auto_awesome',
      iconColor: 'text-[#d0bcff]',
      badge: 'AI'
    }
  ];

  return (
    <aside className="fixed left-0 top-16 bottom-0 w-64 bg-[#0a0e16] border-r border-[#3c4a42] z-40 flex flex-col justify-between overflow-y-auto">
      <div className="p-3">
        <div className="flex items-center justify-between px-1.5 py-1 mb-2 text-[#bbcabf]">
          <span className="text-[10px] uppercase tracking-wider font-mono font-bold">Project Explorer</span>
          <span className="material-symbols-outlined text-sm cursor-pointer hover:text-[#dfe2ee]" title="Explorer Settings">
            tune
          </span>
        </div>

        <nav className="flex flex-col gap-1">
          {navItems.map(item => {
            const isActive = activeView === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onSelectView(item.id)}
                className={`w-full flex items-center justify-between px-3 py-2 text-xs font-mono transition-colors text-left ${
                  isActive
                    ? 'bg-[#262a33] text-[#4edea3] font-bold border-l-2 border-[#4edea3]'
                    : 'text-[#bbcabf] hover:bg-[#1c2028] hover:text-[#dfe2ee]'
                }`}
              >
                <div className="flex items-center gap-2.5 truncate">
                  <span className={`material-symbols-outlined text-base ${item.iconColor || ''}`}>
                    {item.icon}
                  </span>
                  <span className="truncate">{item.label}</span>
                </div>
                {item.badge && (
                  <span className="text-[9px] px-1 bg-[#b090ff]/20 text-[#d0bcff] font-bold">
                    {item.badge}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Telemetry bottom badge */}
      <div className="p-3 border-t border-[#3c4a42] bg-[#181c24]">
        <div className="flex items-center justify-between font-mono">
          <span className="text-[10px] text-[#bbcabf] uppercase font-bold">Telemetry</span>
          <span className="text-xs text-[#4edea3] font-bold">{telemetryLatency}</span>
        </div>
        <div className="w-full bg-[#1c2028] h-1 mt-2 overflow-hidden">
          <div className="bg-[#4edea3] h-1 w-3/4 animate-pulse"></div>
        </div>
        <div className="flex items-center justify-between text-[9px] text-[#86948a] font-mono mt-1.5">
          <span>WebGPU Inference</span>
          <span>FPS: 60.0</span>
        </div>
      </div>
    </aside>
  );
};
