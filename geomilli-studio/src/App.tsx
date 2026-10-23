import React, { useState } from 'react';
import { Navbar } from './components/Navbar';
import { PlaygroundTab } from './components/playground/PlaygroundTab';
import { ImporterTab } from './components/importer/ImporterTab';
import { MetricsTab } from './components/metrics/MetricsTab';

export function App() {
  const [activeTab, setActiveTab] = useState<'playground' | 'importer' | 'metrics'>('playground');

  return (
    <div className="min-h-screen flex flex-col bg-slate-950 text-slate-100">
      {/* Navigation Header */}
      <Navbar activeTab={activeTab} onTabChange={setActiveTab} />

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {activeTab === 'playground' && <PlaygroundTab />}
        {activeTab === 'importer' && <ImporterTab />}
        {activeTab === 'metrics' && <MetricsTab />}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-900 bg-slate-950 py-4 text-center text-xs text-slate-600">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-2">
          <span>Geomilli Open-Source Geospatial Search Engine &bull; Built with Go, Meilisearch & MapLibre</span>
          <span className="font-mono text-slate-500">v0.1.0 (Default region: BE / Belgium)</span>
        </div>
      </footer>
    </div>
  );
}

export default App;
