package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/travayo/auto-complete/internal/countries"
	"github.com/travayo/auto-complete/internal/envload"
	"github.com/travayo/auto-complete/internal/geojsonseq"
	"github.com/travayo/auto-complete/internal/languages"
	"github.com/travayo/auto-complete/internal/osm"
	"github.com/travayo/auto-complete/internal/pipeline"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

type fileDef struct {
	countryCode  string
	countryName  string
	path         string
	tax          *taxonomy.Taxonomy
	enabledLangs map[string]bool
}

func main() {
	envload.Load()

	var inputDir string
	var outputPath string
	var countryPairs string
	var countryFile string
	var geofabrikBase string
	var taxonomyPath string
	var languagesPath string

	flag.StringVar(&inputDir, "input-dir", "raw/osm", "Directory containing *.geojsonseq files")
	flag.StringVar(&outputPath, "output", "staging/places_osm.ndjson", "Output NDJSON path")
	flag.StringVar(&countryPairs, "countries", "", "Optional comma-separated CODE:CountryName override")
	flag.StringVar(&countryFile, "countries-file", "config/countries.yaml", "Optional countries YAML path")
	flag.StringVar(&geofabrikBase, "geofabrik-base", getenv("OSM_GEOFABRIK_BASE_URL", "https://download.geofabrik.de"), "Geofabrik base URL used when countries file is missing")
	flag.StringVar(&taxonomyPath, "taxonomy", "config/taxonomy.yaml", "Path to taxonomy YAML config")
	flag.StringVar(&languagesPath, "languages", "config/languages.yaml", "Path to languages YAML config")
	flag.Parse()

	tax, err := taxonomy.LoadFromFile(taxonomyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load taxonomy: %v\n", err)
		os.Exit(1)
	}

	enabledLangs, err := languages.LoadFromFile(languagesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load languages: %v\n", err)
		os.Exit(1)
	}

	countryMap, err := resolveCountryNames(countryPairs, countryFile, geofabrikBase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve countries: %v\n", err)
		os.Exit(1)
	}

	defs := []fileDef{}
	for code, name := range countryMap {
		defs = append(defs, fileDef{countryCode: code, countryName: name, path: filepath.Join(inputDir, strings.ToLower(code)+"-admin.geojsonseq"), tax: tax, enabledLangs: enabledLangs})
		defs = append(defs, fileDef{countryCode: code, countryName: name, path: filepath.Join(inputDir, strings.ToLower(code)+"-pois.geojsonseq"), tax: tax, enabledLangs: enabledLangs})
	}

	file, writer, err := pipeline.NewJSONLWriter(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open output: %v\n", err)
		os.Exit(1)
	}
	defer pipeline.MustClose(file)

	var count int
	for _, def := range defs {
		if err := transformFile(def, writer, &count); err != nil {
			fmt.Fprintf(os.Stderr, "transform %s: %v\n", def.path, err)
			os.Exit(1)
		}
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d place documents to %s\n", count, outputPath)
}

func transformFile(def fileDef, writer *bufio.Writer, count *int) error {
	f, err := os.Open(def.path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "skip missing input: %s\n", def.path)
			return nil
		}
		return err
	}
	defer f.Close()

	r := geojsonseq.NewReader(f)
	for {
		feature, err := r.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		place, ok := osm.NormalizeFeature(def.countryCode, def.countryName, feature, def.tax, def.enabledLangs)
		if !ok {
			continue
		}
		if err := pipeline.WriteJSONL(writer, place); err != nil {
			return err
		}
		(*count)++
	}
	return nil
}

func parseCountries(raw string) map[string]string {
	out := map[string]string{}
	for _, chunk := range strings.Split(raw, ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.SplitN(chunk, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.ToUpper(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	return out
}

func resolveCountryNames(countryPairs string, countriesFile string, geofabrikBase string) (map[string]string, error) {
	overrides := parseCountries(countryPairs)
	if len(overrides) > 0 {
		return overrides, nil
	}

	resolved, err := countries.Resolve(countriesFile, geofabrikBase)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, c := range resolved {
		out[c.Code] = c.Name
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no countries resolved")
	}
	return out, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
