import React from 'react';
import { Compass, RefreshCw, BarChart2, BookOpen, Github } from 'lucide-react';

interface NavbarProps {
  activeTab: 'playground' | 'importer' | 'metrics';
  onTabChange: (tab: 'playground' | 'importer' | 'metrics') => void;
}

export const Navbar: React.FC<NavbarProps> = ({ activeTab, onTabChange }) => {
  return (
    <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur-md sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo & Branding */}
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-brand-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-brand-500/20 text-white font-bold text-lg">
              🌍
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-base text-slate-100 tracking-tight">
                  Geomilli <span className="text-brand-400 font-medium">Studio</span>
                </span>
                <span className="px-1.5 py-0.2 rounded text-[10px] font-mono bg-brand-500/10 text-brand-400 border border-brand-500/20">
                  v0.1.0
                </span>
              </div>
              <p className="text-[11px] text-slate-400 hidden sm:block">
                OpenStreetMap & Geospatial Search Platform powered by Meilisearch
              </p>
            </div>
          </div>

          {/* Core Tab Navigation */}
          <nav className="flex items-center gap-1 bg-slate-900/90 p-1 rounded-xl border border-slate-800/80 shadow-inner">
            <button
              onClick={() => onTabChange('playground')}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
                activeTab === 'playground'
                  ? 'bg-brand-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-850'
              }`}
            >
              <Compass className="w-4 h-4" />
              <span>Map Explorer</span>
            </button>

            <button
              onClick={() => onTabChange('importer')}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
                activeTab === 'importer'
                  ? 'bg-brand-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-850'
              }`}
            >
              <RefreshCw className="w-4 h-4" />
              <span>Importer Center</span>
            </button>

            <button
              onClick={() => onTabChange('metrics')}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition ${
                activeTab === 'metrics'
                  ? 'bg-brand-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-850'
              }`}
            >
              <BarChart2 className="w-4 h-4" />
              <span>System & Metrics</span>
            </button>
          </nav>

          {/* External Links */}
          <div className="flex items-center gap-3">
            <a
              href="/docs/"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 px-2.5 py-1.5 rounded-lg hover:bg-slate-900 transition"
            >
              <BookOpen className="w-3.5 h-3.5" />
              <span className="hidden md:inline">Docs</span>
            </a>
            <a
              href="https://github.com"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 px-2.5 py-1.5 rounded-lg hover:bg-slate-900 transition"
            >
              <Github className="w-3.5 h-3.5" />
              <span className="hidden md:inline">GitHub</span>
            </a>
          </div>
        </div>
      </div>
    </header>
  );
};
