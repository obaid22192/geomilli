import React from 'react';
import { Database, CheckCircle2, Cloud, Server, AlertCircle } from 'lucide-react';

interface ConnectionBadgeProps {
  meili: {
    mode: 'local' | 'cloud';
    host: string;
    healthy: boolean;
    places_count: number;
    index_size_mb?: number;
  };
}

export const ConnectionBadge: React.FC<ConnectionBadgeProps> = ({ meili }) => {
  const isCloud = meili?.mode === 'cloud';
  const isHealthy = meili?.healthy !== false;

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
      {/* Left: Mode & Host */}
      <div className="flex items-center gap-3">
        <div
          className={`w-11 h-11 rounded-xl flex items-center justify-center ${
            isCloud
              ? 'bg-purple-500/15 text-purple-400 border border-purple-500/30'
              : 'bg-brand-500/15 text-brand-400 border border-brand-500/30'
          }`}
        >
          {isCloud ? <Cloud className="w-6 h-6" /> : <Server className="w-6 h-6" />}
        </div>

        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-bold text-slate-100">
              Meilisearch {isCloud ? 'Cloud Deployment' : 'Local Container'}
            </h3>
            <span
              className={`px-2 py-0.5 rounded-full text-[10px] font-medium flex items-center gap-1 ${
                isHealthy
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                  : 'bg-rose-500/10 text-rose-400 border border-rose-500/30'
              }`}
            >
              {isHealthy ? (
                <>
                  <CheckCircle2 className="w-2.5 h-2.5" />
                  Healthy
                </>
              ) : (
                <>
                  <AlertCircle className="w-2.5 h-2.5" />
                  Unhealthy
                </>
              )}
            </span>
          </div>
          <div className="text-xs font-mono text-slate-400 mt-0.5">
            Host: {meili?.host || 'http://meilisearch:7700'}
          </div>
        </div>
      </div>

      {/* Right: Index stats */}
      <div className="flex items-center gap-6 border-t md:border-t-0 md:border-l border-slate-800 pt-3 md:pt-0 md:pl-6 text-xs">
        <div>
          <span className="text-slate-500 block text-[10px] uppercase font-semibold">
            Active Index
          </span>
          <span className="font-mono text-slate-200 font-bold text-sm flex items-center gap-1.5 mt-0.5">
            <Database className="w-3.5 h-3.5 text-brand-400" />
            places
          </span>
        </div>

        <div>
          <span className="text-slate-500 block text-[10px] uppercase font-semibold">
            Indexed Documents
          </span>
          <span className="font-mono text-slate-100 font-bold text-sm mt-0.5 block">
            {(meili?.places_count ?? 0).toLocaleString()}
          </span>
        </div>

        {meili?.index_size_mb != null && (
          <div>
            <span className="text-slate-500 block text-[10px] uppercase font-semibold">
              Disk Footprint
            </span>
            <span className="font-mono text-slate-100 font-bold text-sm mt-0.5 block">
              {(meili.index_size_mb || 0).toFixed(1)} MB
            </span>
          </div>
        )}
      </div>
    </div>
  );
};
