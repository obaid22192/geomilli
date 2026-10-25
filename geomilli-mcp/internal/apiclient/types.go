package apiclient

// Place models a normalized destination or POI document from Geomilli.
type Place struct {
	ID                string            `json:"id"`
	EntityID          string            `json:"entity_id,omitempty"`
	Class             string            `json:"class"`
	Subclass          string            `json:"subclass,omitempty"`
	NameDisplay       string            `json:"name_display"`
	NamesByLanguage   map[string]string `json:"names_by_language,omitempty"`
	NameAltActive     []string          `json:"name_alt_active,omitempty"`
	NameAltHistorical []string          `json:"name_alt_historical,omitempty"`
	CountryCode       string            `json:"country_code,omitempty"`
	CountryName       string            `json:"country_name,omitempty"`
	AdminHierarchy    []string          `json:"admin_hierarchy,omitempty"`
	ParentEntityID    string            `json:"parent_entity_id,omitempty"`
	Geo               GeoPoint          `json:"_geo"`
	BBox              []float64         `json:"bbox,omitempty"`
	IATACode          string            `json:"iata_code,omitempty"`
	ICAOCode          string            `json:"icao_code,omitempty"`
	StarRating        int               `json:"star_rating,omitempty"`
	Cuisine           string            `json:"cuisine,omitempty"`
	AccommodationType string            `json:"accommodation_type,omitempty"`
	AdminLevel        int               `json:"admin_level,omitempty"`
	Population        int               `json:"population,omitempty"`
	WikidataID        string            `json:"wikidata_id,omitempty"`
	PopularityScore   float64           `json:"popularity_score"`
	Source            string            `json:"source,omitempty"`
	UpdatedAt         string            `json:"updated_at,omitempty"`
	DistanceMeters    *float64          `json:"distance_meters,omitempty"`
}

// GeoPoint represents latitude and longitude coordinates.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// AutocompleteParams specifies query parameters for /v1/autocomplete.
type AutocompleteParams struct {
	Query    string
	Lang     string
	Country  string
	Class    string
	Subclass string
	Lat      *float64
	Lng      *float64
	Limit    int
}

// AutocompleteResponse represents the API response for /v1/autocomplete.
type AutocompleteResponse struct {
	Query            string  `json:"query"`
	Lang             string  `json:"lang,omitempty"`
	ProcessingTimeMs int64   `json:"took_ms"`
	TotalHits        int64   `json:"total_hits"`
	Results          []Place `json:"results"`
}

// SearchParams specifies query parameters for /v1/search.
type SearchParams struct {
	Query  string
	Lang   string
	Filter string
	Sort   string
	Facets string
	Limit  int
	Offset int
}

// SearchResponse represents the response for /v1/search.
type SearchResponse struct {
	Query             string                    `json:"query"`
	ProcessingTimeMs  int64                     `json:"took_ms"`
	Limit             int                       `json:"limit"`
	Offset            int                       `json:"offset"`
	TotalHits         int64                     `json:"total_hits"`
	FacetDistribution map[string]map[string]int `json:"facets,omitempty"`
	Results           []Place                   `json:"results"`
}

// ReverseParams specifies query parameters for /v1/reverse.
type ReverseParams struct {
	Lat          float64
	Lng          float64
	Lang         string
	RadiusMeters *int
}

// ReverseAddress models formatted address components from reverse geocoding.
type ReverseAddress struct {
	Landmark     string `json:"landmark,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	Region       string `json:"region,omitempty"`
	Country      string `json:"country,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
}

// ReverseResponse represents the response for /v1/reverse.
type ReverseResponse struct {
	Lat            float64        `json:"lat"`
	Lng            float64        `json:"lng"`
	RadiusMeters   int            `json:"radius_meters,omitempty"`
	Address        ReverseAddress `json:"address"`
	AdminHierarchy []string       `json:"admin_hierarchy"`
	ClosestPlace   *Place         `json:"closest_place,omitempty"`
	NearestPlaces  []Place        `json:"nearest_places"`
}

// NearbyParams specifies query parameters for /v1/nearby.
type NearbyParams struct {
	Lat          float64
	Lng          float64
	RadiusMeters int
	Class        string
	Subclass     string
	Cuisine      string
	MinStars     int
	Lang         string
	Limit        int
	Offset       int
}

// NearbyResponse represents the response for /v1/nearby.
type NearbyResponse struct {
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	RadiusMeters     int     `json:"radius_meters"`
	ProcessingTimeMs int64   `json:"took_ms"`
	Limit            int     `json:"limit"`
	Offset           int     `json:"offset"`
	TotalHits        int64   `json:"total_hits"`
	Results          []Place `json:"results"`
}

