import React from 'react';
import { Activity, Zap, Clock, ShieldAlert } from 'lucide-react';

interface TelemetryCardsProps {
  totalRequests: number;
  qps: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
}

export const TelemetryCards: React.FC<TelemetryCardsProps> = ({
  totalRequests,
  qps,
  p50Ms,
  p95Ms,
  p99Ms,
}) => {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
      {/* Total Requests */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">Total Requests</span>
          <Activity className="w-4 h-4 text-brand-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {(totalRequests ?? 0).toLocaleString()}
          </div>
          <div className="text-[10px] text-slate-500 mt-1">Processed by geomilli-api</div>
        </div>
      </div>

      {/* QPS */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">Throughput (QPS)</span>
          <Zap className="w-4 h-4 text-amber-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {(qps ?? 0).toFixed(1)} <span className="text-xs font-normal text-slate-400">req/s</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">Real-time throughput</div>
        </div>
      </div>

      {/* p50 Latency */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">p50 Latency (Median)</span>
          <Clock className="w-4 h-4 text-emerald-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-emerald-400">
            {(p50Ms ?? 0).toFixed(1)} <span className="text-xs font-normal text-slate-400">ms</span>
          </div>
          <div className="text-[10px] text-emerald-500/80 mt-1">Sub-10ms target</div>
        </div>
      </div>

      {/* p95 Latency */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">p95 Latency</span>
          <Clock className="w-4 h-4 text-sky-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {(p95Ms ?? 0).toFixed(1)} <span className="text-xs font-normal text-slate-400">ms</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">95% faster than</div>
        </div>
      </div>

      {/* p99 Latency */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 mb-1">
          <span className="text-xs font-medium">p99 Latency</span>
          <ShieldAlert className="w-4 h-4 text-purple-400" />
        </div>
        <div>
          <div className="text-xl font-bold font-mono text-slate-100">
            {(p99Ms ?? 0).toFixed(1)} <span className="text-xs font-normal text-slate-400">ms</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-1">Tail latency bound</div>
        </div>
      </div>
    </div>
  );
};
