# Workspace Rules & Architecture: Geomilli

Authoritative guidelines, architecture constraints, and operational protocols for the **Geomilli** geospatial search platform.

---

## 1. Monorepo Architecture & Topology

Geomilli is an umbrella repository containing decoupled, production-grade microservices:

| Subproject | Directory | Responsibility | Tech Stack |
| :--- | :--- | :--- | :--- |
| **Orchestration & Docs** | `.` & `docs/` | Monorepo root, Docker Compose orchestration, VitePress docs site. | VitePress, Docker Compose, Make |
| **Ingestion Engine** | `geomilli-harvester/` | Streaming OSM ingestion daemon, PBF filtering, Wikidata enrichment, Meilisearch indexing. | Go 1.25, `osmium-tool`, SQLite (WAL) |
| **Search & Geo REST API** | `geomilli-api/` | High-throughput query service (`/autocomplete`, `/search`, `/reverse`, `/nearby`, `/bbox`, `/airports`). | Go 1.25 (stdlib / Chi), Meilisearch Go SDK |
| **MCP Server** | `geomilli-mcp/` | Model Context Protocol adapter exposing geospatial tools, resources, and prompts over stdio and stateless Streamable HTTP (:8090). | Go 1.25, official `modelcontextprotocol/go-sdk` |
| **Studio UI** | `geomilli-studio/` | Interactive developer console, MapLibre GL playground, importer monitor, telemetry. | React 19, TypeScript, Tailwind CSS, Vite |

---

## 2. Core Build & Development Commands

Always execute commands from the repository root:

* **Start Full Cluster**: `make up` (Runs Meilisearch, Harvester daemon, API service, and Studio UI)
* **Start Full Cluster with MCP**: `make up-mcp` *(added by RFC-001)* (Runs full cluster plus `geomilli-mcp` via Docker Compose `--profile mcp`)
* **Run MCP Server**: `make mcp` *(added by RFC-001)* (Runs `geomilli-mcp` locally in stdio mode)
* **Inspect MCP Server**: `make mcp-inspect` *(added by RFC-001)* (Launches MCP Inspector against local `geomilli-mcp`)
* **Stop Full Cluster**: `make down`
* **Run Test Suites**: `make test` (Tests `geomilli-harvester`, `geomilli-api`, and validates documentation build)
* **Build All Binaries & Docs**: `make build` (Compiles Go services, builds docs, and syncs to Studio UI)
* **Build & Sync Documentation**: `make docs-build` (Builds VitePress static site and copies to `geomilli-studio/public/docs/`)
* **Local Docs Dev Server**: `make docs` (Runs local VitePress live preview)

---

## 3. Architectural Constraints & Rules

### A. Osmium C++ Tool Requirement
* **Local Development**: Requires `osmium` CLI installed natively (`brew install osmium-tool` on macOS).
* **Production / Docker**: Containers have `osmium-tool` pre-installed. Do not use Docker-in-Docker wrappers.

### B. SQLite Concurrency (WAL Mode & Busy Timeout)
* The Wikidata and Wikipedia pageviews enrichment cache uses local SQLite databases (`cache.db`).
* **Rule**: SQLite connections must always initialize with WAL mode and a busy timeout:
  ```go
  db.Exec("PRAGMA journal_mode = WAL;")
  db.Exec("PRAGMA busy_timeout = 5000;")
  ```
  This prevents `SQLITE_BUSY` (database is locked) errors during concurrent pipeline execution.

### C. Config-Driven Taxonomy & Language Whitelist
* **Taxonomy**: Matching rules, priority ordering, attributes, and base scores are strictly defined in `config/taxonomy.yaml` (19 canonical classes). **Never hardcode class logic, switch cases, or string matches in Go code.**
* **Languages**: Supported languages are whitelisted in `config/languages.yaml`. Any language code not in the whitelist must be stripped during name normalization to prevent payload and index bloat.
* **Sync Rule**: Keep `config/taxonomy.yaml` and `config/languages.yaml` synchronized across `geomilli-harvester` and `geomilli-api`.

