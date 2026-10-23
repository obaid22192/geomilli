package importer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/geomilli/geomilli-api/internal/config"
)

// Client interacts with geomilli-harvester daemon.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient initializes a new harvester proxy client.
func NewClient(cfg *config.Config) *Client {
	return NewClientWithHTTPClient(cfg, &http.Client{Timeout: 30 * time.Second})
}

// NewClientWithHTTPClient creates a client with a custom http.Client.
func NewClientWithHTTPClient(cfg *config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL:    cfg.HarvesterURL,
		httpClient: httpClient,
	}
}

// ActiveJob models Harvester's active job.
type ActiveJob struct {
	JobID          string   `json:"job_id"`
	Countries      []string `json:"countries"`
	CurrentCountry string   `json:"current_country"`
	Stage          string   `json:"stage"`
	ProgressPct    int      `json:"progress_pct"`
	DocsProcessed  int64    `json:"docs_processed"`
	StartedAt      string   `json:"started_at"`
}

// HarvesterResources models memory and disk stats from Harvester daemon.
type HarvesterResources struct {
	HeapAllocMB        float64 `json:"heap_alloc_mb"`
	SysMemMB           float64 `json:"sys_mem_mb"`
	DiskStagingMB      float64 `json:"disk_staging_mb"`
	DiskCacheMB        float64 `json:"disk_cache_mb"`
	SQLiteCacheEntries int64   `json:"sqlite_cache_entries"`
}

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

type QueueInfo struct {
	Concurrency   int               `json:"concurrency"`
	ActiveWorkers []WorkerSlot      `json:"active_workers"`
	QueuedTasks   []QueuedTask      `json:"queued_tasks"`
	CountryLocks  map[string]string `json:"country_locks"`
}

// HarvesterStatus models GET /api/v1/status response.
type HarvesterStatus struct {
	State     string             `json:"state"` // "idle", "running", "completed", "failed"
	ActiveJob *ActiveJob         `json:"active_job,omitempty"`
	Resources HarvesterResources `json:"resources"`
	History   []any              `json:"history"`
	Queue     *QueueInfo         `json:"queue,omitempty"`
}

// CountryItem models a country record from Harvester.
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

// CancelJobRequest models POST /api/v1/jobs/cancel payload.
type CancelJobRequest struct {
	WorkerID    int    `json:"worker_id,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
}

// CancelJobResponse models POST /api/v1/jobs/cancel response.
type CancelJobResponse struct {
	Cancelled bool   `json:"cancelled"`
	Message   string `json:"message"`
}

// DequeueResponse models POST /api/v1/queue/dequeue response.
type DequeueResponse struct {
	Dequeued bool   `json:"dequeued"`
	Country  string `json:"country,omitempty"`
	Message  string `json:"message"`
}

// LanguageDef models a single language entry.
type LanguageDef struct {
	Code    string `json:"code" yaml:"code"`
	Name    string `json:"name" yaml:"name"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

// LanguagesConfig models GET /api/v1/languages response.
type LanguagesConfig struct {
	Version   int           `json:"version" yaml:"version"`
	Languages []LanguageDef `json:"languages" yaml:"languages"`
}

// CountriesResponse models GET /api/v1/countries response.
type CountriesResponse struct {
	Countries []CountryItem `json:"countries"`
}

// CreateJobRequest models POST /api/v1/jobs request payload.
type CreateJobRequest struct {
	Countries    []string `json:"countries"`
	OnlineEnrich bool     `json:"online_enrich"`
	Mode         string   `json:"mode,omitempty"` // "append" or "replace"
}

// CreateJobResponse models POST /api/v1/jobs response.
type CreateJobResponse struct {
	JobID        string   `json:"job_id"`
	Status       string   `json:"status"`
	Message      string   `json:"message,omitempty"`
	Enqueued     []string `json:"enqueued,omitempty"`
	Deduplicated []string `json:"deduplicated,omitempty"`
	QueueLength  int      `json:"queue_length,omitempty"`
}

// GetStatus queries Harvester daemon status.
func (c *Client) GetStatus(ctx context.Context) (*HarvesterStatus, error) {
	url := fmt.Sprintf("%s/api/v1/status", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("harvester returned %d: %s", resp.StatusCode, string(body))
	}

	var status HarvesterStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode harvester status: %w", err)
	}
	return &status, nil
}

