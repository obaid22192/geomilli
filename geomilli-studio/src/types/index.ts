export interface Place {
  id: string;
  entity_id?: string;
  osm_type?: string;
  osm_id?: number;
  class: string;
  subclass?: string;
  name?: string;
  name_display?: string;
  names_by_language?: Record<string, string>;
  name_alt_active?: string[];
  name_alt_historical?: string[];
  country_code?: string;
  country?: string;
  country_name?: string;
  admin_hierarchy?: string[];
  parent_entity_id?: string;
  state?: string;
  city?: string;
  postcode?: string;
  street?: string;
  housenumber?: string;
  lat?: number;
  lng?: number;
  _geo?: { lat: number; lng: number };
  _geoDistance?: number;
  distance_meters?: number;
  wikidata?: string;
  wikidata_id?: string;
  wikipedia?: string;
  wikipedia_pageviews?: number;
  popularity_score?: number;
  iata?: string;
  iata_code?: string;
  icao?: string;
  icao_code?: string;
  cuisine?: string;
  star_rating?: number;
  accommodation_type?: string;
  admin_level?: number;
  population?: number;
  source?: string;
  updated_at?: string;
  tags?: Record<string, string>;
}

export interface SearchResponse {
  hits?: Place[];
  results?: Place[];
  query?: string;
  processingTimeMs?: number;
  took_ms?: number;
  limit?: number;
  offset?: number;
  estimatedTotalHits?: number;
  totalHits?: number;
  total_hits?: number;
  facetDistribution?: Record<string, Record<string, number>>;
  duration_ms?: number;
  closest_place?: Place;
  nearest_places?: Place[];
  address?: Record<string, any>;
  lat?: number;
  lng?: number;
  min_lat?: number;
  min_lng?: number;
  max_lat?: number;
  max_lng?: number;
}

export interface ImporterJob {
  job_id: string;
  countries: string[];
  current_country: string;
  stage: 'fetch' | 'filter' | 'transform' | 'enrich' | 'index' | 'validate' | 'swap' | 'cleanup' | string;
  progress_pct: number;
  docs_processed: number;
  started_at: string;
}

export interface ImporterResources {
  heap_alloc_mb: number;
  sys_mem_mb: number;
  disk_staging_mb: number;
  disk_cache_mb: number;
  sqlite_cache_entries: number;
}

export interface WorkerSlot {
  id: number;
  country_code: string;
  country_name: string;
  stage: string;
  progress_pct: number;
  docs_processed: number;
  started_at: string;
  status: 'idle' | 'active';
}

export interface QueuedTask {
  id: string;
  country_code: string;
  country_name: string;
  online_enrich: boolean;
  mode: string;
  enqueued_at: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  error?: string;
}

export interface QueueInfo {
  concurrency: number;
  active_workers: WorkerSlot[];
  queued_tasks: QueuedTask[];
  country_locks: Record<string, string>;
}

export interface ImporterStatus {
  state: 'idle' | 'running' | 'completed' | 'failed';
  active_job?: ImporterJob | null;
  resources: ImporterResources;
  history: any[];
  queue?: QueueInfo;
}

export interface CountryInfo {
  code: string;
  name: string;
  continent?: string;
  geofabrik_path: string;
  is_indexed: boolean;
  last_built_at?: string;
  next_run_at?: string;
  sync_interval_hours?: number;
  schedule_status?: 'running' | 'queued' | 'scheduled' | 'overdue' | 'not_scheduled';
  doc_count?: number;
  size_mb?: number;
}

export interface MetricsData {
  total_requests: number;
  qps: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  uptime_seconds?: number;
  meilisearch: {
    mode: 'local' | 'cloud';
    host: string;
    healthy: boolean;
    places_count: number;
    index_size_mb?: number;
  };
  endpoints: Record<string, number>;
  top_queries: { query: string; count: number }[];
  zero_hit_queries: { query: string; count: number }[];
  class_distribution: Record<string, number>;
  country_distribution: Record<string, number>;
}

export type ExplorerMode =
  | 'autocomplete'
  | 'search'
  | 'reverse'
  | 'nearby'
  | 'bbox'
  | 'airports';

export interface LanguageItem {
  code: string;
  name: string;
  enabled: boolean;
}

export interface LanguagesConfig {
  version: number;
  languages: LanguageItem[];
}
