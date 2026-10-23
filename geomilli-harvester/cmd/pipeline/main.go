package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/travayo/auto-complete/internal/countries"
	"github.com/travayo/auto-complete/internal/envload"
)

type FetchManifest struct {
	CountryCode           string `json:"country_code"`
	AdminGeoJSONSeq       string `json:"admin_geojsonseq"`
	AdminGeoJSONSeqSHA256 string `json:"admin_geojsonseq_sha256"`
	PoisGeoJSONSeq        string `json:"pois_geojsonseq"`
	PoisGeoJSONSeqSHA256  string `json:"pois_geojsonseq_sha256"`
}

type BuildManifest struct {
	Job         string `json:"job"`
	CountryCode string `json:"country_code"`
	AdminSHA256 string `json:"admin_sha256"`
	PoisSHA256  string `json:"pois_sha256"`
	UpdatedAt   string `json:"updated_at"`
}

func main() {
	envload.Load()

	var stage string
	var target string
	var onlineEnrich bool
	var concurrency int

	flag.StringVar(&stage, "stage", "run", "Stage to execute: fetch, build-data, index, run, cleanup")
	flag.StringVar(&target, "target", "places", "Target dataset (places only)")
	flag.BoolVar(&onlineEnrich, "online-enrich", os.Getenv("ENRICH_ONLINE") == "true", "Enable online enrichment during build-data/run")
	flag.IntVar(&concurrency, "concurrency", getenvInt("PIPELINE_CONCURRENCY", 4), "Number of countries to build in parallel")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatalf("getwd: %v", err)
	}

	switch stage {
	case "fetch":
		runFetch(root)
	case "build-data":
		runPlacesBuild(root, onlineEnrich, concurrency)
	case "index":
		runPlacesIndex(root)
	case "run":
		runPlacesBuild(root, onlineEnrich, concurrency)
		runPlacesIndex(root)
		runCleanup(root)
		cleanAllData(root)
	case "cleanup":
		runCleanup(root)
	default:
		fatalf("unsupported stage %q", stage)
	}
}

func runFetch(root string) {
	if err := runCmd(root, "bash", "-lc", "./scripts/fetch_osm.sh"); err != nil {
		fatalf("fetch: %v", err)
	}
}