// GetCountries queries available and indexed countries from Harvester.
func (c *Client) GetCountries(ctx context.Context) (*CountriesResponse, error) {
	url := fmt.Sprintf("%s/api/v1/countries", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("harvester returned %d: %s", resp.StatusCode, string(body))
	}

	var list CountriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode countries: %w", err)
	}
	return &list, nil
}

// GetQueue queries the harvester worker queue and active worker slots.
func (c *Client) GetQueue(ctx context.Context) (*QueueInfo, error) {
	url := fmt.Sprintf("%s/api/v1/queue", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("harvester returned %d: %s", resp.StatusCode, string(body))
	}

	var q QueueInfo
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("failed to decode harvester queue: %w", err)
	}
	return &q, nil
}

// CreateJob triggers a new import job on Harvester daemon.
func (c *Client) CreateJob(ctx context.Context, jobReq CreateJobRequest) (*CreateJobResponse, error) {
	url := fmt.Sprintf("%s/api/v1/jobs", c.baseURL)
	body, err := json.Marshal(jobReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create job failed (%d): %s", resp.StatusCode, string(body))
	}

	var res CreateJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode job response: %w", err)
	}
	return &res, nil
}

// Dequeue removes a specific country from the pending queue.
func (c *Client) Dequeue(ctx context.Context, countryCode string) (*DequeueResponse, error) {
	url := fmt.Sprintf("%s/api/v1/queue/dequeue?country=%s", c.baseURL, countryCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dequeue failed (%d): %s", resp.StatusCode, string(body))
	}

	var res DequeueResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode dequeue response: %w", err)
	}
	return &res, nil
}

// ClearQueue removes all pending countries from the harvester queue.
func (c *Client) ClearQueue(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/queue", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clear queue failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// Cancel cancels a running worker or task in the harvester daemon.
func (c *Client) Cancel(ctx context.Context, cancelReq CancelJobRequest) (*CancelJobResponse, error) {
	url := fmt.Sprintf("%s/api/v1/jobs/cancel", c.baseURL)
	body, err := json.Marshal(cancelReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cancel failed (%d): %s", resp.StatusCode, string(body))
	}

	var res CancelJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode cancel response: %w", err)
	}
	return &res, nil
}

// GetLanguages fetches the language configuration from Harvester.
func (c *Client) GetLanguages(ctx context.Context) (*LanguagesConfig, error) {
	url := fmt.Sprintf("%s/api/v1/languages", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get languages (%d): %s", resp.StatusCode, string(body))
	}

	var cfg LanguagesConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode languages: %w", err)
	}
	return &cfg, nil
}

// UpdateLanguages updates or adds languages in Harvester.
func (c *Client) UpdateLanguages(ctx context.Context, payload any) (*LanguagesConfig, error) {
	url := fmt.Sprintf("%s/api/v1/languages", c.baseURL)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("harvester unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update languages failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res struct {
		Updated   bool          `json:"updated"`
		Languages []LanguageDef `json:"languages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &LanguagesConfig{Version: 1, Languages: res.Languages}, nil
}

// RelaySSE proxies the Server-Sent Events stream from Harvester to the client response.
func (c *Client) RelaySSE(w http.ResponseWriter, r *http.Request) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return fmt.Errorf("streaming unsupported")
	}

	url := fmt.Sprintf("%s/api/v1/events", c.baseURL)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use an un-timed HTTP client for long-lived SSE streaming
	sseClient := &http.Client{Timeout: 0}
	resp, err := sseClient.Do(req)
	if err != nil {
		// If harvester is down, output a single SSE event indicating offline status
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", fmt.Sprintf(`{"error":"harvester daemon offline at %s"}`, c.baseURL))
		flusher.Flush()
		return nil
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, writeErr := w.Write(line); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}
