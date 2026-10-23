package enrich

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type CacheRecord struct {
	QID       string
	Sitelinks int
	EnTitle   string
	Pageviews int
	UpdatedAt time.Time
}

type Cache struct {
	db *sql.DB
	mu sync.Mutex
}

func OpenCache(path string) (*Cache, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Configure SQLite pragmas for resilience and performance
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")
	_, _ = db.Exec("PRAGMA journal_mode = WAL;")

	c := &Cache{db: db}
	if err := c.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Cache) migrate() error {
	ddl := `
CREATE TABLE IF NOT EXISTS wikidata_cache (
  qid TEXT PRIMARY KEY,
  sitelinks INTEGER NOT NULL DEFAULT 0,
  en_title TEXT NOT NULL DEFAULT '',
  pageviews INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);`
	_, err := c.db.Exec(ddl)
	return err
}

func (c *Cache) Get(qid string) (CacheRecord, bool, error) {
	row := c.db.QueryRow(`SELECT qid, sitelinks, en_title, pageviews, updated_at FROM wikidata_cache WHERE qid = ?`, qid)
	var rec CacheRecord
	var updatedAt string
	err := row.Scan(&rec.QID, &rec.Sitelinks, &rec.EnTitle, &rec.Pageviews, &updatedAt)
	if err == sql.ErrNoRows {
		return CacheRecord{}, false, nil
	}
	if err != nil {
		return CacheRecord{}, false, err
	}
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return CacheRecord{}, false, fmt.Errorf("parse updated_at: %w", err)
	}
	rec.UpdatedAt = t
	return rec, true, nil
}

func (c *Cache) Put(rec CacheRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = time.Now().UTC()
	}
	_, err := c.db.Exec(
		`INSERT INTO wikidata_cache(qid, sitelinks, en_title, pageviews, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(qid) DO UPDATE SET
		   sitelinks=excluded.sitelinks,
		   en_title=excluded.en_title,
		   pageviews=excluded.pageviews,
		   updated_at=excluded.updated_at`,
		rec.QID,
		rec.Sitelinks,
		rec.EnTitle,
		rec.Pageviews,
		rec.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (c *Cache) Count() (int64, error) {
	if c == nil || c.db == nil {
		return 0, nil
	}
	var count int64
	row := c.db.QueryRow(`SELECT COUNT(*) FROM wikidata_cache`)
	err := row.Scan(&count)
	return count, err
}

func IsFresh(rec CacheRecord, ttlDays int) bool {
	if ttlDays <= 0 {
		return true
	}
	return time.Since(rec.UpdatedAt) < (time.Duration(ttlDays) * 24 * time.Hour)
}
