package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/travayo/auto-complete/internal/envload"
	"github.com/travayo/auto-complete/internal/meili"
	"github.com/travayo/auto-complete/internal/runstate"
)

func main() {
	if err := runIndexer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runIndexer() error {
	envload.Load()

	var indexUID string
	var inputPath string
	var inputGlob string
	var host string
	var apiKey string
	var batchSize int
	var maxInFlight int
	var blueGreen bool
	var lockPath string
	var manifestPath string
	var validateFile string
	var validateLimit int
	var allowEmpty bool

	flag.StringVar(&indexUID, "index", getenv("DEFAULT_INDEX_NAME", "places"), "Meilisearch index name")
	flag.StringVar(&inputPath, "input", getenv("DEFAULT_INDEX_INPUT", ""), "Optional input NDJSON path for direct single-file indexing")
	flag.StringVar(&inputGlob, "input-glob", getenv("DEFAULT_INDEX_GLOB", "staging/places_??.ndjson"), "Glob for NDJSON inputs (e.g. staging/places_??.ndjson)")
	flag.StringVar(&host, "meili-host", getenv("MEILI_HOST", "http://127.0.0.1:7700"), "Meilisearch host")
	flag.StringVar(&apiKey, "meili-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	flag.IntVar(&batchSize, "batch-size", getenvInt("INDEX_BATCH_SIZE", 2000), "Upload batch size")
	flag.IntVar(&maxInFlight, "max-in-flight", getenvInt("INDEX_MAX_IN_FLIGHT", 2), "Max concurrent in-flight indexing tasks")
	flag.BoolVar(&blueGreen, "blue-green", true, "Use shadow index and swap")
	flag.StringVar(&lockPath, "lock-file", "data/locks/index.lock", "File lock path")
	flag.StringVar(&manifestPath, "manifest", "data/manifests/index.json", "Manifest output path")
	flag.StringVar(&validateFile, "validate-file", "", "Golden query validation file path")
	flag.IntVar(&validateLimit, "validate-limit", 10, "Top hits to inspect for each golden query")
	flag.BoolVar(&allowEmpty, "allow-empty", false, "Allow zero indexed documents without failing")
	flag.Parse()

	lock, err := runstate.Acquire(lockPath)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	indexer := meili.New(meili.Config{Host: host, APIKey: apiKey})
	targetIndex := indexUID
	if blueGreen {
		targetIndex = fmt.Sprintf("%s__build_%s", indexUID, time.Now().UTC().Format("20060102150405"))
	}

	defer func() {
		if blueGreen && targetIndex != indexUID {
			fmt.Printf("Cleaning up temporary build index: %s\n", targetIndex)
			if err := indexer.DeleteIndex(targetIndex); err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete build index %s: %v\n", targetIndex, err)
			}
		}
	}()

	if err := indexer.EnsureIndex(targetIndex); err != nil {
		return fmt.Errorf("ensure target index: %w", err)
	}

	if err := indexer.ApplyPlaceSettings(targetIndex); err != nil {
		return fmt.Errorf("apply place settings: %w", err)
	}

	inputLabel := inputPath
	inputHash := ""
	if inputGlob != "" {
		inputLabel = inputGlob
		inputHash, err = runstate.GlobSHA256(inputGlob)
		if err != nil {
			return fmt.Errorf("hash inputs: %w", err)
		}
	} else if inputPath != "" {
		inputHash, err = runstate.FileSHA256(inputPath)
		if err != nil {
			return fmt.Errorf("hash input: %w", err)
		}
	} else {
		return fmt.Errorf("either --input-glob or --input is required")
	}

	var submitted int
	if inputGlob != "" {
		submitted, err = indexer.StreamJSONLGlob(targetIndex, inputGlob, batchSize, maxInFlight)
	} else {
		submitted, err = indexer.StreamJSONL(targetIndex, inputPath, batchSize, maxInFlight)
	}
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}
	if submitted == 0 {
		if !allowEmpty {
			return fmt.Errorf("validation failed: zero documents submitted")
		}
		if err := runstate.WriteManifest(manifestPath, runstate.Manifest{
			Job:         "index",
			InputPath:   inputLabel,
			InputSHA256: inputHash,
			OutputPath:  targetIndex,
			Rows:        0,
		}); err != nil {
			return fmt.Errorf("write manifest: %w", err)
		}
		fmt.Printf("submitted 0 documents for %s; skipping validation and swap (allow-empty=true)\n", indexUID)
		return nil
	}

	docCount, err := indexer.DocumentCount(targetIndex)
	if err != nil {
		return fmt.Errorf("validate index stats: %w", err)
	}
	if docCount == 0 {
		return fmt.Errorf("validation failed: zero documents indexed")
	}

	if validateFile == "" {
		validateFile = "config/golden_queries_places.txt"
	}
	if checks, err := readGoldenQueries(validateFile); err != nil {
		fmt.Fprintf(os.Stderr, "warning: read validation file %s: %v (skipping query checks)\n", validateFile, err)
	} else if len(checks) > 0 {
		if err := indexer.ValidateGoldenQueries(targetIndex, checks, validateLimit); err != nil {
			fmt.Fprintf(os.Stderr, "warning: golden query check warning: %v (proceeding with swap since %d docs indexed)\n", err, docCount)
		} else {
			fmt.Printf("golden queries validated successfully (%d checks passed)\n", len(checks))
		}
	}

	if blueGreen {
		if err := indexer.EnsureIndex(indexUID); err != nil {
			return fmt.Errorf("ensure live index: %w", err)
		}
		if err := indexer.SwapIndexes(indexUID, targetIndex); err != nil {
			return fmt.Errorf("swap indexes: %w", err)
		}
	}

	if err := runstate.WriteManifest(manifestPath, runstate.Manifest{
		Job:         "index",
		InputPath:   inputLabel,
		InputSHA256: inputHash,
		OutputPath:  targetIndex,
		Rows:        int(docCount),
	}); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	fmt.Printf("indexed %d documents into %s (live=%s)\n", docCount, targetIndex, indexUID)
	return nil
}

func getenv(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
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

func readGoldenQueries(path string) ([]meili.QueryExpectation, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	checks := []meili.QueryExpectation{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		q := strings.TrimSpace(parts[0])
		if q == "" {
			continue
		}
		tokens := []string{}
		for _, tok := range strings.Split(parts[1], ",") {
			t := strings.TrimSpace(tok)
			if t != "" {
				tokens = append(tokens, t)
			}
		}
		checks = append(checks, meili.QueryExpectation{Query: q, ExpectedTokens: tokens})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return checks, nil
}
