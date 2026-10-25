# Changelog

All notable changes to the Geomilli monorepo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

### Component Tags

Every changelog entry must be prefixed with one of the following component tags indicating the affected subproject:

- **api**: Search & Geo REST API service (`geomilli-api/`)
- **mcp**: Model Context Protocol adapter service (`geomilli-mcp/`)
- **harvester**: Streaming OSM ingestion and enrichment engine (`geomilli-harvester/`)
- **studio**: Developer console and MapLibre GL playground (`geomilli-studio/`)
- **docs**: Documentation site and API specifications (`docs/`)
- **infra**: Root orchestration, CI/CD workflows, Docker Compose, and build scripts

### Governance for Contributors & Agents

Whenever making user-visible, functional, architectural, or breaking changes, contributors and AI agents **must append** a concise entry under the `## [Unreleased]` section with the appropriate component tag. Releases promote `[Unreleased]` entries to a versioned release header.

---

## [Unreleased]

### Added
- **mcp**: Complete MCP tool catalog (13 Tier-1 tools + 3 gated ops tools) in `geomilli-mcp`:
  - Geocoding: `geocode`, `reverse_geocode`
  - Discovery: `search_places`, `find_nearby`, `places_in_area`, `get_place`
  - Aviation: `find_airports`, `get_airport`
  - Admin Hierarchy: `list_countries`, `list_regions`, `list_cities`
  - Reference: `list_place_categories`, `measure_distance`
  - Operations (gated): `get_service_status`, `list_import_queue`, `list_importable_countries`
- **mcp**: MCP resources for `geomilli://taxonomy`, `geomilli://languages`, `geomilli://countries`, `geomilli://schema/place`, and `geomilli://places/{id}` URI template.
- **mcp**: MCP workflow prompt templates: `explore_area`, `describe_location`, and `plan_airport_transfer`.
- **mcp**: Initial MCP server skeleton (`geomilli-mcp`) supporting stdio and Streamable HTTP transports with Bearer auth, Origin allowlisting, taxonomy caching, and reference tool `list_countries` (RFC-001).
- **api**: Added dynamic `/v1/openapi.json` handler serving runtime OpenAPI 3.1 specification with active build version from `internal/version.Version`.
- **api**: Added automated route coverage and synchronization test suite `openapi_routes_test.go` ensuring router routes, OpenAPI spec, and docs copy stay synchronized.
- **mcp**: Added contract and hardening test suites (`internal/contract`):
  - Golden snapshot schema testing comparing in-memory server tool registrations against `testdata/golden/tools.json` with `-update` flag support.
  - OpenAPI coverage test validating that all 27 routes in canonical `openapi.yaml` are mapped to MCP tools/resources (19 endpoints) or documented in `excludedRoutes` with justifications (8 endpoints).
  - Annotations audit validating non-nil annotations, `readOnlyHint`, `idempotentHint`, and accurate `openWorldHint` across all tools.
  - Boundary validation and fuzz testing verifying resilient handling of hostile or malformed inputs without panics.
- **docs**: Added dedicated Model Context Protocol (MCP) documentation section at `/mcp/` covering architecture & overview, client setup guides (Claude Desktop, Claude Code, Cursor, VS Code, Antigravity), comprehensive 16-tool reference, `geomilli://` context resources, guided workflow prompts, and security & production deployment.
- **infra**: Added `up-mcp`, `mcp`, and `mcp-inspect` Makefile targets for running and inspecting `geomilli-mcp`.
- **infra**: Added `geomilli-mcp` service definition to `docker-compose.yml` gated behind `profiles: ["mcp"]` to maintain zero overhead for default clusters.
- **infra**: Added `test-mcp` job and Dockerfile validation step for `geomilli-mcp` to GitHub Actions CI (`.github/workflows/ci.yml`).
- **infra**: Changelog governance policy, PR template, and CI guard workflow enforcing changelog updates on code changes.
- **infra**: Added `MCP_PORT`, `MCP_AUTH_TOKEN`, and `MCP_ENABLE_OPS_TOOLS` to `.env.example`.

### Changed
- **api**: ⚠️ BREAKING: Standardize API error responses to the `{"error":{"code":"...","message":"...","status":...}}` envelope (RFC-001 §11.2).
- **api**: Meilisearch search and entity query failures now return HTTP 502 `upstream_unavailable` instead of generic HTTP 500 errors.
- **api**: Enforce a maximum radius cap of 100,000 meters (100 km) on `/v1/reverse` and echo effective `radius_meters` across spatial responses.
- **api**: Query parameters (`lat`, `lng`, `radius_meters`, `limit`, `offset`) now use unified parser utilities with automatic clamping to defined ranges.
- **api**: Service version is now injected dynamically at build time via `-ldflags` from `VERSION` (defaults to "dev").
- **studio**: Error handling across Playground, Importer, and LanguageManager updated to parse the RFC-001 error envelope with graceful legacy fallback.
- **docs**: Reconciled API documentation (`overview.md`, `autocomplete-and-search.md`, `geospatial-apis.md`) with actual Go handler implementations, updating response envelopes (`took_ms`, `total_hits`, `facets`, `results`), parameter names (`radius_meters`), limits, and error envelopes.
- **docs**: Removed non-existent `API_AUTH_TOKEN` authentication documentation in favor of network edge proxy guidelines.
- **docs**: Created canonical OpenAPI 3.1 specification at `geomilli-api/api/openapi.yaml` documenting all 27 Chi router routes, synced byte-for-byte to `docs/api/openapi.yaml`.

## [1.0.0] - 2026-10-24

### Added
- **api**: High-throughput geospatial query engine with `/autocomplete`, `/search`, `/reverse`, `/nearby`, `/bbox`, `/airports`, `/countries`, and `/taxonomy` routes.
- **harvester**: Streaming OpenStreetMap PBF ingestion daemon with `osmium-tool` filtering, SQLite WAL caching, and Wikidata/Wikipedia pageviews enrichment.
- **studio**: Interactive developer console with MapLibre GL map playground, real-time importer telemetry, and language management.
- **docs**: VitePress developer documentation site covering architecture, ingestion pipeline, API references, and quickstarts.
- **docs**: Add GitHub Pages deployment links, Geomilli Studio showcase screenshots, and reorder README for API-first architecture with explicit scaling limits.
- **infra**: Multi-service Docker Compose orchestration and Make targets (`make up`, `make down`, `make build`, `make test`).
- **infra**: GitHub Actions CI test matrix, automated GitHub Pages deployment for documentation, and CodeQL security analysis.

### Fixed
- **api**: Ensure deterministic tie-breaking in `topNQueries` and verify query membership in telemetry tests.
- **infra**: Resolve GitHub Actions runner race condition and upgrade Chi router to v5.3.0 to address upstream security vulnerabilities.

[Unreleased]: https://github.com/geomilli/geomilli/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/geomilli/geomilli/releases/tag/v1.0.0
