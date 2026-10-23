# Autocomplete & Search

Geomilli provides two dedicated search endpoints: `GET /v1/autocomplete` for fast, prefix-matching keystroke suggestions, and `GET /v1/search` for rich faceted discovery.

---

## 1. Autocomplete (`GET /v1/autocomplete`)

The autocomplete endpoint is optimized for sub-10ms latency as the user types. It supports prefix queries, localized language fallbacks, geo-proximity sorting, and class filters.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `q` | `string` | **Yes** | Search prefix (e.g. `bru`, `amster`, `eiffel`). |
| `lang` | `string` | No | Language code (`en`, `fr`, `nl`, `de`). Resolves localized names and highlights. |
| `country` | `string` | No | Filter by ISO 3166-1 alpha-2 country code (`BE`, `FR`, `DE`). |
| `class` | `string` | No | Restrict to taxonomy class (`City`, `Airport`, `Hotel`, `Restaurant`). |
| `subclass` | `string` | No | Filter by specific subclass (`museum`, `international`, `hostel`). |
| `lat`, `lng` | `float` | No | Coordinates for geo-proximity distance ranking. |
| `limit` | `int` | No | Number of hits to return (default: `10`, max: `50`). |

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
  "hits": [
    {
      "id": "relation_54094",
      "name": "Bruxelles",
      "class": "City",
      "subclass": "city",
      "country_code": "BE",
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
      "popularity_score": 98.4,
      "_formatted": {
        "name": "<em>Brux</em>elles"
      }
    }
  ],
  "processingTimeMs": 2,
  "query": "Brux",
  "limit": 10
}
```

---

## 2. Full-Text & Faceted Search (`GET /v1/search`)

The search endpoint allows complex boolean filtering, multi-attribute facet counts, custom sort orders, and pagination.

### Query Parameters

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `q` | `string` | No | Full-text query string. If omitted, matches all documents matching filters. |
| `facets` | `string` | No | Comma-separated list of facet fields (e.g., `class,subclass,country_code,cuisine`). |
| `filter` | `string` | No | Meilisearch filter expression (e.g. `country_code = 'BE' AND star_rating >= 4`). |
| `sort` | `string` | No | Sorting rule (`popularity_score:desc`, `_geoPoint(lat,lng):asc`). |
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
  "hits": [
    {
      "id": "way_10293847",
      "name": "Hotel Amigo",
      "class": "Hotel",
      "subclass": "hotel",
      "country_code": "BE",
      "star_rating": 5,
      "_geo": {
        "lat": 50.8462,
        "lng": 4.3519
      },
      "popularity_score": 88.5
    }
  ],
  "facetDistribution": {
    "class": {
      "Hotel": 342
    },
    "star_rating": {
      "4": 280,
      "5": 62
    }
  },
  "totalHits": 342,
  "processingTimeMs": 4,
  "query": "hotel"
}
```
