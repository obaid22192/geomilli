# Geospatial & POI APIs

Geomilli leverages Meilisearch's native geospatial capabilities (`_geoRadius`, `_geoBoundingBox`, `_geoPoint`) to provide rich spatial queries, airport resolution, and administrative hierarchy browsing.

---

## 1. Reverse Geocoding (`GET /v1/reverse`)

Resolves geographic coordinates into structured address components (landmark, neighborhood, city, region, country), enclosing administrative hierarchy, and nearest places within search radius.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `lat` | `float` | **Yes** | Latitude coordinate (-90.0 to 90.0). |
| `lng` | `float` | **Yes** | Longitude coordinate (-180.0 to 180.0). |
| `radius_meters` | `int` | No | Search radius in meters (default: `50000`, max: `100000`). Accepts `radius` as alias. Out-of-bounds values are clamped. |

### Example Request

```bash
curl -G "http://localhost:8080/v1/reverse" \
  --data-urlencode "lat=50.8467" \
  --data-urlencode "lng=4.3525" \
  --data-urlencode "radius_meters=50000"
```

### Example Response

```json
{
  "lat": 50.8467,
  "lng": 4.3525,
  "radius_meters": 50000,
  "address": {
    "landmark": "Grand Place",
    "neighborhood": "Îlot Sacré",
    "city": "Bruxelles - Brussel",
    "region": "Brussels Hoofdstedelijk Gewest",
    "country": "België / Belgique / Belgien",
    "country_code": "BE"
  },
  "admin_hierarchy": [
    "Îlot Sacré",
    "Bruxelles - Brussel",
    "Brussels Hoofdstedelijk Gewest",
    "België / Belgique / Belgien"
  ],
  "closest_place": {
    "id": "node_1189283",
    "entity_id": "node_1189283",
    "class": "TouristAttraction",
    "name_display": "Grand Place",
    "country_code": "BE",
    "country_name": "Belgium",
    "_geo": {
      "lat": 50.8467,
      "lng": 4.3525
    },
    "distance_meters": 12.4,
    "popularity_score": 95.0
  },
  "nearest_places": [
    {
      "id": "node_1189283",
      "entity_id": "node_1189283",
      "class": "TouristAttraction",
      "name_display": "Grand Place",
      "country_code": "BE",
      "country_name": "Belgium",
      "_geo": {
        "lat": 50.8467,
        "lng": 4.3525
      },
      "distance_meters": 12.4,
      "popularity_score": 95.0
    }
  ]
}
```

---

## 2. Nearby POIs (`GET /v1/nearby`)

Finds points of interest and places within a circular geographic radius using Meilisearch's `_geoRadius(lat, lng, radius_in_meters)` filter.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `lat` | `float` | **Yes** | Center point latitude (-90.0 to 90.0). |
| `lng` | `float` | **Yes** | Center point longitude (-180.0 to 180.0). |
| `radius_meters` | `int` | No | Search radius in meters (default: `5000`, max: `100000`). Accepts `radius` as alias. Out-of-bounds values are clamped. |
| `class` | `string` | No | Filter by class (`Restaurant`, `Hotel`, `Cafe`, `TouristAttraction`). |
| `subclass` | `string` | No | Filter by subclass (e.g. `bakery`, `museum`). |
| `cuisine` | `string` | No | Filter by restaurant cuisine (e.g. `italian`, `belgian`). |
| `min_stars`| `int` | No | Minimum star rating for hotels (`1` to `5`). |
| `limit` | `int` | No | Max results to return (default: `20`, max: `100`). |
| `offset` | `int` | No | Pagination offset (default: `0`). |
| `lang` | `string` | No | ISO 639-1 language code for localized name display. |

### Example Request

Find Belgian restaurants within 1.5 km of the Grand Place:

```bash
curl -G "http://localhost:8080/v1/nearby" \
  --data-urlencode "lat=50.8467" \
  --data-urlencode "lng=4.3525" \
  --data-urlencode "radius_meters=1500" \
  --data-urlencode "class=Restaurant" \
  --data-urlencode "cuisine=belgian"
```

### Example Response

```json
{
  "lat": 50.8467,
  "lng": 4.3525,
  "radius_meters": 1500,
  "took_ms": 3,
  "limit": 20,
  "offset": 0,
  "total_hits": 45,
  "results": [
    {
      "id": "node_8923741",
      "entity_id": "node_8923741",
      "class": "Restaurant",
      "subclass": "restaurant",
      "name_display": "Chez Léon",
      "country_code": "BE",
      "country_name": "Belgium",
      "cuisine": "belgian",
      "_geo": {
        "lat": 50.8471,
        "lng": 4.3538
      },
      "distance_meters": 102.5,
      "popularity_score": 78.0
    }
  ]
}
```

---

## 3. Bounding Box Viewport (`GET /v1/bbox`)

