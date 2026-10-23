import React, { useState, useEffect, useRef } from 'react';
import { ImporterStatus, CountryInfo } from '../../types';
import { API_BASE } from '../../utils/api';
import { PipelineFlow } from './PipelineFlow';
import { CountryManager } from './CountryManager';
import { ResourceGauges } from './ResourceGauges';
import { LiveLogs } from './LiveLogs';
import { LanguageManager } from './LanguageManager';

export const ImporterTab: React.FC = () => {
  const [status, setStatus] = useState<ImporterStatus>({
    state: 'idle',
    active_job: null,
    resources: {
      heap_alloc_mb: 0,
      sys_mem_mb: 0,
      disk_staging_mb: 0,
      disk_cache_mb: 0,
      sqlite_cache_entries: 0,
    },
    history: [],
  });

  const [countries, setCountries] = useState<CountryInfo[]>([]);
  const [selectedCountries, setSelectedCountries] = useState<string[]>(['BE']);
  const [logs, setLogs] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [isLangModalOpen, setIsLangModalOpen] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Poll status & countries from API
  const refreshStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/importer/status`);
      if (res.ok) {
        const data = await res.json();
        setStatus(data);
      }
    } catch {
      // Keep state if offline
    }
  };

  const refreshCountries = async () => {
    try {
      const res = await fetch(`${API_BASE}/importer/countries`);
      if (res.ok) {
        const data = await res.json();
        if (data.countries && Array.isArray(data.countries)) {
          setCountries(data.countries);
        }
      }
    } catch {
      // Keep default
    }
  };

  useEffect(() => {
    refreshStatus();
    refreshCountries();
    const interval = setInterval(refreshStatus, 3000);
    return () => clearInterval(interval);
  }, []);

  // SSE Stream for events
  useEffect(() => {
    const sseUrl = `${API_BASE}/importer/events`;
    let es: EventSource | null = null;
    try {
      es = new EventSource(sseUrl);
      es.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed.message) {
            setLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${parsed.message}`]);
          }
          if (parsed.status) {
            setStatus(parsed.status);
          }
        } catch {
          setLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${event.data}`]);
        }
      };

      es.addEventListener('log', (event: any) => {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed.message) {
            const time = parsed.timestamp
              ? new Date(parsed.timestamp).toLocaleTimeString()
              : new Date().toLocaleTimeString();
            setLogs((prev) => [
              ...prev,
              `[${time}] [${(parsed.level || 'info').toUpperCase()}] ${parsed.message}`,
            ]);
            return;
          }
        } catch {
          // fallback
        }
        setLogs((prev) => [...prev, `[${new Date().toLocaleTimeString()}] ${event.data}`]);
      });

      es.addEventListener('progress', (event: any) => {
        try {
          const progressData = JSON.parse(event.data);
          setStatus((prev) => ({
            ...prev,
            active_job: {
              ...(prev.active_job || {
                job_id: 'live_stream',
                countries: ['BE'],
                current_country: 'BE',
                started_at: new Date().toISOString(),
              }),
              ...progressData,
            },
            state: 'running',
          }));
        } catch {
          // ignore
        }
      });

      es.addEventListener('queue', (event: any) => {
        try {
          const qData = JSON.parse(event.data);
          setStatus((prev) => ({
            ...prev,
            queue: qData,
          }));
        } catch {
          // ignore
        }
      });

      eventSourceRef.current = es;
    } catch (err) {
      console.warn('SSE connection failed:', err);
    }

    return () => {
      if (es) es.close();
    };
  }, []);

  // Trigger import
  const handleStartImport = async (onlineEnrich: boolean) => {
    setLoading(true);
    const logLine = `[trigger] Submitting import job for: ${selectedCountries.join(', ')} (online_enrich=${onlineEnrich})`;
    setLogs((prev) => [...prev, logLine]);

    try {
      const res = await fetch(`${API_BASE}/importer/jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          countries: selectedCountries,
          online_enrich: onlineEnrich,
        }),
      });

      if (!res.ok) {
        throw new Error(await res.text());
      }

      const jobData = await res.json();
      const enqueuedCount = jobData.enqueued ? jobData.enqueued.length : selectedCountries.length;
      const dedupCount = jobData.deduplicated ? jobData.deduplicated.length : 0;

      setLogs((prev) => [
        ...prev,
        `[trigger] Job accepted: ${enqueuedCount} countries enqueued into worker pool. ${
          dedupCount > 0 ? `(${dedupCount} already locked/in-progress)` : ''
        }`,
      ]);

      refreshStatus();
    } catch (err: any) {
      setLogs((prev) => [...prev, `[error] Failed to start import: ${err.message}`]);
    } finally {
      setLoading(false);
    }
  };

  const handleSyncDeltas = async () => {
    setLoading(true);
    setLogs((prev) => [
      ...prev,
      `[periodic-sync] Triggering delta ETag check across configured countries...`,
    ]);
    try {
      const res = await fetch(`${API_BASE}/importer/jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          countries: ['ALL'],
          online_enrich: false,
          mode: 'append',
        }),
      });

      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setLogs((prev) => [
        ...prev,
        `[periodic-sync] Delta scan enqueued ${data.enqueued?.length || 0} countries for update check. (${
          data.deduplicated?.length || 0
        } already running/locked).`,
      ]);
      refreshStatus();
    } catch (err: any) {
      setLogs((prev) => [...prev, `[error] Periodic delta sync failed: ${err.message}`]);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleCountry = (code: string) => {
    setSelectedCountries((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]
    );
  };

  const handleSelectGroup = (codes: string[]) => {
    setSelectedCountries((prev) => Array.from(new Set([...prev, ...codes])));
  };

  const handleDeselectGroup = (codes: string[]) => {
    const codeSet = new Set(codes);
    setSelectedCountries((prev) => prev.filter((c) => !codeSet.has(c)));
  };

  const handleSelectAll = () => {
    setSelectedCountries(countries.map((c) => c.code));
  };

  const handleDeselectAll = () => {
    setSelectedCountries([]);
  };

  const handleDequeueCountry = async (countryCode: string) => {
    try {
      const res = await fetch(`${API_BASE}/importer/queue/dequeue?country=${countryCode}`, {
        method: 'POST',
      });
      if (!res.ok) throw new Error(await res.text());
      setLogs((prev) => [
        ...prev,
        `[queue] Dequeued country ${countryCode} from pending queue.`,
      ]);
      refreshStatus();
    } catch (err: any) {
      setLogs((prev) => [...prev, `[error] Failed to dequeue ${countryCode}: ${err.message}`]);
    }
  };

  const handleClearQueue = async () => {
    try {
      const res = await fetch(`${API_BASE}/importer/queue`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error(await res.text());
      setLogs((prev) => [...prev, `[queue] Cleared all pending tasks from worker queue.`]);
      refreshStatus();
    } catch (err: any) {
      setLogs((prev) => [...prev, `[error] Failed to clear queue: ${err.message}`]);
    }
  };

  const handleCancelWorker = async (workerId: number, countryCode?: string) => {
    try {
      const res = await fetch(`${API_BASE}/importer/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ worker_id: workerId, country_code: countryCode }),
      });
      if (!res.ok) throw new Error(await res.text());
      setLogs((prev) => [
        ...prev,
        `[queue] Abort requested for Worker #${workerId}${countryCode ? ` (${countryCode})` : ''}.`,
      ]);
      refreshStatus();
    } catch (err: any) {
      setLogs((prev) => [...prev, `[error] Failed to cancel worker #${workerId}: ${err.message}`]);
    }
  };

  const isJobRunning = status.state === 'running';
  const currentStage = status.active_job?.stage || 'idle';
  const progressPct = status.active_job?.progress_pct || 0;
  const currentCountry = status.active_job?.current_country;
  const docsProcessed = status.active_job?.docs_processed || 0;
  const totalCountries = status.active_job?.countries?.length || 1;

  return (
    <div className="flex flex-col gap-4">
      {/* 1. Resource Gauges */}
      <ResourceGauges resources={status.resources} />

      {/* 2. Visual Pipeline Flow */}
      <PipelineFlow
        currentStage={currentStage}
        isJobRunning={isJobRunning}
        progressPct={progressPct}
        currentCountry={currentCountry}
        docsProcessed={docsProcessed}
        totalCountries={totalCountries}
        jobState={status.state}
        lastError={status.history?.[0]?.error}
        queue={status.queue}
        onDequeueCountry={handleDequeueCountry}
        onClearQueue={handleClearQueue}
        onCancelWorker={handleCancelWorker}
      />

      {/* 3. Country Catalog & Trigger */}
      <CountryManager
        countries={countries}
        selectedCountries={selectedCountries}
        onToggleCountry={handleToggleCountry}
        onSelectAll={handleSelectAll}
        onDeselectAll={handleDeselectAll}
        onSelectGroup={handleSelectGroup}
        onDeselectGroup={handleDeselectGroup}
        onStartImport={handleStartImport}
        onSyncDeltas={handleSyncDeltas}
        onOpenLanguages={() => setIsLangModalOpen(true)}
        countryLocks={status.queue?.country_locks}
        isJobRunning={isJobRunning}
        loading={loading}
      />

      {/* 4. Real-time Terminal Log Viewer */}
      <LiveLogs logs={logs} onClear={() => setLogs([])} />

      {/* 5. Language Configuration Modal */}
      <LanguageManager
        isOpen={isLangModalOpen}
        onClose={() => setIsLangModalOpen(false)}
        indexedCountryCodes={countries.filter((c) => c.is_indexed).map((c) => c.code)}
        onTriggerReindex={(codes) => {
          setSelectedCountries(codes);
          handleStartImport(false);
        }}
      />
    </div>
  );
};
