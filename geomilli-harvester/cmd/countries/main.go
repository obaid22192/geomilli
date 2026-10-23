package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/travayo/auto-complete/internal/countries"
	"github.com/travayo/auto-complete/internal/envload"
)

func main() {
	envload.Load()

	var configPath string
	var baseURL string
	var filterCountries string
	flag.StringVar(&configPath, "config", "config/countries.yaml", "Optional countries config file path")
	flag.StringVar(&baseURL, "base-url", getenv("OSM_GEOFABRIK_BASE_URL", "https://download.geofabrik.de"), "Geofabrik base URL")
	flag.StringVar(&filterCountries, "countries", getenv("COUNTRIES", ""), "Optional comma-separated list of country codes to filter")
	flag.Parse()

	resolved, err := countries.Resolve(configPath, baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve countries: %v\n", err)
		os.Exit(1)
	}

	var filterMap map[string]bool
	if filterCountries != "" {
		filterMap = make(map[string]bool)
		for _, part := range strings.Split(filterCountries, ",") {
			part = strings.ToUpper(strings.TrimSpace(part))
			if part != "" {
				filterMap[part] = true
			}
		}
	}

	for _, c := range resolved {
		if filterMap != nil && !filterMap[c.Code] {
			continue
		}
		fmt.Printf("%s|%s|%s\n", c.Code, c.Name, c.GeofabrikPath)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

