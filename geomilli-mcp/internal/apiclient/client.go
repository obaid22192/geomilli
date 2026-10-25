package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/version"
)

// APIError represents a structured error returned by the Geomilli REST API.
type APIError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error (%d %s): %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Message)
}

// Client is a typed client for the Geomilli REST API.
type Client struct {
	baseURL    string
	apiToken   string
	timeout    time.Duration
	httpClient *http.Client
	userAgent  string
}

// NewClient initializes a new Geomilli REST API client.
func NewClient(baseURL, apiToken string, timeout time.Duration, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}
	v := version.Version
	if v == "" {
		v = "dev"
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   strings.TrimSpace(apiToken),
		timeout:    timeout,
		httpClient: httpClient,
		userAgent:  fmt.Sprintf("geomilli-mcp/%s", v),
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	u := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Geomilli-Client", "mcp")
	req.Header.Set("Accept", "application/json")
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, respBytes)
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decoding JSON response: %w", err)
		}
	}

	return nil
}

type newEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}

type legacyEnvelope struct {
	Error   bool   `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func parseAPIError(statusCode int, body []byte) *APIError {
	// Try parsing new envelope {"error": {"code": "...", "message": "...", "status": 400}}
	var n newEnvelope
	if err := json.Unmarshal(body, &n); err == nil && (n.Error.Message != "" || n.Error.Code != "") {
		s := n.Error.Status
		if s == 0 {
			s = statusCode
		}
		return &APIError{
			Status:  s,
			Code:    n.Error.Code,
			Message: n.Error.Message,
		}
	}

	// Try parsing legacy envelope {"error": true, "status": 400, "message": "..."}
	var l legacyEnvelope
	if err := json.Unmarshal(body, &l); err == nil && l.Error && l.Message != "" {
		s := l.Status
		if s == 0 {
			s = statusCode
		}
		return &APIError{
			Status:  s,
			Message: l.Message,
		}
	}

	// Fallback to raw text
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(statusCode)
	} else if len(msg) > 256 {
		msg = msg[:256] + "..."
	}

	return &APIError{
		Status:  statusCode,
		Message: msg,
	}
}

// Autocomplete performs prefix matching suggestions with filters and proximity boost.
func (c *Client) Autocomplete(ctx context.Context, params AutocompleteParams) (*AutocompleteResponse, error) {
	q := url.Values{}
	if params.Query != "" {
		q.Set("q", params.Query)
	}
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}
	if params.Country != "" {
		q.Set("country", params.Country)
	}
	if params.Class != "" {
		q.Set("class", params.Class)
	}
	if params.Subclass != "" {
		q.Set("subclass", params.Subclass)
	}
	if params.Lat != nil && params.Lng != nil {
		q.Set("lat", strconv.FormatFloat(*params.Lat, 'f', -1, 64))
		q.Set("lng", strconv.FormatFloat(*params.Lng, 'f', -1, 64))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}

	var res AutocompleteResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/autocomplete", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Search performs full-text and faceted search queries.
func (c *Client) Search(ctx context.Context, params SearchParams) (*SearchResponse, error) {
	q := url.Values{}
	if params.Query != "" {
		q.Set("q", params.Query)
	}
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}
	if params.Filter != "" {
		q.Set("filter", params.Filter)
	}
	if params.Sort != "" {
		q.Set("sort", params.Sort)
	}
	if params.Facets != "" {
		q.Set("facets", params.Facets)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	var res SearchResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/search", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Reverse resolves latitude and longitude coordinates to address and administrative hierarchy.
func (c *Client) Reverse(ctx context.Context, params ReverseParams) (*ReverseResponse, error) {
	q := url.Values{}
	q.Set("lat", strconv.FormatFloat(params.Lat, 'f', -1, 64))
	q.Set("lng", strconv.FormatFloat(params.Lng, 'f', -1, 64))
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}
	if params.RadiusMeters != nil && *params.RadiusMeters > 0 {
		q.Set("radius_meters", strconv.Itoa(*params.RadiusMeters))
	}

	var res ReverseResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/reverse", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Nearby queries POIs and destinations within a geo-radius.
func (c *Client) Nearby(ctx context.Context, params NearbyParams) (*NearbyResponse, error) {
	q := url.Values{}
	q.Set("lat", strconv.FormatFloat(params.Lat, 'f', -1, 64))
	q.Set("lng", strconv.FormatFloat(params.Lng, 'f', -1, 64))
	if params.RadiusMeters > 0 {
		q.Set("radius_meters", strconv.Itoa(params.RadiusMeters))
	}
	if params.Class != "" {
		q.Set("class", params.Class)
	}
	if params.Subclass != "" {
		q.Set("subclass", params.Subclass)
	}
	if params.Cuisine != "" {
		q.Set("cuisine", params.Cuisine)
	}
	if params.MinStars > 0 {
		q.Set("min_stars", strconv.Itoa(params.MinStars))
	}
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", strconv.Itoa(params.Offset))
	}

	var res NearbyResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/nearby", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// BBox queries places within a bounding box viewport.
func (c *Client) BBox(ctx context.Context, params BBoxParams) (*BBoxResponse, error) {
	q := url.Values{}
	q.Set("min_lat", strconv.FormatFloat(params.MinLat, 'f', -1, 64))
	q.Set("min_lng", strconv.FormatFloat(params.MinLng, 'f', -1, 64))
	q.Set("max_lat", strconv.FormatFloat(params.MaxLat, 'f', -1, 64))
	q.Set("max_lng", strconv.FormatFloat(params.MaxLng, 'f', -1, 64))
	if params.Class != "" {
		q.Set("class", params.Class)
	}
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}

	var res BBoxResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/bbox", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPlaceByID retrieves place details by Meilisearch ID or OSM entity ID.
func (c *Client) GetPlaceByID(ctx context.Context, id, lang string, includeNearby bool) (*PlaceDetailsResponse, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}
	if includeNearby {
		q.Set("include_nearby", "true")
	}

	path := fmt.Sprintf("/v1/places/%s", url.PathEscape(id))
	var res PlaceDetailsResponse
	if err := c.doRequest(ctx, http.MethodGet, path, q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPlaceByWikidataID retrieves place details by Wikidata QID (e.g. Q207).
func (c *Client) GetPlaceByWikidataID(ctx context.Context, qid, lang string, includeNearby bool) (*PlaceDetailsResponse, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}
	if includeNearby {
		q.Set("include_nearby", "true")
	}

	path := fmt.Sprintf("/v1/places/wikidata/%s", url.PathEscape(qid))
	var res PlaceDetailsResponse
	if err := c.doRequest(ctx, http.MethodGet, path, q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SearchAirports searches airports by text query, country, IATA/ICAO code, or proximity.
func (c *Client) SearchAirports(ctx context.Context, params SearchAirportsParams) (*AirportsResponse, error) {
	q := url.Values{}
	if params.Query != "" {
		q.Set("q", params.Query)
	}
	if params.Country != "" {
		q.Set("country", params.Country)
	}
	if params.IATA != "" {
		q.Set("iata", params.IATA)
	}
	if params.ICAO != "" {
		q.Set("icao", params.ICAO)
	}
	if params.Code != "" {
		q.Set("code", params.Code)
	}
	if params.Lat != nil && params.Lng != nil {
		q.Set("lat", strconv.FormatFloat(*params.Lat, 'f', -1, 64))
		q.Set("lng", strconv.FormatFloat(*params.Lng, 'f', -1, 64))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Lang != "" {
		q.Set("lang", params.Lang)
	}

	var res AirportsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/airports", q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetAirportByCode retrieves an airport by its 3-letter IATA code or 4-letter ICAO code.
func (c *Client) GetAirportByCode(ctx context.Context, code, lang string) (*Place, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}

	path := fmt.Sprintf("/v1/airports/%s", url.PathEscape(code))
	var res Place
	if err := c.doRequest(ctx, http.MethodGet, path, q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ListCountries returns all indexed countries with entity counts.
func (c *Client) ListCountries(ctx context.Context) (*CountriesListResponse, error) {
	var res CountriesListResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/countries", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ListRegions returns regions/states for a given ISO country code.
func (c *Client) ListRegions(ctx context.Context, countryCode, lang string) (*RegionsListResponse, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}

	path := fmt.Sprintf("/v1/countries/%s/regions", url.PathEscape(countryCode))
	var res RegionsListResponse
	if err := c.doRequest(ctx, http.MethodGet, path, q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ListCities returns cities for a given ISO country code.
func (c *Client) ListCities(ctx context.Context, countryCode string, limit, offset int, lang string) (*CitiesListResponse, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	if lang != "" {
		q.Set("lang", lang)
	}

	path := fmt.Sprintf("/v1/countries/%s/cities", url.PathEscape(countryCode))
	var res CitiesListResponse
	if err := c.doRequest(ctx, http.MethodGet, path, q, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetTaxonomy returns taxonomy class definitions and match rules.
func (c *Client) GetTaxonomy(ctx context.Context) (*TaxonomyConfig, error) {
	var res TaxonomyConfig
	if err := c.doRequest(ctx, http.MethodGet, "/v1/taxonomy", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetLanguages returns supported languages configuration.
func (c *Client) GetLanguages(ctx context.Context) (*LanguagesConfig, error) {
	var res LanguagesConfig
	if err := c.doRequest(ctx, http.MethodGet, "/v1/languages", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetHealth returns the health status of the API service and connected Meilisearch.
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	var res HealthResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/health", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetImporterStatus proxies Harvester daemon status.
func (c *Client) GetImporterStatus(ctx context.Context) (*ImporterStatusResponse, error) {
	var res ImporterStatusResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/importer/status", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetImporterQueue proxies Harvester daemon active and queued tasks.
func (c *Client) GetImporterQueue(ctx context.Context) (*ImporterQueueResponse, error) {
	var res ImporterQueueResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v1/importer/queue", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetImporterCountries proxies Harvester available and indexed country list.
func (c *Client) GetImporterCountries(ctx context.Context) (*ImporterCountriesResponse, error) {
	// Harvester endpoint /v1/importer/countries returns either []CountryItem directly
	// or {"countries": [...]} depending on error/success wrapper.
	// We handle direct slice or object decoding:
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	u := fmt.Sprintf("%s/v1/importer/countries", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Geomilli-Client", "mcp")
	req.Header.Set("Accept", "application/json")
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBytes)
	}

	// Try unmarshaling as []CountryItem first
	var list []CountryItem
	if err := json.Unmarshal(respBytes, &list); err == nil {
		return &ImporterCountriesResponse{Countries: list}, nil
	}

	// Otherwise try unmarshaling as ImporterCountriesResponse object
	var obj ImporterCountriesResponse
	if err := json.Unmarshal(respBytes, &obj); err != nil {
		return nil, fmt.Errorf("decoding importer countries response: %w", err)
	}
	return &obj, nil
}

// GetMetrics returns real-time API telemetry snapshot.
func (c *Client) GetMetrics(ctx context.Context) (*TelemetrySnapshot, error) {
	var res TelemetrySnapshot
	if err := c.doRequest(ctx, http.MethodGet, "/v1/metrics", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
