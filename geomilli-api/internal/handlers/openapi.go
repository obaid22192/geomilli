package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/geomilli/geomilli-api/internal/version"
	"gopkg.in/yaml.v3"
)

var (
	cachedSpecLock    sync.RWMutex
	cachedSpecVersion string
	cachedSpecJSON    []byte
)

// loadOpenAPIYAML searches common paths for api/openapi.yaml across test, development, and container runtimes.
func loadOpenAPIYAML() ([]byte, error) {
	candidates := []string{
		"../../api/openapi.yaml",
		"../api/openapi.yaml",
		"api/openapi.yaml",
		"geomilli-api/api/openapi.yaml",
		"/app/api/openapi.yaml",
	}

	for _, c := range candidates {
		if data, err := os.ReadFile(c); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// Walk upwards from working directory
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 5; i++ {
			p := filepath.Join(dir, "api", "openapi.yaml")
			if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
				return data, nil
			}
			p2 := filepath.Join(dir, "geomilli-api", "api", "openapi.yaml")
			if data, err := os.ReadFile(p2); err == nil && len(data) > 0 {
				return data, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Walk relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, rel := range []string{"api/openapi.yaml", "../api/openapi.yaml", "../../api/openapi.yaml"} {
			p := filepath.Join(exeDir, rel)
			if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("openapi.yaml specification file not found")
}

// GetOpenAPI returns the dynamic OpenAPI 3.1 specification JSON with info.version set to version.Version.
func (h *Handler) GetOpenAPI(w http.ResponseWriter, r *http.Request) {
	cachedSpecLock.RLock()
	if cachedSpecJSON != nil && cachedSpecVersion == version.Version {
		data := cachedSpecJSON
		cachedSpecLock.RUnlock()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	cachedSpecLock.RUnlock()

	cachedSpecLock.Lock()
	defer cachedSpecLock.Unlock()

	// Double-check under write lock
	if cachedSpecJSON != nil && cachedSpecVersion == version.Version {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cachedSpecJSON)
		return
	}

	yamlData, err := loadOpenAPIYAML()
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalError, "Failed to load OpenAPI spec: "+err.Error())
		return
	}

	var spec map[string]any
	if err := yaml.Unmarshal(yamlData, &spec); err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalError, "Failed to parse OpenAPI spec: "+err.Error())
		return
	}

	// Dynamically set info.version from internal/version.Version
	if info, ok := spec["info"].(map[string]any); ok {
		info["version"] = version.Version
	} else {
		spec["info"] = map[string]any{
			"title":   "Geomilli API",
			"version": version.Version,
		}
	}

	jsonData, err := json.Marshal(spec)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalError, "Failed to serialize OpenAPI spec to JSON: "+err.Error())
		return
	}

	cachedSpecVersion = version.Version
	cachedSpecJSON = jsonData

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonData)
}
