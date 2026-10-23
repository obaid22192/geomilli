import React from 'react';
import {
  Download,
  Filter,
  Cpu,
  Database,
  RefreshCw,
  CheckCircle2,
  Loader2,
  AlertTriangle,
  Check,
  Layers,
  Clock,
  Lock,
  X,
  Trash2,
  Square,
} from 'lucide-react';
import { QueueInfo } from '../../types';

interface PipelineFlowProps {
  currentStage: string;
  isJobRunning: boolean;
  progressPct: number;
  currentCountry?: string;
  docsProcessed?: number;
  totalCountries?: number;
  jobState?: 'idle' | 'running' | 'completed' | 'failed';
  lastError?: string;
  lastCompletedJob?: any;
  queue?: QueueInfo;
  onDequeueCountry?: (countryCode: string) => void;
  onClearQueue?: () => void;
  onCancelWorker?: (workerId: number, countryCode?: string) => void;
}

const STAGES = [
  {
    id: 'fetch',
    label: '1. Fetch OSM PBF',
    shortDesc: 'Geofabrik stream',
    fullDesc: 'Downloading compressed .osm.pbf dump from Geofabrik mirrors with HTTP byte-range resume and ETag verification.',
    icon: Download,
  },
  {
    id: 'filter',
    label: '2. Tag Filtering',
    shortDesc: 'Osmium C++ binary',
    fullDesc: 'Streaming Osmium tags-filter to isolate administrative hierarchies (admin_level 2-8) and whitelisted POI classes.',
    icon: Filter,
  },
  {
    id: 'transform',
    label: '3. Stream Transform',
    shortDesc: 'O(1) RAM to NDJSON',
    fullDesc: 'Normalizing multi-language names and calculating centroids into disk-backed newline-delimited JSON with bounded memory.',
    icon: Cpu,
  },
  {
    id: 'enrich',
    label: '4. Wikidata Enrich',
    shortDesc: 'SQLite WAL Cache',
    fullDesc: 'Correlating Wikidata sitelinks and Wikimedia 30-day pageviews into popularity scores with local SQLite WAL caching.',
    icon: Database,
  },
  {
    id: 'swap',
    label: '5. Blue/Green Swap',
    shortDesc: 'Zero-Downtime Meili',
    fullDesc: 'Indexing into isolated build index, running golden validation queries, and atomically swapping aliases in Meilisearch.',
    icon: RefreshCw,
  },
];

const STAGE_EXPLANATIONS: Record<string, string> = {
  fetch: 'Stage 1/5: Fetching OSM PBF extracts from Geofabrik mirrors. ETag verification prevents duplicate downloads.',
  filter: 'Stage 2/5: Filtering raw OSM data with osmium-tool into admin polygons and POI features.',
  transform: 'Stage 3/5: Normalizing place schemas and whitelist languages into disk-buffered NDJSON format.',
  enrich: 'Stage 4/5: Enriching POIs with Wikipedia sitelinks and pageview metrics via SQLite WAL cache.',
  index: 'Stage 5/5: Ingesting documents into isolated Meilisearch staging index in bounded batches.',
  validate: 'Stage 5/5: Running golden test queries to verify search precision before swapping.',
  swap: 'Stage 5/5: Atomic index alias swap completed with zero downtime.',
  cleanup: 'Post-process: Cleaning temporary raw PBF and GeoJSON files to conserve disk space.',
};

