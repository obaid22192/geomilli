package meili

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ms "github.com/meilisearch/meilisearch-go"
)

type Config struct {
	Host   string
	APIKey string
}

type Indexer struct {
	client ms.ServiceManager
	host   string
	apiKey string
}

func New(cfg Config) *Indexer {
	host := strings.TrimRight(cfg.Host, "/")
	return &Indexer{client: ms.New(host, ms.WithAPIKey(cfg.APIKey)), host: host, apiKey: cfg.APIKey}
}

func (i *Indexer) EnsureIndex(indexUID string) error {
	body := map[string]string{"uid": indexUID, "primaryKey": "id"}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, i.host+"/indexes", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(bodyBytes), "index_already_exists") {
		return nil
	}
	return fmt.Errorf("ensure index %s: status=%d body=%s", indexUID, resp.StatusCode, string(bodyBytes))
}

func (i *Indexer) ApplyPlaceSettings(indexUID string) error {
	idx := i.client.Index(indexUID)
	_, err := idx.UpdateSettings(&ms.Settings{
		SearchableAttributes: []string{
			"name_display",
			"name_alt_active",
			"iata_code",
			"icao_code",
			"class",
			"subclass",
			"admin_hierarchy",
			"country_name",
		},
		FilterableAttributes: []string{
			"class",
			"subclass",
			"country_code",
			"admin_level",
			"_geo",
			"star_rating",
			"cuisine",
			"iata_code",
			"icao_code",
			"accommodation_type",
		},
		SortableAttributes: []string{"popularity_score", "_geo"},
		RankingRules: []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
	})
	return err
}

func (i *Indexer) StreamJSONL(indexUID string, inputPath string, batchSize int, maxInFlight int) (int, error) {
	if batchSize <= 0 {
		batchSize = 2000
	}
	if maxInFlight <= 0 {
		maxInFlight = 2
	}
	idx := i.client.Index(indexUID)
	return i.streamJSONLFiles(idx, []string{inputPath}, batchSize, maxInFlight, 1)
}

func (i *Indexer) StreamJSONLGlob(indexUID string, inputGlob string, batchSize int, maxInFlight int) (int, error) {
	if batchSize <= 0 {
		batchSize = 2000
	}
	if maxInFlight <= 0 {
		maxInFlight = 2
	}

	matches, err := filepath.Glob(inputGlob)
	if err != nil {
		return 0, fmt.Errorf("invalid input glob %q: %w", inputGlob, err)
	}
	if len(matches) == 0 {
		fmt.Printf("no files matched input glob %s\n", inputGlob)
		return 0, nil
	}

	sort.Strings(matches)
	validFiles := make([]string, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			return 0, err
		}
		if info.IsDir() {
			continue
		}
		validFiles = append(validFiles, path)
	}

	if len(validFiles) == 0 {
		fmt.Printf("no regular files matched input glob %s\n", inputGlob)
		return 0, nil
	}

	idx := i.client.Index(indexUID)
	// Bounded worker pool: default 2 concurrent files, 256KB buffer per file
	concurrentFiles := 2
	total, err := i.streamJSONLFiles(idx, validFiles, batchSize, maxInFlight, concurrentFiles)
	if err != nil {
		return 0, err
	}

	fmt.Printf("submitted %d docs from glob %s to index %s\n", total, inputGlob, indexUID)
	return total, nil
}

