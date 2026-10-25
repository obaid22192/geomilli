# Security & Production Deployment

Geomilli MCP is engineered from the ground up to be safe, reliable, and production-ready in both local development environments and multi-tenant cloud architectures.

---

## 1. Security Architecture

### Principle of Least Privilege

Geomilli MCP partitions capabilities into two distinct operational tiers:

* **Tier 1 (Public & Read-Only)**: Always enabled. Includes 13 geospatial search, geocoding, and reference tools. All Tier-1 tools are strictly read-only, idempotent, and have zero side-effects.
* **Tier 2 (Operations & Telemetry)**: Disabled by default. Contains read-only administrative introspection tools (`get_service_status`, `list_import_queue`, `list_importable_countries`). Enabled only when explicitly requested via `MCP_ENABLE_OPS_TOOLS=true`.

> [!IMPORTANT]
> **No Mutating Operations in MCP**: High-risk mutating operations (such as queue clearing, job cancellation, and language whitelist changes) are deliberately excluded from MCP tool exposure to prevent autonomous agents from triggering unintended cluster modifications.

### Protection Against Injection & Hallucination

* **No Raw Query DSL Exposure**: Geomilli MCP never exposes Meilisearch's internal filter syntax or query language to LLMs. All tool inputs use strongly typed, structured schemas. The Go adapter validates parameters and constructs safe internal queries.
* **Input Validation & Clamping**: Coordinates are strictly bounded to valid ranges (Latitude: `[-90, 90]`, Longitude: `[-180, 180]`). Search radii and limits are clamped to safe maximums (`radius_meters` $\le 50\,000$ for nearby, $\le 100\,000$ for reverse; `limit` $\le 25$).
* **Config-Derived Enums**: Taxonomy classes and language codes are validated against cached metadata dynamically loaded from the API. Invalid values produce clear, self-correcting error messages rather than upstream failures.

---

## 2. Transport Security

Geomilli MCP implements defense-in-depth protections tailored to both supported transport protocols:

### `stdio` Transport

* **Zero Network Exposure**: Runs as a local child process communicating over Unix standard input/output pipes. No network ports are opened.
* **Stream Isolation**: All diagnostic, tracing, and operational logs are strictly written to standard error (`stderr`) using Go's `log/slog` structured logger. Standard output (`stdout`) is reserved exclusively for JSON-RPC MCP messages, completely preventing protocol desynchronization.

### Streamable `http` Transport

The HTTP transport exposes a stateless Streamable HTTP endpoint on `POST /mcp` along with an unauthenticated health probe on `GET /healthz`.

#### Constant-Time Bearer Authentication

When configured, incoming requests must include an `Authorization: Bearer <token>` header. The token is validated using Go's `crypto/subtle.ConstantTimeCompare` to eliminate timing side-channel attacks:

```go
func constantTimeCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

#### Non-Loopback Enforcement

To prevent accidental public exposure of unauthenticated endpoints, `geomilli-mcp` includes an automated startup safety check:
* If bound to loopback (`127.0.0.1` or `localhost`), `MCP_AUTH_TOKEN` is optional.
* If bound to any non-loopback address (e.g. `0.0.0.0` or a public interface), the server **refuses to start** unless `MCP_AUTH_TOKEN` is explicitly set with a non-empty secret token.

#### DNS-Rebinding & Origin Validation

Web-based LLM clients or browsers visiting malicious websites could theoretically attempt DNS-rebinding attacks against local services. Geomilli MCP implements the MCP specification's Origin validation:
* Specify permitted origins via `MCP_ALLOWED_ORIGINS` (comma-separated list, e.g. `http://localhost:3000,https://my-agent.internal`).
* If `MCP_ALLOWED_ORIGINS` is configured, incoming requests with an unrecognized `Origin` header are rejected with HTTP 403 Forbidden.

---

## 3. Production Deployment Topologies

### Topology A: Docker Compose Cluster (Recommended for Teams)

In a containerized environment, run `geomilli-mcp` behind Docker Compose with the `mcp` profile:

