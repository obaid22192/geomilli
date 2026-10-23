import React from 'react';
import { Layers, Search, AlertCircle, BarChart3, Globe } from 'lucide-react';

interface ChartsProps {
  endpoints: Record<string, number>;
  topQueries: { query: string; count: number }[];
  zeroHitQueries: { query: string; count: number }[];
  classDistribution: Record<string, number>;
  countryDistribution: Record<string, number>;
}

export const Charts: React.FC<ChartsProps> = ({
  endpoints,
  topQueries,
  zeroHitQueries,
  classDistribution,
  countryDistribution,
}) => {
  // Compute max for relative bars
  const totalEndpointCalls = Object.values(endpoints).reduce((a, b) => a + b, 0) || 1;
  const maxClassCount = Math.max(...Object.values(classDistribution), 1);
  const maxCountryCount = Math.max(...Object.values(countryDistribution), 1);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {/* 1. Endpoint Usage Breakdown */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
            <BarChart3 className="w-4 h-4 text-brand-400" />
            <span>Endpoint Traffic Distribution</span>
          </h4>
          <span className="text-[11px] text-slate-500">
            {totalEndpointCalls.toLocaleString()} calls
          </span>
        </div>

        {Object.keys(endpoints).length === 0 ? (
          <div className="text-xs text-slate-500 italic py-6 text-center">No traffic recorded yet</div>
        ) : (
          <div className="flex flex-col gap-2 mt-1">
            {Object.entries(endpoints).map(([endpoint, count]) => {
              const pct = Math.round((count / totalEndpointCalls) * 100);
              return (
                <div key={endpoint} className="text-xs">
                  <div className="flex items-center justify-between text-slate-300 font-mono text-[11px] mb-1">
                    <span className="truncate">{endpoint}</span>
                    <span className="text-slate-400">
                      {count.toLocaleString()} ({pct}%)
                    </span>
                  </div>
                  <div className="w-full bg-slate-950 h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-brand-500 h-1.5 rounded-full"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 2. Taxonomy Class Distribution */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
            <Layers className="w-4 h-4 text-purple-400" />
            <span>Document Breakdown by Class</span>
          </h4>
          <span className="text-[11px] text-slate-500">19 OSM classes</span>
        </div>

        {Object.keys(classDistribution).length === 0 ? (
          <div className="text-xs text-slate-500 italic py-6 text-center">No indexed documents yet</div>
        ) : (
          <div className="flex flex-col gap-2 mt-1 max-h-56 overflow-y-auto pr-1">
            {Object.entries(classDistribution).map(([cls, count]) => {
              const pct = Math.round((count / maxClassCount) * 100);
              return (
                <div key={cls} className="text-xs">
                  <div className="flex items-center justify-between text-slate-300 text-[11px] mb-1">
                    <span>{cls}</span>
                    <span className="font-mono text-slate-400">{count.toLocaleString()} docs</span>
                  </div>
                  <div className="w-full bg-slate-950 h-1.5 rounded-full overflow-hidden">
                    <div
                      className="bg-purple-500 h-1.5 rounded-full"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 3. Top Searched Queries */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
            <Search className="w-4 h-4 text-emerald-400" />
            <span>Top Popular Search Queries</span>
          </h4>
          <span className="text-[11px] text-slate-500">Volume</span>
        </div>

        <div className="flex flex-col gap-1.5 max-h-48 overflow-y-auto">
          {topQueries.length === 0 ? (
            <div className="text-xs text-slate-600 italic py-2">No recorded queries yet</div>
          ) : (
            topQueries.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-2 rounded-lg bg-slate-950/60 border border-slate-800/80 text-xs"
              >
                <div className="flex items-center gap-2">
                  <span className="text-slate-600 font-mono text-[10px] w-4">{idx + 1}.</span>
                  <span className="text-slate-200 font-medium font-mono">"{item.query}"</span>
                </div>
                <span className="text-emerald-400 font-mono text-xs">{item.count} hits</span>
              </div>
            ))
          )}
        </div>
      </div>

      {/* 4. Zero-Hit Queries & Country Breakdown */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
            <AlertCircle className="w-4 h-4 text-amber-400" />
            <span>Zero-Hit Queries (Synonym / Gap Radar)</span>
          </h4>
          <span className="text-[11px] text-slate-500">Misses</span>
        </div>

        <div className="flex flex-col gap-1.5 max-h-48 overflow-y-auto">
          {zeroHitQueries.length === 0 ? (
            <div className="text-xs text-emerald-500/80 italic py-2">
              All tested queries returned hits!
            </div>
          ) : (
            zeroHitQueries.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-2 rounded-lg bg-slate-950/60 border border-slate-800/80 text-xs"
              >
                <span className="text-slate-300 font-mono">"{item.query}"</span>
                <span className="text-amber-400/90 font-mono text-xs">{item.count} misses</span>
              </div>
            ))
          )}
        </div>
      </div>

      {/* 5. Documents by Country */}
      <div className="md:col-span-2 bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold text-slate-200 flex items-center gap-1.5">
            <Globe className="w-4 h-4 text-sky-400" />
            <span>Indexed Document Distribution by Country</span>
          </h4>
          <span className="text-[11px] text-slate-500">Meilisearch Index</span>
        </div>

        {Object.keys(countryDistribution).length === 0 ? (
          <div className="text-xs text-slate-500 italic py-6 text-center">No countries indexed yet</div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2">
            {Object.entries(countryDistribution).map(([code, count]) => (
              <div
                key={code}
                className="p-3 rounded-lg bg-slate-950 border border-slate-800 flex flex-col justify-between"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-bold font-mono text-brand-400">{code}</span>
                  <span className="w-2 h-2 rounded-full bg-brand-500"></span>
                </div>
                <div className="text-sm font-bold font-mono text-slate-200">
                  {count.toLocaleString()}
                </div>
                <div className="text-[10px] text-slate-500 mt-0.5">documents</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
