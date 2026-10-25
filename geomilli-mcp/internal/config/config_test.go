package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	// Clear relevant env vars
	envVars := []string{
		"GEOMILLI_API_URL", "GEOMILLI_API_TOKEN", "MCP_TRANSPORT",
		"MCP_HTTP_ADDR", "MCP_AUTH_TOKEN", "MCP_ALLOWED_ORIGINS",
		"MCP_ENABLE_OPS_TOOLS", "MCP_DEFAULT_LANG", "MCP_REQUEST_TIMEOUT",
		"MCP_TAXONOMY_TTL",
	}
	for _, k := range envVars {
		os.Unsetenv(k)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("unexpected error loading defaults: %v", err)
	}

	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("expected APIURL=%q, got %q", DefaultAPIURL, cfg.APIURL)
	}
	if cfg.Transport != DefaultTransport {
		t.Errorf("expected Transport=%q, got %q", DefaultTransport, cfg.Transport)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("expected HTTPAddr=%q, got %q", DefaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("expected RequestTimeout=%v, got %v", DefaultRequestTimeout, cfg.RequestTimeout)
	}
	if cfg.TaxonomyTTL != DefaultTaxonomyTTL {
		t.Errorf("expected TaxonomyTTL=%v, got %v", DefaultTaxonomyTTL, cfg.TaxonomyTTL)
	}
	if cfg.EnableOpsTools != false {
		t.Errorf("expected EnableOpsTools=false, got %v", cfg.EnableOpsTools)
	}
}

func TestConfigFlagsOverride(t *testing.T) {
	os.Setenv("GEOMILLI_API_URL", "http://env-api:8080")
	defer os.Unsetenv("GEOMILLI_API_URL")

	args := []string{
		"--api-url=http://flag-api:8080",
		"--api-token=secret-token",
		"--transport=http",
		"--http-addr=127.0.0.1:9090",
		"--enable-ops-tools",
		"--lang=fr",
	}

	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIURL != "http://flag-api:8080" {
		t.Errorf("expected flag override for APIURL, got %q", cfg.APIURL)
	}
	if cfg.APIToken != "secret-token" {
		t.Errorf("expected APIToken=secret-token, got %q", cfg.APIToken)
	}
	if cfg.Transport != "http" {
		t.Errorf("expected Transport=http, got %q", cfg.Transport)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Errorf("expected HTTPAddr=127.0.0.1:9090, got %q", cfg.HTTPAddr)
	}
	if !cfg.EnableOpsTools {
		t.Errorf("expected EnableOpsTools=true")
	}
	if cfg.DefaultLang != "fr" {
		t.Errorf("expected DefaultLang=fr, got %q", cfg.DefaultLang)
	}
}

func TestConfigValidation_NonLoopbackWithoutToken(t *testing.T) {
	nonLoopbacks := []string{
		"0.0.0.0:8090",
		"192.168.1.100:8090",
		"geomilli.example.com:8090",
	}

	for _, addr := range nonLoopbacks {
		cfg := &Config{
			APIURL:         "http://localhost:8080",
			Transport:      "http",
			HTTPAddr:       addr,
			AuthToken:      "", // Missing token!
			RequestTimeout: 5 * time.Second,
			TaxonomyTTL:    5 * time.Minute,
		}
		err := cfg.Validate()
		if err == nil {
			t.Errorf("expected error for non-loopback HTTPAddr %q without auth token, got nil", addr)
		}
	}

	// But WITH token, it should be valid
	cfg := &Config{
		APIURL:         "http://localhost:8080",
		Transport:      "http",
		HTTPAddr:       "0.0.0.0:8090",
		AuthToken:      "my-strong-token",
		RequestTimeout: 5 * time.Second,
		TaxonomyTTL:    5 * time.Minute,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error for non-loopback with auth token: %v", err)
	}
}

func TestConfigValidation_LoopbackWithoutTokenAllowed(t *testing.T) {
	loopbacks := []string{
		"127.0.0.1:8090",
		"localhost:8090",
		"[::1]:8090",
	}

	for _, addr := range loopbacks {
		cfg := &Config{
			APIURL:         "http://localhost:8080",
			Transport:      "http",
			HTTPAddr:       addr,
			AuthToken:      "",
			RequestTimeout: 5 * time.Second,
			TaxonomyTTL:    5 * time.Minute,
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected loopback %q without token to be valid, got: %v", addr, err)
		}
	}
}

func TestConfigValidation_InvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{
			name: "invalid api url",
			mutate: func(c *Config) {
				c.APIURL = "not-a-url"
			},
			wantErr: true,
		},
		{
			name: "ftp api url",
			mutate: func(c *Config) {
				c.APIURL = "ftp://localhost:8080"
			},
			wantErr: true,
		},
		{
			name: "invalid transport",
			mutate: func(c *Config) {
				c.Transport = "websocket"
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			mutate: func(c *Config) {
				c.RequestTimeout = 0
			},
			wantErr: true,
		},
		{
			name: "zero taxonomy ttl",
			mutate: func(c *Config) {
				c.TaxonomyTTL = 0
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				APIURL:         "http://localhost:8080",
				Transport:      "stdio",
				RequestTimeout: 5 * time.Second,
				TaxonomyTTL:    5 * time.Minute,
			}
			tc.mutate(cfg)
			err := cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
