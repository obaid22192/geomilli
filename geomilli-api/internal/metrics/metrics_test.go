package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	c := NewCollector()

	// Record varying latencies: 1ms, 2ms, ... 100ms
	for i := 1; i <= 100; i++ {
		query := ""
		hits := 1
		if i%2 == 0 {
			query = "brussels"
		} else if i%5 == 0 {
			query = "unknown_place"
			hits = 0
		}
		c.Record("/v1/autocomplete", 200, time.Duration(i)*time.Millisecond, query, hits)
	}

	snap := c.Snapshot(MeiliInfo{
		Status:    "healthy",
		Mode:      "local",
		BaseURL:   "http://localhost:7700",
		LatencyMs: 1.5,
		IndexDocs: 150000,
	})

	if snap.TotalRequests != 100 {
		t.Fatalf("expected 100 total requests, got %d", snap.TotalRequests)
	}
	if snap.EndpointBreakdown["/v1/autocomplete"] != 100 {
		t.Fatalf("expected 100 endpoint requests, got %d", snap.EndpointBreakdown["/v1/autocomplete"])
	}

	// p50 should be around 50ms, p95 around 95ms, p99 around 99ms
	if snap.Latency.P50 < 45 || snap.Latency.P50 > 55 {
		t.Fatalf("unexpected p50: %f", snap.Latency.P50)
	}
	if snap.Latency.P95 < 90 || snap.Latency.P95 > 98 {
		t.Fatalf("unexpected p95: %f", snap.Latency.P95)
	}

	if len(snap.TopQueries) == 0 || snap.TopQueries[0].Query != "brussels" {
		t.Fatalf("expected top query brussels, got %+v", snap.TopQueries)
	}
	if len(snap.ZeroHitQueries) == 0 || snap.ZeroHitQueries[0].Query != "unknown_place" {
		t.Fatalf("expected zero hit query unknown_place, got %+v", snap.ZeroHitQueries)
	}
}

func TestConcurrentRecord(t *testing.T) {
	c := NewCollector()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Record("/v1/search", 200, 10*time.Millisecond, "antwerp", 5)
			}
		}(i)
	}

	wg.Wait()
	snap := c.Snapshot(MeiliInfo{})
	if snap.TotalRequests != 1000 {
		t.Fatalf("expected 1000 requests, got %d", snap.TotalRequests)
	}
}
