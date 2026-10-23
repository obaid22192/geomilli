package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/importer"
	"github.com/geomilli/geomilli-api/internal/meili"
	"github.com/geomilli/geomilli-api/internal/metrics"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler holds dependencies for all API endpoints.
type Handler struct {
	cfg       *config.Config
	meili     *meili.Client
	importer  *importer.Client
	collector *metrics.Collector
}

// NewHandler initializes a new Handler.
func NewHandler(cfg *config.Config, meiliClient *meili.Client, importerClient *importer.Client, collector *metrics.Collector) *Handler {
	return &Handler{
		cfg:       cfg,
		meili:     meiliClient,
		importer:  importerClient,
		collector: collector,
	}
}

// RegisterRoutes registers all 10 endpoint groups on the given Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Base Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(appMiddleware.CORS(h.cfg.AllowedOrigins))
	r.Use(appMiddleware.RequestLogger)
	r.Use(appMiddleware.TelemetryRecorder(h.collector))

	// Root redirect / info
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        "Geomilli API",
			"version":     "1.0.0",
			"docs":        "/v1/openapi.json",
			"health":      "/v1/health",
			"status":      "online",
			"meili_mode":  h.getMeiliMode(),
		})
	})

	// /v1 Subrouter
	r.Route("/v1", func(v1 chi.Router) {
		// 1. Autocomplete
		v1.Get("/autocomplete", h.Autocomplete)

		// 2. Full-Text & Faceted Search
		v1.Get("/search", h.Search)

		// 3. Reverse Geocoding
		v1.Get("/reverse", h.Reverse)

		// 4. Nearby POIs (Geo-Radius)
		v1.Get("/nearby", h.Nearby)

		// 5. Bounding Box Viewport
		v1.Get("/bbox", h.BBox)

		// 6. Places & Entity Lookup
		v1.Get("/places/{id}", h.GetPlaceByID)
		v1.Get("/places/wikidata/{qid}", h.GetPlaceByWikidataID)

		// 7. Aviation / Airports
		v1.Get("/airports", h.SearchAirports)
		v1.Get("/airports/{code}", h.GetAirportByCode)

		// 8. Administrative Hierarchy
		v1.Get("/countries", h.ListCountries)
		v1.Get("/countries/{code}/regions", h.ListCountryRegions)
		v1.Get("/countries/{code}/cities", h.ListCountryCities)

		// 9. Taxonomy & Schema Metadata
		v1.Get("/taxonomy", h.GetTaxonomy)
		v1.Get("/languages", h.GetLanguages)
		v1.Put("/languages", h.UpdateLanguages)

		// 10. Importer Proxy & Observability
		v1.Get("/importer/status", h.ImporterStatus)
		v1.Get("/importer/countries", h.ImporterCountries)
		v1.Get("/importer/queue", h.ImporterQueue)
		v1.Delete("/importer/queue", h.ImporterClearQueue)
		v1.Post("/importer/queue/dequeue", h.ImporterDequeue)
		v1.Post("/importer/cancel", h.ImporterCancel)
		v1.Post("/importer/jobs", h.ImporterCreateJob)
		v1.Get("/importer/events", h.ImporterEvents)

		v1.Get("/metrics", h.GetMetrics)
		v1.Get("/health", h.GetHealth)
		v1.Get("/openapi.json", h.GetOpenAPI)
	})
}

func (h *Handler) getMeiliMode() string {
	if h.meili.IsCloud() {
		return "cloud"
	}
	return "local"
}

// writeJSON is a helper to serialize data to JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError serializes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   true,
		"status":  status,
		"message": message,
	})
}

// localizePlace replaces NameDisplay with language-specific name if available.
func localizePlace(p *schema.Place, lang string) {
	if lang == "" || p == nil || len(p.NamesByLanguage) == 0 {
		return
	}
	if loc, ok := p.NamesByLanguage[lang]; ok && loc != "" {
		p.NameDisplay = loc
	}
}

// localizePlaces localizes an array of places in-place.
func localizePlaces(places []schema.Place, lang string) {
	if lang == "" {
		return
	}
	for i := range places {
		localizePlace(&places[i], lang)
	}
}