```yaml
# docker-compose.yml snippet
services:
  geomilli-mcp:
    build:
      context: ./geomilli-mcp
      dockerfile: Dockerfile
    container_name: geomilli-mcp
    restart: unless-stopped
    profiles:
      - mcp
    ports:
      - "${MCP_PORT:-8090}:8090"
    depends_on:
      geomilli-api:
        condition: service_started
    environment:
      - GEOMILLI_API_URL=http://geomilli-api:8080
      - MCP_TRANSPORT=http
      - MCP_HTTP_ADDR=0.0.0.0:8090
      - MCP_AUTH_TOKEN=${MCP_AUTH_TOKEN:-dev-mcp-token}
      - MCP_ALLOWED_ORIGINS=${MCP_ALLOWED_ORIGINS:-}
      - MCP_ENABLE_OPS_TOOLS=${MCP_ENABLE_OPS_TOOLS:-false}
```

Start the cluster:
```bash
make up-mcp
```

### Topology B: Kubernetes / Cloud Container Deployment

Deploy `geomilli-mcp` as a Kubernetes Deployment or AWS ECS task:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: geomilli-mcp
  labels:
    app: geomilli-mcp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: geomilli-mcp
  template:
    metadata:
      labels:
        app: geomilli-mcp
    spec:
      containers:
        - name: mcp
          image: geomilli-mcp:1.0.0
          ports:
            - containerPort: 8090
          env:
            - name: GEOMILLI_API_URL
              value: "http://geomilli-api.internal:8080"
            - name: MCP_TRANSPORT
              value: "http"
            - name: MCP_HTTP_ADDR
              value: "0.0.0.0:8090"
            - name: MCP_AUTH_TOKEN
              valueFrom:
                secretKeyRef:
                  name: geomilli-secrets
                  key: mcp-auth-token
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8090
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8090
            initialDelaySeconds: 2
            periodSeconds: 5
```

### Topology C: Reverse Proxy with TLS (Nginx / Caddy)

Terminate SSL/TLS and enforce rate limiting at the network edge:

```text
# Caddyfile example
mcp.geomilli.internal {
    reverse_proxy geomilli-mcp:8090 {
        header_up X-Forwarded-Proto {scheme}
    }
}
```

---

## 4. Horizontal Scalability & Performance

* **Stateless by Design**: `geomilli-mcp` maintains no user sessions, no websocket state, and no local disk caches. Every tool call maps directly to an upstream REST request.
* **Lightweight Resource Footprint**: The static Go binary consumes less than 30MB of RAM under normal operating conditions.
* **In-Memory TTL Cache**: Taxonomy definitions and language whitelists are cached in memory for 10 minutes (`MCP_TAXONOMY_TTL`) to avoid redundant upstream calls during high-frequency tool use.
* **Cold-Start Resilience**: If the upstream API is temporarily unreachable during startup, the taxonomy cache gracefully falls back to permissive string validation, ensuring the MCP server remains operational.

---

## 5. Environment Variables & Configuration Reference

| Variable | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `GEOMILLI_API_URL` | String | `http://localhost:8080` | URL of the upstream `geomilli-api` REST service. |
| `GEOMILLI_API_TOKEN` | String | _none_ | Upstream Bearer token forwarded in `Authorization: Bearer <token>`. |
| `MCP_TRANSPORT` | String | `stdio` | Transport protocol: `stdio` or `http`. |
| `MCP_HTTP_ADDR` | String | `127.0.0.1:8090` | Listen host and port for HTTP transport. |
| `MCP_AUTH_TOKEN` | String | _none_ | Required Bearer token for client authentication in HTTP mode. Mandatory for non-loopback addresses. |
| `MCP_ALLOWED_ORIGINS` | String | _none_ | Comma-separated list of permitted `Origin` header values for DNS-rebinding protection. |
| `MCP_ENABLE_OPS_TOOLS` | Boolean | `false` | When `true`, enables Tier-2 operations and monitoring tools. |
| `MCP_DEFAULT_LANG` | String | _none_ | Default ISO language code (e.g. `en`, `fr`) applied when tool input omits `lang`. |
| `MCP_REQUEST_TIMEOUT` | Duration | `10s` | Upstream REST request timeout (e.g. `5s`, `10s`). |
| `MCP_TAXONOMY_TTL` | Duration | `10m` | Refresh interval for cached taxonomy and languages metadata. |
