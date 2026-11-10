import React from 'react';
import { 
  Play, 
  CircleDot, 
  RefreshCw, 
  ArrowRightLeft, 
  ShieldCheck, 
  Trash2, 
  FileCode2, 
  Activity, 
  Split, 
  Sliders,
  Download
} from 'lucide-react';
import { ProxyMode } from '../types/api';

interface NavbarProps {
  currentTab: string;
  setCurrentTab: (tab: string) => void;
  mode: ProxyMode;
  onModeChange: (newMode: ProxyMode) => void;
  activeCassette: string;
  interactionsCount: number;
  onPurge: () => void;
  onOpenCAModal: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  currentTab,
  setCurrentTab,
  mode,
  onModeChange,
  activeCassette,
  interactionsCount,
  onPurge,
  onOpenCAModal,
}) => {
  const getModeColor = (m: ProxyMode) => {
    switch (m) {
      case 'record':
        return 'bg-rose-500/20 text-rose-300 border-rose-500/40 hover:bg-rose-500/30';
      case 'replay':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40 hover:bg-emerald-500/30';
      case 'auto':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/40 hover:bg-amber-500/30';
      case 'passthrough':
        return 'bg-slate-500/20 text-slate-300 border-slate-500/40 hover:bg-slate-500/30';
    }
  };

  const navItems = [
    { id: 'waterfall', label: 'Waterfall', icon: Activity, count: interactionsCount },
    { id: 'cassettes', label: 'Cassettes', icon: FileCode2 },
    { id: 'diff', label: 'Diff Inspector', icon: Split },
    { id: 'redaction', label: 'Redaction Sandbox', icon: ShieldCheck },
    { id: 'chaos', label: 'Chaos Simulator', icon: Sliders },
  ];

  return (
    <header className="border-b border-slate-800 bg-[#0c1220]/90 backdrop-blur sticky top-0 z-30 px-4 py-2.5">
      <div className="max-w-7xl mx-auto flex items-center justify-between gap-4">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400 shadow-sm shadow-indigo-500/10">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect width="20" height="14" x="2" y="5" rx="2"/>
              <circle cx="8" cy="12" r="2"/>
              <circle cx="16" cy="12" r="2"/>
              <path d="M6 12h4"/>
              <path d="M14 12h4"/>
            </svg>
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-semibold text-slate-100 tracking-tight text-base">CassetteCraft</span>
              <span className="text-[10px] font-mono uppercase bg-indigo-950/60 text-indigo-300 border border-indigo-700/50 px-1.5 py-0.5 rounded">v1.0</span>
            </div>
            <div className="text-xs text-slate-400 flex items-center gap-1.5 font-mono">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              <span>{activeCassette || 'default.yaml'}</span>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <nav className="flex items-center gap-1 bg-slate-900/80 border border-slate-800 p-1 rounded-lg">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = currentTab === item.id;
            return (
              <button
                key={item.id}
                onClick={() => setCurrentTab(item.id)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                  isActive
                    ? 'bg-indigo-600 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{item.label}</span>
                {item.count !== undefined && item.count > 0 && (
                  <span className={`text-[10px] px-1.5 py-0.2 rounded-full font-mono ${
                    isActive ? 'bg-indigo-700 text-indigo-100' : 'bg-slate-800 text-slate-300'
                  }`}>
                    {item.count}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Mode Switcher & Actions */}
        <div className="flex items-center gap-2">
          {/* Mode Switcher */}
          <div className="flex items-center bg-slate-900 border border-slate-800 rounded-lg p-0.5">
            {(['record', 'replay', 'auto', 'passthrough'] as ProxyMode[]).map((m) => {
              const isSelected = mode === m;
              return (
                <button
                  key={m}
                  onClick={() => onModeChange(m)}
                  className={`text-xs capitalize font-medium px-2.5 py-1 rounded-md transition-all flex items-center gap-1.5 ${
                    isSelected
                      ? getModeColor(m) + ' border shadow-xs font-semibold'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  {m === 'record' && <CircleDot className="w-3 h-3" />}
                  {m === 'replay' && <Play className="w-3 h-3" />}
                  {m === 'auto' && <RefreshCw className="w-3 h-3" />}
                  {m === 'passthrough' && <ArrowRightLeft className="w-3 h-3" />}
                  <span>{m}</span>
                </button>
              );
            })}
          </div>

          {/* CA Certificate Button */}
          <button
            onClick={onOpenCAModal}
            className="flex items-center gap-1.5 bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors"
            title="Download Root CA Certificate"
          >
            <Download className="w-3.5 h-3.5 text-indigo-400" />
            <span>Root CA</span>
          </button>

          {/* Purge Buffer */}
          <button
            onClick={onPurge}
            className="p-1.5 text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors border border-transparent hover:border-rose-500/20"
            title="Purge live stream buffer"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>
  );
};
