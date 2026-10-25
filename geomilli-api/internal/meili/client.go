package meili

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/schema"
)

// Client interacts with the Meilisearch HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	indexName  string
	isCloud    bool
	httpClient *http.Client
}

// NewClient initializes a new Meilisearch client.
func NewClient(cfg *config.Config) *Client {
	return NewClientWithHTTPClient(cfg, &http.Client{
		Timeout: 10 * time.Second,
	})
}

// NewClientWithHTTPClient initializes a client with a custom http.Client (e.g. for testing).
func NewClientWithHTTPClient(cfg *config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    cfg.MeiliHost,
		apiKey:     cfg.MeiliAPIKey,
		indexName:  cfg.MeiliIndex,
		isCloud:    cfg.IsCloud,
		httpClient: httpClient,
	}
}

// SearchRequest holds parameters for Meilisearch /indexes/{index}/search.
type SearchRequest struct {
	Q                     string   `json:"q"`
	Filter                any      `json:"filter,omitempty"` // string or []string
	Sort                  []string `json:"sort,omitempty"`
	Facets                []string `json:"facets,omitempty"`
	Limit                 int      `json:"limit,omitempty"`
	Offset                int      `json:"offset,omitempty"`
	AttributesToHighlight []string `json:"attributesToHighlight,omitempty"`
}

// SearchResponse models the standard Meilisearch search response.
type SearchResponse struct {
	Hits               []json.RawMessage         `json:"hits"`
	Query              string                    `json:"query"`
	ProcessingTimeMs   int64                     `json:"processingTimeMs"`
	Limit              int                       `json:"limit"`
	Offset             int                       `json:"offset"`
	EstimatedTotalHits int64                     `json:"estimatedTotalHits"`
	TotalHits          int64                     `json:"totalHits"`
	FacetDistribution  map[string]map[string]int `json:"facetDistribution,omitempty"`
}

// PlacesSearchResult holds typed Place objects.
type PlacesSearchResult struct {
	Hits              []schema.Place            `json:"hits"`
	Query             string                    `json:"query"`
	ProcessingTimeMs  int64                     `json:"processing_time_ms"`
	Limit             int                       `json:"limit"`
	Offset            int                       `json:"offset"`
	TotalHits         int64                     `json:"total_hits"`
	FacetDistribution map[string]map[string]int `json:"facet_distribution,omitempty"`
}

// IndexStats models Meilisearch index stats.
type IndexStats struct {
	NumberOfDocuments int64            `json:"numberOfDocuments"`
	IsIndexing        bool             `json:"isIndexing"`
	FieldDistribution map[string]int64 `json:"fieldDistribution"`
}

// HealthStatus models /health response.
type HealthStatus struct {
	Status string `json:"status"`
}

// VersionInfo models /version response.
type VersionInfo struct {
	PkgVersion string `json:"pkgVersion"`
}

// Search executes a raw search on the configured index.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	url := fmt.Sprintf("%s/indexes/%s/search", c.baseURL, c.indexName)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("meilisearch search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "index_not_found") {
			// Index has not been created or populated yet; treat gracefully as zero hits.
			return &SearchResponse{
				Hits:               []json.RawMessage{},
				Query:              req.Q,
				ProcessingTimeMs:   0,
				Limit:              req.Limit,
				Offset:             req.Offset,
				EstimatedTotalHits: 0,
				TotalHits:          0,
				FacetDistribution:  map[string]map[string]int{},
			}, nil
		}
		return nil, fmt.Errorf("meilisearch returned status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("meilisearch returned status %d: %s", resp.StatusCode, string(body))
	}

	var res SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &res, nil
}

// SearchPlaces searches and unmarshals hits into []schema.Place.
func (c *Client) SearchPlaces(ctx context.Context, req SearchRequest) (*PlacesSearchResult, error) {
	raw, err := c.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	total := raw.EstimatedTotalHits
	if raw.TotalHits > 0 {
		total = raw.TotalHits
	}

	places := make([]schema.Place, 0, len(raw.Hits))
	for _, hitBytes := range raw.Hits {
		var p schema.Place
		if err := json.Unmarshal(hitBytes, &p); err == nil {
			places = append(places, p)
		}
	}

	return &PlacesSearchResult{
		Hits:              places,
		Query:             raw.Query,
		ProcessingTimeMs:  raw.ProcessingTimeMs,
		Limit:             raw.Limit,
		Offset:            raw.Offset,
		TotalHits:         total,
		FacetDistribution: raw.FacetDistribution,
	}, nil
}

// GetDocument retrieves a single document by its Meilisearch ID.
func (c *Client) GetDocument(ctx context.Context, id string) (*schema.Place, error) {
	url := fmt.Sprintf("%s/indexes/%s/documents/%s", c.baseURL, c.indexName, id)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get document failed with status %d: %s", resp.StatusCode, string(body))
	}

	var place schema.Place
	if err := json.NewDecoder(resp.Body).Decode(&place); err != nil {
		return nil, err
	}
	return &place, nil
}

// FindOneByFilter executes a filtered search with limit=1 to find an exact match (e.g. wikidata_id, entity_id).
func (c *Client) FindOneByFilter(ctx context.Context, filter string) (*schema.Place, error) {
	res, err := c.SearchPlaces(ctx, SearchRequest{
		Filter: filter,
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Hits) == 0 {
		return nil, nil
	}
	return &res.Hits[0], nil
}

// GetStats returns document count and indexing stats.
func (c *Client) GetStats(ctx context.Context) (*IndexStats, error) {
	url := fmt.Sprintf("%s/indexes/%s/stats", c.baseURL, c.indexName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "index_not_found") {
			return &IndexStats{
				NumberOfDocuments: 0,
				IsIndexing:        false,
				FieldDistribution: map[string]int64{},
			}, nil
		}
		return nil, fmt.Errorf("stats failed with status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stats failed with status %d: %s", resp.StatusCode, string(body))
	}

	var stats IndexStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// EnsureIndex creates the index with primaryKey="id" if it doesn't already exist.
func (c *Client) EnsureIndex(ctx context.Context) error {
	url := fmt.Sprintf("%s/indexes", c.baseURL)
	payload := map[string]string{"uid": c.indexName, "primaryKey": "id"}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "index_already_exists") {
		return nil
	}
	return fmt.Errorf("ensure index status %d: %s", resp.StatusCode, string(body))
}

// Health checks if Meilisearch is reachable and healthy.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HealthStatus{Status: "unhealthy"}, fmt.Errorf("health returned status %d", resp.StatusCode)
	}

	var status HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return &HealthStatus{Status: "available"}, nil
	}
	return &status, nil
}

// Version returns Meilisearch version.
func (c *Client) Version(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/version", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ver VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&ver); err != nil {
		return "", err
	}
	return ver.PkgVersion, nil
}

// IsCloud returns true if client is connected to Meilisearch Cloud.
func (c *Client) IsCloud() bool {
	return c.isCloud
}

// BaseURL returns the Meilisearch base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// IndexName returns the configured index name.
func (c *Client) IndexName() string {
	return c.indexName
}

// HaversineDistance calculates the great-circle distance between two points in meters.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	cVal := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * cVal
}
