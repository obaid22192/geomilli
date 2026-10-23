# Geomilli API (`geomilli-api`)

High-performance, sub-10ms OpenStreetMap geospatial search, autocomplete, reverse geocoding, and POI discovery REST API powered by **Meilisearch** and written in Go 1.25.

---

## Features

- ⚡ **Sub-10ms Autosuggest**: Prefix autocomplete with localized multilingual names and geo-proximity sorting.
- 🔍 **Full-Text & Faceted Search**: Multi-class filters, dynamic facet distributions, sorting, and pagination.
- 📍 **Geospatial Intelligence**:
  - **Reverse Geocoding**: Resolves nearest place, enclosing admin hierarchy (`Neighborhood` → `City` → `Region` → `Country`).
  - **Nearby POIs**: Radius search (`_geoRadius`) sorted by distance in meters and popularity score.
  - **Bounding Box**: Viewport query (`_geoBoundingBox`) for map markers and interactive clustering.
- 🛫 **Aviation & Airport Lookup**: Dedicated search and lookups by IATA (`BRU`) or ICAO (`EBBR`) codes.
- 🏛️ **Administrative Hierarchy**: Countries, provinces/regions, and cities ranked by population and popularity.
- 🏷️ **Dynamic Taxonomy**: Schema definitions for 19 classes, tags, and multilingual support.
- 📊 **Zero-Dependency Ring-Buffer Telemetry**:
  - p50, p95, p99 latency percentiles
  - Active QPS sliding window
  - Top search queries and zero-hit queries
  - Meilisearch health (`local` vs `cloud`) and Go runtime memory stats
- 🔄 **Importer Control Proxy**: Direct relay for `geomilli-harvester` status, country catalog, job dispatch, and live SSE event streams.

---

## Architecture

```text
                  ┌───────────────────────────────┐
                  │    HTTP Clients / Studio UI    │
                  └───────────────┬───────────────┘
                                  │ :8080
                                  ▼
               ┌─────────────────────────────────────┐
               │            geomilli-api             │
               │  - Chi Router & CORS Middleware     │
               │  - Ring-Buffer Telemetry Collector  │
               └──────────┬────────────────┬─────────┘
                          │                │
            Meilisearch   │                │ Harvester Proxy
             REST Client  │                │ Client (:8085)
                          ▼                ▼
            ┌───────────────────┐    ┌────────────────────┐
            │ Meilisearch Engine│    │ geomilli-harvester │
            │ (Local or Cloud)  │    │ (Daemon & Jobs)    │
            └───────────────────┘    └────────────────────┘
```

---

## Configuration

All configuration is driven by environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for the HTTP server to listen on. |
| `MEILI_HOST` | `http://127.0.0.1:7700` | Base URL for Meilisearch instance (Local or Meilisearch Cloud). |
| `MEILI_API_KEY` | `""` | Master or search API key for Meilisearch. |
| `MEILI_INDEX` | `places` | Name of the primary places search index. |
| `MEILI_MODE` | `""` | Optional override: `local` or `cloud`. Auto-detected by host if blank. |
| `HARVESTER_URL` | `http://127.0.0.1:8085` | Address of `geomilli-harvester` daemon control API. |
| `CORS_ALLOWED_ORIGINS`| `*` | Comma-separated list of allowed CORS origins. |

---

## API Endpoints Reference

### 1. Autocomplete (`GET /v1/autocomplete`)
Fast prefix suggestions with proximity boost and multilingual support.
```bash
curl "http://localhost:8080/v1/autocomplete?q=bru&lang=fr&country=BE&lat=50.85&lng=4.35&limit=5"
```
**Parameters**:
- `q`: Search prefix query.
- `lang`: Language code (`en`, `fr`, `nl`, `de`, etc.) to return localized `name_display`.
- `country`: Country ISO code filter (e.g. `BE`).
- `class`: Taxonomy class filter (e.g. `City`, `Airport`, `Hotel`).
- `subclass`: Subclass filter.
- `lat`, `lng`: Coordinates for proximity sort (`_geoPoint:asc`).
- `limit`: Maximum results (default `10`, max `100`).