### D. Meilisearch Schema & Searchable Attributes
* **Index Configuration**: Configured in `geomilli-harvester/internal/meili/indexer.go`:
  * `SearchableAttributes`:
    ```go
    []string{
        "name_display",
        "name_alt_active",
        "iata_code",
        "icao_code",
        "class",
        "subclass",
        "admin_hierarchy",
        "country_name",
    }
    ```
    *Note: Keeping `iata_code`, `icao_code`, `class`, `subclass`, and `country_name` searchable ensures users can search airports by code (e.g. `BRU`, `CRL`), search by category (`Stadium`, `Beach`, `TrainStation`), and search by country name.*
  * `FilterableAttributes`:
    ```go
    []string{
        "class",
        "subclass",
        "country_code",
        "admin_level",
        "_geo",
        "star_rating",
        "cuisine",
        "iata_code",
        "icao_code",
        "accommodation_type",
    }
    ```
* **Blue/Green Atomic Index Swapping**:
  * Build new data into an isolated shadow index: `places__build_<timestamp>`.
  * On completion, swap atomically into `places` via `POST /swap-indexes`.
  * Clean up temporary shadow build indices in a Go `defer` block to prevent dangling indices if an error occurs.

### E. Enrichment Resiliency & Circuit Breaker
* Handle Wikimedia Pageviews `404 Not Found` API responses silently (`return 0, nil`) to avoid log flooding.
* Automatically trip the circuit breaker and disable online HTTP enrichment after 5 consecutive network errors, falling back to cached or un-enriched records.

### F. Studio UI & API Routing
* **API Origin**: The Go API backend runs on `:8080`.
* In `geomilli-studio`, Vite / Nginx proxies `/v1` requests to the Go API.
* The cURL generator in `PlaygroundTab.tsx` must point to the backend API origin (`http://localhost:8080`), not the UI port (`:3000` / `:3333`).

### G. Scalability & Instance Constraints
* **`geomilli-api`**: Stateless service with zero in-memory locking. Can scale horizontally without limitations across multiple instances behind a load balancer.
* **`geomilli-harvester`**: Single-instance bound. Relies on local NVMe disk filesystem (`raw/`, `staging/`) and an embedded SQLite database (`cache.db`) for enrichment caching. **Cannot run multiple concurrent instances** (to prevent SQLite lock conflicts and file races). Scale vertically with fast NVMe disk IO and multi-core CPU, adjusting internal thread parallelism via `HARVESTER_CONCURRENCY`.

### H. MCP Server Constraints
* **Adapter-Only Architecture**: `geomilli-mcp` is an adapter service. It must **only** communicate with the upstream `geomilli-api` REST interface (`/v1/*`), never with Meilisearch or SQLite directly.
* **No Raw Filter Exposure**: Never expose raw Meilisearch filter syntax or query DSL to LLMs. All tool inputs must use structured parameters; the API handles filter translation.
* **Config-Derived Tool Enums**: Tool input schemas and enums (e.g. `class`, `subclass`, `lang`) must be derived dynamically from `/v1/taxonomy` and `/v1/languages` at startup/cache refresh, never hardcoded in Go code.
* **stdio Protocol Safety**: All application logging must go strictly to `stderr` (`log/slog`). Writing anything to `stdout` corrupts the stdio MCP protocol transport.
* **Gated Operations Tools**: Operations tools are disabled by default and gated behind `MCP_ENABLE_OPS_TOOLS=true`. In v1, only read-only ops tools (`get_service_status`, `list_import_queue`, `list_importable_countries`) are exposed.
* **Stateless & Horizontally Scalable**: `geomilli-mcp` maintains zero in-memory business state and can scale horizontally across multiple instances or processes behind a load balancer.
* **Tool Annotations Required**: Every tool definition must explicitly declare its behavior annotations: `readOnlyHint`, `destructiveHint`, `idempotentHint`, and `openWorldHint`.

