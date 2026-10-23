package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/travayo/auto-complete/internal/envload"
)

func main() {
	envload.Load()

	var host string
	var apiKey string
	var keepBuilds int
	var keepManifestDays int
	flag.StringVar(&host, "meili-host", getenv("MEILI_HOST", "http://127.0.0.1:7700"), "Meilisearch host")
	flag.StringVar(&apiKey, "meili-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	flag.IntVar(&keepBuilds, "keep-builds", 3, "Number of shadow build indexes to keep per base index")
	flag.IntVar(&keepManifestDays, "keep-manifest-days", 14, "Delete manifests older than this many days")
	flag.Parse()

	if err := cleanupManifests("data/manifests", keepManifestDays); err != nil {
		fmt.Fprintf(os.Stderr, "cleanup manifests: %v\n", err)
		os.Exit(1)
	}
	if err := cleanupBuildIndexes(host, apiKey, keepBuilds); err != nil {
		fmt.Fprintf(os.Stderr, "cleanup build indexes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("cleanup complete")
}

func cleanupManifests(dir string, keepDays int) error {
	if keepDays <= 0 {
		return nil
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			if remErr := os.Remove(path); remErr != nil {
				return remErr
			}
		}
		return nil
	})
}

func cleanupBuildIndexes(host string, apiKey string, keepBuilds int) error {
	if keepBuilds < 0 {
		keepBuilds = 0
	}
	host = strings.TrimRight(host, "/")
	req, err := http.NewRequest(http.MethodGet, host+"/indexes", nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("list indexes status=%d", resp.StatusCode)
	}
	var payload struct {
		Results []struct {
			UID string `json:"uid"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}

	grouped := map[string][]string{}
	for _, idx := range payload.Results {
		uid := idx.UID
		parts := strings.Split(uid, "__build_")
		if len(parts) != 2 {
			continue
		}
		base := parts[0]
		grouped[base] = append(grouped[base], uid)
	}

	for _, uids := range grouped {
		sort.Strings(uids)
		if len(uids) <= keepBuilds {
			continue
		}
		toDelete := uids[:len(uids)-keepBuilds]
		for _, uid := range toDelete {
			if err := deleteIndex(host, apiKey, uid); err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteIndex(host, apiKey, uid string) error {
	req, err := http.NewRequest(http.MethodDelete, host+"/indexes/"+uid, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delete index %s status=%d", uid, resp.StatusCode)
	}
	return nil
}

func getenv(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