// BBoxParams specifies query parameters for /v1/bbox.
type BBoxParams struct {
	MinLat float64
	MinLng float64
	MaxLat float64
	MaxLng float64
	Class  string
	Lang   string
	Limit  int
}

// BBoxResponse represents the response for /v1/bbox.
type BBoxResponse struct {
	MinLat           float64 `json:"min_lat"`
	MinLng           float64 `json:"min_lng"`
	MaxLat           float64 `json:"max_lat"`
	MaxLng           float64 `json:"max_lng"`
	ProcessingTimeMs int64   `json:"took_ms"`
	TotalHits        int64   `json:"total_hits"`
	Results          []Place `json:"results"`
}

// PlaceDetailsResponse models the detailed place response with optional nearby POIs.
type PlaceDetailsResponse struct {
	Place
	NearbyPlaces []Place `json:"nearby_places,omitempty"`
}

// SearchAirportsParams specifies query parameters for /v1/airports.
type SearchAirportsParams struct {
	Query   string
	Country string
	IATA    string
	ICAO    string
	Code    string
	Lat     *float64
	Lng     *float64
	Limit   int
	Lang    string
}

// AirportsResponse models the search response for /v1/airports.
type AirportsResponse struct {
	Query            string  `json:"query,omitempty"`
	ProcessingTimeMs int64   `json:"took_ms"`
	TotalHits        int64   `json:"total_hits"`
	Results          []Place `json:"results"`
}

// CountrySummary models an indexed country with its place count.
type CountrySummary struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	PlaceCount  int    `json:"place_count"`
}

// CountriesListResponse models /v1/countries response.
type CountriesListResponse struct {
	TotalCountries int              `json:"total_countries"`
	Countries      []CountrySummary `json:"countries"`
}

// RegionsListResponse models /v1/countries/{code}/regions response.
type RegionsListResponse struct {
	CountryCode string  `json:"country_code"`
	TotalHits   int64   `json:"total_hits"`
	Regions     []Place `json:"regions"`
}

// CitiesListResponse models /v1/countries/{code}/cities response.
type CitiesListResponse struct {
	CountryCode string  `json:"country_code"`
	Limit       int     `json:"limit"`
	Offset      int     `json:"offset"`
	TotalHits   int64   `json:"total_hits"`
	Cities      []Place `json:"cities"`
}

// AttributeDefinition defines a mapped attribute on a taxonomy class.
type AttributeDefinition struct {
	Tag       string `json:"tag"`
	Field     string `json:"field"`
	Transform string `json:"transform,omitempty"`
}

// MatchRule defines an OSM tag matching rule.
type MatchRule struct {
	Key           string   `json:"key"`
	Values        []string `json:"values"`
	RequireAnyTag []string `json:"require_any_tag,omitempty"`
	Geometry      []string `json:"geometry,omitempty"`
}

// ClassDefinition defines a single taxonomy class.
type ClassDefinition struct {
	Name       string                `json:"name"`
	BaseScore  float64               `json:"base_score"`
	Layer      string                `json:"layer"`
	Match      []MatchRule           `json:"match"`
	Attributes []AttributeDefinition `json:"attributes,omitempty"`
}

// TaxonomyConfig holds taxonomy classes and match definitions.
type TaxonomyConfig struct {
	Version int               `json:"version"`
	Classes []ClassDefinition `json:"classes"`
}

// LanguageItem models a supported language entry.
type LanguageItem struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// LanguagesConfig holds the languages whitelist.
type LanguagesConfig struct {
	Version   int            `json:"version"`
	Languages []LanguageItem `json:"languages"`
}

// HealthResponse models the /v1/health response.
type HealthResponse struct {
	Status      string         `json:"status"`
	Version     string         `json:"version"`
	Timestamp   string         `json:"timestamp"`
	Meilisearch map[string]any `json:"meilisearch"`
}

// HarvesterResources models memory and disk stats from Harvester daemon.
type HarvesterResources struct {
	HeapAllocMB        float64 `json:"heap_alloc_mb"`
	SysMemMB           float64 `json:"sys_mem_mb"`
	DiskStagingMB      float64 `json:"disk_staging_mb"`
	DiskCacheMB        float64 `json:"disk_cache_mb"`
	SQLiteCacheEntries int64   `json:"sqlite_cache_entries"`
}

// ActiveJob models Harvester active job.
type ActiveJob struct {
	JobID          string   `json:"job_id"`
	Countries      []string `json:"countries"`
	CurrentCountry string   `json:"current_country"`
	Stage          string   `json:"stage"`
	ProgressPct    int      `json:"progress_pct"`
	DocsProcessed  int64    `json:"docs_processed"`
	StartedAt      string   `json:"started_at"`
}

