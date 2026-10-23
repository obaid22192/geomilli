package config_test

import (
	"os"
	"testing"

	"github.com/geomilli/geomilli-api/internal/config"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("PORT")
	os.Unsetenv("API_PORT")
	os.Unsetenv("MEILI_HOST")
	os.Unsetenv("MEILI_API_KEY")
	os.Unsetenv("MEILI_INDEX")
	os.Unsetenv("HARVESTER_URL")
	os.Unsetenv("MEILI_MODE")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := config.Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.MeiliHost != "http://127.0.0.1:7700" {
		t.Errorf("expected default meili host http://127.0.0.1:7700, got %s", cfg.MeiliHost)
	}
	if cfg.MeiliIndex != "places" {
		t.Errorf("expected default meili index 'places', got %s", cfg.MeiliIndex)
	}
	if cfg.HarvesterURL != "http://127.0.0.1:8085" {
		t.Errorf("expected default harvester url http://127.0.0.1:8085, got %s", cfg.HarvesterURL)
	}
	if cfg.IsCloud {
		t.Errorf("expected local mode by default, got cloud")
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected allowed origins ['*'], got %v", cfg.AllowedOrigins)
	}
}

func TestConfigCustomPorts(t *testing.T) {
	os.Setenv("PORT", "9000")
	cfg := config.Load()
	if cfg.Port != "9000" {
		t.Errorf("expected port 9000 from PORT, got %s", cfg.Port)
	}

	// API_PORT should take precedence over PORT
	os.Setenv("API_PORT", "9090")
	cfg = config.Load()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090 from API_PORT precedence, got %s", cfg.Port)
	}
	os.Unsetenv("API_PORT")
	os.Unsetenv("PORT")
}

func TestConfigCloudDetection(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		host     string
		expected bool
	}{
		{"explicit cloud mode", "cloud", "http://127.0.0.1:7700", true},
		{"explicit local mode", "local", "https://ms-test.meilisearch.io", false},
		{"heuristic cloud domain", "", "https://ms-abc123.sfo.meilisearch.io", true},
		{"heuristic local 127.0.0.1", "", "http://127.0.0.1:7700", false},
		{"heuristic local localhost", "", "http://localhost:7700", false},
		{"heuristic local docker service", "", "http://meilisearch:7700", false},
		{"heuristic external remote ip", "", "http://198.51.100.1:7700", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("MEILI_MODE", tc.mode)
			os.Setenv("MEILI_HOST", tc.host)
			cfg := config.Load()
			if cfg.IsCloud != tc.expected {
				t.Errorf("for host=%s mode=%s: expected IsCloud=%v, got %v", tc.host, tc.mode, tc.expected, cfg.IsCloud)
			}
		})
	}
	os.Unsetenv("MEILI_MODE")
	os.Unsetenv("MEILI_HOST")
}

func TestConfigAllowedOrigins(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://app.geomilli.com, http://localhost:3000 , https://preview.example.com")
	cfg := config.Load()
	if len(cfg.AllowedOrigins) != 3 {
		t.Fatalf("expected 3 origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://app.geomilli.com" ||
		cfg.AllowedOrigins[1] != "http://localhost:3000" ||
		cfg.AllowedOrigins[2] != "https://preview.example.com" {
		t.Errorf("unexpected origins parsed: %v", cfg.AllowedOrigins)
	}
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
}