Renders places and markers directly inside an interactive map viewport (e.g., MapLibre GL, Leaflet, or Google Maps) using `_geoBoundingBox`.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `min_lat` | `float` | **Yes** | Southwest bounding latitude (-90.0 to 90.0). |
| `min_lng` | `float` | **Yes** | Southwest bounding longitude (-180.0 to 180.0). |
| `max_lat` | `float` | **Yes** | Northeast bounding latitude (-90.0 to 90.0). |
| `max_lng` | `float` | **Yes** | Northeast bounding longitude (-180.0 to 180.0). |
| `class` | `string` | No | Filter to specific taxonomy class (e.g. `TouristAttraction`, `Hotel`). |
| `limit` | `int` | No | Max markers to return (default: `100`, max: `500`). |
| `lang` | `string` | No | Language code for name localization. |

### Example Request

```bash
curl -G "http://localhost:8080/v1/bbox" \
  --data-urlencode "min_lat=50.8400" \
  --data-urlencode "min_lng=4.3400" \
  --data-urlencode "max_lat=50.8600" \
  --data-urlencode "max_lng=4.3700" \
  --data-urlencode "class=Hotel"
```

### Example Response

```json
{
  "min_lat": 50.8400,
  "min_lng": 4.3400,
  "max_lat": 50.8600,
  "max_lng": 4.3700,
  "took_ms": 5,
  "total_hits": 87,
  "results": [
    {
      "id": "way_10293847",
      "entity_id": "way_10293847",
      "class": "Hotel",
      "subclass": "hotel",
      "name_display": "Hotel Amigo",
      "country_code": "BE",
      "country_name": "Belgium",
      "star_rating": 5,
      "_geo": {
        "lat": 50.8462,
        "lng": 4.3519
      },
      "popularity_score": 88.5
    }
  ]
}
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
  "entity_id": "relation_2378912",
  "class": "Airport",
  "subclass": "international",
  "name_display": "Brussels Airport",
  "iata_code": "BRU",
  "icao_code": "EBBR",
  "country_code": "BE",
  "country_name": "Belgium",
  "_geo": {
    "lat": 50.9014,
    "lng": 4.4844
  },
  "wikidata_id": "Q207853",
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
| `limit` | `int` | Maximum results to return (default: `20`, max: `100`). |
| `lang` | `string` | Language code for name localization. |

### Airport Search Response

```json
{
  "query": "Charleroi",
  "took_ms": 2,
  "total_hits": 1,
  "results": [
    {
      "id": "relation_98234",
      "entity_id": "relation_98234",
      "class": "Airport",
      "subclass": "international",
      "name_display": "Brussels South Charleroi Airport",
      "iata_code": "CRL",
      "icao_code": "EBCI",
      "country_code": "BE",
      "country_name": "Belgium",
      "_geo": {
        "lat": 50.4592,
        "lng": 4.4538
      },
      "popularity_score": 85.0
    }
  ]
}
```

---

## 5. Administrative Hierarchy & Place Lookup

Geomilli structures geographic administrative divisions into a queryable hierarchy.

### List Countries (`GET /v1/countries`)

Returns all indexed countries and total entity counts:

```json
{
  "total_countries": 1,
  "countries": [
    {
      "country_code": "BE",
      "country_name": "Belgium",
      "place_count": 184520
    }
  ]
}
```

### List Regions (`GET /v1/countries/{code}/regions`)

Lists administrative level 4 regions (provinces, states):

```json
{
  "country_code": "BE",
  "total_hits": 3,
  "regions": [
    {
      "id": "relation_54094",
      "entity_id": "relation_54094",
      "class": "Region",
      "name_display": "Brussels Hoofdstedelijk Gewest",
      "country_code": "BE",
      "admin_level": 4,
      "_geo": { "lat": 50.8466, "lng": 4.3528 },
      "popularity_score": 90.0
    }
  ]
}
```

### List Cities (`GET /v1/countries/{code}/cities`)

Lists cities and municipalities sorted by population. Supports `limit` (default: 50, max: 200) and `offset` (default: 0):

```json
{
  "country_code": "BE",
  "limit": 50,
  "offset": 0,
  "total_hits": 581,
  "cities": [
    {
      "id": "relation_54094",
      "entity_id": "relation_54094",
      "class": "City",
      "name_display": "Bruxelles - Brussel",
      "country_code": "BE",
      "population": 1208542,
      "_geo": { "lat": 50.8466, "lng": 4.3528 },
      "popularity_score": 98.4
    }
  ]
}
```

### Place Entity Lookup (`GET /v1/places/{id}`)

Fetches a place document by ID (e.g. `relation_54094`). Optional query parameter `include_nearby=true` returns nearby POIs within 2 km:

```bash
curl -s "http://localhost:8080/v1/places/relation_54094?include_nearby=true" | jq .
```

### Wikidata QID Lookup (`GET /v1/places/wikidata/{qid}`)

Fetches a place document by its Wikidata item identifier (e.g. `Q31`):

```bash
curl -s "http://localhost:8080/v1/places/wikidata/Q31" | jq .
```
