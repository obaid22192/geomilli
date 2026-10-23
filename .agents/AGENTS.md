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
| **Studio UI** | `geomilli-studio/` | Interactive developer console, MapLibre GL playground, importer monitor, telemetry. | React 19, TypeScript, Tailwind CSS, Vite |

---

## 2. Core Build & Development Commands

Always execute commands from the repository root:

* **Start Full Cluster**: `make up` (Runs Meilisearch, Harvester daemon, API service, and Studio UI)
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

---

## 4. Documentation Maintenance Protocol

To ensure developer documentation (`docs/`) and Studio UI documentation never become stale:

### When to Update Documentation:
1. **API Endpoints or Query Parameters Change**: Update `docs/api/overview.md`, `docs/api/autocomplete-and-search.md`, and `docs/api/geospatial-apis.md`.
2. **Taxonomy or Language Rules Change**: Update `docs/harvester/taxonomy-and-languages.md`.
3. **Pipeline or Ingestion Engine Changes**: Update `docs/guide/architecture.md` and `docs/guide/streaming-and-memory.md`.
4. **Studio Features or Controls Change**: Update `docs/studio/playground-and-metrics.md`.

### Execution Steps:
1. **Edit Markdown Files**: Make corresponding documentation updates in `docs/`.
2. **Mermaid Diagrams**: Any diagrams must use standard ````mermaid` blocks (client-side rendering is handled automatically via `vitepress-plugin-mermaid`).
3. **Rebuild & Synchronize**:
   ```bash
   make docs-build
   ```
   This command builds the VitePress static bundle and synchronizes `docs/.vitepress/dist/` into `geomilli-studio/public/docs/`.
4. **Verify**: Run `make test` to verify that all unit tests and the documentation build pass with zero errors.