// WorkerSlot models Harvester worker slot.
type WorkerSlot struct {
	ID            int    `json:"id"`
	CountryCode   string `json:"country_code"`
	CountryName   string `json:"country_name"`
	Stage         string `json:"stage"`
	ProgressPct   int    `json:"progress_pct"`
	DocsProcessed int    `json:"docs_processed"`
	StartedAt     string `json:"started_at"`
	Status        string `json:"status"`
}

// QueuedTask models Harvester queued task.
type QueuedTask struct {
	ID           string `json:"id"`
	CountryCode  string `json:"country_code"`
	CountryName  string `json:"country_name"`
	OnlineEnrich bool   `json:"online_enrich"`
	Mode         string `json:"mode"`
	EnqueuedAt   string `json:"enqueued_at"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

// QueueInfo models queue information in importer status.
type QueueInfo struct {
	Concurrency   int               `json:"concurrency"`
	ActiveWorkers []WorkerSlot      `json:"active_workers"`
	QueuedTasks   []QueuedTask      `json:"queued_tasks"`
	CountryLocks  map[string]string `json:"country_locks"`
}

// ImporterStatusResponse models /v1/importer/status response.
type ImporterStatusResponse struct {
	State     string             `json:"state"`
	ActiveJob *ActiveJob         `json:"active_job,omitempty"`
	Resources HarvesterResources `json:"resources"`
	History   []any              `json:"history"`
	Queue     *QueueInfo         `json:"queue,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// CountryItem models a country record from the harvester importer.
type CountryItem struct {
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Continent         string  `json:"continent"`
	GeofabrikPath     string  `json:"geofabrik_path"`
	IsIndexed         bool    `json:"is_indexed"`
	LastBuiltAt       string  `json:"last_built_at,omitempty"`
	NextRunAt         string  `json:"next_run_at,omitempty"`
	SyncIntervalHours int     `json:"sync_interval_hours,omitempty"`
	ScheduleStatus    string  `json:"schedule_status,omitempty"`
	DocCount          int64   `json:"doc_count,omitempty"`
	SizeMB            float64 `json:"size_mb,omitempty"`
}

// ImporterCountriesResponse models /v1/importer/countries response.
type ImporterCountriesResponse struct {
	Countries []CountryItem `json:"countries"`
	Error     string        `json:"error,omitempty"`
}

// ImporterQueueResponse models /v1/importer/queue response.
type ImporterQueueResponse struct {
	Concurrency   int               `json:"concurrency"`
	ActiveWorkers []WorkerSlot      `json:"active_workers"`
	QueuedTasks   []QueuedTask      `json:"queued_tasks"`
	CountryLocks  map[string]string `json:"country_locks"`
	Error         string            `json:"error,omitempty"`
}

// LatencyPercentiles models latency distribution in milliseconds.
type LatencyPercentiles struct {
	P50 float64 `json:"p50_ms"`
	P95 float64 `json:"p95_ms"`
	P99 float64 `json:"p99_ms"`
}

// QueryCount models query frequency for top/zero-hit queries.
type QueryCount struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

// MeiliInfo models Meilisearch status in metrics.
type MeiliInfo struct {
	Status      string  `json:"status"`
	Mode        string  `json:"mode"`
	BaseURL     string  `json:"base_url"`
	LatencyMs   float64 `json:"latency_ms"`
	IndexDocs   int64   `json:"index_docs"`
	EstimatedMB float64 `json:"estimated_index_mb"`
}

// SystemMemory models memory and runtime statistics.
type SystemMemory struct {
	HeapAllocMB   float64 `json:"heap_alloc_mb"`
	SysMemMB      float64 `json:"sys_mem_mb"`
	NumGoroutines int     `json:"num_goroutines"`
}

// TelemetrySnapshot models the /v1/metrics response.
type TelemetrySnapshot struct {
	Timestamp         string             `json:"timestamp"`
	TotalRequests     int64              `json:"total_requests"`
	ActiveQPS         float64            `json:"active_qps"`
	Latency           LatencyPercentiles `json:"latency"`
	EndpointBreakdown map[string]int64   `json:"endpoint_breakdown"`
	StatusCodes       map[int]int64      `json:"status_codes"`
	TopQueries        []QueryCount       `json:"top_queries"`
	ZeroHitQueries    []QueryCount       `json:"zero_hit_queries"`
	Meilisearch       MeiliInfo          `json:"meilisearch"`
	System            SystemMemory       `json:"system"`
}
