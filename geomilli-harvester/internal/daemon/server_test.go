package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDaemonStatus(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if status.State != "idle" {
		t.Errorf("expected state idle, got %s", status.State)
	}
	if status.Resources.HeapAllocMB < 0 {
		t.Errorf("expected positive heap alloc, got %f", status.Resources.HeapAllocMB)
	}
}

func TestDaemonJobsConflict(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	// Manually set runner to running
	srv.runner.state = "running"
	srv.runner.countryLocks["BE"] = "running"
	srv.runner.activeJob = &ActiveJobInfo{
		JobID:          "test_job",
		Countries:      []string{"BE"},
		CurrentCountry: "BE",
		Stage:          "fetch",
		StartedAt:      NowISO(),
	}

	body := []byte(`{"countries": ["BE"], "online_enrich": false, "mode": "replace"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleJobs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", resp.StatusCode)
	}
}

func TestDaemonEventsSSE(t *testing.T) {
	b := NewBroadcaster()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		b.ServeHTTP(w, req)
		close(done)
	}()

	// Wait for context to cancel
	<-done

	out := w.Body.String()
	if !strings.Contains(out, ": connected") {
		t.Errorf("expected initial SSE comment ': connected', got %s", out)
	}
}

func TestDaemonHealth(t *testing.T) {
	srv := NewServer(ServerConfig{Port: ":0"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status ok, got %v", res["status"])
	}
}

func TestDaemonCountries(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    "../..",
		AutoImport: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/countries", nil)
	w := httptest.NewRecorder()
	srv.handleCountries(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var res CountriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode countries response: %v", err)
	}
	if len(res.Countries) == 0 {
		t.Errorf("expected non-empty countries list")
	}
}

func TestDaemonJobValidation(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	// Invalid JSON
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader([]byte("{invalid-json")))
	wBadJSON := httptest.NewRecorder()
	srv.handleJobs(wBadJSON, reqBadJSON)
	if wBadJSON.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", wBadJSON.Result().StatusCode)
	}

	// Empty countries
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader([]byte(`{"countries":[]}`)))
	wEmpty := httptest.NewRecorder()
	srv.handleJobs(wEmpty, reqEmpty)
	if wEmpty.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty countries, got %d", wEmpty.Result().StatusCode)
	}
}

func TestDaemonQueueDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	// Lock BE as running
	srv.runner.countryLocks["BE"] = "running"

	// Request both BE and NL
	body := []byte(`{"countries": ["BE", "NL"], "online_enrich": false, "mode": "append"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleJobs(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}

	var res CreateJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Enqueued) != 1 || res.Enqueued[0] != "NL" {
		t.Errorf("expected NL to be enqueued, got %v", res.Enqueued)
	}
	if len(res.Deduplicated) != 1 || res.Deduplicated[0] != "BE" {
		t.Errorf("expected BE to be deduplicated, got %v", res.Deduplicated)
	}

	// Verify GET /api/v1/queue
	reqQ := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	wQ := httptest.NewRecorder()
	srv.handleQueue(wQ, reqQ)
	if wQ.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for queue, got %d", wQ.Result().StatusCode)
	}

	var q QueueInfo
	if err := json.NewDecoder(wQ.Body).Decode(&q); err != nil {
		t.Fatalf("failed to decode queue info: %v", err)
	}
	if q.CountryLocks["BE"] != "running" {
		t.Errorf("expected BE to be locked as running, got %s", q.CountryLocks["BE"])
	}
}

func TestDaemonQueueDequeueAndCancel(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	// Add queued task
	srv.runner.countryLocks["NL"] = "queued"
	srv.runner.queuedTasks = append(srv.runner.queuedTasks, QueuedTask{
		ID:          "task_test_nl",
		CountryCode: "NL",
		Status:      "queued",
	})

	// Dequeue NL
	reqDeq := httptest.NewRequest(http.MethodPost, "/api/v1/queue/dequeue?country=NL", nil)
	wDeq := httptest.NewRecorder()
	srv.handleDequeue(wDeq, reqDeq)
	if wDeq.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for dequeue, got %d", wDeq.Result().StatusCode)
	}

	if _, ok := srv.runner.countryLocks["NL"]; ok {
		t.Errorf("expected NL to be unlocked after dequeue")
	}

	// Add 2 queued tasks and test ClearQueue
	srv.runner.countryLocks["FR"] = "queued"
	srv.runner.countryLocks["DE"] = "queued"
	srv.runner.queuedTasks = append(srv.runner.queuedTasks,
		QueuedTask{ID: "t1", CountryCode: "FR", Status: "queued"},
		QueuedTask{ID: "t2", CountryCode: "DE", Status: "queued"},
	)

	reqClear := httptest.NewRequest(http.MethodDelete, "/api/v1/queue", nil)
	wClear := httptest.NewRecorder()
	srv.handleQueue(wClear, reqClear)
	if wClear.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for clear queue, got %d", wClear.Result().StatusCode)
	}

	if len(srv.runner.queuedTasks) != 0 {
		t.Errorf("expected 0 queued tasks, got %d", len(srv.runner.queuedTasks))
	}
}

func TestDaemonCountrySchedule(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    "../..",
		AutoImport: false,
	})

	srv.runner.countryLocks["BE"] = "running"
	srv.runner.countryLocks["NL"] = "queued"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/countries", nil)
	w := httptest.NewRecorder()
	srv.handleCountries(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}

	var res CountriesResponse
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, c := range res.Countries {
		if c.Code == "BE" && c.ScheduleStatus != "running" {
			t.Errorf("expected BE schedule_status to be running, got %s", c.ScheduleStatus)
		}
		if c.Code == "NL" && c.ScheduleStatus != "queued" {
			t.Errorf("expected NL schedule_status to be queued, got %s", c.ScheduleStatus)
		}
	}
}

func TestDaemonLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	langFile := filepath.Join(configDir, "languages.yaml")
	if err := os.WriteFile(langFile, []byte("version: 1\nlanguages:\n  - code: en\n    name: English\n    enabled: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(ServerConfig{
		Port:       ":0",
		RootDir:    tmpDir,
		AutoImport: false,
	})

	// GET /api/v1/languages
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/languages", nil)
	wGet := httptest.NewRecorder()
	srv.handleLanguages(wGet, reqGet)
	if wGet.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Result().StatusCode)
	}

	// PUT /api/v1/languages (add Thai)
	putBody := []byte(`{"code": "th", "name": "Thai", "enabled": true}`)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/languages", bytes.NewReader(putBody))
	wPut := httptest.NewRecorder()
	srv.handleLanguages(wPut, reqPut)
	if wPut.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wPut.Result().StatusCode, wPut.Body.String())
	}

	// Verify file was updated
	data, err := os.ReadFile(langFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "th") {
		t.Errorf("expected languages.yaml to contain 'th', got: %s", string(data))
	}
}
