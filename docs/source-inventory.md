# Source Inventory

This document defines exactly what is fetched from our data source, what fields are consumed, and where records flow.

---

## Geofabrik OSM Extracts (PBF)

- **Source name**: Geofabrik country extracts
- **Purpose**: Unified places, airports, accommodations, and discovery POI backbone for autosuggest destinations
- **Fetch method**: `scripts/fetch_osm.sh` via `curl` from `https://download.geofabrik.de/{geofabrik_path}-latest.osm.pbf`
- **Fetch controls**: lock file + ETag check + manifest write. If local ETag matches and output files exist, download and osmium tags-filtering are skipped entirely.
- **Schedule**: Weekly recommended
- **Artifact format**: `.osm.pbf`, then filtered `.osm.pbf`, then `.geojsonseq`
- **Fields consumed**:
  - `name`, `name:*` (all translations)
  - `alt_name`, `loc_name`, `official_name`, `short_name`, `old_name`, `int_name`
  - `admin_level`, `boundary`, `place`
  - `railway`, `tourism`, `leisure`, `amenity`, `historic`, `natural`, `shop`
  - `population`, `wikidata`, `wikipedia`
  - `iata`, `icao` (airports)
  - `stars` (hotels)
  - `cuisine` (restaurants)
  - geometry and bbox
- **Entities produced**: 19 classes (Airport, Hotel, City, Region, Neighborhood, TrainStation, TouristAttraction, Restaurant, etc.)
- **Stable ID strategy**: `osm-{type}-{id}` from exported feature id
- **Refresh model**: Incremental per-country build with full index swap.
- **Downstream index target**: `places` index in Meilisearch
- **Known risks**:
  - admin-level semantics vary by country.
  - multilingual name coverage is uneven in low-profile regions.

---

## Wikidata & Wikipedia Pageviews Enrichment

- **Source name**: Wikidata API/SPARQL & Wikimedia pageviews API
- **Purpose**: Place enrichment (sitelinks count, labels, monthly Wikipedia pageviews) to derive a `popularity_score`
- **Fetch method**: API calls with local cache (`data/cache/enrichment.sqlite`)
- **Schedule**: Cache-first during country build-data runs. If a country's `.geojsonseq` checksums haven't changed, the entire enrichment is skipped for that country.
- **Fields consumed**:
  - Sitelinks count (Wikidata)
  - Monthly pageviews (Wikimedia REST API)
- **Downstream index target**: Mapped to `popularity_score` on `Place` documents.
