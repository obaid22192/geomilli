package metrics

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

const (
	ringBufferSize   = 4096
	slidingWindowSec = 60
	maxTrackedQueries = 100
)

// Collector tracks API telemetry using ring buffers and counters.
type Collector struct {
	mu sync.RWMutex

	totalRequests   int64
	endpointCounts  map[string]int64
	statusCounts    map[int]int64
	topQueries      map[string]int64
	zeroHitQueries  map[string]int64

	// Ring buffer for latency in milliseconds
	latencies  [ringBufferSize]float64
	latencyIdx int
	latencyLen int

	// Sliding window for QPS: timestamps of requests in seconds
	requestTimes []time.Time
}

// NewCollector initializes a new telemetry collector.
func NewCollector() *Collector {
	return &Collector{
		endpointCounts: make(map[string]int64),
		statusCounts:   make(map[int]int64),
		topQueries:     make(map[string]int64),
		zeroHitQueries: make(map[string]int64),
		requestTimes:   make([]time.Time, 0, 1000),
	}
}

// Record logs a single request's telemetry.
func (c *Collector) Record(endpoint string, statusCode int, duration time.Duration, query string, hits int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalRequests++
	c.endpointCounts[endpoint]++
	c.statusCounts[statusCode]++

	// Record latency in ms in ring buffer
	ms := float64(duration.Microseconds()) / 1000.0
	c.latencies[c.latencyIdx] = ms
	c.latencyIdx = (c.latencyIdx + 1) % ringBufferSize
	if c.latencyLen < ringBufferSize {
		c.latencyLen++
	}

	// Sliding window for QPS
	now := time.Now()
	c.requestTimes = append(c.requestTimes, now)
	cutoff := now.Add(-slidingWindowSec * time.Second)
	// Prune old request times
	i := 0
	for ; i < len(c.requestTimes); i++ {
		if c.requestTimes[i].After(cutoff) {
			break
		}
	}
	if i > 0 {
		c.requestTimes = c.requestTimes[i:]
	}

	// Query tracking
	if query != "" {
		c.topQueries[query]++
		if len(c.topQueries) > maxTrackedQueries*2 {
			c.pruneMap(c.topQueries)
		}

		if hits == 0 {
			c.zeroHitQueries[query]++
			if len(c.zeroHitQueries) > maxTrackedQueries*2 {
				c.pruneMap(c.zeroHitQueries)
			}
		}
	}
}

func (c *Collector) pruneMap(m map[string]int64) {
	type entry struct {
		k string
		v int64
	}
	entries := make([]entry, 0, len(m))
	for k, v := range m {
		entries = append(entries, entry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].v > entries[j].v
	})
	for i := maxTrackedQueries; i < len(entries); i++ {
		delete(m, entries[i].k)
	}
}

// LatencyPercentiles holds p50, p95, and p99 values in milliseconds.
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
	Status        string  `json:"status"` // "healthy", "unreachable"
	Mode          string  `json:"mode"`   // "local", "cloud"
	BaseURL       string  `json:"base_url"`
	LatencyMs     float64 `json:"latency_ms"`
	IndexDocs     int64   `json:"index_docs"`
	EstimatedMB   float64 `json:"estimated_index_mb"`
}

// SystemMemory models memory and runtime statistics.
type SystemMemory struct {
	HeapAllocMB   float64 `json:"heap_alloc_mb"`
	SysMemMB      float64 `json:"sys_mem_mb"`
	NumGoroutines int     `json:"num_goroutines"`
}

// TelemetrySnapshot holds full telemetry report.
type TelemetrySnapshot struct {
	Timestamp          string             `json:"timestamp"`
	TotalRequests      int64              `json:"total_requests"`
	ActiveQPS          float64            `json:"active_qps"`
	Latency            LatencyPercentiles `json:"latency"`
	EndpointBreakdown  map[string]int64   `json:"endpoint_breakdown"`
	StatusCodes        map[int]int64      `json:"status_codes"`
	TopQueries         []QueryCount       `json:"top_queries"`
	ZeroHitQueries     []QueryCount       `json:"zero_hit_queries"`
	Meilisearch        MeiliInfo          `json:"meilisearch"`
	System             SystemMemory       `json:"system"`
}

// Snapshot generates the current telemetry report.
func (c *Collector) Snapshot(meili MeiliInfo) TelemetrySnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Calculate QPS over sliding window
	now := time.Now()
	cutoff := now.Add(-slidingWindowSec * time.Second)
	recentCount := 0
	for _, t := range c.requestTimes {
		if t.After(cutoff) {
			recentCount++
		}
	}
	qps := float64(recentCount) / float64(slidingWindowSec)
	qps = math.Round(qps*100) / 100

	// Calculate latency percentiles
	percentiles := LatencyPercentiles{}
	if c.latencyLen > 0 {
		sorted := make([]float64, c.latencyLen)
		copy(sorted, c.latencies[:c.latencyLen])
		sort.Float64s(sorted)

		p50Idx := int(float64(c.latencyLen) * 0.50)
		p95Idx := int(float64(c.latencyLen) * 0.95)
		p99Idx := int(float64(c.latencyLen) * 0.99)

		if p50Idx >= c.latencyLen {
			p50Idx = c.latencyLen - 1
		}
		if p95Idx >= c.latencyLen {
			p95Idx = c.latencyLen - 1
		}
		if p99Idx >= c.latencyLen {
			p99Idx = c.latencyLen - 1
		}

		percentiles.P50 = math.Round(sorted[p50Idx]*100) / 100
		percentiles.P95 = math.Round(sorted[p95Idx]*100) / 100
		percentiles.P99 = math.Round(sorted[p99Idx]*100) / 100
	}

	// Copy maps
	epCopy := make(map[string]int64, len(c.endpointCounts))
	for k, v := range c.endpointCounts {
		epCopy[k] = v
	}

	scCopy := make(map[int]int64, len(c.statusCounts))
	for k, v := range c.statusCounts {
		scCopy[k] = v
	}

	topList := topNQueries(c.topQueries, 10)
	zeroList := topNQueries(c.zeroHitQueries, 10)

	// Memory stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	sysMem := SystemMemory{
		HeapAllocMB:   math.Round((float64(mem.HeapAlloc)/(1024*1024))*100) / 100,
		SysMemMB:      math.Round((float64(mem.Sys)/(1024*1024))*100) / 100,
		NumGoroutines: runtime.NumGoroutine(),
	}

	return TelemetrySnapshot{
		Timestamp:         now.UTC().Format(time.RFC3339),
		TotalRequests:     c.totalRequests,
		ActiveQPS:         qps,
		Latency:           percentiles,
		EndpointBreakdown: epCopy,
		StatusCodes:       scCopy,
		TopQueries:        topList,
		ZeroHitQueries:    zeroList,
		Meilisearch:       meili,
		System:            sysMem,
	}
}

func topNQueries(m map[string]int64, n int) []QueryCount {
	list := make([]QueryCount, 0, len(m))
	for k, v := range m {
		list = append(list, QueryCount{Query: k, Count: v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Query < list[j].Query
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}
