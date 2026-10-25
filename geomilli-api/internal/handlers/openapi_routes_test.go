package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/geomilli/geomilli-api/internal/version"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// TestOpenAPIRoutesCoverage asserts that every path & HTTP method registered in the Chi router
// via RegisterRoutes is fully documented in geomilli-api/api/openapi.yaml.
func TestOpenAPIRoutesCoverage(t *testing.T) {
	r, _ := setupTestRouter()

	type routeKey struct {
		method string
		path   string
	}

	routerRoutes := make(map[routeKey]bool)
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		routerRoutes[routeKey{
			method: strings.ToLower(method),
			path:   route,
		}] = true
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk Chi router: %v", err)
	}

	yamlData, err := loadOpenAPIYAML()
	if err != nil {
		t.Fatalf("Failed to load OpenAPI specification YAML: %v", err)
	}

	var spec map[string]any
	if err := yaml.Unmarshal(yamlData, &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI YAML: %v", err)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI spec missing 'paths' mapping")
	}

	var missing []string
	for rKey := range routerRoutes {
		pathItem, exists := paths[rKey.path]
		if !exists {
			missing = append(missing, rKey.method+" "+rKey.path+" (missing path)")
			continue
		}

		pathMap, isMap := pathItem.(map[string]any)
		if !isMap {
			missing = append(missing, rKey.method+" "+rKey.path+" (invalid path item)")
			continue
		}

		if _, methodExists := pathMap[rKey.method]; !methodExists {
			missing = append(missing, rKey.method+" "+rKey.path+" (missing operation)")
		}
	}

	if len(missing) > 0 {
		t.Errorf("The following %d routes registered in router are missing from openapi.yaml:\n%s",
			len(missing), strings.Join(missing, "\n"))
	}
}

// TestOpenAPISync asserts that docs/api/openapi.yaml is byte-for-byte identical to geomilli-api/api/openapi.yaml.
func TestOpenAPISync(t *testing.T) {
	canonicalPath := "../../api/openapi.yaml"
	docsPath := "../../../docs/api/openapi.yaml"

	canonicalBytes, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("Failed to read canonical OpenAPI spec at %s: %v", canonicalPath, err)
	}

	docsBytes, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatalf("Failed to read documentation OpenAPI spec at %s: %v", docsPath, err)
	}

	if !bytes.Equal(canonicalBytes, docsBytes) {
		t.Errorf("Documentation OpenAPI spec (%s) is not byte-for-byte identical to canonical spec (%s)", docsPath, canonicalPath)
	}
}

// TestOpenAPIEndpointVersion asserts that GET /v1/openapi.json returns valid JSON
// with info.version dynamically matching version.Version.
func TestOpenAPIEndpointVersion(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Initial request with current version
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected application/json content type, got: %s", contentType)
	}

	var spec map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatalf("Missing 'info' object in openapi.json")
	}

	if info["version"] != version.Version {
		t.Errorf("Expected info.version == %q, got %q", version.Version, info["version"])
	}

	// 2. Dynamic version check: change version and assert response reflects it
	oldVersion := version.Version
	defer func() {
		version.Version = oldVersion
	}()

	testVersion := "2.1.0-alpha.test"
	version.Version = testVersion

	req2 := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 on dynamic check, got %d", rr2.Code)
	}

	var spec2 map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &spec2); err != nil {
		t.Fatalf("Response is not valid JSON on dynamic check: %v", err)
	}

	info2, ok := spec2["info"].(map[string]any)
	if !ok {
		t.Fatalf("Missing 'info' object on dynamic check")
	}

	if info2["version"] != testVersion {
		t.Errorf("Expected dynamically updated info.version == %q, got %q", testVersion, info2["version"])
	}
}

// TestOpenAPILoadFallback verifies path discovery in loadOpenAPIYAML.
func TestOpenAPILoadFallback(t *testing.T) {
	data, err := loadOpenAPIYAML()
	if err != nil {
		t.Fatalf("loadOpenAPIYAML returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("loadOpenAPIYAML returned empty bytes")
	}
}
