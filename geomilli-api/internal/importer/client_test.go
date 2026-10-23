package importer_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/importer"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestImporterClient_GetStatus(t *testing.T) {
	cfg := &config.Config{HarvesterURL: "http://mock-harvester:8085"}
	httpClient := &http.Client{
		Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/v1/status" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"state": "running",
					"active_job": {
						"job_id": "job_be_001",
						"countries": ["BE"],
						"current_country": "BE",
						"stage": "enrich",
						"progress_pct": 45,
						"docs_processed": 50000,
						"started_at": "2026-10-23T04:00:00Z"
					},
					"resources": {
						"heap_alloc_mb": 25.4,
						"sys_mem_mb": 50.1,
						"disk_staging_mb": 110.0,
						"disk_cache_mb": 15.2,
						"sqlite_cache_entries": 3200
					},
					"history": []
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}
	client := importer.NewClientWithHTTPClient(cfg, httpClient)

	status, err := client.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.State != "running" {
		t.Errorf("expected state 'running', got %s", status.State)
	}
	if status.ActiveJob == nil || status.ActiveJob.JobID != "job_be_001" {
		t.Errorf("expected active job 'job_be_001', got %+v", status.ActiveJob)
	}
	if status.Resources.HeapAllocMB != 25.4 {
		t.Errorf("expected heap alloc 25.4, got %f", status.Resources.HeapAllocMB)
	}
}

func TestImporterClient_GetCountries(t *testing.T) {
	cfg := &config.Config{HarvesterURL: "http://mock-harvester:8085"}
	httpClient := &http.Client{
		Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"countries": [
						{"code": "BE", "name": "Belgium", "geofabrik_path": "europe/belgium", "is_indexed": true},
						{"code": "NL", "name": "Netherlands", "geofabrik_path": "europe/netherlands", "is_indexed": false}
					]
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	}
	client := importer.NewClientWithHTTPClient(cfg, httpClient)

	resp, err := client.GetCountries(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Countries) != 2 {
		t.Fatalf("expected 2 countries, got %d", len(resp.Countries))
	}
	if resp.Countries[0].Code != "BE" || !resp.Countries[0].IsIndexed {
		t.Errorf("expected BE indexed=true, got %+v", resp.Countries[0])
	}
}

func TestImporterClient_CreateJob(t *testing.T) {
	cfg := &config.Config{HarvesterURL: "http://mock-harvester:8085"}
	httpClient := &http.Client{
		Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/jobs" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(bytes.NewBufferString(`{"job_id": "job_new_123", "status": "started", "message": "Import job queued"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	client := importer.NewClientWithHTTPClient(cfg, httpClient)

	resp, err := client.CreateJob(context.Background(), importer.CreateJobRequest{
		Countries:    []string{"BE", "LU"},
		OnlineEnrich: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.JobID != "job_new_123" || resp.Status != "started" {
		t.Errorf("unexpected job response: %+v", resp)
	}
}

func TestImporterClient_HarvesterOffline(t *testing.T) {
	cfg := &config.Config{HarvesterURL: "http://mock-harvester:8085"}
	httpClient := &http.Client{
		Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}
	client := importer.NewClientWithHTTPClient(cfg, httpClient)

	_, err := client.GetStatus(context.Background())
	if err == nil {
		t.Fatal("expected error when harvester offline, got nil")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/importer/events", nil)
	err = client.RelaySSE(rec, req)
	if err != nil {
		t.Fatalf("expected graceful handling in RelaySSE, got err: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "harvester daemon offline") {
		t.Errorf("expected offline notice in SSE fallback, got %s", rec.Body.String())
	}
}

func TestImporterClient_QueueAndCancel(t *testing.T) {
	cfg := &config.Config{HarvesterURL: "http://mock-harvester:8085"}
	httpClient := &http.Client{
		Transport: mockTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/api/v1/queue/dequeue" && r.URL.Query().Get("country") == "NL" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"dequeued": true, "country": "NL", "message": "country dequeued"}`)),
					Header:     make(http.Header),
				}, nil
			}
			if r.URL.Path == "/api/v1/queue" && r.Method == http.MethodDelete {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"cleared": true, "message": "all tasks cleared"}`)),
					Header:     make(http.Header),
				}, nil
			}
			if r.URL.Path == "/api/v1/jobs/cancel" && r.Method == http.MethodPost {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"cancelled": true, "message": "worker cancelled"}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "not found"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	client := importer.NewClientWithHTTPClient(cfg, httpClient)

	deq, err := client.Dequeue(context.Background(), "NL")
	if err != nil {
		t.Fatalf("unexpected dequeue error: %v", err)
	}
	if !deq.Dequeued || deq.Country != "NL" {
		t.Errorf("expected dequeued NL, got %+v", deq)
	}

	err = client.ClearQueue(context.Background())
	if err != nil {
		t.Fatalf("unexpected clear queue error: %v", err)
	}

	canc, err := client.Cancel(context.Background(), importer.CancelJobRequest{WorkerID: 1})
	if err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if !canc.Cancelled {
		t.Errorf("expected cancelled true, got %+v", canc)
	}
}
