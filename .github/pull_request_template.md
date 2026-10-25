## Summary

<!-- Briefly describe the purpose of this PR and what changes were made. -->

## Affected Component(s)

<!-- Select all components affected by this pull request -->
- [ ] `api` (Search & Geo REST API)
- [ ] `mcp` (Model Context Protocol server)
- [ ] `harvester` (Streaming OSM ingestion & enrichment)
- [ ] `studio` (Interactive developer console UI)
- [ ] `docs` (Documentation site & OpenAPI specification)
- [ ] `infra` (Docker Compose, CI workflows, build scripts)

## Checklist

Before submitting this PR, please confirm that you have completed the following:

- [ ] **Changelog**: I have added an entry under `## [Unreleased]` in `CHANGELOG.md` with the appropriate component tag (`**api**`, `**mcp**`, `**harvester**`, `**studio**`, `**docs**`, `**infra**`), OR I have applied the `skip-changelog` label with a justification below.
- [ ] **Documentation**: Developer documentation in `docs/` has been updated (or no documentation updates were required).
- [ ] **OpenAPI Specification**: `geomilli-api/api/openapi.yaml` has been updated to reflect any API route, parameter, or envelope changes.
- [ ] **MCP Impact**: If API endpoints or parameters changed, MCP tool mappings, schemas, and `docs/mcp/` documentation were evaluated and updated.
- [ ] **Testing**: All relevant unit tests pass locally (`go test -race ./...` or `make test`).
- [ ] **Docs Verification**: Documentation builds cleanly (`make docs-build`).

### Skip Changelog Justification (if applicable)

<!-- If this PR does not modify CHANGELOG.md, explain why it qualifies for the skip-changelog label (e.g. chore, internal test fix). -->
