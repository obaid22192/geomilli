# Autocomplete & Search

Geomilli provides two dedicated search endpoints: `GET /v1/autocomplete` for fast, prefix-matching keystroke suggestions, and `GET /v1/search` for rich faceted discovery.

---

## 1. Autocomplete (`GET /v1/autocomplete`)

The autocomplete endpoint is optimized for sub-10ms latency as the user types. It supports prefix queries, localized language fallbacks, geo-proximity sorting, and class filters.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `q` | `string` | No | Search prefix (e.g. `bru`, `amster`, `eiffel`). |
| `lang` | `string` | No | ISO 639-1 language code (`en`, `fr`, `nl`, `de`). Resolves localized names and highlights. |
| `country` | `string` | No | Filter by ISO 3166-1 alpha-2 country code (`BE`, `FR`, `DE`). |
| `class` | `string` | No | Restrict to taxonomy class (`City`, `Airport`, `Hotel`, `Restaurant`). |
| `subclass` | `string` | No | Filter by specific subclass (`museum`, `international`, `hostel`). |
| `lat`, `lng` | `float` | No | Coordinates for geo-proximity distance ranking. |
| `limit` | `int` | No | Number of hits to return (default: `10`, max: `100`). |

### Example Request

```bash
curl -G "http://localhost:8080/v1/autocomplete" \
  --data-urlencode "q=Brux" \
  --data-urlencode "lang=fr" \
  --data-urlencode "class=City"
```

### Example Response

```json
{
  "query": "Brux",
  "lang": "fr",
  "took_ms": 2,
  "total_hits": 1,
  "results": [
    {
      "id": "relation_54094",
      "entity_id": "relation_54094",
      "class": "City",
      "subclass": "city",
      "name_display": "Bruxelles",
      "country_code": "BE",
      "country_name": "Belgium",
      "admin_hierarchy": [
        "Brussels Hoofdstedelijk Gewest",
        "Belgium"
      ],
      "names_by_language": {
        "en": "Brussels",
        "fr": "Bruxelles",
        "nl": "Brussel",
        "de": "Brüssel"
      },
      "_geo": {
        "lat": 50.8466,
        "lng": 4.3528
      },
      "popularity_score": 98.4
    }
  ]
}
```

---

## 2. Full-Text & Faceted Search (`GET /v1/search`)

The search endpoint allows complex boolean filtering, multi-attribute facet counts, custom sort orders, and pagination.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `q` | `string` | No | Full-text query string. If omitted, matches all documents matching filters. |
| `lang` | `string` | No | ISO 639-1 language code for localized name display. |
| `facets` | `string` | No | Comma-separated list of facet fields (e.g. `class,subclass,country_code,cuisine`) or `*`. |
| `filter` | `string` | No | Meilisearch filter expression (e.g. `country_code = 'BE' AND star_rating >= 4`). |
| `sort` | `string` | No | Sorting rule (`popularity_score:desc`, `_geoPoint(lat,lng):asc`). |
| `class` | `string` | No | Filter by taxonomy class (`Hotel`, `Restaurant`, etc.). |
| `subclass` | `string` | No | Filter by specific subclass. |
| `country` | `string` | No | Filter by ISO 3166-1 alpha-2 country code (`BE`, `NL`, `FR`). |
| `cuisine` | `string` | No | Filter restaurants by cuisine type (e.g. `belgian`, `italian`). |
| `star_rating` | `int` | No | Filter hotels by minimum star rating (`1` to `5`). |
| `offset` | `int` | No | Pagination offset (default: `0`). |
| `limit` | `int` | No | Page size (default: `20`, max: `100`). |

### Example Request with Facets & Filtering

Search for 4-star hotels in Belgium, sorted by popularity:

```bash
curl -G "http://localhost:8080/v1/search" \
  --data-urlencode "q=hotel" \
  --data-urlencode "filter=country_code = 'BE' AND class = 'Hotel' AND star_rating >= 4" \
  --data-urlencode "facets=class,subclass,star_rating" \
  --data-urlencode "sort=popularity_score:desc"
```

### Example Response

```json
{
  "query": "hotel",
  "took_ms": 4,
  "limit": 20,
  "offset": 0,
  "total_hits": 342,
  "facets": {
    "class": {
      "Hotel": 342
    },
    "star_rating": {
      "4": 280,
      "5": 62
    }
  },
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
