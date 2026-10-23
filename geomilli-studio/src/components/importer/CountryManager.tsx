import React, { useState, useMemo } from 'react';
import { CountryInfo } from '../../types';
import {
  Globe,
  Search,
  CheckSquare,
  Square,
  Play,
  ShieldCheck,
  Sparkles,
  ChevronDown,
  ChevronRight,
  Info,
  RotateCcw,
  Layers,
  ArrowRight,
  Loader2,
  Clock,
  Lock,
  RefreshCw,
  AlertTriangle,
  Languages,
} from 'lucide-react';

interface CountryManagerProps {
  countries: CountryInfo[];
  selectedCountries: string[];
  onToggleCountry: (code: string) => void;
  onSelectAll: () => void;
  onDeselectAll: () => void;
  onSelectGroup: (codes: string[]) => void;
  onDeselectGroup: (codes: string[]) => void;
  onStartImport: (onlineEnrich: boolean) => void;
  onSyncDeltas?: () => void;
  onOpenLanguages?: () => void;
  countryLocks?: Record<string, string>;
  isJobRunning: boolean;
  loading: boolean;
}

const formatRelativeSchedule = (isoString?: string): string => {
  if (!isoString) return '';
  const target = new Date(isoString).getTime();
  const diffMs = target - Date.now();
  if (diffMs <= 0) return 'due now';
  const hours = Math.round(diffMs / (1000 * 60 * 60));
  if (hours < 1) {
    const mins = Math.max(1, Math.round(diffMs / (1000 * 60)));
    return `in ${mins}m`;
  }
  if (hours < 24) {
    return `in ${hours}h`;
  }
  const days = Math.round(hours / 24);
  return `in ${days}d`;
};

