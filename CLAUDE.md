# CLAUDE.md — Geomilli Project Guidelines

> [!NOTE]
> The single authoritative source of truth for repository architecture, build commands, constraints, and documentation protocols is maintained in [AGENTS.md](file:///.agents/AGENTS.md).
> All AI agents (Claude, Antigravity, etc.) must adhere to the rules outlined in that file.

---

## Quick Reference

### Core Commands
* **Run Test Suites**: `make test` (tests `geomilli-harvester`, `geomilli-api`, and validates docs build)
* **Build Binaries & Docs**: `make build`
* **Build & Sync Docs**: `make docs-build` (compiles VitePress site and syncs to `geomilli-studio/public/docs/`)
* **Start Services**: `make up` (Docker Compose local cluster)
* **Stop Services**: `make down`

### Key Architectural Constraints
1. **Config-Driven**: Class matching, taxonomy definitions (19 canonical classes), and language whitelists are defined in `config/taxonomy.yaml` and `config/languages.yaml`. Do not hardcode class logic in Go code.
2. **SQLite WAL Mode**: Local enrichment databases must always enable WAL mode and a 5000ms busy timeout.
3. **Meilisearch Schema**:
   * `SearchableAttributes`: `name_display`, `name_alt_active`, `iata_code`, `icao_code`, `class`, `subclass`, `admin_hierarchy`, `country_name`.
   * `FilterableAttributes`: `class`, `subclass`, `country_code`, `admin_level`, `_geo`, `star_rating`, `cuisine`, `iata_code`, `icao_code`, `accommodation_type`.
4. **Blue/Green Swapping**: Use shadow indices (`places__build_<timestamp>`) and swap atomically via `POST /swap-indexes`. Always clean up temporary indices in a `defer` block.
5. **Scalability Boundaries**: `geomilli-api` scales horizontally without limitations. `geomilli-harvester` is single-instance bound to local disk and SQLite; scale it vertically (fast NVMe, CPU, `HARVESTER_CONCURRENCY`).
6. **Documentation Protocol**: Whenever API endpoints, taxonomy, or pipeline features are modified, update `docs/`, run `make docs-build`, and verify with `make test`.

For complete architectural details, see [.agents/AGENTS.md](file:///.agents/AGENTS.md).