func (i *Indexer) streamJSONLFiles(idx ms.IndexManager, filePaths []string, batchSize int, maxInFlight int, concurrentFiles int) (int, error) {
	if concurrentFiles <= 0 {
		concurrentFiles = 1
	}
	if concurrentFiles > len(filePaths) {
		concurrentFiles = len(filePaths)
	}
	if maxInFlight <= 0 {
		maxInFlight = 2
	}
	if batchSize <= 0 {
		batchSize = 2000
	}

	docChan := make(chan map[string]interface{}, batchSize*2)
	errChan := make(chan error, len(filePaths))
	pathChan := make(chan string, len(filePaths))

	for _, p := range filePaths {
		pathChan <- p
	}
	close(pathChan)

	var readerWg sync.WaitGroup
	var totalMalformed atomic.Int64

	// Bounded reader pool: at most concurrentFiles (default 2) open simultaneously
	for w := 0; w < concurrentFiles; w++ {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()

			// Reusable 256 KB buffer per worker (instead of unbounded 16 MB per file)
			scanBuf := make([]byte, 0, 64*1024)

			for path := range pathChan {
				f, err := os.Open(path)
				if err != nil {
					errChan <- fmt.Errorf("open %s: %w", path, err)
					return
				}

				scanner := bufio.NewScanner(f)
				scanner.Buffer(scanBuf, 256*1024)
				lineNo := 0
				fileMalformed := 0

				for scanner.Scan() {
					lineNo++
					line := scanner.Bytes()
					if len(line) == 0 {
						continue
					}
					var doc map[string]interface{}
					if err := json.Unmarshal(line, &doc); err != nil {
						fileMalformed++
						totalMalformed.Add(1)
						fmt.Fprintf(os.Stderr, "[index] warning: skipping malformed JSON line %s:%d: %v\n", path, lineNo, err)
						continue
					}
					docChan <- doc
				}

				scanErr := scanner.Err()
				_ = f.Close() // Close file immediately to free file descriptor and memory

				if scanErr != nil {
					errChan <- fmt.Errorf("read %s: %w", path, scanErr)
					return
				}
				if fileMalformed > 0 {
					fmt.Fprintf(os.Stderr, "[index] warning: skipped %d malformed lines from %s\n", fileMalformed, path)
				}
			}
		}()
	}

	// Close docChan once all readers complete
	go func() {
		readerWg.Wait()
		close(docChan)
		close(errChan)
	}()

	inFlight := make(chan int64, maxInFlight)
	total := 0

	submitBatch := func(b []map[string]interface{}) error {
		if len(b) == 0 {
			return nil
		}

		if len(inFlight) >= maxInFlight {
			oldTaskUID := <-inFlight
			if err := i.waitForTask(oldTaskUID, 5*time.Minute); err != nil {
				return err
			}
		}

		task, err := idx.AddDocuments(b)
		if err != nil {
			return err
		}
		if task != nil {
			inFlight <- task.TaskUID
		}
		total += len(b)
		return nil
	}

	// Main batching loop
	batch := make([]map[string]interface{}, 0, batchSize)
	for doc := range docChan {
		batch = append(batch, doc)
		if len(batch) >= batchSize {
			bCopy := make([]map[string]interface{}, len(batch))
			copy(bCopy, batch)
			batch = batch[:0]
			if err := submitBatch(bCopy); err != nil {
				return 0, err
			}
		}
	}

	// Flush remaining documents
	if len(batch) > 0 {
		if err := submitBatch(batch); err != nil {
			return 0, err
		}
	}

	// Drain remaining in-flight tasks
	close(inFlight)
	for taskUID := range inFlight {
		if err := i.waitForTask(taskUID, 5*time.Minute); err != nil {
			return 0, err
		}
	}

	// Check for any reader errors
	for err := range errChan {
		if err != nil {
			return 0, err
		}
	}

	malformedCount := totalMalformed.Load()
	if malformedCount > 0 {
		fmt.Fprintf(os.Stderr, "[index] warning: skipped %d malformed lines across all input files\n", malformedCount)
	}

	return total, nil
}

func (i *Indexer) waitForTask(taskUID int64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, message, err := i.taskStatus(taskUID)
		if err != nil {
			return err
		}
		switch status {
		case "succeeded":
			return nil
		case "failed", "canceled":
			if message != "" {
				return fmt.Errorf("task %d %s: %s", taskUID, status, message)
			}
			return fmt.Errorf("task %d %s", taskUID, status)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("task %d timeout after %s (last status=%s)", taskUID, timeout, status)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (i *Indexer) taskStatus(taskUID int64) (status string, message string, err error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/tasks/%d", i.host, taskUID), nil)
	if err != nil {
		return "", "", err
	}
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("task %d status=%d body=%s", taskUID, resp.StatusCode, string(body))
	}

	var payload struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}

	msg := strings.TrimSpace(payload.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(payload.Error.Code)
	}
	return payload.Status, msg, nil
}

func (i *Indexer) DocumentCount(indexUID string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, i.host+"/indexes/"+indexUID+"/stats", nil)
	if err != nil {
		return 0, err
	}
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("stats for %s: status=%d body=%s", indexUID, resp.StatusCode, string(body))
	}
	var payload struct {
		NumberOfDocuments int64 `json:"numberOfDocuments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.NumberOfDocuments, nil
}

func (i *Indexer) SwapIndexes(a string, b string) error {
	body := []map[string][]string{{"indexes": []string{a, b}}}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, i.host+"/swap-indexes", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("swap indexes: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

func (i *Indexer) DeleteIndex(indexUID string) error {
	req, err := http.NewRequest(http.MethodDelete, i.host+"/indexes/"+indexUID, nil)
	if err != nil {
		return err
	}
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete index %s status=%d body=%s", indexUID, resp.StatusCode, string(body))
	}
	return nil
}
