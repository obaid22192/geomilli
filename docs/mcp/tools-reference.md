# MCP Tools Reference

The Geomilli MCP server provides **16 specialized tools** designed specifically for LLM and agentic workflows. Every tool uses structured parameters, provides built-in validation with actionable error messages, and returns token-efficient results.

---

## Tool Categories

| Category | Tools | Description |
| :--- | :--- | :--- |
| **[Geocoding & Location](#geocoding--location-lookups)** | `geocode`, `reverse_geocode` | Forward query to coordinates and reverse coordinate to address/place. |
| **[Spatial Discovery & POIs](#spatial-discovery--poi-search)** | `search_places`, `find_nearby`, `places_in_area`, `get_place` | Structured POI search, radial proximity queries, bounding boxes, and ID lookups. |
| **[Aviation Intelligence](#aviation--transit-intelligence)** | `find_airports`, `get_airport` | Search and inspect international and regional airports by IATA/ICAO code or proximity. |
| **[Administrative Hierarchy](#administrative-hierarchy)** | `list_countries`, `list_regions`, `list_cities` | Explore countries, provinces, and settlements ordered by population and popularity. |
| **[Reference & Utilities](#reference--utilities)** | `list_place_categories`, `measure_distance` | Inspect the 19 taxonomy classes or calculate exact Haversine great-circle distances. |
| **[Operations & Telemetry](#operations--monitoring-tier-2)** | `get_service_status`, `list_import_queue`, `list_importable_countries` | Inspect cluster health, harvester status, and ingestion pipelines (gated by `MCP_ENABLE_OPS_TOOLS`). |

---

## Tool Annotations Explained

In accordance with the Model Context Protocol specification, every Geomilli tool explicitly declares its behavioral hints:

* **`readOnlyHint`**: `true` indicates the tool performs read-only operations with no mutations or side-effects.
* **`destructiveHint`**: `true` indicates the operation permanently modifies or deletes data. All Tier-1 and read-only Tier-2 tools set this to `false` or omit it.
* **`idempotentHint`**: `true` indicates repeated calls with identical parameters yield identical states.
* **`openWorldHint`**: `true` indicates the tool searches dynamic, open-world geographic data where results depend on external data updates. `false` indicates a closed-world query (e.g. fixed metadata, taxonomy, or mathematical calculations).

---

## Geocoding & Location Lookups

### 1. `geocode`

Forward geocoding: resolves place names, addresses, landmarks, or natural language queries into candidate coordinates with enclosing administrative context and popularity ranking.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/autocomplete`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `query` | `string` | **Yes** | — | Search query, address, or place name (e.g. `"Atomium"`, `"Brussels"`, `"Eiffel Tower"`). |
| `country` | `string` | No | — | ISO 3166-1 alpha-2 country code to restrict candidates (e.g. `"BE"`). |
| `class` | `string` | No | — | Filter by taxonomy class (e.g. `"Attraction"`, `"City"`, `"Airport"`). |
| `near_lat` | `number` | No | — | Latitude coordinate for geo-proximity ranking boost (-90 to 90). |
| `near_lng` | `number` | No | — | Longitude coordinate for geo-proximity ranking boost (-180 to 180). |
| `lang` | `string` | No | — | Preferred ISO language code for localized display names (e.g. `"en"`, `"fr"`, `"nl"`). |
| `limit` | `integer` | No | `5` | Maximum candidates to return (1 to 10). |

#### Example Invocation

```json
{
  "query": "Grand Place",
  "country": "BE",
  "lang": "en",
  "limit": 2
}
```

#### Sample Response

```json
{
  "total_hits": 2,
  "places": [
    {
      "id": "osm_w157053075",
      "name": "Grand Place",
      "class": "Attraction",
      "subclass": "square",
      "lat": 50.8467,
      "lng": 4.3524,
      "country_code": "BE",
      "admin_hierarchy": ["Pentagone", "Brussels", "Brussels-Capital Region", "Belgium"],
      "wikidata_id": "Q215429",
      "popularity_score": 0.94
    }
  ]
}
```

---

### 2. `reverse_geocode`

Reverse geocoding: identifies the geographic address, hierarchical administrative jurisdiction, and closest known place for a coordinate pair.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/reverse`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `lat` | `number` | **Yes** | — | Latitude coordinate in decimal degrees (-90 to 90). |
| `lng` | `number` | **Yes** | — | Longitude coordinate in decimal degrees (-180 to 180). |
| `radius_meters` | `integer` | No | `50000` | Search radius in meters (1 to 100,000). Clamped automatically if exceeded. |
| `lang` | `string` | No | — | Preferred ISO language code for localized display names. |

#### Example Invocation

```json
{
  "lat": 50.8949,
  "lng": 4.3415,
  "radius_meters": 1000,
  "lang": "en"
}
```

#### Sample Response

```json
{
  "address": "Atomium, Square de l'Atomium, 1020 Brussels, Belgium",
  "admin_hierarchy": ["Laeken", "Brussels", "Brussels-Capital Region", "Belgium"],
  "radius_meters": 1000,
  "closest_place": {
    "id": "osm_n123456",
    "name": "Atomium",
    "class": "Attraction",
    "subclass": "monument",
    "lat": 50.89494,
    "lng": 4.34153,
    "country_code": "BE",
    "distance_meters": 12.4
  }
}
```

---

## Spatial Discovery & POI Search

### 3. `search_places`

Full-text search across places with structured filtering by taxonomy class, country, cuisine, and hotel star ratings.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/search`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `query` | `string` | No | `""` | Search text or keyword. |
| `class` | `string` | No | — | Taxonomy class (e.g. `"Restaurant"`, `"Hotel"`, `"Attraction"`). |
| `subclass` | `string` | No | — | Fine-grained classification (e.g. `"museum"`, `"italian"`, `"hostel"`). |
| `country` | `string` | No | — | ISO 3166-1 alpha-2 country code (e.g. `"BE"`). |
| `cuisine` | `string` | No | — | Dining cuisine type (e.g. `"italian"`, `"french"`, `"japanese"`). |
| `star_rating` | `integer` | No | — | Exact lodging star rating (1 to 5). |
| `lang` | `string` | No | — | Preferred ISO language code. |
| `limit` | `integer` | No | `10` | Maximum results to return (1 to 25). |
| `offset` | `integer` | No | `0` | Pagination offset. |
| `include_facets` | `boolean` | No | `false` | Whether to include taxonomy and country facet distributions. |

---

### 4. `find_nearby`

Radial spatial search: finds points of interest, dining, lodging, or landmarks within a given distance from coordinates, ordered by physical distance.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/nearby`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `lat` | `number` | **Yes** | — | Center latitude coordinate (-90 to 90). |
| `lng` | `number` | **Yes** | — | Center longitude coordinate (-180 to 180). |
| `radius_meters` | `integer` | No | `5000` | Search radius in meters (1 to 50,000). |
| `class` | `string` | No | — | Filter by taxonomy category (e.g. `"Restaurant"`, `"Hotel"`). |
| `subclass` | `string` | No | — | Subclass tag filter. |
| `cuisine` | `string` | No | — | Cuisine filter for dining places. |
| `min_stars` | `integer` | No | — | Minimum lodging star rating (1 to 5). |
| `limit` | `integer` | No | `10` | Maximum results to return (1 to 25). |
| `lang` | `string` | No | — | Preferred ISO language code. |

#### Sample Response

```json
{
  "total_hits": 18,
  "places": [
    {
      "id": "osm_n987654",
      "name": "Le Bistro",
      "class": "Restaurant",
      "subclass": "belgian",
      "cuisine": "belgian",
      "lat": 50.8451,
      "lng": 4.3512,
      "country_code": "BE",
      "distance_meters": 142
    }
  ]
}
```

---

### 5. `places_in_area`

Bounding-box spatial query: retrieves entities located inside a geographic bounding box (`[min_lat, min_lng, max_lat, max_lng]`).

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/bbox`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `min_lat` | `number` | **Yes** | — | Southern latitude boundary (-90 to 90). |
| `min_lng` | `number` | **Yes** | — | Western longitude boundary (-180 to 180). |
| `max_lat` | `number` | **Yes** | — | Northern latitude boundary (-90 to 90). |
| `max_lng` | `number` | **Yes** | — | Eastern longitude boundary (-180 to 180). |
| `class` | `string` | No | — | Filter by taxonomy class. |
| `limit` | `integer` | No | `10` | Maximum places to return (1 to 25). |
| `lang` | `string` | No | — | Preferred language code. |

---

### 6. `get_place`

Deep place inspection: retrieves full details for a place using its Geomilli OSM primary ID (`osm_n123`, `osm_w456`, `osm_r789`) or its Wikidata entity QID (`Q207`).

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/places/{id}` or `GET /v1/places/wikidata/{qid}`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `id` | `string` | No | — | Primary place identifier (e.g. `"osm_n123456"`). |
| `wikidata_id` | `string` | No | — | Wikidata entity QID (e.g. `"Q207"`). |
| `include_nearby` | `boolean` | No | `false` | Whether to fetch nearby places in the same response. |
| `lang` | `string` | No | — | Preferred ISO language code. |
| `verbose` | `boolean` | No | `false` | When true, returns the raw uncompressed document including all language maps and aliases. |

> [!IMPORTANT]
> Exactly one of `id` or `wikidata_id` must be provided.

---

## Aviation & Transit Intelligence

### 7. `find_airports`

Search for international, domestic, or regional airports by name, IATA code, ICAO code, country, or coordinate proximity.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: true`
* **Upstream Endpoint**: `GET /v1/airports`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `query` | `string` | No | — | Search string or airport name. |
| `country` | `string` | No | — | ISO 3166-1 alpha-2 country code (e.g. `"BE"`). |
| `iata` | `string` | No | — | 3-letter IATA code (e.g. `"BRU"`). |
| `icao` | `string` | No | — | 4-letter ICAO code (e.g. `"EBBR"`). |
| `near_lat` | `number` | No | — | Latitude coordinate for proximity sorting. |
| `near_lng` | `number` | No | — | Longitude coordinate for proximity sorting. |
| `limit` | `integer` | No | `10` | Maximum airports to return (1 to 25). |

---

### 8. `get_airport`

Retrieve complete specifications and metadata for a specific airport by its 3-letter IATA code or 4-letter ICAO code.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/airports/{code}`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `code` | `string` | **Yes** | — | Airport 3-letter IATA code (e.g. `"BRU"`, `"CDG"`) or 4-letter ICAO code (e.g. `"EBBR"`, `"LFPG"`). |

#### Sample Response

```json
{
  "airport": {
    "id": "osm_w38548549",
    "name": "Brussels Airport",
    "class": "Airport",
    "subclass": "international",
    "iata_code": "BRU",
    "icao_code": "EBBR",
    "lat": 50.9014,
    "lng": 4.4844,
    "country_code": "BE",
    "admin_hierarchy": ["Zaventem", "Flemish Brabant", "Flanders", "Belgium"],
    "popularity_score": 0.96
  }
}
```

---

## Administrative Hierarchy

### 9. `list_countries`

List all countries currently indexed in the Geomilli database along with total place counts. Use this to determine where data is available before running localized queries.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/countries`
* **Parameters**: None (`{}`)

---

### 10. `list_regions`

List administrative subdivisions (states, provinces, regions) for an ISO country code.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/countries/{code}/regions`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `country_code` | `string` | **Yes** | — | ISO 3166-1 alpha-2 country code (e.g. `"BE"`, `"FR"`). |

---

### 11. `list_cities`

List major cities and settlements for a country, ordered by resident population and popularity score.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/countries/{code}/cities`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `country_code` | `string` | **Yes** | — | ISO 3166-1 alpha-2 country code (e.g. `"BE"`). |
| `limit` | `integer` | No | `20` | Maximum cities to return (1 to 50). |
| `offset` | `integer` | No | `0` | Pagination offset. |

---

## Reference & Utilities

### 12. `list_place_categories`

Inspect the 19 canonical taxonomy categories defined in `config/taxonomy.yaml`, including layer (`poi` or `admin`), baseline ranking score, and known subclasses.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/taxonomy`
* **Parameters**: None (`{}`)

---

### 13. `measure_distance`

Calculate the exact great-circle distance between two locations using the Haversine formula. Coordinates can be provided directly (`{lat, lng}`) or resolved automatically by passing Geomilli place IDs.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: Composite (resolves places via `/v1/places/{id}` if IDs are passed, then computes Haversine formula)

#### Parameters

| Name | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `from` | `string` or `object` | **Yes** | Starting point: place ID string (e.g. `"osm_w157053075"`) or coordinate object `{"lat": 50.8467, "lng": 4.3524}`. |
| `to` | `string` or `object` | **Yes** | Destination point: place ID string or coordinate object. |

#### Sample Response

```json
{
  "from_name": "Grand Place",
  "to_name": "Atomium",
  "distance_meters": 5420.5,
  "distance_km": 5.42,
  "distance_miles": 3.37
}
```

---

## Operations & Monitoring (Tier 2)

> [!WARNING]
> Tier-2 operations tools are disabled by default. Set `MCP_ENABLE_OPS_TOOLS=true` in your server environment to enable them.

### 14. `get_service_status`

Retrieve consolidated operational health, harvester ingestion daemon state, and search telemetry metrics for the Geomilli cluster.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoints**: `GET /v1/health`, `GET /v1/importer/status`, `GET /v1/metrics`
* **Parameters**: None (`{}`)

---

### 15. `list_import_queue`

List the Harvester ingestion daemon queue, active worker slots, and pending country import tasks.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/importer/queue`
* **Parameters**: None (`{}`)

---

### 16. `list_importable_countries`

List all countries available for OpenStreetMap data download and ingestion by the Harvester daemon, including indexing status and download sizes.

* **Annotations**: `readOnlyHint: true`, `idempotentHint: true`, `openWorldHint: false`
* **Upstream Endpoint**: `GET /v1/importer/countries`

#### Parameters

| Name | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `search` | `string` | No | — | Optional search query to filter countries by name or code. |
