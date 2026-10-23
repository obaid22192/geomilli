# Geomilli Studio 🌍

**Geomilli Studio** is the official web UI, interactive map explorer, and operations dashboard for the **Geomilli** open-source geospatial search and OpenStreetMap intelligence platform.

Built with **React 18**, **TypeScript**, **Vite**, **Tailwind CSS**, and **MapLibre GL JS**.

---

## Features

### 1. Interactive API Playground & Map Explorer
- **6 Geospatial Exploration Modes**:
  - `Autocomplete`: Instant sub-10ms prefix search with geo-proximity boosting.
  - `Search & Facets`: Full-text faceted discovery with class, subclass, and attribute filtering.
  - `Nearby POIs`: Circular radial radius searches (`_geoRadius`) around custom coordinates.
  - `Reverse Geocode`: Resolve latitude and longitude into street, neighborhood, city, and country hierarchies.
  - `Bounding Box Viewport`: Live viewport querying (`_geoBoundingBox`) synced with pan and zoom.
  - `Airports API`: Dedicated search for commercial airports by IATA or ICAO codes.
- **Interactive MapLibre GL Map**:
  - Dark-mode cartographic styling via vector/raster tiles (CartoDB Dark Matter / OSM) with zero API keys required.
  - Interactive pin dropping for reverse geocoding and radius calculations.
  - Rich tooltips and markers for search results with auto-framing bounds.
- **Taxonomy & Attribute Filters**:
  - 19 OSM taxonomy classes (City, Airport, Hotel, Restaurant, TrainStation, Museum, Tourism, etc.).
  - Cuisine types for dining and star ratings for hotels.
  - Multilingual localized names (`en`, `fr`, `nl`, `de`, `es`).
- **Developer Inspector**:
  - Live cURL command generator (clickable copy).
  - Raw JSON response viewer.
  - Dual HTTP latency and Meilisearch engine query duration badges.

### 2. Importer Control Center
- **Live Visual Pipeline Flow**: Real-time stage visualizer tracing `Fetch PBF` → `Tag Filter` → `Stream Transform` → `Wikidata Enrich` → `Blue/Green Swap`.
- **SSE Stream Progress**: Real-time progress percentage, document counts, and country stage status.
- **Country Manager**: Searchable catalog of Geofabrik country extracts (defaults to Belgium `BE`) with online Wikidata enrichment toggle.
- **Resource Efficiency Gauges**: Live telemetry of Go Heap RAM (demonstrating low $O(1)$ memory streaming under 50MB), Disk Staging usage, and SQLite WAL cache entries.
- **Live Terminal Log Stream**: Dark monospace terminal streaming real-time harvester log output.

### 3. System Metrics & Usage Dashboard
- **Meilisearch Connection Status**: Instant detection of Local Container vs Meilisearch Cloud, active index health, document count, and disk footprint.
- **Real-Time API Telemetry**: Total request volume, throughput (QPS), and latency percentiles ($p50$, $p95$, $p99$).
- **Visual Analytics**:
  - Endpoint traffic distribution bars.
  - Top popular searched queries.
  - Zero-hit query radar (for discovering synonym and taxonomy coverage gaps).
  - Document distribution breakdown by Taxonomy Class and Country code.

---

## Quickstart

### Local Development

```bash
# 1. Install dependencies
npm install

# 2. Run Vite development server (proxies /v1 to http://localhost:8080)
npm run dev
```

Visit [http://localhost:3000](http://localhost:3000) in your browser.

### Production Build

```bash
# Type check and build static assets into dist/
npm run build
```

### Docker Production Container

```bash
# Build multi-stage Nginx container
docker build -t geomilli-studio .

# Run container on port 3000
docker run -p 3000:3000 geomilli-studio
```

---

## Configuration

| Environment Variable | Default | Purpose |
| :--- | :--- | :--- |
| `VITE_API_BASE_URL` | `/v1` | Root URL path for all Geomilli API requests. |
| `VITE_API_PROXY_TARGET` | `http://localhost:8080` | Target URL used by Vite dev server to proxy `/v1/*` calls. |
