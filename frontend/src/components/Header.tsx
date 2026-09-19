import React, { useState } from 'react';
import { ActiveView } from '../types';
import { es } from '../i18n/es';

interface HeaderProps {
  activeView: ActiveView;
  onSelectView: (view: ActiveView) => void;
  onExport: (type: 'svg' | 'png' | 'xmi' | 'plantuml' | 'sql' | 'zip') => void | Promise<void>;
  projectName?: string;
  onBackToProjects?: () => void;
}

export const Header: React.FC<HeaderProps> = ({ activeView, onSelectView, onExport, projectName = 'E-Commerce Domain Model', onBackToProjects }) => {
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [showProjectMenu, setShowProjectMenu] = useState(false);
  const [showBranchMenu, setShowBranchMenu] = useState(false);
  const [currentBranch, setCurrentBranch] = useState('feat/payment-order-v2');

  const branches = [
    { name: 'feat/payment-order-v2', hash: '#7f3a9e', current: true },
    { name: 'main', hash: '#a1c490', current: false },
    { name: 'feat/graphql-federation', hash: '#e4b102', current: false }
  ];

  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-16 bg-[#0a0e16]/95 backdrop-blur-md border-b border-[#3c4a42]">
      <div className="w-full h-16 px-4 flex items-center justify-between gap-4">
        {/* Left branding & context selectors */}
        <div className="flex items-center gap-3">
          {/* Logo */}
          <div className="flex items-center gap-2 cursor-pointer" onClick={() => onSelectView('uml-canvas')}>
            <div className="w-8 h-8 rounded bg-gradient-to-br from-[#10b981] via-[#03b5d3] to-[#b090ff] p-[1.5px] flex items-center justify-center">
              <div className="w-full h-full bg-[#0a0e16] rounded flex items-center justify-center">
                <span className="material-symbols-outlined text-[#4edea3] text-lg">schema</span>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="font-heading font-bold text-base text-[#dfe2ee] tracking-tight uppercase">AI UML</span>
              <span className="text-[10px] px-1.5 py-0.5 bg-[#1c2028] text-[#4edea3] font-bold border border-[#3c4a42] font-mono">v2.4</span>
            </div>
          </div>

          <div className="h-4 w-px bg-[#3c4a42] hidden sm:block"></div>

          {/* Project Switcher */}
          <div className="relative hidden md:block">
            <button 
              onClick={() => setShowProjectMenu(!showProjectMenu)}
              className="flex items-center gap-1.5 px-2.5 py-1 bg-[#181c24] border border-[#3c4a42] hover:border-[#86948a] text-xs transition-colors"
            >
              <span className="material-symbols-outlined text-sm text-[#4cd7f6]">folder_open</span>
              <span className="text-[#dfe2ee] font-semibold">{projectName}</span>
              <span className="material-symbols-outlined text-sm text-[#bbcabf]">arrow_drop_down</span>
            </button>

            {showProjectMenu && (
              <div className="absolute left-0 mt-1 w-64 bg-[#1c2028] border border-[#3c4a42] shadow-2xl p-1 z-50 text-xs font-mono">
                <div className="px-2 py-1 text-[10px] text-[#86948a] uppercase">Active Domains</div>
                <button 
                  onClick={() => setShowProjectMenu(false)}
                  className="w-full text-left px-2 py-1.5 text-[#4edea3] bg-[#262a33] flex items-center justify-between"
                >
                  <span>Modelo de dominio e-commerce</span>
                  <span className="material-symbols-outlined text-xs">check</span>
                </button>
                <button 
                  onClick={() => setShowProjectMenu(false)}
                  className="w-full text-left px-2 py-1.5 text-[#dfe2ee] hover:bg-[#262a33] transition-colors"
                >
                  <span>Core bancario fintech</span>
                </button>
                <button 
                  onClick={() => setShowProjectMenu(false)}
                  className="w-full text-left px-2 py-1.5 text-[#dfe2ee] hover:bg-[#262a33] transition-colors"
                >
                  <span>Microservicios de telemetría IoT</span>
                </button>
              </div>
            )}
          </div>

          {/* Branch Switcher */}
          <div className="relative hidden lg:block">
            <button 
              onClick={() => setShowBranchMenu(!showBranchMenu)}
              className="flex items-center gap-1.5 px-2.5 py-1 bg-[#181c24] border border-[#3c4a42] hover:border-[#86948a] text-xs font-mono transition-colors"
            >
              <span className="material-symbols-outlined text-sm text-[#bbcabf]">fork_right</span>
              <span className="text-[#dfe2ee]">{currentBranch}</span>
              <span className="text-[10px] px-1 bg-[#262a33] text-[#bbcabf]">#7f3a9e</span>
              <span className="material-symbols-outlined text-sm text-[#bbcabf]">expand_more</span>
            </button>

            {showBranchMenu && (
              <div className="absolute left-0 mt-1 w-60 bg-[#1c2028] border border-[#3c4a42] shadow-2xl p-1 z-50 text-xs font-mono">
                <div className="px-2 py-1 text-[10px] text-[#86948a] uppercase">Ramas de Git</div>
                {branches.map(b => (
                  <button 
                    key={b.name}
                    onClick={() => {
                      setCurrentBranch(b.name);
                      setShowBranchMenu(false);
                    }}
                    className={`w-full text-left px-2 py-1.5 flex items-center justify-between hover:bg-[#262a33] ${b.name === currentBranch ? 'text-[#4edea3] font-bold' : 'text-[#dfe2ee]'}`}
                  >
                    <span>{b.name}</span>
                    <span className="text-[10px] text-[#86948a]">{b.hash}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Center Navigation Tabs */}
        <div className="flex items-center gap-2">
          <nav className="flex items-center h-16 overflow-x-auto">
            <button
              onClick={() => onSelectView('uml-canvas')}
              className={`h-16 flex items-center px-3.5 text-xs font-mono uppercase tracking-wider transition-colors ${
                activeView === 'uml-canvas'
                  ? 'bg-[#1c2028] text-[#4edea3] border-b-2 border-[#4edea3] font-bold'
                  : 'text-[#bbcabf] hover:text-[#dfe2ee] hover:bg-[#181c24]'
              }`}
            >
              <span className="material-symbols-outlined text-sm mr-1.5">schema</span>
              <span>{es.navigation.umlCanvas}</span>
            </button>

            <button
              onClick={() => onSelectView('backend-db-generator')}
              className={`h-16 flex items-center px-3.5 text-xs font-mono uppercase tracking-wider transition-colors ${
                activeView === 'backend-db-generator'
                  ? 'bg-[#1c2028] text-[#4edea3] border-b-2 border-[#4edea3] font-bold'
                  : 'text-[#bbcabf] hover:text-[#dfe2ee] hover:bg-[#181c24]'
              }`}
            >
              <span className="material-symbols-outlined text-sm mr-1.5 text-[#4cd7f6]">code</span>
              <span>{es.navigation.backend}</span>
            </button>

            <button
              onClick={() => onSelectView('git-diff-versions')}
              className={`h-16 flex items-center px-3.5 text-xs font-mono uppercase tracking-wider transition-colors ${
                activeView === 'git-diff-versions'
                  ? 'bg-[#1c2028] text-[#4edea3] border-b-2 border-[#4edea3] font-bold'
                  : 'text-[#bbcabf] hover:text-[#dfe2ee] hover:bg-[#181c24]'
              }`}
            >
              <span className="material-symbols-outlined text-sm mr-1.5 text-[#d0bcff]">history_toggle_off</span>
              <span>{es.navigation.diff}</span>
            </button>

            <button
              onClick={() => onSelectView('team-room-live')}
              className={`h-16 flex items-center px-3.5 text-xs font-mono uppercase tracking-wider transition-colors ${
                activeView === 'team-room-live'
                  ? 'bg-[#1c2028] text-[#4edea3] border-b-2 border-[#4edea3] font-bold'
                  : 'text-[#bbcabf] hover:text-[#dfe2ee] hover:bg-[#181c24]'
              }`}
            >
              <span className="material-symbols-outlined text-sm mr-1.5 text-[#4edea3]">groups</span>
              <span>Sesión colaborativa (en vivo)</span>
            </button>
          </nav>
        </div>

        {/* Right side controls */}
        <div className="flex items-center gap-3">
          {/* WebGPU Status Pill */}
          <div className="hidden xl:flex items-center gap-1.5 px-2.5 py-1 bg-[#181c24] border border-[#3c4a42]">
            <span className="inline-block w-2 h-2 rounded-full bg-[#4edea3] animate-pulse"></span>
            <span className="text-[10px] text-[#bbcabf] uppercase tracking-wider font-mono font-bold">
              Motor local: WebGPU ONNX activo
            </span>
          </div>

          {/* Export Dropdown */}
          <div className="relative">
            <button
              onClick={() => setShowExportMenu(!showExportMenu)}
              className="flex items-center gap-1.5 px-3 py-1 bg-[#1c2028] hover:bg-[#262a33] text-[#dfe2ee] border border-[#3c4a42] transition-colors text-xs font-mono uppercase tracking-wider font-semibold"
              type="button"
              aria-haspopup="menu"
              aria-expanded={showExportMenu}
              aria-controls="diagram-export-menu"
            >
              <span className="material-symbols-outlined text-sm text-[#4cd7f6]">ios_share</span>
              <span>{es.export.export}</span>
              <span className="material-symbols-outlined text-sm">expand_more</span>
            </button>

            {showExportMenu && (
              <div id="diagram-export-menu" className="absolute right-0 mt-1 w-56 bg-[#262a33] border border-[#3c4a42] shadow-2xl p-1 z-50 font-mono text-xs" role="menu" aria-label={es.export.exportMenu}>
                <button
                  onClick={() => { void onExport('svg'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#dfe2ee] hover:bg-[#1c2028] hover:text-[#4edea3] transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                  aria-label={es.export.downloadSvg}
                >
                  <span>{es.export.diagramSvg}</span>
                  <span className="text-[10px] text-[#86948a]">.svg</span>
                </button>
                <button
                  onClick={() => { void onExport('png'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#dfe2ee] hover:bg-[#1c2028] hover:text-[#4edea3] transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                  aria-label={es.export.downloadPng}
                >
                  <span>{es.export.diagramPng}</span>
                  <span className="text-[10px] text-[#86948a]">.png</span>
                </button>
                <div className="h-px bg-[#3c4a42] my-1"></div>
                <button
                  onClick={() => { onExport('xmi'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#dfe2ee] hover:bg-[#1c2028] hover:text-[#4edea3] transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                >
                  <span>{es.export.xmi}</span>
                  <span className="text-[10px] text-[#86948a]">.xmi</span>
                </button>
                <button
                  onClick={() => { onExport('plantuml'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#dfe2ee] hover:bg-[#1c2028] hover:text-[#4edea3] transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                >
                  <span>{es.export.plantuml}</span>
                  <span className="text-[10px] text-[#86948a]">.puml</span>
                </button>
                <button
                  onClick={() => { onExport('sql'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#dfe2ee] hover:bg-[#1c2028] hover:text-[#4edea3] transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                >
                  <span>{es.export.sql}</span>
                  <span className="text-[10px] text-[#86948a]">.sql</span>
                </button>
                <div className="h-px bg-[#3c4a42] my-1"></div>
                <button
                  onClick={() => { onExport('zip'); setShowExportMenu(false); }}
                  className="w-full text-left px-2.5 py-1.5 text-[#4edea3] hover:bg-[#1c2028] font-bold transition-colors flex items-center justify-between"
                  type="button"
                  role="menuitem"
                >
                  <span>{es.export.zip}</span>
                  <span className="material-symbols-outlined text-xs">archive</span>
                </button>
              </div>
            )}
          </div>

          <div className="h-4 w-px bg-[#3c4a42]"></div>

          {/* User Profile Avatar with Mic Badge */}
          <button type="button" onClick={onBackToProjects} className="relative flex items-center cursor-pointer group rounded-full focus:outline-none focus:ring-2 focus:ring-[#4edea3]" title={es.export.backToProjects} aria-label={es.export.backToProjects}>
            <div className="w-8 h-8 rounded-full bg-[#1c2028] border border-[#3c4a42] flex items-center justify-center text-xs font-bold text-[#4edea3]">
              ME
            </div>
            <span className="absolute -bottom-1 -right-1 flex items-center justify-center w-4 h-4 bg-[#b090ff] text-[#4600a7] rounded-full ring-2 ring-[#0f131c]">
              <span className="material-symbols-outlined text-[10px]">mic</span>
            </span>
          </button>
        </div>
      </div>
    </header>
  );
};
