# Service Overview: What We Are Trying to Achieve

This document captures the intent and goals of the `auto-complete` service.

## One-line summary

Build and maintain clean, searchable Meilisearch datasets that will power a
travel **autosuggest / autocomplete** experience — so a user typing a few
characters instantly gets relevant places, airports, hotels, and tourist attractions.

## The product goal

When a user starts typing in a travel search box, they should get fast,
relevant suggestions for:

- **Places**: cities, regions, neighborhoods, islands.
- **Airports**: airport and airfield locations (enriched with IATA codes).
- **Accommodations**: hotels, hostels, apartments, guest houses.
- **Discovery**: tourist attractions, stadiums, historic sites, parks, restaurants, shopping malls.

This repo owns the **data side** of that experience. The actual HTTP
autosuggest API that serves queries to end users is intentionally **out of
scope** here. We are building the pipeline that keeps the search indices
correct, fresh, and production-ready.

## What this service actually does

It is a **data pipeline**, not a web service. It runs in stages:

1. **Fetch** raw source data
   - OSM (Geofabrik country extracts) for places, admin boundaries, and POIs.
   - Uses ETag checks and local manifests so unchanged countries are skipped entirely.

2. **Transform** raw data into normalized NDJSON documents
   - Executed per-country to allow horizontal scalability and incremental build caching.
   - Extracts 19 distinct entity classes, mapping custom fields like `iata_code`, `star_rating`, and `cuisine`.
   - Normalizes multilingual names (`name:*` tags) and alt names into a searchable flat index.

3. **Enrich** place records
   - Adds popularity/quality signals (Wikidata/Wikipedia-derived pageviews and sitelinks) via a local SQLite cache.
   - Unchanged country builds reuse their previously enriched files, skipping CPU-intensive network/database hits.

4. **Index** into Meilisearch using blue/green deploys
   - Streams all per-country enriched documents directly into one shadow build index.
   - Builds a shadow index, validates it (golden queries + doc counts), then atomically swaps it into the live index name.
   - Live index never serves partial/broken data.

5. **Cleanup / operate**
   - Retain only recent shadow builds, prune old manifests.

## Why it is built this way (the real objectives)

- **Relevance**: normalize multilingual names, alt names, and search terms so suggestions match how people actually type.
- **Freshness**: re-runnable on a schedule without manual babysitting.
- **Planet-Scale Readiness**: Incremental country builds allow processing the entire planet in small, cached slices instead of one giant file.
- **Safety / zero-downtime updates**: blue/green indexing means a bad build never replaces a good live index.
- **Correctness guarantees**: golden-query validation + non-empty checks catch broken data before it goes live.
- **Operational resilience**: locks prevent duplicate runs, and cached fallbacks survive upstream outages.

## What "done / healthy" looks like

- A live `places` index exists with a non-trivial doc count and passes its golden queries.
- All document IDs are valid Meilisearch IDs.
- Old shadow `__build_` indices are pruned.
- `make run` completes end-to-end without manual intervention.

## Out of scope (for this repo)

- The user-facing autosuggest HTTP API / query service.
- Frontend search UI.
- Ranking experiments beyond the baseline Meilisearch settings.
