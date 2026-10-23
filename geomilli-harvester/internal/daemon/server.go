package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/travayo/auto-complete/internal/countries"
	"github.com/travayo/auto-complete/internal/languages"
)

type ServerConfig struct {
	Port         string
	RootDir      string
	MeiliHost    string
	MeiliKey     string
	AutoImport   bool
	CountriesEnv string
}

type Server struct {
	cfg         ServerConfig
	runner      *Runner
	broadcaster *Broadcaster
}

func NewServer(cfg ServerConfig) *Server {
	if cfg.Port == "" {
		cfg.Port = "8085"
	}
	if !strings.HasPrefix(cfg.Port, ":") {
		cfg.Port = ":" + cfg.Port
	}
	if cfg.RootDir == "" {
		var err error
		cfg.RootDir, err = os.Getwd()
		if err != nil {
			cfg.RootDir = "."
		}
	}
	if cfg.MeiliHost == "" {
		cfg.MeiliHost = "http://127.0.0.1:7700"
	}
	if cfg.CountriesEnv == "" {
		cfg.CountriesEnv = "BE"
	}

	b := NewBroadcaster()
	r := NewRunner(cfg.RootDir, cfg.MeiliHost, cfg.MeiliKey, b)

	return &Server{
		cfg:         cfg,
		runner:      r,
		broadcaster: b,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/status", s.cors(s.handleStatus))
	mux.HandleFunc("/api/v1/countries", s.cors(s.handleCountries))
	mux.HandleFunc("/api/v1/jobs", s.cors(s.handleJobs))
	mux.HandleFunc("/api/v1/jobs/cancel", s.cors(s.handleCancelJob))
	mux.HandleFunc("/api/v1/queue", s.cors(s.handleQueue))
	mux.HandleFunc("/api/v1/queue/dequeue", s.cors(s.handleDequeue))
	mux.HandleFunc("/api/v1/queue/clear", s.cors(s.handleClearQueue))
	mux.HandleFunc("/api/v1/jobs/periodic-sync", s.cors(s.handlePeriodicSync))
	mux.HandleFunc("/api/v1/languages", s.cors(s.handleLanguages))
	mux.HandleFunc("/api/v1/events", s.handleEvents)
	mux.HandleFunc("/health", s.cors(s.handleHealth))
	mux.HandleFunc("/api/v1/health", s.cors(s.handleHealth))

	server := &http.Server{
		Addr:    s.cfg.Port,
		Handler: mux,
	}

	fmt.Printf("[harvester] daemon listening on %s (meili=%s)\n", s.cfg.Port, s.cfg.MeiliHost)

	// Check for auto-import on startup in background
	go s.checkAutoImport()

	return server.ListenAndServe()
}

func (s *Server) checkAutoImport() {
	time.Sleep(1 * time.Second) // wait for server to bind

	docCount, err := s.checkIndexDocCount("places")
	if err == nil && docCount > 0 {
		fmt.Printf("[harvester] places index already active with %d documents\n", docCount)
		s.runner.SetCompleted("Live index verified", int(docCount))
		s.broadcaster.BroadcastLog("info", fmt.Sprintf("Live places index verified (%d documents active in Meilisearch)", docCount))
		return
	}

	shouldImport := s.cfg.AutoImport || err != nil || docCount == 0

	if shouldImport {
		var target []string
		for _, part := range strings.Split(s.cfg.CountriesEnv, ",") {
			part = strings.ToUpper(strings.TrimSpace(part))
			if part != "" {
				target = append(target, part)
			}
		}
		if len(target) == 0 {
			target = []string{"BE"}
		}

		fmt.Printf("[harvester] auto-import triggered on startup for countries: %v\n", target)
		s.broadcaster.BroadcastLog("info", fmt.Sprintf("Auto-import triggered on startup for countries: %v", target))
		_, _ = s.runner.StartJob(target, false, "replace")
	}
}

func (s *Server) checkIndexDocCount(indexUID string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/indexes/%s/stats", strings.TrimRight(s.cfg.MeiliHost, "/"), indexUID), nil)
	if err != nil {
		return 0, err
	}
	if s.cfg.MeiliKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.MeiliKey)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}

	var payload struct {
		NumberOfDocuments int64 `json:"numberOfDocuments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.NumberOfDocuments, nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.runner.GetStatus()
	if status.State != "running" {
		if docCount, err := s.checkIndexDocCount("places"); err == nil && docCount > 0 {
			if status.State == "failed" || status.State == "idle" {
				s.runner.SetCompleted("Live index verified", int(docCount))
				status = s.runner.GetStatus()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) handleCountries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configPath := filepath.Join(s.cfg.RootDir, "config", "countries.yaml")
	baseURL := os.Getenv("OSM_GEOFABRIK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://download.geofabrik.de"
	}
	resolved, err := countries.Resolve(configPath, baseURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve countries: %v", err), http.StatusInternalServerError)
		return
	}

	out := make([]CountryInfo, 0, len(resolved))
	syncIntervalHours := 24
	if envSync := os.Getenv("DELTA_SYNC_INTERVAL_HOURS"); envSync != "" {
		if n, err := strconv.Atoi(envSync); err == nil && n > 0 {
			syncIntervalHours = n
		}
	}
	q := s.runner.GetQueue()
	meiliOnline := true

	for _, c := range resolved {
		lcCode := strings.ToLower(c.Code)
		buildManifest := filepath.Join(s.cfg.RootDir, "data", "manifests", fmt.Sprintf("build-places-%s.json", lcCode))

		isIndexed := false
		lastBuiltAt := ""

		if data, err := os.ReadFile(buildManifest); err == nil {
			var bm struct {
				UpdatedAt string `json:"updated_at"`
			}
			if err := json.Unmarshal(data, &bm); err == nil {
				lastBuiltAt = bm.UpdatedAt
				isIndexed = true
			}
		}

		// Also check Meilisearch directly if available
		if !isIndexed && meiliOnline {
			if count, err := s.checkCountryDocCount(c.Code); err == nil && count > 0 {
				isIndexed = true
			} else if err != nil {
				meiliOnline = false
			}
		}

		// Compute delta schedule status
		scheduleStatus := "idle"
		nextRunAt := "Not yet run"
		if lock, ok := q.CountryLocks[c.Code]; ok {
			if lock == "running" {
				scheduleStatus = "running"
				nextRunAt = "In Progress"
			} else if lock == "queued" {
				scheduleStatus = "queued"
				nextRunAt = "Queued"
			}
		} else if lastBuiltAt != "" {
			if parsed, err := time.Parse(time.RFC3339, lastBuiltAt); err == nil {
				nextTime := parsed.Add(time.Duration(syncIntervalHours) * time.Hour)
				nextRunAt = nextTime.UTC().Format(time.RFC3339)
				if nextTime.Before(time.Now().UTC()) {
					scheduleStatus = "overdue"
				} else {
					scheduleStatus = "scheduled"
				}
			}
		}

		out = append(out, CountryInfo{
			Code:              c.Code,
			Name:              c.Name,
			Continent:         c.Continent,
			GeofabrikPath:     c.GeofabrikPath,
			IsIndexed:         isIndexed,
			LastBuiltAt:       lastBuiltAt,
			NextRunAt:         nextRunAt,
			SyncIntervalHours: syncIntervalHours,
			ScheduleStatus:    scheduleStatus,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(CountriesResponse{Countries: out})
}

func (s *Server) checkCountryDocCount(code string) (int64, error) {
	payload := map[string]interface{}{
		"filter": fmt.Sprintf("country_code = '%s'", strings.ToUpper(code)),
		"limit":  1,
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/indexes/places/search", strings.TrimRight(s.cfg.MeiliHost, "/")), bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.MeiliKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.MeiliKey)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("search failed with %d", resp.StatusCode)
	}

	var res struct {
		EstimatedTotalHits int64 `json:"estimatedTotalHits"`
		TotalHits          int64 `json:"totalHits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	if res.TotalHits > 0 {
		return res.TotalHits, nil
	}
	return res.EstimatedTotalHits, nil
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		cleared, _ := s.runner.ClearQueue()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Cleared %d tasks from queue", cleared),
			"cleared": cleared,
		})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queue := s.runner.GetQueue()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(queue)
}

