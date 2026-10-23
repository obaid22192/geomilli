package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/geomilli/geomilli-api/internal/taxonomy"
)

// GetTaxonomy returns all 19 taxonomy classes, layer classifications, base scores, match criteria, and attributes.
func (h *Handler) GetTaxonomy(w http.ResponseWriter, r *http.Request) {
	tax, err := taxonomy.GetTaxonomy()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load taxonomy: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tax)
}

// GetLanguages returns the whitelist of supported language codes and their full names.
// It prioritizes dynamic configuration from Harvester daemon, falling back to embedded config.
func (h *Handler) GetLanguages(w http.ResponseWriter, r *http.Request) {
	if h.importer != nil {
		if cfg, err := h.importer.GetLanguages(r.Context()); err == nil && cfg != nil && len(cfg.Languages) > 0 {
			writeJSON(w, http.StatusOK, cfg)
			return
		}
	}

	langs, err := taxonomy.GetLanguages()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load languages: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, langs)
}

// UpdateLanguages toggles or adds language support via Harvester daemon.
func (h *Handler) UpdateLanguages(w http.ResponseWriter, r *http.Request) {
	var body any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	if h.importer == nil {
		writeError(w, http.StatusServiceUnavailable, "Harvester service client not configured")
		return
	}

	res, err := h.importer.UpdateLanguages(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to update languages: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"updated":   true,
		"languages": res.Languages,
		"message":   "Language configuration updated. Re-index countries to apply new language extractions.",
	})
}
