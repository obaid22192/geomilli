# Meta Prompt: Meilisearch Index Audit & Program Correctness Check

Copy everything in the fenced block below into a new chat session. It gives the
assistant enough context about this pipeline to (1) list the indices on the
local Meilisearch instance and (2) verify the program is behaving correctly.

---

```text
You are auditing a Go data pipeline called "auto-complete" that builds two
Meilisearch datasets for a travel autosuggest API.

## Workspace
- Repo root: /Users/a670090449/Documents/travayo-v2/auto-complete
- Language: Go 1.25 (uses modernc.org/sqlite, meilisearch-go)
- Search engine: Meilisearch running locally
- Config is loaded from `.env` at repo root.
  - MEILI_HOST (default http://127.0.0.1:7700)
  - MEILI_API_KEY (read from .env; do NOT print the key value back to me)

## What the pipeline produces
Two live indices:
- `places`  : cities, regions, neighborhoods, airports, train stations,
              attractions, stadiums (OSM + OurAirports sources)
- `hotels`  : hotel properties (Agoda feed)

## How indexing works (important for correctness reasoning)
- The indexer uses BLUE/GREEN indexing.
- For each run it builds a shadow index named `<base>__build_<UTCtimestamp>`,
  e.g. `places__build_20260616231238`.
- It uploads docs to the shadow, waits for Meilisearch task completion,
  runs golden-query validation, then atomically swaps the shadow into the
  live base name via the `/swap-indexes` endpoint.
- Hotels indexing runs with `--allow-empty=true`: if 0 docs are produced it
  SKIPS validation and swap (so a live `hotels` index may legitimately not
  exist yet if the hotels feed has produced 0 rows).
- Orchestrator entrypoint: `go run ./cmd/pipeline --stage run --target all`
  (or `make run`). Stages: fetch | build-data | index | run | cleanup.
- Cleanup entrypoint: `make cleanup` -> `cmd/cleanup`.
  - Groups indices by `<base>__build_` prefix.
  - Keeps the newest N shadow builds per base (`--keep-builds`, default 3).
  - Deletes older shadow builds. It does NOT delete live base indices.
  - Also prunes manifests in `data/manifests/` older than `--keep-manifest-days`.

## Known problem to investigate
The pipeline has been run many times and has accumulated too many leftover
`__build_<timestamp>` shadow indices that were never cleaned up. There may also
be `hotels__build_*` shadows that were never swapped to a live `hotels` index
because the hotels feed produced 0 documents.

## Your tasks
1. Fetch the list of index names on the local Meilisearch.
   - Read MEILI_HOST and MEILI_API_KEY from `.env`.
   - Call `GET {MEILI_HOST}/indexes?limit=1000` with
     `Authorization: Bearer {MEILI_API_KEY}` if the key is set.
   - Print, for each index: uid, primaryKey, and numberOfDocuments
     (numberOfDocuments comes from `GET {host}/indexes/{uid}/stats`).

2. Classify each index as:
   - LIVE base (`places`, `hotels`)
   - SHADOW build (`<base>__build_<timestamp>`)
   - UNKNOWN/orphan (anything else)

3. Check program correctness against these expectations:
   - A live `places` index exists and has a non-trivial document count
     (expected on the order of ~19k docs for the configured countries; treat
     0 docs as a failure).
   - `places` primaryKey is `id`.
   - Document IDs are valid Meilisearch IDs (alphanumeric, `-`, `_`, <=511
     bytes). Sample a few docs via
     `POST {host}/indexes/places/search` with body {"q":"","limit":5} and
     confirm each `id` looks valid (no `<nil>`, no spaces/slashes).
   - Run the golden queries in `config/golden_queries_places.txt`
     (format: `query|expected_token_1,expected_token_2`, `#` = comment) against
     the live `places` index and confirm each query returns hits containing the
     expected tokens.
   - Count shadow `__build_` indices per base. If more than 3 per base exist,
     report it as cleanup debt.
   - Note whether a live `hotels` index exists. If only `hotels__build_*`
     shadows exist and no live `hotels`, explain that this is expected when the
     hotels feed yields 0 docs (allow-empty mode) — not necessarily a bug.

4. Produce a concise report:
   - Table of all indices (uid | type | docs | primaryKey)
   - PASS/FAIL for each correctness expectation above
   - Cleanup recommendation. To prune shadows safely, the intended command is:
       make cleanup            # keeps newest 3 builds per base
     or to keep only the latest build:
       go run ./cmd/cleanup --keep-builds 1
   - Do NOT delete the live `places` (or live `hotels`) index.

## Guardrails
- Treat the local Meilisearch as the source of truth; the `.env` host/key are
  the connection details.
- Never print the API key value.
- Do not delete any index unless I explicitly ask. First show me the report and
  the proposed cleanup command.
```

---

## Quick reference (for the human, not the prompt)

- List indices manually:
  ```bash
  KEY=$(grep -E '^MEILI_API_KEY=' .env | cut -d= -f2-)
  HOST=$(grep -E '^MEILI_HOST=' .env | cut -d= -f2-)
  curl -s -H "Authorization: Bearer ${KEY}" "${HOST}/indexes?limit=1000"
  ```
- Prune leftover shadow indices (keeps newest 3 per base by default):
  ```bash
  make cleanup
  # or keep only the newest build per base:
  go run ./cmd/cleanup --keep-builds 1
  ```
- Cleanup never removes live base indices (`places`, `hotels`); it only trims
  `<base>__build_<timestamp>` shadows.
