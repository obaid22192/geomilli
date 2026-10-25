package config

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAPIURL         = "http://localhost:8080"
	DefaultTransport      = "stdio"
	DefaultHTTPAddr       = "127.0.0.1:8090"
	DefaultRequestTimeout = 10 * time.Second
	DefaultTaxonomyTTL    = 10 * time.Minute
)

// Config holds all runtime configuration for geomilli-mcp.
type Config struct {
	// Upstream Geomilli REST API URL (e.g., "http://localhost:8080").
	APIURL string

	// Upstream Geomilli API bearer token (forwarded as Authorization: Bearer).
	APIToken string

	// MCP transport mode: "stdio" or "http".
	Transport string

	// HTTP address to bind if Transport == "http" (e.g., "127.0.0.1:8090").
	HTTPAddr string

	// Bearer auth token required by incoming clients for the HTTP transport.
	AuthToken string

	// Allowed Origin header values for HTTP transport (DNS rebinding / CORS protection).
	AllowedOrigins []string

	// Whether to register Tier-2 operations tools.
	EnableOpsTools bool

	// Default language code for localized names if omitted by caller.
	DefaultLang string

	// Per-request timeout for upstream API calls.
	RequestTimeout time.Duration

	// Cache TTL for taxonomy and languages metadata.
	TaxonomyTTL time.Duration
}

// Load loads and validates configuration from environment variables and CLI arguments.
// Flags take precedence over environment variables, which take precedence over defaults.
func Load(args []string) (*Config, error) {
	cfg := &Config{
		APIURL:         getEnv("GEOMILLI_API_URL", DefaultAPIURL),
		APIToken:       getEnv("GEOMILLI_API_TOKEN", ""),
		Transport:      getEnv("MCP_TRANSPORT", DefaultTransport),
		HTTPAddr:       getEnv("MCP_HTTP_ADDR", DefaultHTTPAddr),
		AuthToken:      getEnv("MCP_AUTH_TOKEN", ""),
		AllowedOrigins: parseCommaSeparated(getEnv("MCP_ALLOWED_ORIGINS", "")),
		EnableOpsTools: getEnvBool("MCP_ENABLE_OPS_TOOLS", false),
		DefaultLang:    getEnv("MCP_DEFAULT_LANG", ""),
		RequestTimeout: getEnvDuration("MCP_REQUEST_TIMEOUT", DefaultRequestTimeout),
		TaxonomyTTL:    getEnvDuration("MCP_TAXONOMY_TTL", DefaultTaxonomyTTL),
	}

	fs := flag.NewFlagSet("geomilli-mcp", flag.ContinueOnError)

	var (
		apiURLFlag         string
		apiTokenFlag       string
		transportFlag      string
		httpAddrFlag       string
		enableOpsToolsFlag bool
		langFlag           string
	)

	fs.StringVar(&apiURLFlag, "api-url", "", "Upstream Geomilli REST API URL")
	fs.StringVar(&apiTokenFlag, "api-token", "", "Upstream Geomilli API bearer token")
	fs.StringVar(&transportFlag, "transport", "", "MCP transport (stdio or http)")
	fs.StringVar(&httpAddrFlag, "http-addr", "", "HTTP listen address when transport=http")
	fs.BoolVar(&enableOpsToolsFlag, "enable-ops-tools", false, "Enable operations tools")
	fs.StringVar(&langFlag, "lang", "", "Default language code for localized names")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if apiURLFlag != "" {
		cfg.APIURL = apiURLFlag
	}
	if apiTokenFlag != "" {
		cfg.APIToken = apiTokenFlag
	}
	if transportFlag != "" {
		cfg.Transport = transportFlag
	}
	if httpAddrFlag != "" {
		cfg.HTTPAddr = httpAddrFlag
	}
	// fs.BoolVar sets enableOpsToolsFlag to true only if passed explicitly
	// We check if the flag was visited:
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "enable-ops-tools" {
			cfg.EnableOpsTools = enableOpsToolsFlag
		}
	})
	if langFlag != "" {
		cfg.DefaultLang = langFlag
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that the configuration values are well-formed and meet security constraints.
func (c *Config) Validate() error {
	c.APIURL = strings.TrimRight(c.APIURL, "/")
	parsedURL, err := url.Parse(c.APIURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("invalid GEOMILLI_API_URL: must be a valid HTTP/HTTPS URL, got %q", c.APIURL)
	}

	c.Transport = strings.ToLower(strings.TrimSpace(c.Transport))
	if c.Transport != "stdio" && c.Transport != "http" {
		return fmt.Errorf("invalid MCP_TRANSPORT: must be 'stdio' or 'http', got %q", c.Transport)
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("invalid MCP_REQUEST_TIMEOUT: must be greater than 0, got %v", c.RequestTimeout)
	}

	if c.TaxonomyTTL <= 0 {
		return fmt.Errorf("invalid MCP_TAXONOMY_TTL: must be greater than 0, got %v", c.TaxonomyTTL)
	}

	if c.Transport == "http" {
		host, _, err := net.SplitHostPort(c.HTTPAddr)
		if err != nil {
			// In case only a port or address without port was given
			return fmt.Errorf("invalid MCP_HTTP_ADDR: must be host:port, got %q: %w", c.HTTPAddr, err)
		}

		if !IsLoopbackHost(host) && strings.TrimSpace(c.AuthToken) == "" {
			return fmt.Errorf("MCP_AUTH_TOKEN is required when binding HTTP transport to a non-loopback address (%s)", c.HTTPAddr)
		}
	}

	return nil
}

// IsLoopbackHost returns true if the host is a known loopback hostname or IP address.
func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		d, err := time.ParseDuration(val)
		if err == nil {
			return d
		}
	}
	return defaultVal
}

func parseCommaSeparated(val string) []string {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
