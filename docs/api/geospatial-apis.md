# Geospatial & POI APIs

Geomilli leverages Meilisearch's native geospatial capabilities (`_geoRadius`, `_geoBoundingBox`, `_geoPoint`) to provide rich spatial queries, airport resolution, and administrative hierarchy browsing.

---

## 1. Reverse Geocoding (`GET /v1/reverse`)

Resolves geographic coordinates into an enclosing administrative hierarchy (Country, Region, City, Neighborhood) and identifies the nearest landmark or street address.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `lat` | `float` | **Yes** | Latitude (-90.0 to 90.0). |
| `lng` | `float` | **Yes** | Longitude (-180.0 to 180.0). |
| `limit` | `int` | No | Number of nearby candidate places to return (default: `5`). |

### Example Request

```bash
curl -G "http://localhost:8080/v1/reverse" \
  --data-urlencode "lat=50.8467" \
  --data-urlencode "lng=4.3525"
```

### Example Response

```json
{
  "hierarchy": {
    "country": {
      "id": "relation_52411",
      "name": "België / Belgique / Belgien",
      "country_code": "BE"
    },
    "region": {
      "id": "relation_54094",
      "name": "Brussels Hoofdstedelijk Gewest",
      "class": "Region"
    },
    "city": {
      "id": "relation_54094",
      "name": "Bruxelles - Brussel",
      "class": "City"
    },
    "neighborhood": {
      "id": "node_983726",
      "name": "Îlot Sacré",
      "class": "Neighborhood"
    }
  },
  "nearest_place": {
    "id": "node_1189283",
    "name": "Grand Place",
    "class": "TouristAttraction",
    "distance_meters": 12.4
  }
}
```

---

## 2. Nearby POIs (`GET /v1/nearby`)

Finds points of interest within a circular geographic radius using Meilisearch's `_geoRadius(lat, lng, distance_in_meters)` filter.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `lat` | `float` | **Yes** | Center point latitude. |
| `lng` | `float` | **Yes** | Center point longitude. |
| `radius` | `int` | No | Search radius in meters (default: `1000`, max: `50000`). |
| `class` | `string` | No | Filter by class (`Restaurant`, `Hotel`, `Cafe`, `TouristAttraction`). |
| `subclass` | `string` | No | Filter by subclass (e.g. `bakery`, `museum`). |
| `cuisine` | `string` | No | Filter by restaurant cuisine (e.g. `italian`, `belgian`). |
| `min_stars`| `int` | No | Minimum star rating for hotels (`1` to `5`). |
| `limit` | `int` | No | Max results to return (default: `20`). |

### Example Request

Find Belgian restaurants within 1.5 km of the Grand Place:

```bash
curl -G "http://localhost:8080/v1/nearby" \
  --data-urlencode "lat=50.8467" \
  --data-urlencode "lng=4.3525" \
  --data-urlencode "radius=1500" \
  --data-urlencode "class=Restaurant" \
  --data-urlencode "cuisine=belgian"
```

---

## 3. Bounding Box Viewport (`GET /v1/bbox`)

Renders places and markers directly inside an interactive map viewport (e.g., MapLibre GL, Leaflet, or Google Maps) using `_geoBoundingBox`.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `min_lat` | `float` | **Yes** | Southwest bounding latitude. |
| `min_lng` | `float` | **Yes** | Southwest bounding longitude. |
| `max_lat` | `float` | **Yes** | Northeast bounding latitude. |
| `max_lng` | `float` | **Yes** | Northeast bounding longitude. |
| `class` | `string` | No | Filter to specific taxonomy classes (comma-separated). |
| `limit` | `int` | No | Max markers to return (default: `100`, max: `500`). |

### Example Request

```bash
curl -G "http://localhost:8080/v1/bbox" \
  --data-urlencode "min_lat=50.8400" \
  --data-urlencode "min_lng=4.3400" \
  --data-urlencode "max_lat=50.8600" \
  --data-urlencode "max_lng=4.3700" \
  --data-urlencode "class=TouristAttraction,Hotel"
```

---

## 4. Aviation & Airports API

Geomilli isolates airport facilities, airstrips, and heliports with specialized IATA and ICAO indices.

### Lookup by Code (`GET /v1/airports/{code}`)

Resolves 3-letter IATA codes (e.g. `BRU`, `AMS`, `CDG`, `JFK`) or 4-letter ICAO codes (e.g. `EBBR`, `EHAM`):

```bash
curl -s "http://localhost:8080/v1/airports/BRU" | jq .
```

Response:
```json
{
  "id": "relation_2378912",
  "name": "Brussels Airport",
  "class": "Airport",
  "subclass": "international",
  "iata": "BRU",
  "icao": "EBBR",
  "country_code": "BE",
  "_geo": {
    "lat": 50.9014,
    "lng": 4.4844
  },
  "wikidata": "Q207853",
  "popularity_score": 96.2
}
```

### Search Airports (`GET /v1/airports`)

Supports searching and filtering airports by name text, IATA/ICAO code, country, or proximity:

```bash
# 1. List all airports for a country
curl -s "http://localhost:8080/v1/airports?country=BE&limit=25" | jq .

# 2. Search by keyword or airport name (e.g. "Charleroi")
curl -s "http://localhost:8080/v1/airports?q=Charleroi&country=BE" | jq .

# 3. Search directly by 3-letter IATA or 4-letter ICAO code
curl -s "http://localhost:8080/v1/airports?iata=BRU" | jq .
curl -s "http://localhost:8080/v1/airports?code=CRL" | jq .

# 4. Proximity search around GPS coordinates
curl -s "http://localhost:8080/v1/airports?lat=50.9014&lng=4.4844&limit=10" | jq .
```

| Query Parameter | Type | Description |
| :--- | :--- | :--- |
| `q` | `string` | Free-text search matching airport names or embedded IATA/ICAO codes. |
| `country` | `string` | ISO 2-letter country code filter (e.g. `BE`, `NL`, `FR`). |
| `code` | `string` | Exact match on either `iata_code` or `icao_code`. |
| `iata` | `string` | Exact match on 3-letter IATA code (e.g. `BRU`, `ANR`, `CRL`). |
| `icao` | `string` | Exact match on 4-letter ICAO code (e.g. `EBBR`, `EBCI`). |
| `lat`, `lng` | `float` | Geo-coordinates to rank airports by proximity. |
| `limit` | `int` | Maximum results to return (default: 20, max: 100). |

---

## 5. Administrative Hierarchy & Place Lookup

- `GET /v1/countries`: Returns all indexed countries with entity counts.
- `GET /v1/countries/{code}/regions`: Lists admin level 4 regions (provinces, states).
- `GET /v1/countries/{code}/cities`: Lists cities ranked by population.
- `GET /v1/places/{id}`: Fetches the raw normalized place document by ID (e.g. `relation_54094`).
- `GET /v1/places/wikidata/{qid}`: Fetches place by Wikidata entity ID (e.g. `Q31`).
