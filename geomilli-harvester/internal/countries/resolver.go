package countries

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

type Country struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Continent     string `json:"continent"`
	GeofabrikPath string `json:"geofabrik_path"`
}

type configFile struct {
	Countries []struct {
		Code          string `yaml:"code"`
		GeofabrikPath string `yaml:"geofabrik_path"`
		Name          string `yaml:"name"`
		Continent     string `yaml:"continent"`
	} `yaml:"countries"`
}

func Resolve(configPath string, geofabrikBase string) ([]Country, error) {
	if countries, ok, err := fromConfig(configPath); err != nil {
		return nil, err
	} else if ok {
		return countries, nil
	}
	return fromGeofabrikIndex(geofabrikBase)
}

func fromConfig(path string) ([]Country, bool, error) {
	if path == "" {
		return nil, false, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var cfg configFile
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, false, err
	}
	if len(cfg.Countries) == 0 {
		return nil, false, nil
	}
	out := make([]Country, 0, len(cfg.Countries))
	for _, c := range cfg.Countries {
		code := strings.ToUpper(strings.TrimSpace(c.Code))
		path := strings.TrimSpace(c.GeofabrikPath)
		if code == "" || path == "" {
			continue
		}
		name := strings.TrimSpace(c.Name)
		if name == "" {
			name = titleWords(strings.ReplaceAll(lastSegment(path), "-", " "))
		}
		continent := strings.TrimSpace(c.Continent)
		if continent == "" {
			continent = deriveContinent(path)
		}
		out = append(out, Country{Code: code, Name: name, Continent: continent, GeofabrikPath: path})
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, true, nil
}

func fromGeofabrikIndex(base string) ([]Country, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "https://download.geofabrik.de"
	}
	url := base + "/index-v1-nogeom.json"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("geofabrik index status: %d", resp.StatusCode)
	}
	var payload struct {
		Features []struct {
			Properties struct {
				Name string            `json:"name"`
				ISO2 []string          `json:"iso3166-1:alpha2"`
				URLs map[string]string `json:"urls"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	seen := map[string]Country{}
	for _, f := range payload.Features {
		if len(f.Properties.ISO2) == 0 {
			continue
		}
		pbfURL := strings.TrimSpace(f.Properties.URLs["pbf"])
		if pbfURL == "" {
			continue
		}
		geopath, ok := pathFromPBF(base, pbfURL)
		if !ok {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(f.Properties.ISO2[0]))
		if len(code) != 2 {
			continue
		}
		name := strings.TrimSpace(f.Properties.Name)
		if name == "" {
			name = titleWords(strings.ReplaceAll(lastSegment(geopath), "-", " "))
		}
		if prev, exists := seen[code]; exists {
			if len(geopath) < len(prev.GeofabrikPath) {
				seen[code] = Country{Code: code, Name: name, GeofabrikPath: geopath}
			}
			continue
		}
		seen[code] = Country{Code: code, Name: name, GeofabrikPath: geopath}
	}
	out := make([]Country, 0, len(seen))
	for _, c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func pathFromPBF(base, pbfURL string) (string, bool) {
	prefix := strings.TrimRight(base, "/") + "/"
	if !strings.HasPrefix(pbfURL, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(pbfURL, prefix)
	rel = strings.TrimSuffix(rel, "-latest.osm.pbf")
	rel = strings.TrimSuffix(rel, ".osm.pbf")
	rel = strings.TrimSuffix(rel, "/")
	if rel == "" {
		return "", false
	}
	return rel, true
}

func lastSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func titleWords(s string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	for i, p := range parts {
		r := []rune(p)
		if len(r) == 0 {
			continue
		}
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

func deriveContinent(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasPrefix(lower, "europe/"):
		return "Europe"
	case strings.HasPrefix(lower, "asia/"):
		return "Asia"
	case strings.HasPrefix(lower, "north-america/"):
		return "North America"
	case strings.HasPrefix(lower, "south-america/"):
		return "South America"
	case strings.HasPrefix(lower, "africa/"):
		return "Africa"
	case strings.HasPrefix(lower, "australia-oceania/") || strings.HasPrefix(lower, "oceania/"):
		return "Oceania"
	case strings.HasPrefix(lower, "central-america/"):
		return "Central America & Caribbean"
	case strings.HasPrefix(lower, "antarctica"):
		return "Antarctica"
	default:
		return "Other"
	}
}
