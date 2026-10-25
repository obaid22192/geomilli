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
* **Start with MCP**: `make up-mcp` *(added by RFC-001)* (Docker Compose cluster + MCP server)
* **Run MCP Server**: `make mcp` *(added by RFC-001)* (runs `geomilli-mcp` in stdio mode)
* **Inspect MCP Server**: `make mcp-inspect` *(added by RFC-001)* (launches MCP Inspector)
* **Stop Services**: `make down`

### Key Architectural Constraints
1. **Config-Driven**: Class matching, taxonomy definitions (19 canonical classes), and language whitelists are defined in `config/taxonomy.yaml` and `config/languages.yaml`. Do not hardcode class logic in Go code.
2. **SQLite WAL Mode**: Local enrichment databases must always enable WAL mode and a 5000ms busy timeout.
3. **Meilisearch Schema**:
   * `SearchableAttributes`: `name_display`, `name_alt_active`, `iata_code`, `icao_code`, `class`, `subclass`, `admin_hierarchy`, `country_name`.
   * `FilterableAttributes`: `class`, `subclass`, `country_code`, `admin_level`, `_geo`, `star_rating`, `cuisine`, `iata_code`, `icao_code`, `accommodation_type`.
4. **Blue/Green Swapping**: Use shadow indices (`places__build_<timestamp>`) and swap atomically via `POST /swap-indexes`. Always clean up temporary indices in a `defer` block.
5. **Scalability Boundaries**: `geomilli-api` and `geomilli-mcp` scale horizontally without limitations. `geomilli-harvester` is single-instance bound to local disk and SQLite; scale it vertically (fast NVMe, CPU, `HARVESTER_CONCURRENCY`).
6. **MCP Server Constraints**: `geomilli-mcp` is an adapter service communicating strictly with `geomilli-api` REST routes (never Meilisearch/SQLite directly). Never expose raw filter DSL. Enums are dynamically derived from `/v1/taxonomy` and `/v1/languages`. Logs go strictly to `stderr` (stdio transport safety). Ops tools are gated behind `MCP_ENABLE_OPS_TOOLS` (read-only in v1). Every tool has annotations.
7. **Canonical OpenAPI Spec**: Keep `geomilli-api/api/openapi.yaml` strictly in sync with all API routes, parameters, and envelopes.
8. **Mandatory Changelog Policy**: Every user-visible or contract change must add an entry under `## [Unreleased]` in `CHANGELOG.md` with a component tag (`**api**`, `**mcp**`, `**harvester**`, `**studio**`, `**docs**`, `**infra**`). Verified by CI.
9. **Documentation Protocol**: Whenever API endpoints, MCP tools, taxonomy, or pipeline features change, update `docs/`, run `make docs-build`, and verify with `make test`.

### Definition of Done (DoD)
Before completing any task, self-verify:
- [ ] Code adheres to architectural constraints (WAL mode, config-driven taxonomy, adapter-only MCP).
- [ ] Tests pass (`make test` or `go test -race ./...`).
- [ ] Documentation updated in `docs/` and verified with `make docs-build`.
- [ ] OpenAPI spec (`geomilli-api/api/openapi.yaml`) synchronized with API changes.
- [ ] MCP impact evaluated and updated if API endpoints or schemas changed.
- [ ] Entry added under `## [Unreleased]` in `CHANGELOG.md` with appropriate component tag.

For complete architectural details, see [.agents/AGENTS.md](file:///.agents/AGENTS.md) and [.agents/rfcs/RFC-001-mcp-server.md](file:///.agents/rfcs/RFC-001-mcp-server.md).
