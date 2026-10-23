package config

import (
	"os"
	"strings"
)

// Config holds runtime configuration for geomilli-api.
type Config struct {
	Port           string
	MeiliHost      string
	MeiliAPIKey    string
	MeiliIndex     string
	HarvesterURL   string
	IsCloud        bool
	AllowedOrigins []string
}

// Load loads configuration from environment variables with defaults.
func Load() *Config {
	port := getEnv("API_PORT", getEnv("PORT", "8080"))
	meiliHost := getEnv("MEILI_HOST", "http://127.0.0.1:7700")
	meiliAPIKey := getEnv("MEILI_API_KEY", "")
	meiliIndex := getEnv("MEILI_INDEX", "places")
	harvesterURL := getEnv("HARVESTER_URL", "http://127.0.0.1:8085")

	// Detect if Meili is Cloud or Local
	mode := strings.ToLower(os.Getenv("MEILI_MODE"))
	isCloud := false
	if mode == "cloud" {
		isCloud = true
	} else if mode == "local" {
		isCloud = false
	} else {
		// Heuristic detection based on host URL
		isCloud = strings.Contains(meiliHost, "meilisearch.io") ||
			(!strings.Contains(meiliHost, "localhost") &&
				!strings.Contains(meiliHost, "127.0.0.1") &&
				!strings.Contains(meiliHost, "meilisearch:") &&
				!strings.HasPrefix(meiliHost, "http://meilisearch"))
	}

	originsStr := getEnv("CORS_ALLOWED_ORIGINS", "*")
	var origins []string
	for _, o := range strings.Split(originsStr, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	return &Config{
		Port:           port,
		MeiliHost:      strings.TrimRight(meiliHost, "/"),
		MeiliAPIKey:    meiliAPIKey,
		MeiliIndex:     meiliIndex,
		HarvesterURL:   strings.TrimRight(harvesterURL, "/"),
		IsCloud:        isCloud,
		AllowedOrigins: origins,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
