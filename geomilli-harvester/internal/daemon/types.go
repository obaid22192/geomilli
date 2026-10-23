package daemon

import "time"

type ActiveJobInfo struct {
	JobID          string   `json:"job_id"`
	Countries      []string `json:"countries"`
	CurrentCountry string   `json:"current_country"`
	Stage          string   `json:"stage"` // fetch | filter | transform | enrich | index | validate | swap | cleanup
	ProgressPct    int      `json:"progress_pct"`
	DocsProcessed  int      `json:"docs_processed"`
	StartedAt      string   `json:"started_at"`
}

type ResourcesInfo struct {
	HeapAllocMB        float64 `json:"heap_alloc_mb"`
	SysMemMB           float64 `json:"sys_mem_mb"`
	DiskStagingMB      float64 `json:"disk_staging_mb"`
	DiskCacheMB        float64 `json:"disk_cache_mb"`
	SQLiteCacheEntries int64   `json:"sqlite_cache_entries"`
}

type JobHistory struct {
	JobID         string   `json:"job_id"`
	Countries     []string `json:"countries"`
	Status        string   `json:"status"` // completed | failed
	StartedAt     string   `json:"started_at"`
	CompletedAt   string   `json:"completed_at"`
	DocsProcessed int      `json:"docs_processed"`
	Error         string   `json:"error,omitempty"`
}

type WorkerSlot struct {
	ID            int    `json:"id"`
	CountryCode   string `json:"country_code"`
	CountryName   string `json:"country_name"`
	Stage         string `json:"stage"` // fetch | filter | transform | enrich | index | validate | swap | cleanup
	ProgressPct   int    `json:"progress_pct"`
	DocsProcessed int    `json:"docs_processed"`
	StartedAt     string `json:"started_at"`
	Status        string `json:"status"` // idle | active
}

type QueuedTask struct {
	ID           string `json:"id"`
	CountryCode  string `json:"country_code"`
	CountryName  string `json:"country_name"`
	OnlineEnrich bool   `json:"online_enrich"`
	Mode         string `json:"mode"`
	EnqueuedAt   string `json:"enqueued_at"`
	Status       string `json:"status"` // queued | running | completed | failed
	Error        string `json:"error,omitempty"`
}

type QueueInfo struct {
	Concurrency   int               `json:"concurrency"`
	ActiveWorkers []WorkerSlot      `json:"active_workers"`
	QueuedTasks   []QueuedTask      `json:"queued_tasks"`
	CountryLocks  map[string]string `json:"country_locks"` // code -> "running" | "queued"
}

type StatusResponse struct {
	State     string         `json:"state"` // idle | running | completed | failed
	ActiveJob *ActiveJobInfo `json:"active_job"`
	Resources ResourcesInfo  `json:"resources"`
	History   []JobHistory   `json:"history"`
	Queue     *QueueInfo     `json:"queue,omitempty"`
}

type CountryInfo struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	Continent         string `json:"continent"`
	GeofabrikPath     string `json:"geofabrik_path"`
	IsIndexed         bool   `json:"is_indexed"`
	LastBuiltAt       string `json:"last_built_at,omitempty"`
	NextRunAt         string `json:"next_run_at,omitempty"`
	SyncIntervalHours int    `json:"sync_interval_hours"`
	ScheduleStatus    string `json:"schedule_status"` // "running" | "queued" | "scheduled" | "overdue" | "idle"
}

type CountriesResponse struct {
	Countries []CountryInfo `json:"countries"`
}

type CancelJobRequest struct {
	CountryCode string `json:"country_code,omitempty"`
	WorkerID    int    `json:"worker_id,omitempty"`
}

type CancelJobResponse struct {
	Message   string   `json:"message"`
	Cancelled []string `json:"cancelled"`
	Dequeued  []string `json:"dequeued"`
}

type DequeueResponse struct {
	Message     string `json:"message"`
	CountryCode string `json:"country_code,omitempty"`
	Remaining   int    `json:"remaining"`
}

type CreateJobRequest struct {
	Countries    []string `json:"countries"`
	OnlineEnrich bool     `json:"online_enrich"`
	Mode         string   `json:"mode"` // append | replace
}

type CreateJobResponse struct {
	JobID        string   `json:"job_id"`
	Message      string   `json:"message"`
	Status       string   `json:"status"`
	Enqueued     []string `json:"enqueued,omitempty"`
	Deduplicated []string `json:"deduplicated,omitempty"`
	QueueLength  int      `json:"queue_length,omitempty"`
}

type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type ProgressEvent struct {
	JobID          string `json:"job_id"`
	Stage          string `json:"stage"`
	CurrentCountry string `json:"current_country"`
	ProgressPct    int    `json:"progress_pct"`
	DocsProcessed  int    `json:"docs_processed"`
	Timestamp      string `json:"timestamp"`
}

func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