export const CountryManager: React.FC<CountryManagerProps> = ({
  countries,
  selectedCountries,
  onToggleCountry,
  onSelectAll,
  onDeselectAll,
  onSelectGroup,
  onDeselectGroup,
  onStartImport,
  onSyncDeltas,
  onOpenLanguages,
  countryLocks = {},
  isJobRunning,
  loading,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [onlineEnrich, setOnlineEnrich] = useState(false);
  const [showEnrichExplainer, setShowEnrichExplainer] = useState(true);
  const [showResumptionExplainer, setShowResumptionExplainer] = useState(false);
  const [collapsedContinents, setCollapsedContinents] = useState<Record<string, boolean>>({});

  // Group countries by continent
  const continentGroups = useMemo(() => {
    const groups: Record<string, CountryInfo[]> = {};
    const lowerSearch = searchTerm.toLowerCase();

    for (const c of countries) {
      if (
        searchTerm &&
        !c.name.toLowerCase().includes(lowerSearch) &&
        !c.code.toLowerCase().includes(lowerSearch) &&
        !(c.continent || '').toLowerCase().includes(lowerSearch)
      ) {
        continue;
      }

      const continent = c.continent || 'Europe';
      if (!groups[continent]) {
        groups[continent] = [];
      }
      groups[continent].push(c);
    }

    return groups;
  }, [countries, searchTerm]);

  const continentNames = Object.keys(continentGroups);

  const toggleContinentCollapse = (continent: string) => {
    setCollapsedContinents((prev) => ({
      ...prev,
      [continent]: !prev[continent],
    }));
  };

  const handleSelectContinent = (continent: string) => {
    const codes = (continentGroups[continent] || []).map((c) => c.code);
    onSelectGroup(codes);
  };

  const handleDeselectContinent = (continent: string) => {
    const codes = (continentGroups[continent] || []).map((c) => c.code);
    onDeselectGroup(codes);
  };

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm flex flex-col gap-4">
      {/* Header & Main Ingestion Controls */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 border-b border-slate-800 pb-4">
        <div>
          <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <Globe className="w-4 h-4 text-brand-400" />
            <span>Global Geofabrik Catalog & Continent Selector</span>
            <span className="text-xs font-normal text-slate-400 bg-slate-800 px-2 py-0.5 rounded-full">
              {selectedCountries.length} of {countries.length} countries selected
            </span>
          </h3>
          <p className="text-xs text-slate-400 mt-1">
            Stream individual countries, entire continents, or the whole world. Default testbed is Belgium (BE).
          </p>
        </div>

        {/* Action Controls */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Online Enrich Toggle */}
          <label className="flex items-center gap-2 text-xs text-slate-200 cursor-pointer bg-slate-950 px-3 py-2 rounded-lg border border-slate-800 hover:border-slate-700 transition">
            <input
              type="checkbox"
              checked={onlineEnrich}
              onChange={(e) => setOnlineEnrich(e.target.checked)}
              className="rounded text-brand-500 focus:ring-0 bg-slate-900 border-slate-700 w-3.5 h-3.5"
            />
            <span className="flex items-center gap-1.5 font-medium">
              <Sparkles className="w-3.5 h-3.5 text-amber-400" />
              Online Wikidata Enrich
            </span>
          </label>

          {/* Languages Whitelist Button */}
          {onOpenLanguages && (
            <button
              onClick={onOpenLanguages}
              type="button"
              className="flex items-center gap-1.5 px-3 py-2 text-xs font-semibold rounded-lg bg-slate-950 hover:bg-slate-800 text-slate-200 border border-slate-700/80 hover:border-slate-600 transition shadow-sm"
              title="Configure language whitelist and translation extraction"
            >
              <Languages className="w-3.5 h-3.5 text-brand-400" />
              <span>Languages</span>
            </button>
          )}

          {/* Periodic Delta Sync Button */}
          {onSyncDeltas && (
            <button
              onClick={onSyncDeltas}
              disabled={loading}
              title="Checks remote ETags on Geofabrik and enqueues delta updates for modified countries"
              className="px-3 py-2 bg-slate-950 hover:bg-slate-800 disabled:opacity-50 text-slate-200 text-xs font-semibold rounded-lg transition border border-slate-700/80 shadow-sm flex items-center gap-1.5"
            >
              <RotateCcw className="w-3.5 h-3.5 text-cyan-400" />
              <span>Periodic Delta Sync</span>
            </button>
          )}

          {/* Trigger Import Button */}
          <button
            onClick={() => onStartImport(onlineEnrich)}
            disabled={selectedCountries.length === 0 || loading}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:bg-slate-800 disabled:text-slate-600 text-white text-xs font-semibold rounded-lg transition shadow-sm flex items-center gap-2"
          >
            <Play className="w-3.5 h-3.5 fill-white" />
            {loading ? 'Submitting...' : `Enqueue ${selectedCountries.length} Countries`}
          </button>
        </div>
      </div>

      {/* Explainer Banners */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {/* 1. Online Wikidata Enrichment Explanation Card */}
        <div className="bg-slate-950/80 border border-amber-500/20 rounded-xl p-3.5 text-xs text-slate-300">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1.5 font-semibold text-amber-400">
              <Sparkles className="w-4 h-4 text-amber-400" />
              <span>Online Wikidata & Pageviews Enrichment</span>
            </div>
            <button
              onClick={() => setShowEnrichExplainer(!showEnrichExplainer)}
              className="text-[11px] text-slate-400 hover:text-slate-200"
            >
              {showEnrichExplainer ? 'Hide Details' : 'Show Details'}
            </button>
          </div>

          <p className="text-[11px] text-slate-400 leading-relaxed mb-2">
            Without enrichment, places rely solely on static OSM tag heuristics (base score ~30-50). When enabled, Harvester queries the live Wikidata API & Wikimedia Pageviews to compute real-world global relevance.
          </p>

          {showEnrichExplainer && (
            <div className="bg-slate-900/90 border border-slate-800 rounded-lg p-2.5 space-y-2 text-[11px]">
              <div className="font-semibold text-slate-200 flex items-center gap-1">
                <span>Before vs. After Example:</span>
                <span className="font-mono text-brand-300">"Grand Place, Brussels"</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[10px]">
                <div className="bg-slate-950 p-2 rounded border border-rose-500/20">
                  <div className="text-rose-400 font-bold mb-0.5">Offline Heuristics Only:</div>
                  <div className="text-slate-400">Score: <strong className="text-slate-200">35</strong> (generic square)</div>
                  <div className="text-slate-500 mt-1">Ranks equally with a nondescript local parking lot or side alley.</div>
                </div>
                <div className="bg-slate-950 p-2 rounded border border-emerald-500/20">
                  <div className="text-emerald-400 font-bold mb-0.5">Enriched via Wikidata Q180901:</div>
                  <div className="text-slate-400">Score: <strong className="text-emerald-300">92.4</strong> (64 sitelinks, 250k views)</div>
                  <div className="text-slate-400 mt-1">Instantly ranks #1 for "grand pl" query across the viewport.</div>
                </div>
              </div>
              <div className="text-[10px] text-slate-500 italic">
                Cached in local SQLite WAL (`data/cache/enrichment.sqlite`) with 5-strike circuit breaker.
              </div>
            </div>
          )}
        </div>

        {/* 2. Checkpointing & Resumption Card */}
        <div className="bg-slate-950/80 border border-cyan-500/20 rounded-xl p-3.5 text-xs text-slate-300">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1.5 font-semibold text-cyan-400">
              <RotateCcw className="w-4 h-4 text-cyan-400" />
              <span>Idempotent Resumption & Checkpointing</span>
            </div>
            <button
              onClick={() => setShowResumptionExplainer(!showResumptionExplainer)}
              className="text-[11px] text-slate-400 hover:text-slate-200"
            >
              {showResumptionExplainer ? 'Hide Details' : 'Show Details'}
            </button>
          </div>

          <p className="text-[11px] text-slate-400 leading-relaxed mb-2">
            If an import is disrupted or cancelled, Harvester never starts from scratch. It intelligently resumes exactly where it left off.
          </p>

          {showResumptionExplainer ? (
            <div className="bg-slate-900/90 border border-slate-800 rounded-lg p-2.5 space-y-1.5 text-[11px]">
              <div className="flex items-start gap-1.5">
                <span className="text-cyan-400 font-mono text-[10px]">•</span>
                <span className="text-slate-300"><strong className="text-slate-100">Country Checkpoints:</strong> Completed country NDJSON files in `staging/` are detected and skipped.</span>
              </div>
              <div className="flex items-start gap-1.5">
                <span className="text-cyan-400 font-mono text-[10px]">•</span>
                <span className="text-slate-300"><strong className="text-slate-100">Persistent SQLite WAL:</strong> All Wikidata sitelinks and Wikimedia pageviews are cached locally.</span>
              </div>
              <div className="flex items-start gap-1.5">
                <span className="text-cyan-400 font-mono text-[10px]">•</span>
                <span className="text-slate-300"><strong className="text-slate-100">HTTP Resume:</strong> Raw PBF downloads resume via HTTP byte ranges and ETag validation.</span>
              </div>
            </div>
          ) : (
            <div className="text-[10px] text-slate-500 italic mt-3">
              Safely interrupt or restart containers at any time; completed stages and cache are preserved.
            </div>
          )}
        </div>
      </div>

      {/* Filter & Global Select Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-1">
        <div className="relative flex-1 max-w-sm">
          <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Filter by country, ISO-2, or continent..."
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-brand-500"
          />
        </div>

        {/* Global Select Buttons */}
        <div className="flex items-center gap-2 text-xs">
          <button
            onClick={onSelectAll}
            className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded border border-slate-700 transition font-medium flex items-center gap-1"
          >
            <CheckSquare className="w-3.5 h-3.5 text-brand-400" />
            Select Entire World ({countries.length})
          </button>
          <button
            onClick={onDeselectAll}
            className="px-2.5 py-1 bg-slate-950 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded border border-slate-800 transition"
          >
            Clear Selection
          </button>
        </div>
      </div>

      {/* Continent Groups & Country Cards */}
      <div className="space-y-4 max-h-[500px] overflow-y-auto pr-1">
        {continentNames.length === 0 ? (
          <div className="text-center py-8 text-xs text-slate-500">
            No countries found matching "{searchTerm}"
          </div>
        ) : (
          continentNames.map((continent) => {
            const groupCountries = continentGroups[continent] || [];
            const isCollapsed = collapsedContinents[continent] || false;
            const selectedInContinent = groupCountries.filter((c) =>
              selectedCountries.includes(c.code)
            );
            const allContinentSelected =
              groupCountries.length > 0 && selectedInContinent.length === groupCountries.length;

            return (
              <div
                key={continent}
                className="border border-slate-800/90 bg-slate-950/40 rounded-xl overflow-hidden"
              >
                {/* Continent Group Header */}
                <div className="bg-slate-850/80 px-4 py-2.5 flex items-center justify-between border-b border-slate-800">
                  <div
                    onClick={() => toggleContinentCollapse(continent)}
                    className="flex items-center gap-2 cursor-pointer select-none"
                  >
                    {isCollapsed ? (
                      <ChevronRight className="w-4 h-4 text-slate-400" />
                    ) : (
                      <ChevronDown className="w-4 h-4 text-slate-400" />
                    )}
                    <Layers className="w-4 h-4 text-brand-400" />
                    <span className="text-xs font-bold text-slate-200 uppercase tracking-wider">
                      {continent}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      ({selectedInContinent.length}/{groupCountries.length} selected)
                    </span>
                  </div>

                  <div className="flex items-center gap-2 text-xs">
                    <button
                      onClick={() =>
                        allContinentSelected
                          ? handleDeselectContinent(continent)
                          : handleSelectContinent(continent)
                      }
                      className="px-2 py-0.5 text-[11px] rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700/80 transition"
                    >
                      {allContinentSelected ? 'Deselect Continent' : `Select All in ${continent}`}
                    </button>
                  </div>
                </div>

                {/* Country Grid for Continent */}
                {!isCollapsed && (
                  <div className="p-3 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
                    {groupCountries.map((c) => {
                      const lockState = countryLocks[c.code];
                      const isLocked = Boolean(lockState);
                      const isSelected = selectedCountries.includes(c.code);

                      return (
                        <div
                          key={c.code}
                          onClick={() => {
                            if (!isLocked) onToggleCountry(c.code);
                          }}
                          className={`p-2.5 rounded-lg border transition flex items-center justify-between gap-2 select-none ${
                            isLocked
                              ? 'bg-slate-950/90 border-slate-700/60 opacity-80 cursor-not-allowed'
                              : isSelected
                              ? 'bg-slate-850 border-brand-500/70 text-slate-100 shadow-sm ring-1 ring-brand-500/20 cursor-pointer'
                              : 'bg-slate-950/80 border-slate-800/80 hover:border-slate-700 text-slate-400 cursor-pointer'
                          }`}
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            {isLocked ? (
                              <Lock className="w-4 h-4 text-amber-400 flex-shrink-0" />
                            ) : isSelected ? (
                              <CheckSquare className="w-4 h-4 text-brand-400 flex-shrink-0" />
                            ) : (
                              <Square className="w-4 h-4 text-slate-600 flex-shrink-0" />
                            )}
                            <div className="truncate">
                              <span className="text-xs font-medium text-slate-200 block truncate">
                                {c.name}
                              </span>
                              <div className="flex items-center gap-1.5 text-[10px] font-mono text-slate-500">
                                <span>{c.code}</span>
                                <span>·</span>
                                {c.schedule_status === 'overdue' ? (
                                  <span className="text-amber-400 font-medium">⚠️ Delta Due</span>
                                ) : c.schedule_status === 'scheduled' && c.next_run_at ? (
                                  <span className="text-sky-400" title={`Next automated delta sync: ${new Date(c.next_run_at).toLocaleString()}`}>
                                    Next: {formatRelativeSchedule(c.next_run_at)}
                                  </span>
                                ) : (
                                  <span className="truncate">{c.geofabrik_path}</span>
                                )}
                              </div>
                            </div>
                          </div>

                          <div className="flex items-center gap-1 flex-shrink-0">
                            {lockState === 'running' ? (
                              <span
                                title="Worker actively processing this country"
                                className="px-1.5 py-0.5 rounded text-[10px] bg-amber-500/15 text-amber-400 border border-amber-500/40 flex items-center gap-1 font-medium"
                              >
                                <Loader2 className="w-2.5 h-2.5 animate-spin" />
                                Running
                              </span>
                            ) : lockState === 'queued' ? (
                              <span
                                title="Waiting in async worker queue"
                                className="px-1.5 py-0.5 rounded text-[10px] bg-cyan-500/15 text-cyan-400 border border-cyan-500/40 flex items-center gap-1 font-medium"
                              >
                                <Clock className="w-2.5 h-2.5" />
                                Queued
                              </span>
                            ) : c.schedule_status === 'overdue' ? (
                              <span
                                title={`Delta update is overdue. Last built: ${c.last_built_at || 'never'}`}
                                className="px-1.5 py-0.5 rounded text-[10px] bg-rose-500/15 text-rose-300 border border-rose-500/40 flex items-center gap-1 font-medium"
                              >
                                <AlertTriangle className="w-2.5 h-2.5 text-rose-400" />
                                Due
                              </span>
                            ) : c.is_indexed ? (
                              <span
                                title={`Already indexed in Meilisearch${c.next_run_at ? ` · Next delta: ${new Date(c.next_run_at).toLocaleString()}` : ''}`}
                                className="px-1.5 py-0.5 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-0.5 font-medium"
                              >
                                <ShieldCheck className="w-3 h-3" />
                                Indexed
                              </span>
                            ) : null}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};
