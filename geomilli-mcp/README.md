# Geomilli Model Context Protocol (MCP) Server

`geomilli-mcp` is a thin, high-performance, stateless Model Context Protocol (MCP) adapter that exposes the [Geomilli](https://github.com/geomilli/geomilli) geospatial search platform and OpenStreetMap knowledge base directly to AI assistants, LLM applications, and autonomous agents (such as Claude Desktop, Claude Code, Cursor, and Antigravity).

Built with the official [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) (v1.x) in Go 1.25.

---

## Architecture & Principles

- **Stateless Adapter**: Sits between MCP clients and the upstream `geomilli-api` REST service (`/v1/*`). Never interacts with Meilisearch or SQLite directly.
- **Dual Transports**:
  - `stdio` (default): Fast, local subprocess execution via standard I/O. All application and diagnostic logging is strictly routed to `stderr` (`log/slog`) to prevent protocol corruption.
  - `http`: Modern Streamable HTTP transport running at `/mcp`. Features constant-time Bearer token authentication, Origin header validation (DNS-rebinding protection), and an unauthenticated `/healthz` probe.
- **Intent-Driven Tools**: 16 strongly typed tools with structured schemas, parameter clamping, automatic JSON-schema inference, and token-efficient compact outputs (`structuredContent` + short text summary).
- **Config-Driven**: Taxonomy classes and supported languages are cached dynamically from `/v1/taxonomy` and `/v1/languages` with TTL refresh and cold-start fallback.
- **Resources & Prompts**: Exposes `geomilli://` context resources and guided workflow prompts (`explore_area`, `describe_location`, `plan_airport_transfer`).

---

## Tool Catalog

### Tier 1: Public, Read-Only Tools (Always Enabled)

| Tool | Upstream Route | Key Inputs | Description |
| :--- | :--- | :--- | :--- |
| `geocode` | `GET /v1/autocomplete` | `query`*, `country`, `class`, `near_lat`, `near_lng`, `lang`, `limit` | Forward geocoding: resolve names and addresses into coordinates and admin hierarchy. |
| `reverse_geocode` | `GET /v1/reverse` | `lat`*, `lng`*, `radius_meters`, `lang` | Reverse geocoding: coordinates to address, hierarchy, and closest landmark. |
| `search_places` | `GET /v1/search` | `query`, `class`, `subclass`, `country`, `cuisine`, `star_rating`, `lang`, `limit`, `offset` | Full-text search with structured facet filters. |
| `find_nearby` | `GET /v1/nearby` | `lat`*, `lng`*, `radius_meters`, `class`, `subclass`, `cuisine`, `min_stars`, `limit` | Proximity search around coordinates, sorted by physical distance. |
| `places_in_area` | `GET /v1/bbox` | `min_lat`*, `min_lng`*, `max_lat`*, `max_lng`*, `class`, `limit`, `lang` | Viewport bounding-box spatial queries. |
| `get_place` | `GET /v1/places/{id}` | `id` xor `wikidata_id`, `include_nearby`, `lang`, `verbose` | Deep inspection by OSM primary identifier or Wikidata QID. |
| `find_airports` | `GET /v1/airports` | `query`, `country`, `iata`, `icao`, `near_lat`, `near_lng`, `limit` | Search airports by name, codes, country, or coordinate proximity. |
| `get_airport` | `GET /v1/airports/{code}` | `code`* (IATA or ICAO) | Complete airport specifications by 3-letter IATA or 4-letter ICAO code. |
| `list_countries` | `GET /v1/countries` | — | List all indexed countries with place counts. |
| `list_regions` | `GET /v1/countries/{code}/regions` | `country_code`* | List administrative regions, provinces, or states for a country. |
| `list_cities` | `GET /v1/countries/{code}/cities` | `country_code`*, `limit`, `offset` | List major cities ordered by population and popularity. |
| `list_place_categories` | `GET /v1/taxonomy` | — | Inspect the 19 canonical taxonomy categories and scoring weights. |
| `measure_distance` | Composite / Haversine | `from`*, `to`* (place IDs or `{lat, lng}`) | Calculate exact great-circle distance between two locations. |

### Tier 2: Operations & Telemetry (Gated by `MCP_ENABLE_OPS_TOOLS=true`)

| Tool | Upstream Route | Annotations | Description |
| :--- | :--- | :--- | :--- |
| `get_service_status` | `/v1/health` + `/v1/importer/status` + `/v1/metrics` | Read-only | Consolidated health, harvester ingestion state, and telemetry metrics. |
| `list_import_queue` | `GET /v1/importer/queue` | Read-only | Harvester queue, worker status, and pending country import tasks. |
| `list_importable_countries` | `GET /v1/importer/countries` | Read-only | Available countries for OSM ingestion with index and sync status. |

---

## Context Resources & Prompts

### Resources (`geomilli://`)

* `geomilli://taxonomy`: Canonical taxonomy JSON (classes, subclasses, scoring weights).
* `geomilli://languages`: Supported language whitelist JSON.
* `geomilli://countries`: Indexed countries with place counts.
* `geomilli://schema/place`: Authoritative Markdown documentation of the `Place` data model.
* `geomilli://places/{id}`: Dynamic URI template for direct place document lookups.

### Guided Workflow Prompts

* `explore_area` (`place`*, `interests`): Forward geocode a place and discover points of interest, dining, and cultural sights.
* `describe_location` (`lat`*, `lng`*): Reverse geocode coordinates and generate a vivid geographical and administrative narrative.
* `plan_airport_transfer` (`city`*): Find airports serving a city, measure distances, and evaluate transfer logistics.

---

## Configuration Reference

| Flag | Environment Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `--api-url` | `GEOMILLI_API_URL` | `http://localhost:8080` | Upstream Geomilli REST API origin |
| `--api-token` | `GEOMILLI_API_TOKEN` | _none_ | Forwarded as `Authorization: Bearer <token>` to upstream API |
| `--transport` | `MCP_TRANSPORT` | `stdio` | Transport type (`stdio` or `http`) |
| `--http-addr` | `MCP_HTTP_ADDR` | `127.0.0.1:8090` | Listen address for HTTP transport |
| — | `MCP_AUTH_TOKEN` | _none_ | Required Bearer token for HTTP clients (mandatory if binding non-loopback) |
| — | `MCP_ALLOWED_ORIGINS` | _none_ | Comma-separated allowlist of valid `Origin` headers (DNS-rebinding protection) |
| `--enable-ops-tools` | `MCP_ENABLE_OPS_TOOLS` | `false` | Enable Tier-2 operations and monitoring tools |
| `--lang` | `MCP_DEFAULT_LANG` | _none_ | Default ISO language code for localized place names |
| — | `MCP_REQUEST_TIMEOUT` | `10s` | Per-call timeout for upstream REST API queries |
| — | `MCP_TAXONOMY_TTL` | `10m` | Refresh interval for cached taxonomy and languages metadata |

---

## Quickstart

### Build and Test

```bash
cd geomilli-mcp
make build
make test
```

### Running Locally with stdio

```bash
./bin/geomilli-mcp --transport stdio --api-url http://localhost:8080
```

### Running with Docker Compose

Start the full Geomilli stack with the MCP server enabled:

```bash
# From repository root:
make up-mcp
```

### Testing with MCP Inspector

Anthropic's interactive MCP Inspector lets you test tools, resources, and prompts in a browser UI:

```bash
# From repository root:
make mcp-inspect
```

---

## Client Integration Examples

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "geomilli": {
      "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
      "args": ["--api-url", "http://localhost:8080"]
    }
  }
}
```

### Claude Code (CLI)

```bash
claude mcp add geomilli -- /absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp --api-url http://localhost:8080
```

### Cursor (`.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "geomilli": {
      "command": "/absolute/path/to/geomilli/geomilli-mcp/bin/geomilli-mcp",
      "args": ["--api-url", "http://localhost:8080"]
    }
  }
}
```

---

## Further Documentation

For full documentation and advanced deployment patterns, visit the official Geomilli Documentation:
- [MCP Architecture & Overview](/mcp/overview)
- [Client Setup Guide](/mcp/client-setup)
- [Tools Reference](/mcp/tools-reference)
- [Resources & Prompts](/mcp/resources-and-prompts)
- [Security & Production Deployment](/mcp/security-and-deployment)