---

### 2. Full-Text & Faceted Search (`GET /v1/search`)
Search across entities with dynamic facet distributions and filtering.
```bash
curl "http://localhost:8080/v1/search?q=ghent&facets=*&limit=10"
```
**Parameters**:
- `q`: Query string.
- `facets`: Comma-separated list of facet fields or `*` (`class,subclass,country_code,cuisine,star_rating,accommodation_type`).
- `filter`: Meilisearch filter string (e.g. `class = "Restaurant" AND cuisine = "italian"`).
- `sort`: Comma-separated sort expressions (default `popularity_score:desc`).
- `offset`: Pagination offset (default `0`).
- `limit`: Results per page (default `20`, max `100`).

---

### 3. Reverse Geocoding (`GET /v1/reverse`)
Resolves latitude and longitude coordinates to the closest landmark and enclosing administrative hierarchy.
```bash
curl "http://localhost:8080/v1/reverse?lat=50.8467&lng=4.3525"
```
**Parameters**:
- `lat`, `lng`: Latitude and longitude (required).
- `radius_meters`: Search radius in meters (default `50000`).

---

### 4. Nearby POIs (`GET /v1/nearby`)
Finds places within a given radius, sorted by distance in meters.
```bash
curl "http://localhost:8080/v1/nearby?lat=50.8503&lng=4.3517&radius_meters=3000&class=Restaurant"
```
**Parameters**:
- `lat`, `lng`: Coordinate center point (required).
- `radius_meters`: Radius in meters (default `5000`, max `100000`).
- `class`: Optional taxonomy class filter.
- `subclass`: Optional subclass filter.
- `cuisine`: Optional cuisine filter.
- `min_stars`: Minimum hotel star rating.
- `limit`: Max results (default `20`).
- `offset`: Pagination offset.

---

### 5. Bounding Box Viewport (`GET /v1/bbox`)
Fetches places within a map bounding box for marker clustering.
```bash
curl "http://localhost:8080/v1/bbox?min_lat=50.80&min_lng=4.30&max_lat=50.90&max_lng=4.40"
```
**Parameters**:
- `min_lat`, `min_lng`, `max_lat`, `max_lng`: Bounding coordinates (required).
- `class`: Optional class filter.
- `limit`: Max results (default `100`, max `500`).

---

### 6. Place & Entity Details (`GET /v1/places/{id}`)
Lookup by Meilisearch ID, OSM entity ID, or Wikidata QID:
```bash
curl "http://localhost:8080/v1/places/node:123?include_nearby=true"
curl "http://localhost:8080/v1/places/wikidata/Q31"
```
**Parameters**:
- `include_nearby`: `true` to fetch POIs within 2km of this location.

---

### 7. Aviation / Airports (`GET /v1/airports`)
```bash
curl "http://localhost:8080/v1/airports?q=Brussels"
curl "http://localhost:8080/v1/airports/BRU"
curl "http://localhost:8080/v1/airports/EBBR"
```

---

### 8. Administrative Hierarchy (`GET /v1/countries`)
```bash
curl "http://localhost:8080/v1/countries"
curl "http://localhost:8080/v1/countries/BE/regions"
curl "http://localhost:8080/v1/countries/BE/cities?limit=25"
```

---

### 9. Taxonomy & Schema Metadata (`GET /v1/taxonomy`)
```bash
curl "http://localhost:8080/v1/taxonomy"
curl "http://localhost:8080/v1/languages"
```

---

### 10. Observability & Harvester Proxy
```bash
curl "http://localhost:8080/v1/health"
curl "http://localhost:8080/v1/metrics"
curl "http://localhost:8080/v1/openapi.json"
curl "http://localhost:8080/v1/importer/status"
curl "http://localhost:8080/v1/importer/countries"
curl -N "http://localhost:8080/v1/importer/events"
```

---

## Development

```bash
# Run unit tests with race detection
make test

# Build production binary
make build

# Run server locally
make run

# Build Docker container
make docker-build
```
