import React, { useState, useEffect } from 'react';
import { MetricsData } from '../../types';
import { API_BASE } from '../../utils/api';
import { ConnectionBadge } from './ConnectionBadge';
import { TelemetryCards } from './TelemetryCards';
import { Charts } from './Charts';

const INITIAL_METRICS: MetricsData = {
  total_requests: 0,
  qps: 0,
  p50_ms: 0,
  p95_ms: 0,
  p99_ms: 0,
  uptime_seconds: 0,
  meilisearch: {
    mode: 'local',
    host: '',
    healthy: false,
    places_count: 0,
    index_size_mb: 0,
  },
  endpoints: {},
  top_queries: [],
  zero_hit_queries: [],
  class_distribution: {},
  country_distribution: {},
};

export const MetricsTab: React.FC = () => {
  const [metrics, setMetrics] = useState<MetricsData>(INITIAL_METRICS);

  const fetchMetrics = async () => {
    try {
      const res = await fetch(`${API_BASE}/metrics`);
      if (res.ok) {
        const raw = await res.json();
        const mapped: MetricsData = {
          total_requests: raw.total_requests || 0,
          qps: raw.qps ?? raw.active_qps ?? 0,
          p50_ms: raw.p50_ms ?? raw.latency?.p50_ms ?? 0,
          p95_ms: raw.p95_ms ?? raw.latency?.p95_ms ?? 0,
          p99_ms: raw.p99_ms ?? raw.latency?.p99_ms ?? 0,
          uptime_seconds: raw.uptime_seconds || 0,
          meilisearch: {
            mode: raw.meilisearch?.mode || 'local',
            host: raw.meilisearch?.base_url || raw.meilisearch?.host || 'http://meilisearch:7700',
            healthy: raw.meilisearch?.status === 'available' || raw.meilisearch?.healthy === true,
            places_count: raw.meilisearch?.index_docs ?? raw.meilisearch?.places_count ?? 0,
            index_size_mb: raw.meilisearch?.estimated_index_mb ?? raw.meilisearch?.index_size_mb ?? 0,
          },
          endpoints: raw.endpoints || raw.endpoint_breakdown || {},
          top_queries: raw.top_queries || [],
          zero_hit_queries: raw.zero_hit_queries || [],
          class_distribution: raw.class_distribution || {},
          country_distribution: raw.country_distribution || {},
        };
        setMetrics(mapped);
      }
    } catch {
      // Keep state
    }
  };

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 3500);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex flex-col gap-4">
      {/* 1. Meilisearch Connection & Health Badge */}
      <ConnectionBadge meili={metrics.meilisearch} />

      {/* 2. Top Telemetry Cards */}
      <TelemetryCards
        totalRequests={metrics.total_requests}
        qps={metrics.qps}
        p50Ms={metrics.p50_ms}
        p95Ms={metrics.p95_ms}
        p99Ms={metrics.p99_ms}
      />

      {/* 3. Charts Grid */}
      <Charts
        endpoints={metrics.endpoints || {}}
        topQueries={metrics.top_queries || []}
        zeroHitQueries={metrics.zero_hit_queries || []}
        classDistribution={metrics.class_distribution || {}}
        countryDistribution={metrics.country_distribution || {}}
      />
    </div>
  );
};