func (s *Server) handleClearQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cleared, _ := s.runner.ClearQueue()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Cleared %d tasks from queue", cleared),
		"cleared": cleared,
	})
}

func (s *Server) handleDequeue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CountryCode string `json:"country_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.CountryCode == "" {
		req.CountryCode = r.URL.Query().Get("country")
	}
	if req.CountryCode == "" {
		http.Error(w, "country_code required", http.StatusBadRequest)
		return
	}

	_, err := s.runner.DequeueCountry(req.CountryCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	q := s.runner.GetQueue()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DequeueResponse{
		Message:     fmt.Sprintf("Country %s dequeued successfully", req.CountryCode),
		CountryCode: req.CountryCode,
		Remaining:   len(q.QueuedTasks),
	})
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CancelJobRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.CountryCode == "" {
		req.CountryCode = r.URL.Query().Get("country")
	}

	if req.WorkerID > 0 {
		if err := s.runner.StopWorker(req.WorkerID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CancelJobResponse{
			Message:   fmt.Sprintf("Worker #%d cancelled", req.WorkerID),
			Cancelled: []string{fmt.Sprintf("worker_%d", req.WorkerID)},
		})
		return
	}

	if req.CountryCode != "" {
		action, err := s.runner.CancelCountry(req.CountryCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CancelJobResponse{
			Message:   fmt.Sprintf("Country %s %s", req.CountryCode, action),
			Cancelled: []string{req.CountryCode},
		})
		return
	}

	// Stop all active workers and clear queue
	resp, _ := s.runner.StopAll()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePeriodicSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enqueue all configured countries to perform an ETag/delta check
	configPath := filepath.Join(s.cfg.RootDir, "config", "countries.yaml")
	baseURL := os.Getenv("OSM_GEOFABRIK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://download.geofabrik.de"
	}
	resolved, err := countries.Resolve(configPath, baseURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve countries: %v", err), http.StatusInternalServerError)
		return
	}

	var targets []string
	for _, c := range resolved {
		targets = append(targets, c.Code)
	}

	resp, err := s.runner.EnqueueJob(targets, false, "append")
	if err != nil {
		http.Error(w, fmt.Sprintf("enqueue periodic sync: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json body: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Countries) == 0 {
		http.Error(w, "at least one country code is required", http.StatusBadRequest)
		return
	}

	resp, err := s.runner.EnqueueJob(req.Countries, req.OnlineEnrich, req.Mode)
	if err != nil {
		http.Error(w, fmt.Sprintf("enqueue job: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(resp.Enqueued) == 0 && len(resp.Deduplicated) > 0 {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":        "job already in progress",
			"message":      resp.Message,
			"deduplicated": resp.Deduplicated,
		})
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.broadcaster.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "geomilli-harvester",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	langFile := filepath.Join(s.cfg.RootDir, "config", "languages.yaml")

	if r.Method == http.MethodGet {
		cfg, err := languages.LoadFullConfig(langFile)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load languages: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
		return
	}

	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		var req struct {
			Code      string                  `json:"code"`
			Name      string                  `json:"name"`
			Enabled   bool                    `json:"enabled"`
			Languages []languages.LanguageDef `json:"languages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
			return
		}

		var updatedCfg *languages.Config
		var err error
		if len(req.Languages) > 0 {
			updatedCfg = &languages.Config{
				Version:   1,
				Languages: req.Languages,
			}
			err = languages.SaveToFile(langFile, updatedCfg)
		} else if req.Code != "" {
			updatedCfg, err = languages.AddOrUpdateLanguage(langFile, req.Code, req.Name, req.Enabled)
		} else {
			http.Error(w, "missing 'code' or 'languages' in request", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("failed to save languages: %v", err), http.StatusInternalServerError)
			return
		}

		s.broadcaster.BroadcastLog("info", fmt.Sprintf("Language configuration updated in %s", langFile))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"updated":   true,
			"languages": updatedCfg.Languages,
			"message":   "Language configuration updated successfully. Re-index countries to apply new language extractions.",
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