func runPlacesBuild(root string, onlineEnrich bool, concurrency int) {
	configPath := filepath.Join(root, "config", "countries.yaml")
	baseURL := os.Getenv("OSM_GEOFABRIK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://download.geofabrik.de"
	}
	resolved, err := countries.Resolve(configPath, baseURL)
	if err != nil {
		fatalf("resolve countries: %v", err)
	}

	var filterMap map[string]bool
	if filterCountries := os.Getenv("COUNTRIES"); filterCountries != "" {
		filterMap = make(map[string]bool)
		for _, part := range strings.Split(filterCountries, ",") {
			part = strings.ToUpper(strings.TrimSpace(part))
			if part != "" {
				filterMap[part] = true
			}
		}
	}

	if concurrency <= 0 {
		concurrency = 1
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var buildErrors []error

	for _, c := range resolved {
		if filterMap != nil && !filterMap[c.Code] {
			continue
		}

		wg.Add(1)
		go func(country countries.Country) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := processCountry(root, country, onlineEnrich)
			if err != nil {
				mu.Lock()
				buildErrors = append(buildErrors, fmt.Errorf("country %s: %w", country.Code, err))
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	if len(buildErrors) > 0 {
		fmt.Fprintf(os.Stderr, "[pipeline] build completed with %d error(s):\n", len(buildErrors))
		for _, err := range buildErrors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("[pipeline] per-country datasets are ready under staging/")
}

func processCountry(root string, c countries.Country, onlineEnrich bool) error {
	lcCode := strings.ToLower(c.Code)

	// 1. Fetch OSM data for this country only
	fmt.Printf("[pipeline] fetching OSM data for country %s (%s)...\n", c.Name, c.Code)
	if err := runCmd(root, "bash", "-lc", "./scripts/fetch_osm.sh "+strings.ToUpper(c.Code)); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	fetchManifestPath := filepath.Join(root, "data", "manifests", fmt.Sprintf("fetch-osm-%s.json", lcCode))
	fetchBytes, err := os.ReadFile(fetchManifestPath)
	if err != nil {
		fmt.Printf("[pipeline] fetch manifest missing for %s, processing unconditionally\n", c.Code)
		if err := buildCountry(root, c, onlineEnrich); err != nil {
			return err
		}
		cleanCountryRawData(root, c.Code)
		return nil
	}

	var fm FetchManifest
	if err := json.Unmarshal(fetchBytes, &fm); err != nil {
		fmt.Printf("[pipeline] warning: failed to parse fetch manifest for %s: %v\n", c.Code, err)
		if err := buildCountry(root, c, onlineEnrich); err != nil {
			return err
		}
		cleanCountryRawData(root, c.Code)
		return nil
	}

	buildManifestPath := filepath.Join(root, "data", "manifests", fmt.Sprintf("build-places-%s.json", lcCode))
	osmOut := filepath.Join(root, "staging", fmt.Sprintf("places_osm_%s.ndjson", lcCode))
	enrichedOut := filepath.Join(root, "staging", fmt.Sprintf("places_%s.ndjson", lcCode))

	buildBytes, err := os.ReadFile(buildManifestPath)
	upToDate := false
	if !onlineEnrich && err == nil {
		var bm BuildManifest
		if err := json.Unmarshal(buildBytes, &bm); err == nil {
			if bm.AdminSHA256 == fm.AdminGeoJSONSeqSHA256 &&
				bm.PoisSHA256 == fm.PoisGeoJSONSeqSHA256 &&
				fileExists(osmOut) && fileExists(enrichedOut) {
				upToDate = true
			}
		}
	}

	if upToDate {
		fmt.Printf("[pipeline] country %s build is up-to-date, skipping transform and enrich\n", c.Code)
		cleanCountryRawData(root, c.Code)
		return nil
	}

	if err := buildCountry(root, c, onlineEnrich); err != nil {
		return err
	}
	cleanCountryRawData(root, c.Code)

	bm := BuildManifest{
		Job:         "build-places",
		CountryCode: c.Code,
		AdminSHA256: fm.AdminGeoJSONSeqSHA256,
		PoisSHA256:  fm.PoisGeoJSONSeqSHA256,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if bmBytes, err := json.MarshalIndent(bm, "", "  "); err == nil {
		_ = os.WriteFile(buildManifestPath, bmBytes, 0o644)
	}
	return nil
}

func buildCountry(root string, c countries.Country, onlineEnrich bool) error {
	lcCode := strings.ToLower(c.Code)
	fmt.Printf("[pipeline] processing country %s (%s)...\n", c.Name, c.Code)

	osmOut := filepath.Join("staging", fmt.Sprintf("places_osm_%s.ndjson", lcCode))
	enrichedOut := filepath.Join("staging", fmt.Sprintf("places_%s.ndjson", lcCode))

	if err := runBinary(root, "transform-places",
		"--countries", fmt.Sprintf("%s:%s", c.Code, c.Name),
		"--output", osmOut,
	); err != nil {
		return err
	}

	enrichArgs := []string{"--input", osmOut, "--output", enrichedOut}
	if onlineEnrich {
		enrichArgs = append(enrichArgs, "--online=true")
	}
	return runBinary(root, "enrich-places", enrichArgs...)
}

func runPlacesIndex(root string) {
	_ = os.MkdirAll(filepath.Join(root, "data", "manifests"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "data", "locks"), 0o755)
	args := []string{
		"--index", "places",
		"--input-glob", "staging/places_??.ndjson",
		"--blue-green=true",
		"--validate-file", "config/golden_queries_places.txt",
		"--lock-file", "data/locks/index-places.lock",
		"--manifest", "data/manifests/index-places.json",
	}
	if err := runBinary(root, "index", args...); err != nil {
		fatalf("index: %v", err)
	}
}

func runCleanup(root string) {
	if err := runBinary(root, "cleanup", "--keep-builds", "3", "--keep-manifest-days", "14"); err != nil {
		fatalf("cleanup: %v", err)
	}
}

func runBinary(root, name string, args ...string) error {
	if bin := filepath.Join("/usr/local/bin", name); fileExists(bin) {
		return runCmd(root, bin, args...)
	}
	cmdArgs := append([]string{"run", "./cmd/" + name}, args...)
	return runCmd(root, "go", cmdArgs...)
}

func runCmd(root, name string, args ...string) error {
	fmt.Printf("[pipeline] %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return fallback
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func cleanCountryRawData(root, countryCode string) {
	lcCode := strings.ToLower(countryCode)
	files := []string{
		filepath.Join(root, "raw", "osm", lcCode+"-latest.osm.pbf"),
		filepath.Join(root, "raw", "osm", lcCode+"-latest.osm.pbf.etag"),
		filepath.Join(root, "raw", "osm", lcCode+"-admin.osm.pbf"),
		filepath.Join(root, "raw", "osm", lcCode+"-pois.osm.pbf"),
		filepath.Join(root, "raw", "osm", lcCode+"-admin.geojsonseq"),
		filepath.Join(root, "raw", "osm", lcCode+"-pois.geojsonseq"),
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			fmt.Printf("[pipeline] warning: failed to delete raw file %s: %v\n", f, err)
		}
	}
	fmt.Printf("[pipeline] cleaned up raw files for %s\n", countryCode)
}

// cleanAllData removes all raw OSM files and per-country staging files after
// indexing is complete. The data is now in Meilisearch so these intermediate
// files are no longer needed. The raw/osm directory itself is preserved (never
// deleted) to prevent VirtioFS inode tracking issues on macOS Docker.
func cleanAllData(root string) {
	// Clean raw/osm contents (but keep the directory itself).
	rawDir := filepath.Join(root, "raw", "osm")
	entries, err := os.ReadDir(rawDir)
	if err == nil {
		for _, e := range entries {
			p := filepath.Join(rawDir, e.Name())
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				fmt.Printf("[pipeline] warning: failed to delete %s: %v\n", p, err)
			}
		}
		fmt.Println("[pipeline] cleaned up raw/osm files")
	}

	// Clean per-country staging files (places_osm_XX.ndjson, places_XX.ndjson).
	stagingDir := filepath.Join(root, "staging")
	stagingEntries, err := os.ReadDir(stagingDir)
	if err == nil {
		for _, e := range stagingEntries {
			name := e.Name()
			// Remove per-country intermediate files.
			if strings.HasPrefix(name, "places_osm_") || strings.HasPrefix(name, "places_") {
				p := filepath.Join(stagingDir, name)
				if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
					fmt.Printf("[pipeline] warning: failed to delete %s: %v\n", p, err)
				}
			}
		}
		fmt.Println("[pipeline] cleaned up per-country staging files")
	}
}
