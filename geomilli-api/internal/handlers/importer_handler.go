package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geomilli/geomilli-api/internal/importer"
)

// ImporterStatus proxies the harvester daemon's GET /api/v1/status.
func (h *Handler) ImporterStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.importer.GetStatus(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"state": "offline",
			"error": err.Error(),
			"resources": map[string]any{
				"heap_alloc_mb": 0,
				"sys_mem_mb":    0,
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// ImporterCountries proxies the harvester daemon's GET /api/v1/countries.
func (h *Handler) ImporterCountries(w http.ResponseWriter, r *http.Request) {
	countries, err := h.importer.GetCountries(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":     err.Error(),
			"countries": []any{},
		})
		return
	}
	writeJSON(w, http.StatusOK, countries)
}

// ImporterQueue proxies the harvester daemon's GET /api/v1/queue.
func (h *Handler) ImporterQueue(w http.ResponseWriter, r *http.Request) {
	queue, err := h.importer.GetQueue(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, queue)
}

// ImporterCreateJob triggers a new import job on the harvester daemon.
func (h *Handler) ImporterCreateJob(w http.ResponseWriter, r *http.Request) {
	var jobReq importer.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&jobReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if len(jobReq.Countries) == 0 {
		writeError(w, http.StatusBadRequest, "Must specify at least one country in 'countries' array")
		return
	}

	res, err := h.importer.CreateJob(r.Context(), jobReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to start harvester job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, res)
}

// ImporterClearQueue clears all pending countries from the harvester queue.
func (h *Handler) ImporterClearQueue(w http.ResponseWriter, r *http.Request) {
	if err := h.importer.ClearQueue(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, "Failed to clear queue: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cleared": true,
		"message": "All pending queue tasks removed",
	})
}

// ImporterDequeue removes a specific country from the queue.
func (h *Handler) ImporterDequeue(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		country = r.URL.Query().Get("country_code")
	}
	if country == "" && r.Body != nil {
		var req struct {
			Country     string `json:"country"`
			CountryCode string `json:"country_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Country != "" {
				country = req.Country
			} else {
				country = req.CountryCode
			}
		}
	}

	if country == "" {
		writeError(w, http.StatusBadRequest, "Missing 'country' parameter")
		return
	}

	res, err := h.importer.Dequeue(r.Context(), country)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to dequeue: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ImporterCancel stops an active worker slot or cancels a country job.
func (h *Handler) ImporterCancel(w http.ResponseWriter, r *http.Request) {
	var cancelReq importer.CancelJobRequest

	if r.Header.Get("Content-Type") == "application/json" && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&cancelReq)
	}
	if cancelReq.CountryCode == "" {
		cancelReq.CountryCode = r.URL.Query().Get("country")
		if cancelReq.CountryCode == "" {
			cancelReq.CountryCode = r.URL.Query().Get("country_code")
		}
	}
	if cancelReq.WorkerID == 0 && r.URL.Query().Get("worker_id") != "" {
		var wid int
		if _, err := fmt.Sscanf(r.URL.Query().Get("worker_id"), "%d", &wid); err == nil {
			cancelReq.WorkerID = wid
		}
	}

	res, err := h.importer.Cancel(r.Context(), cancelReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to cancel job: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ImporterEvents relays Server-Sent Events from the harvester daemon to the frontend.
func (h *Handler) ImporterEvents(w http.ResponseWriter, r *http.Request) {
	if err := h.importer.RelaySSE(w, r); err != nil {
		// Connection closed by client or network error
		return
	}
}
