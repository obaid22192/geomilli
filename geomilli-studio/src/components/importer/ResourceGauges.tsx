import React from 'react';
import { ImporterResources } from '../../types';
import { Cpu, HardDrive, Database, Zap } from 'lucide-react';

interface ResourceGaugesProps {
  resources: ImporterResources;
}

export const ResourceGauges: React.FC<ResourceGaugesProps> = ({ resources }) => {
  const heap = resources?.heap_alloc_mb || 0;
  const sysMem = resources?.sys_mem_mb || 0;
  const diskStaging = resources?.disk_staging_mb || 0;
  const diskCache = resources?.disk_cache_mb || 0;
  const sqliteEntries = resources?.sqlite_cache_entries || 0;

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {/* Go Heap RAM */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">Go Heap RAM</span>
          <Cpu className="w-4 h-4 text-brand-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {heap.toFixed(1)} <span className="text-xs font-normal text-slate-400">MB</span>
          </div>
          <div className="flex items-center gap-1.5 mt-1 text-[10px] text-emerald-400">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            O(1) Streaming Low RAM
          </div>
        </div>
      </div>

      {/* Sys Memory */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">Process RSS</span>
          <Zap className="w-4 h-4 text-amber-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {sysMem.toFixed(1)} <span className="text-xs font-normal text-slate-400">MB</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">Total process overhead</div>
        </div>
      </div>

      {/* Disk Staging */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">Disk Staging</span>
          <HardDrive className="w-4 h-4 text-sky-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {diskStaging.toFixed(1)} <span className="text-xs font-normal text-slate-400">MB</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">Intermediates cleaned on finish</div>
        </div>
      </div>

      {/* SQLite WAL Cache */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">SQLite WAL Cache</span>
          <Database className="w-4 h-4 text-purple-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {sqliteEntries.toLocaleString()} <span className="text-xs font-normal text-slate-400">keys</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">
            {diskCache.toFixed(1)} MB cache database
          </div>
        </div>
      </div>
    </div>
  );
};
