# API Overview

`geomilli-api` is a lightweight, high-performance Go REST service exposing the complete capabilities of your indexed OpenStreetMap data.

---

## 1. Base URL & Versioning

All API routes are versioned under the `/v1` prefix:

```text
http://localhost:8080/v1
```

In production deployments behind reverse proxies or cloud load balancers, SSL/TLS should be terminated at the edge.

---

## 2. Authentication

By default, `geomilli-api` operates in open developer mode. When deployed to public environments, you can enforce Bearer token authentication via environment configuration:

```dotenv
API_AUTH_TOKEN=your_secure_bearer_token
```

When enabled, clients must provide the `Authorization` header:

```http
Authorization: Bearer your_secure_bearer_token
```

---

## 3. Standard Response Format

Endpoints return standardized JSON responses. Search and query endpoints include execution time and total hit counts:

```json
{
  "hits": [ ... ],
  "query": "Brux",
  "processingTimeMs": 2,
  "limit": 10,
  "offset": 0,
  "estimatedTotalHits": 1
}
```

### Response Headers
- `Content-Type: application/json; charset=utf-8`
- `X-Processing-Time-Ms: 2`: Execution time inside Meilisearch engine.
- `X-Response-Time-Ms: 3.4`: Total round-trip time inside the Go HTTP handler.

---

## 4. Error Format

When an error occurs, the API returns a structured error object:

```json
{
  "error": {
    "code": "invalid_parameter",
    "message": "Parameter 'lat' must be a valid float between -90 and 90",
    "status": 400
  }
}
```

### HTTP Status Codes:
- `200 OK`: Request succeeded.
- `400 Bad Request`: Missing or malformed query parameter.
- `401 Unauthorized`: Missing or invalid Bearer token.
- `404 Not Found`: Resource, place ID, or airport code not found.
- `500 Internal Server Error`: Meilisearch engine unreachable or backend failure.

---

## 5. OpenAPI 3.1 Specification

The complete interactive OpenAPI specification is located at [`/api/openapi.yaml`](/api/openapi.yaml) and can be loaded into Swagger UI, Postman, or Insomnia.