---

## 4. Documentation Maintenance Protocol

To ensure developer documentation (`docs/`) and Studio UI documentation never become stale:

### When to Update Documentation:
1. **API Endpoints or Query Parameters Change**:
   - Update `docs/api/overview.md`, `docs/api/autocomplete-and-search.md`, and `docs/api/geospatial-apis.md`.
   - Keep `geomilli-api/api/openapi.yaml` (canonical OpenAPI 3.1 specification) strictly synchronized.
   - Assess MCP impact: update `geomilli-mcp` tool mappings, `docs/mcp/tools-reference.md`, and golden schema snapshots.
2. **MCP Tools, Resources, or Prompts Change**:
   - Update `docs/mcp/*` (`overview.md`, `client-setup.md`, `tools-reference.md`, `resources-and-prompts.md`, `security-and-deployment.md`).
3. **Taxonomy or Language Rules Change**: Update `docs/harvester/taxonomy-and-languages.md`.
4. **Pipeline or Ingestion Engine Changes**: Update `docs/guide/architecture.md` and `docs/guide/streaming-and-memory.md`.
5. **Studio Features or Controls Change**: Update `docs/studio/playground-and-metrics.md`.
6. **Every User-Visible Change**:
   - Add a concise entry under `## [Unreleased]` in `CHANGELOG.md` prefixed with the appropriate component tag (`**api**`, `**mcp**`, `**harvester**`, `**studio**`, `**docs**`, `**infra**`).

### Execution Steps:
1. **Edit Markdown Files**: Make corresponding documentation updates in `docs/`.
2. **Mermaid Diagrams**: Any diagrams must use standard ````mermaid` blocks (client-side rendering is handled automatically via `vitepress-plugin-mermaid`).
3. **Rebuild & Synchronize**:
   ```bash
   make docs-build
   ```
   This command builds the VitePress static bundle and synchronizes `docs/.vitepress/dist/` into `geomilli-studio/public/docs/`.
4. **Verify**: Run `make test` to verify that all unit tests and the documentation build pass with zero errors.

---

## 5. Definition of Done (DoD) Checklist

Before reporting completion or opening a pull request, all agents and contributors must self-verify this checklist:

- [ ] **Architectural Compliance**: Code adheres to monorepo rules (WAL mode with 5000ms timeout for SQLite, config-driven taxonomy/languages, adapter-only MCP architecture).
- [ ] **Tests Pass**: Module tests and test suites pass locally (`go test -race ./...` for modified Go services; `npm run build` for studio/docs; `make test`).
- [ ] **Documentation Updated**: Developer documentation in `docs/` is updated if endpoints, schemas, parameters, or configurations changed.
- [ ] **OpenAPI Spec Synchronized**: `geomilli-api/api/openapi.yaml` matches the exact router endpoints, parameter names, limits, and response structures.
- [ ] **MCP Impact Assessed**: If API endpoints or parameters changed, `geomilli-mcp` tool mappings, schemas, and `docs/mcp/` documentation are updated.
- [ ] **Changelog Entry Added**: An entry is appended under `## [Unreleased]` in `CHANGELOG.md` with the corresponding component tag (`**api**`, `**mcp**`, `**harvester**`, `**studio**`, `**docs**`, `**infra**`), or `skip-changelog` is justified.
- [ ] **Docs Build Verified**: `make docs-build` succeeds with zero errors, and static assets sync cleanly to `geomilli-studio/public/docs/`.

---

## 6. Architecture RFCs

Major architectural proposals, designs, and delivery plans are maintained under `.agents/rfcs/`:
* [RFC-001: Geomilli MCP Server](file:///.agents/rfcs/RFC-001-mcp-server.md) — Architecture, tools catalog, and delivery plan for the Model Context Protocol adapter service.
