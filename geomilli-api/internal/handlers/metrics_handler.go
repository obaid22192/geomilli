package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/geomilli/geomilli-api/internal/metrics"
)

// GetMetrics returns real-time ring-buffer telemetry, QPS, latency percentiles, and system memory.
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	meiliInfo := metrics.MeiliInfo{
		Status:  "unreachable",
		Mode:    h.getMeiliMode(),
		BaseURL: h.meili.BaseURL(),
	}

	start := time.Now()
	health, err := h.meili.Health(ctx)
	if err == nil && health != nil {
		meiliInfo.Status = health.Status
		meiliInfo.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0

		// Also fetch doc count
		stats, errStats := h.meili.GetStats(ctx)
		if errStats == nil && stats != nil {
			meiliInfo.IndexDocs = stats.NumberOfDocuments
			// Rough estimate: ~1KB per indexed place document in LMDB
			meiliInfo.EstimatedMB = float64(stats.NumberOfDocuments) * 0.001
		}
	}

	snapshot := h.collector.Snapshot(meiliInfo)
	writeJSON(w, http.StatusOK, snapshot)
}

// GetHealth returns the health status of the API service and its connection to Meilisearch.
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	health, err := h.meili.Health(ctx)
	latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

	meiliStatus := "available"
	if err != nil || health == nil {
		meiliStatus = "unreachable"
	}

	statusCode := http.StatusOK
	apiStatus := "healthy"
	if meiliStatus == "unreachable" {
		statusCode = http.StatusServiceUnavailable
		apiStatus = "degraded"
	}

	writeJSON(w, statusCode, map[string]any{
		"status":     apiStatus,
		"version":    "1.0.0",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"meilisearch": map[string]any{
			"status":     meiliStatus,
			"mode":       h.getMeiliMode(),
			"base_url":   h.meili.BaseURL(),
			"latency_ms": latencyMs,
		},
	})
}
