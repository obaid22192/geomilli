# API Overview

`geomilli-api` is a lightweight, high-performance Go REST service exposing the complete capabilities of your indexed OpenStreetMap data.

---

## 1. Base URL & Ports

The Go REST API service listens on port `:8080`. Root service status and version information are available at `/`, while operational API routes are versioned under the `/v1` prefix:

```text
http://localhost:8080/v1
```

In the full cluster setup, `geomilli-studio` runs on port `:3000` (or `:3333`) and proxies `/v1` requests directly to the API backend on `:8080`. In production deployments behind reverse proxies or cloud load balancers, SSL/TLS should be terminated at the network edge.

---

## 2. Authentication

The API currently runs without authentication. Protect it at the network edge (reverse proxy, API gateway, or private network). Built-in Bearer authentication is planned.

---

## 3. Standard Response Format

Endpoints return standardized JSON responses. Search and query endpoints include execution time, pagination limits, and total hit counts:

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
      "_geo": {
        "lat": 50.8466,
        "lng": 4.3528
      },
      "popularity_score": 98.4
    }
  ]
}
```

### Response Headers
- `Content-Type: application/json; charset=utf-8`
- `X-Processing-Time-Ms: 2`: Execution time inside the Meilisearch engine.
- `X-Response-Time-Ms: 3.4`: Total round-trip time inside the Go HTTP handler.

---

## 4. Error Format

When an error occurs, the API returns a structured RFC-001 §11.2 error envelope:

```json
{
  "error": {
    "code": "invalid_parameter",
    "message": "Both 'lat' and 'lng' query parameters are required",
    "status": 400
  }
}
```

### Standard Error Codes

| `code` | HTTP Status | Description |
| :--- | :--- | :--- |
| `missing_parameter` | `400 Bad Request` | A required query or path parameter is absent |
| `invalid_parameter` | `400 Bad Request` | A parameter is malformed, non-numeric, or out of range |
| `invalid_body` | `400 Bad Request` | The request body contains malformed JSON |
| `not_found` | `404 Not Found` | Requested place ID, Wikidata QID, or airport code was not found |
| `upstream_unavailable` | `502 Bad Gateway` / `503 Service Unavailable` | Meilisearch engine or Harvester daemon is unreachable |
| `internal_error` | `500 Internal Server Error` | An unexpected internal failure occurred |
| `unauthorized` | `401 Unauthorized` | *Reserved for future token authentication* |

---

## 5. OpenAPI 3.1 Specification

The canonical OpenAPI 3.1 specification is available in multiple formats:
- **Interactive documentation file**: [`/docs/api/openapi.yaml`](/docs/api/openapi.yaml)
- **Live JSON endpoint**: `GET http://localhost:8080/v1/openapi.json` (reflects the runtime binary build version)
