package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/travayo/auto-complete/internal/enrich"
	"github.com/travayo/auto-complete/internal/envload"
	"github.com/travayo/auto-complete/internal/pipeline"
	"github.com/travayo/auto-complete/internal/schema"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

func main() {
	envload.Load()

	var inputPath string
	var outputPath string
	var cacheDB string
	var online bool
	var ttlDays int
	var qps float64
	var taxonomyPath string
	var workers int
	flag.StringVar(&inputPath, "input", "staging/places_osm_xx.ndjson", "Input NDJSON path (typically per-country debug input)")
	flag.StringVar(&outputPath, "output", "staging/places_xx.ndjson", "Output NDJSON path (typically per-country debug output)")
	flag.StringVar(&cacheDB, "cache-db", getenv("ENRICH_CACHE_DB", "data/cache/enrichment.sqlite"), "SQLite enrichment cache path")
	flag.BoolVar(&online, "online", os.Getenv("ENRICH_ONLINE") == "true", "Allow outbound Wikidata/Wikipedia requests for cache misses")
	flag.IntVar(&ttlDays, "ttl-days", getenvInt("ENRICH_CACHE_TTL_DAYS", 30), "Cache TTL in days")
	flag.Float64Var(&qps, "qps", getenvFloat("ENRICH_QPS", 3.0), "Max outbound request rate per second")
	flag.StringVar(&taxonomyPath, "taxonomy", "config/taxonomy.yaml", "Path to taxonomy YAML config")
	flag.IntVar(&workers, "workers", getenvInt("ENRICH_WORKERS", 8), "Number of concurrent enrichment workers")
	flag.Parse()

	tax, err := taxonomy.LoadFromFile(taxonomyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load taxonomy: %v\n", err)
		os.Exit(1)
	}

	cache, err := enrich.OpenCache(cacheDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open cache: %v\n", err)
		os.Exit(1)
	}
	defer cache.Close()

	client := enrich.NewClient(15*time.Second, qps)

	in, err := os.Open(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer in.Close()

	outFile, outWriter, err := pipeline.NewJSONLWriter(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open output: %v\n", err)
		os.Exit(1)
	}
	defer pipeline.MustClose(outFile)

	fmt.Printf("[enrich-places] starting enrichment: input=%s output=%s cache=%s online=%t workers=%d qps=%.2f\n", inputPath, outputPath, cacheDB, online, workers, qps)

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	var cacheHits atomic.Int64
	var cacheMisses atomic.Int64
	var networkCalls atomic.Int64
	var consecutiveErrors atomic.Int32
	var onlineSafe atomic.Bool
	onlineSafe.Store(online)

	if workers <= 0 {
		workers = 1
	}

	// Bounded channels: O(workers) memory footprint
	jobChan := make(chan schema.Place, workers*2)
	resChan := make(chan schema.Place, workers*2)

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for place := range jobChan {
				sitelinks := 0
				pageviews := 0
				if place.WikidataID != "" {
					rec, found, err := cache.Get(place.WikidataID)
					if err != nil {
						fmt.Fprintf(os.Stderr, "cache read warning (%s): %v\n", place.WikidataID, err)
					}
					if found && enrich.IsFresh(rec, ttlDays) {
						cacheHits.Add(1)
						sitelinks = rec.Sitelinks
						pageviews = rec.Pageviews
					} else {
						cacheMisses.Add(1)
						if onlineSafe.Load() {
							networkCalls.Add(1)
							s, title, err := client.FetchWikidataSummary(place.WikidataID)
							if err != nil {
								fmt.Fprintf(os.Stderr, "wikidata warning (%s): %v\n", place.WikidataID, err)
								consecutive := consecutiveErrors.Add(1)
								if consecutive >= 5 {
									if onlineSafe.CompareAndSwap(true, false) {
										fmt.Fprintf(os.Stderr, "[enrich-places] warning: %d consecutive network errors. Disabling online enrichment to prevent hanging the pipeline.\n", consecutive)
									}
								}
							} else {
								consecutiveErrors.Store(0)
								sitelinks = s
								pv := 0
								if title != "" {
									networkCalls.Add(1)
									if got, err := client.FetchPageviews("en.wikipedia.org", title); err == nil {
										pv = got
									} else {
										fmt.Fprintf(os.Stderr, "pageviews warning (%s): %v\n", place.WikidataID, err)
									}
								}
								pageviews = pv
								_ = cache.Put(enrich.CacheRecord{
									QID:       place.WikidataID,
									Sitelinks: sitelinks,
									EnTitle:   title,
									Pageviews: pageviews,
									UpdatedAt: time.Now().UTC(),
								})
							}
						}
					}
				}

				place.PopularityScore = enrich.DerivePopularity(place, sitelinks, pageviews, tax)
				place.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

				resChan <- place
			}
		}()
	}

	// Producer goroutine
	go func() {
		defer close(jobChan)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var place schema.Place
			if err := json.Unmarshal(line, &place); err != nil {
				fmt.Fprintf(os.Stderr, "skip invalid json line: %v\n", err)
				continue
			}
			jobChan <- place
		}
	}()

	// Wait for workers and close results channel
	go func() {
		wg.Wait()
		close(resChan)
	}()

	// Bounded streaming writer (main goroutine) - direct streaming without unbounded reorder buffer
	writeCount := 0
	for place := range resChan {
		if err := pipeline.WriteJSONL(outWriter, place); err != nil {
			fmt.Fprintf(os.Stderr, "write output: %v\n", err)
			os.Exit(1)
		}
		writeCount++
		if writeCount%500 == 0 {
			fmt.Printf("[enrich-places] processed %d records (hits=%d, misses=%d, net=%d)\n",
				writeCount, cacheHits.Load(), cacheMisses.Load(), networkCalls.Load())
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan input: %v\n", err)
		os.Exit(1)
	}
	if err := outWriter.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("enriched %d place docs into %s\n", writeCount, outputPath)
	fmt.Printf("enrich stats: cache_hits=%d cache_misses=%d network_calls=%d online=%t ttl_days=%d\n",
		cacheHits.Load(), cacheMisses.Load(), networkCalls.Load(), onlineSafe.Load(), ttlDays)
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
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getenvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