export const PipelineFlow: React.FC<PipelineFlowProps> = ({
  currentStage,
  isJobRunning,
  progressPct,
  currentCountry,
  docsProcessed = 0,
  totalCountries = 1,
  jobState = 'idle',
  lastError,
  queue,
  onDequeueCountry,
  onClearQueue,
  onCancelWorker,
}) => {
  const getStageIndex = (stageId: string) => {
    if (stageId === 'index' || stageId === 'validate' || stageId === 'swap') return 4;
    const idx = STAGES.findIndex((s) => s.id === stageId);
    return idx >= 0 ? idx : -1;
  };

  const activeIndex = getStageIndex(currentStage);
  const activeExplanation =
    STAGE_EXPLANATIONS[currentStage] ||
    (isJobRunning
      ? `Processing stage: ${currentStage}`
      : jobState === 'failed'
      ? 'Pipeline stopped due to an error. Check details below.'
      : 'Importer is idle. Ready to stream chosen countries into Meilisearch.');

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm flex flex-col gap-4">
      {/* Header & Active State */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <span>Streaming Ingestion Pipeline</span>
            {isJobRunning ? (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1.5 animate-pulse">
                <Loader2 className="w-3 h-3 animate-spin" />
                Active: {currentStage.toUpperCase()}
                {currentCountry && ` (${currentCountry})`}
              </span>
            ) : jobState === 'failed' ? (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30 flex items-center gap-1.5">
                <AlertTriangle className="w-3 h-3 text-rose-400" />
                Execution Failed
              </span>
            ) : jobState === 'completed' ? (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1.5">
                <Check className="w-3 h-3 text-emerald-400" />
                Completed
              </span>
            ) : (
              <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-400 border border-slate-700">
                Idle / Ready
              </span>
            )}
          </h3>
          <p className="text-xs text-slate-400 mt-1">
            {activeExplanation}
          </p>
        </div>

        {isJobRunning && (
          <div className="flex items-center gap-4 bg-slate-950 px-3.5 py-2 rounded-lg border border-slate-800 self-start sm:self-auto">
            <div>
              <div className="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Docs Streamed</div>
              <div className="text-sm font-bold font-mono text-emerald-400">
                {docsProcessed.toLocaleString()}
              </div>
            </div>
            <div className="h-7 w-[1px] bg-slate-800" />
            <div className="text-right">
              <div className="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Total Progress</div>
              <div className="text-base font-bold font-mono text-brand-400">{progressPct}%</div>
            </div>
          </div>
        )}
      </div>

      {/* Failure Banner */}
      {jobState === 'failed' && (
        <div className="bg-rose-950/40 border border-rose-500/40 rounded-xl p-3.5 text-xs text-rose-300 flex items-start gap-3">
          <AlertTriangle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-1">
            <div className="font-semibold text-rose-200">Ingestion Stage Failed:</div>
            <div className="text-[11px] text-rose-300 font-mono bg-rose-950/80 px-2 py-1 rounded border border-rose-500/30">
              {lastError || 'Process terminated with non-zero exit status'}
            </div>
            <p className="text-[10px] text-slate-400 pt-0.5">
              Review live terminal logs below. You can safely restart ingestion by clicking "Import Selected Countries" below (Harvester will resume from cache).
            </p>
          </div>
        </div>
      )}

      {/* Pipeline Stages Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-5 gap-3">
        {STAGES.map((stage, idx) => {
          const Icon = stage.icon;
          const isCurrent =
            isJobRunning &&
            (stage.id === currentStage ||
              (stage.id === 'swap' &&
                (currentStage === 'index' || currentStage === 'validate' || currentStage === 'swap')));
          const isDone = isJobRunning ? idx < activeIndex : jobState === 'completed' || activeIndex >= 0;

          return (
            <div
              key={stage.id}
              className={`relative flex flex-col p-3 rounded-xl border transition-all ${
                isCurrent
                  ? 'bg-brand-950/40 border-brand-500 shadow-md ring-1 ring-brand-500/50'
                  : isDone
                  ? 'bg-slate-900 border-slate-700/80 text-slate-300'
                  : 'bg-slate-950/60 border-slate-800 text-slate-500'
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <div
                  className={`w-8 h-8 rounded-lg flex items-center justify-center transition flex-shrink-0 ${
                    isCurrent
                      ? 'bg-brand-600 text-white shadow-lg shadow-brand-500/30 animate-pulse'
                      : isDone
                      ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                      : 'bg-slate-800 text-slate-400'
                  }`}
                >
                  {isDone && !isCurrent ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  ) : isCurrent ? (
                    <Icon className="w-4 h-4 animate-spin-slow" />
                  ) : (
                    <Icon className="w-4 h-4" />
                  )}
                </div>
                <div className="text-xs font-semibold text-slate-200 leading-tight">
                  {stage.label}
                </div>
              </div>

              <div className="text-[11px] text-slate-400 leading-relaxed">
                {stage.fullDesc}
              </div>
            </div>
          );
        })}
      </div>

      {/* Progress Bar with Indicator */}
      {isJobRunning && (
        <div className="space-y-1">
          <div className="flex justify-between text-[11px] text-slate-400 font-mono">
            <span>
              Processing: <strong className="text-slate-200">{currentCountry || 'BE'}</strong> ({docsProcessed.toLocaleString()} docs written)
            </span>
            <span>{progressPct}% overall</span>
          </div>
          <div className="w-full bg-slate-950 rounded-full h-2.5 overflow-hidden border border-slate-800">
            <div
              className="bg-gradient-to-r from-brand-600 to-emerald-500 h-2.5 rounded-full transition-all duration-300 shadow-sm"
              style={{ width: `${Math.max(4, progressPct)}%` }}
            />
          </div>
        </div>
      )}

      {/* Concurrent Worker Pool & Async Queue */}
      {queue && (
        <div className="mt-2 pt-3 border-t border-slate-800/80 space-y-3">
          <div className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <Layers className="w-4 h-4 text-brand-400" />
              <span className="font-semibold text-slate-200">Async Worker Pool</span>
              <span className="text-[11px] text-slate-500 font-mono">
                (Concurrency: {queue.concurrency || 2} · Deduplication Locked)
              </span>
            </div>
            <div className="flex items-center gap-2 text-[11px]">
              <span className="text-slate-400">
                Queued:{' '}
                <strong className="text-brand-300 font-mono">
                  {queue.queued_tasks ? queue.queued_tasks.length : 0}
                </strong>
              </span>
              <span className="text-slate-600">·</span>
              <span className="text-slate-400">
                Active Locks:{' '}
                <strong className="text-emerald-400 font-mono">
                  {queue.country_locks ? Object.keys(queue.country_locks).length : 0}
                </strong>
              </span>
            </div>
          </div>

          {/* Worker Slots Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2.5">
            {queue.active_workers &&
              queue.active_workers.map((worker) => {
                const isActive = worker.status === 'active';
                return (
                  <div
                    key={worker.id}
                    className={`p-2.5 rounded-lg border text-xs transition ${
                      isActive
                        ? 'bg-slate-950 border-emerald-500/40 shadow-sm ring-1 ring-emerald-500/20'
                        : 'bg-slate-950/60 border-slate-800 text-slate-500'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="font-semibold text-slate-300 text-[11px]">
                        Worker #{worker.id}
                      </span>
                      {isActive ? (
                        <div className="flex items-center gap-1.5">
                          <span className="px-1.5 py-0.5 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1 font-medium animate-pulse">
                            <Loader2 className="w-2.5 h-2.5 animate-spin" />
                            Active
                          </span>
                          {onCancelWorker && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                onCancelWorker(worker.id, worker.country_code);
                              }}
                              title={`Abort / Cancel Worker #${worker.id} (${worker.country_code})`}
                              className="px-1.5 py-0.5 rounded bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30 transition flex items-center gap-0.5 text-[10px] font-medium"
                            >
                              <Square className="w-2.5 h-2.5 fill-rose-500/40 text-rose-400" />
                              Stop
                            </button>
                          )}
                        </div>
                      ) : (
                        <span className="text-[10px] text-slate-600 font-mono">Idle</span>
                      )}
                    </div>

                    {isActive ? (
                      <div className="space-y-1 text-[11px]">
                        <div className="flex items-center justify-between font-mono">
                          <span className="text-slate-200 font-medium">
                            {worker.country_name || worker.country_code} ({worker.country_code})
                          </span>
                          <span className="text-brand-400">{worker.progress_pct}%</span>
                        </div>
                        <div className="flex items-center justify-between text-[10px] text-slate-400">
                          <span className="capitalize">{worker.stage}</span>
                          <span>{worker.docs_processed.toLocaleString()} docs</span>
                        </div>
                        <div className="w-full bg-slate-900 rounded-full h-1 overflow-hidden">
                          <div
                            className="bg-emerald-500 h-1 rounded-full transition-all duration-300"
                            style={{ width: `${Math.max(5, worker.progress_pct)}%` }}
                          />
                        </div>
                      </div>
                    ) : (
                      <div className="text-[11px] text-slate-500 italic py-1">
                        Waiting for next country in queue...
                      </div>
                    )}
                  </div>
                );
              })}
          </div>

          {/* Queued Tasks Bar */}
          {queue.queued_tasks && queue.queued_tasks.length > 0 && (
            <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800/90 text-xs flex flex-wrap items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-[11px] text-slate-400 flex items-center gap-1 font-medium">
                  <Clock className="w-3.5 h-3.5 text-brand-400" />
                  Pending Queue ({queue.queued_tasks.length}):
                </span>
                {queue.queued_tasks.map((task) => (
                  <span
                    key={task.id}
                    className="px-2 py-0.5 rounded text-[11px] bg-slate-900 border border-slate-700 text-slate-300 font-mono flex items-center gap-1.5"
                  >
                    <Lock className="w-3 h-3 text-amber-400" />
                    <span>{task.country_name || task.country_code} ({task.country_code})</span>
                    {onDequeueCountry && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onDequeueCountry(task.country_code);
                        }}
                        title={`Dequeue ${task.country_code}`}
                        className="text-slate-500 hover:text-rose-400 ml-0.5 p-0.5 hover:bg-slate-800 rounded transition"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    )}
                  </span>
                ))}
              </div>

              {onClearQueue && (
                <button
                  onClick={onClearQueue}
                  title="Clear all pending tasks from the worker queue"
                  className="px-2 py-0.5 text-[10px] rounded bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30 transition flex items-center gap-1 font-medium"
                >
                  <Trash2 className="w-3 h-3" />
                  Clear Queue
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
