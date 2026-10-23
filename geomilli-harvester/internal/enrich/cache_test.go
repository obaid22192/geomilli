package enrich_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/travayo/auto-complete/internal/enrich"
)

func TestCacheWALAndConcurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "geomilli_cache_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_cache.sqlite")
	cache, err := enrich.OpenCache(dbPath)
	if err != nil {
		t.Fatalf("OpenCache failed: %v", err)
	}
	defer cache.Close()

	// 1. Verify PRAGMA journal_mode is WAL
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open raw db for pragma check: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("failed to read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
	}

	// 2. Test Put and Get
	rec := enrich.CacheRecord{
		QID:       "Q12345",
		Sitelinks: 42,
		EnTitle:   "Grand_Place",
		Pageviews: 150000,
		UpdatedAt: time.Now().UTC(),
	}
	if err := cache.Put(rec); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, found, err := cache.Get("Q12345")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("expected record to be found")
	}
	if got.QID != rec.QID || got.Sitelinks != rec.Sitelinks || got.EnTitle != rec.EnTitle || got.Pageviews != rec.Pageviews {
		t.Errorf("got record mismatch: %+v vs %+v", got, rec)
	}

	// 3. Test Count
	count, err := cache.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// 4. Test IsFresh
	if !enrich.IsFresh(got, 30) {
		t.Errorf("record created just now should be fresh")
	}

	staleRec := enrich.CacheRecord{
		UpdatedAt: time.Now().AddDate(0, 0, -40),
	}
	if enrich.IsFresh(staleRec, 30) {
		t.Errorf("record 40 days old should not be fresh with 30d TTL")
	}

	// 5. Concurrent Writes without SQLITE_BUSY
	const numGoroutines = 50
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wRec := enrich.CacheRecord{
				QID:       fmt.Sprintf("Q%06d", workerID+1000),
				Sitelinks: workerID * 2,
				EnTitle:   fmt.Sprintf("Place_%d", workerID),
				Pageviews: workerID * 100,
				UpdatedAt: time.Now().UTC(),
			}
			if err := cache.Put(wRec); err != nil {
				errCh <- fmt.Errorf("worker %d Put failed: %w", workerID, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for wErr := range errCh {
		t.Errorf("concurrent write error: %v", wErr)
	}

	finalCount, err := cache.Count()
	if err != nil {
		t.Fatalf("final Count failed: %v", err)
	}
	if finalCount != int64(numGoroutines+1) {
		t.Errorf("expected %d entries, got %d", numGoroutines+1, finalCount)
	}
}
