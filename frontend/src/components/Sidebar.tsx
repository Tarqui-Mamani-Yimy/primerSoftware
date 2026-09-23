import React from 'react';
import { ActiveView } from '../types';
import { es } from '../i18n/es';

interface SidebarProps {
  activeView: ActiveView;
  onSelectView: (view: ActiveView) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeView,
  onSelectView
}) => {
  const navItems = [
    {
      id: 'uml-canvas' as ActiveView,
      label: es.navigation.umlCanvas,
      icon: 'account_tree',
      iconColor: null as string | null,
      badge: null
    },
    {
      id: 'backend-db-generator' as ActiveView,
      label: es.navigation.backend,
      icon: 'database',
      iconColor: null as string | null,
      badge: null
    }
  ];

  return (
    <aside className="fixed left-0 top-16 bottom-0 w-64 bg-[#0a0e16] border-r border-[#3c4a42] z-40 flex flex-col justify-between overflow-y-auto">
      <div className="p-3">
        <div className="flex items-center justify-between px-1.5 py-1 mb-2 text-[#bbcabf]">
          <span className="text-[10px] uppercase tracking-wider font-mono font-bold">{es.navigation.projectExplorer}</span>
          <span className="material-symbols-outlined text-sm cursor-pointer hover:text-[#dfe2ee]" title={es.navigation.explorerSettings} aria-label={es.navigation.explorerSettings}>
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
    </aside>
  );
};
