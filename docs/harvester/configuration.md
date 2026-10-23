# Harvester Configuration

`geomilli-harvester` is the ingestion engine responsible for downloading OpenStreetMap data from Geofabrik, filtering tags via `osmium`, normalizing taxonomy entities, performing Wikidata enrichment, and indexing into Meilisearch.

---

## 1. Environment Variables

The harvester is configured entirely through environment variables or CLI flags:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `COUNTRIES` | `BE` | Comma-separated ISO alpha-2 country codes (e.g. `BE`, `BE,NL,LU`, or `ALL`). |
| `MEILI_HOST` | `http://127.0.0.1:7700` | Target Meilisearch server URL. |
| `MEILI_API_KEY` | `masterKey123` | Meilisearch Admin / Master API key. |
| `ONLINE_ENRICH` | `false` | Enable online Wikidata and Wikipedia Pageviews API queries. When `false`, uses offline heuristics. |
| `PORT` or `HARVESTER_PORT`| `8085` | HTTP and SSE control daemon port. |
| `DATA_DIR` | `./data` | Base directory for disk caching, SQLite WAL databases, and staging files. |
| `OSM_GEOFABRIK_BASE_URL` | `https://download.geofabrik.de` | Geofabrik mirror base URL. |
| `ENRICH_QPS` | `3` | Maximum rate-limited QPS for external Wikipedia API calls. |
| `ENRICH_CACHE_TTL_DAYS` | `30` | Retention window for SQLite enrichment cache. |

---

## 2. Country Selection & The Default (`BE`)

To ensure a friction-free local developer experience, Geomilli defaults to **Belgium (`BE`)** (`europe/belgium-latest.osm.pbf`):
- PBF Download Size: ~70 MB
- Filtered GeoJSON: ~180,000 places & amenities
- Processing Time: ~60–90 seconds on standard laptops

### Configuring Additional Countries
To index other countries, set `COUNTRIES` in `.env`:

```dotenv
# Ingest the Benelux union
COUNTRIES=BE,NL,LU

# Ingest all supported countries
# COUNTRIES=ALL
```

Country definitions and URL paths are resolved against `config/countries.yaml`.

---

## 3. Operational Modes: Daemon vs CLI

The harvester binary provides two operational patterns:

### A. Daemon Mode (`harvester serve`)
Used in Docker Compose and persistent container deployments. It boots a lightweight HTTP server on port `8085`:
- Listens for trigger jobs from `geomilli-studio` or `geomilli-api`.
- Emits real-time Server-Sent Events (SSE) tracking pipeline progress, records/sec, and memory footprint.
- Runs an initial import on startup if the `places` index does not exist.

### B. One-Shot CLI Mode (`harvester run`)
Ideal for CI/CD pipelines, Airflow tasks, or Kubernetes CronJobs:

```bash
# Ingest Belgium and exit
geomilli-harvester run --countries=BE --online-enrich=true

# Custom index name without swapping
geomilli-harvester run --countries=NL --index-name=places_nl
```

---

## 4. Harvester Control API

When running in daemon mode, the harvester exposes an internal HTTP API:

- `GET /api/v1/countries`: Returns supported country codes, their status (`indexed`, `cached`, `queued`), and file sizes.
- `POST /api/v1/jobs`: Triggers an asynchronous import job.
  ```json
  {
    "countries": ["BE", "LU"],
    "online_enrich": false,
    "mode": "append"
  }
  ```
- `GET /api/v1/status`: Returns current stage, progress percentage, records processed, and Go runtime memory stats (`HeapAlloc`, `Sys`, `NumGC`).
- `GET /api/v1/events`: Subscribes to real-time Server-Sent Events (SSE) for UI progress bars.
