# 60-Second Quickstart

Get a production-grade geospatial search engine running locally in less than a minute. By default, Geomilli is pre-configured to ingest and index **Belgium (`BE`)** from Geofabrik, allowing you to test autocomplete, search, and reverse geocoding without large downloads.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose v2+
- 2 GB available RAM (Geomilli itself runs in < 100 MB RAM)
- Ports `3000` (Studio), `8080` (API), `8085` (Harvester), and `7700` (Meilisearch) available

---

## 1. Clone & Start the Stack

```bash
# Clone the umbrella repository
git clone https://github.com/geomilli/geomilli.git
cd geomilli

# Copy unified environment file (defaults to Belgium: COUNTRIES=BE)
cp .env.example .env

# Spin up all services with Docker Compose
docker compose up -d --build
```

The stack starts four services:

| Service | Port | Description |
| :--- | :--- | :--- |
| **`geomilli-studio`** | `http://localhost:3000` | Interactive Web UI, API Playground & Importer Dashboard |
| **`meilisearch-ui`** | `http://localhost:7701` | Official Meilisearch Web Management UI (`eyeix/meilisearch-ui`) |
| **`geomilli-api`** | `http://localhost:8080` | High-performance Go REST API |
| **`geomilli-harvester`** | `http://localhost:8085` | Streaming OSM ingestion daemon & SSE stream |
| **`meilisearch`** | `http://localhost:7700` | Fast search engine (Local container) |

---

## 2. Ingestion Status

On startup, `geomilli-harvester` detects if the `places` index exists. If empty, it automatically triggers the ingestion pipeline for Belgium (`BE`):

1. **Fetch**: Streams `belgium-latest.osm.pbf` from Geofabrik mirrors.
2. **Filter**: Osmium filters relevant OSM tags (places, boundaries, amenities).
3. **Transform**: Normalizes entities into 19 taxonomy classes with bounding boxes.
4. **Enrich**: Queries Wikidata & Wikipedia Pageviews (cached in SQLite WAL).
5. **Index**: Streams NDJSON batches into Meilisearch shadow index and performs an atomic swap.

You can monitor the live import progress in real time:

- Open **Geomilli Studio** in your browser at `http://localhost:3000` and click the **Importer** tab.
- Or watch the harvester logs:
  ```bash
  docker compose logs -f geomilli-harvester
  ```

---

## 3. Test Autocomplete via cURL

Once ingestion completes (typically 60–90 seconds for Belgium), query the API:

```bash
# Sub-10ms prefix autocomplete
curl -s "http://localhost:8080/v1/autocomplete?q=Brux" | jq .
```

Sample response:
```json
{
  "hits": [
    {
      "id": "relation_54094",
      "name": "Bruxelles - Brussel",
      "class": "City",
      "subclass": "city",
      "country_code": "BE",
      "names_by_language": {
        "en": "Brussels",
        "fr": "Bruxelles",
        "nl": "Brussel"
      },
      "_geo": {
        "lat": 50.8466,
        "lng": 4.3528
      },
      "popularity_score": 98.4
    }
  ],
  "processingTimeMs": 3,
  "query": "Brux"
}
```

---

## 4. Explore Geospatial APIs

Test reverse geocoding at the Grand Place in Brussels:

```bash
curl -s "http://localhost:8080/v1/reverse?lat=50.8467&lng=4.3525" | jq .
```

Find nearby restaurants within 500 meters:

```bash
curl -s "http://localhost:8080/v1/nearby?lat=50.8467&lng=4.3525&radius=500&class=Restaurant" | jq .
```

Look up Brussels Airport:

```bash
curl -s "http://localhost:8080/v1/airports/BRU" | jq .
```

---

## 5. Next Steps

- Learn about the [System Architecture](/guide/architecture).
- Understand our [O(1) Streaming & Memory](/guide/streaming-and-memory) design.
- Learn how to connect to [Meilisearch Cloud](/guide/meilisearch-local-vs-cloud).
- Explore the [Taxonomy & Supported Classes](/harvester/taxonomy-and-languages).
