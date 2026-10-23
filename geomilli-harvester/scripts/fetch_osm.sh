#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "${ROOT_DIR}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "${ROOT_DIR}/.env"
  set +a
fi
CONFIG_FILE="${ROOT_DIR}/config/countries.yaml"
RAW_DIR="${ROOT_DIR}/raw/osm"
GEOFABRIK_BASE_URL="${OSM_GEOFABRIK_BASE_URL:-https://download.geofabrik.de}"
CURL_RETRY_ARGS=(--retry 5 --retry-all-errors --retry-delay 2 --retry-max-time 300)
CURL_PROBE_ARGS=(--retry 3 --retry-all-errors --retry-delay 2 --connect-timeout 15 --max-time 30)
MANIFEST_DIR="${ROOT_DIR}/data/manifests"
LOCK_DIR="${ROOT_DIR}/data/locks"
mkdir -p "${RAW_DIR}" "${MANIFEST_DIR}" "${LOCK_DIR}"

# Helper to calculate SHA256 across Linux (sha256sum), macOS (shasum), or openssl
calc_sha256() {
  local target="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${target}" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${target}" | awk '{print $1}'
  else
    openssl dgst -sha256 "${target}" | awk '{print $NF}'
  fi
}

# Helper to invoke the countries binary or fallback to go run
run_countries() {
  if command -v countries >/dev/null 2>&1; then
    countries "$@"
  elif [[ -x "/usr/local/bin/countries" ]]; then
    /usr/local/bin/countries "$@"
  elif [[ -x "${ROOT_DIR}/bin/countries" ]]; then
    "${ROOT_DIR}/bin/countries" "$@"
  elif command -v go >/dev/null 2>&1; then
    (cd "${ROOT_DIR}" && go run ./cmd/countries "$@")
  else
    echo "[fetch_osm] ERROR: neither countries binary nor go compiler found" >&2
    exit 1
  fi
}

# Helper to invoke the gen-filters binary or fallback to go run
run_gen_filters() {
  if command -v gen-filters >/dev/null 2>&1; then
    gen-filters "$@"
  elif [[ -x "/usr/local/bin/gen-filters" ]]; then
    /usr/local/bin/gen-filters "$@"
  elif [[ -x "${ROOT_DIR}/bin/gen-filters" ]]; then
    "${ROOT_DIR}/bin/gen-filters" "$@"
  elif command -v go >/dev/null 2>&1; then
    (cd "${ROOT_DIR}" && go run ./cmd/gen-filters "$@")
  else
    echo "[fetch_osm] ERROR: neither gen-filters binary nor go compiler found" >&2
    exit 1
  fi
}

# Require osmium to be installed locally (brew install osmium-tool).
# In production (K8s/Docker), the container image has osmium pre-installed.
if ! command -v osmium >/dev/null 2>&1; then
  echo "[fetch_osm] ERROR: osmium binary not found." >&2
  echo "[fetch_osm] Install it with: brew install osmium-tool" >&2
  exit 1
fi
echo "[fetch_osm] using local osmium binary ($(osmium --version 2>&1 | head -1))"

suffix="global"
if [[ $# -gt 0 ]]; then
  suffix="$(echo "$1" | tr '[:upper:]' '[:lower:]')"
fi
LOCK_FILE="${LOCK_DIR}/fetch-osm-${suffix}.lock"
if ! ( set -o noclobber; echo "$$" >"${LOCK_FILE}" ) 2>/dev/null; then
  lock_pid="$(cat "${LOCK_FILE}" 2>/dev/null || true)"
  if [[ -n "${lock_pid}" ]] && kill -0 "${lock_pid}" 2>/dev/null; then
    echo "[fetch_osm] lock already held: ${LOCK_FILE} (pid=${lock_pid})" >&2
    exit 1
  fi
  echo "[fetch_osm] removing stale lock: ${LOCK_FILE}" >&2
  rm -f "${LOCK_FILE}"
  if ! ( set -o noclobber; echo "$$" >"${LOCK_FILE}" ) 2>/dev/null; then
    echo "[fetch_osm] unable to acquire lock: ${LOCK_FILE}" >&2
    exit 1
  fi
fi
trap 'rm -f "${LOCK_FILE}"' EXIT

filter_args=()
if [[ -n "${COUNTRIES:-}" ]]; then
  filter_args+=(--countries "${COUNTRIES}")
fi
if [[ $# -gt 0 ]]; then
  filter_args+=(--countries "$1")
fi

while IFS='|' read -r code country_name geopath; do
  [[ -z "${code}" || -z "${geopath}" ]] && continue
  lc_code="$(echo "${code}" | tr '[:upper:]' '[:lower:]')"
  url="${GEOFABRIK_BASE_URL}/${geopath}-latest.osm.pbf"

  src_pbf="${RAW_DIR}/${lc_code}-latest.osm.pbf"
  admin_pbf="${RAW_DIR}/${lc_code}-admin.osm.pbf"
  pois_pbf="${RAW_DIR}/${lc_code}-pois.osm.pbf"
  admin_geojsonseq="${RAW_DIR}/${lc_code}-admin.geojsonseq"
  pois_geojsonseq="${RAW_DIR}/${lc_code}-pois.geojsonseq"
  etag_file="${RAW_DIR}/${lc_code}-latest.osm.pbf.etag"
  manifest_file="${MANIFEST_DIR}/fetch-osm-${lc_code}.json"

  remote_etag="$(curl -fsSI "${CURL_PROBE_ARGS[@]}" "${url}" | tr -d '\r' | awk -F': ' 'tolower($1)=="etag" {print $2; exit}' || true)"
  local_etag=""
  if [[ -f "${etag_file}" ]]; then
    local_etag="$(cat "${etag_file}")"
  fi
  manifest_etag="${local_etag:-}"

  if [[ -f "${src_pbf}" && -f "${admin_geojsonseq}" && -f "${pois_geojsonseq}" && -f "${manifest_file}" ]]; then
    if [[ -n "${remote_etag}" && "${remote_etag}" == "${local_etag}" ]]; then
      echo "[fetch_osm] ${code} unchanged ETag and outputs exist, skipping processing entirely"
      continue
    elif [[ -z "${remote_etag}" ]]; then
      echo "[fetch_osm] ${code} ETag unavailable but outputs exist, skipping processing entirely"
      continue
    fi
  fi

  if [[ -n "${remote_etag}" && "${remote_etag}" == "${local_etag}" && -f "${src_pbf}" ]]; then
    echo "[fetch_osm] ${code} unchanged ETag, skipping download"
    manifest_etag="${remote_etag}"
  elif [[ -z "${remote_etag}" && -f "${src_pbf}" ]]; then
    echo "[fetch_osm] ${code} ETag unavailable, reusing cached snapshot ${src_pbf}"
  else
    echo "[fetch_osm] downloading ${code} from ${url}"
    download_tmp="${src_pbf}.download"
    if curl -fL "${CURL_RETRY_ARGS[@]}" "${url}" -o "${download_tmp}"; then
      mv "${download_tmp}" "${src_pbf}"
      if [[ -n "${remote_etag}" ]]; then
        echo "${remote_etag}" > "${etag_file}"
        manifest_etag="${remote_etag}"
      fi
    else
      rm -f "${download_tmp}"
      if [[ -f "${src_pbf}" ]]; then
        echo "[fetch_osm] download failed, reusing cached snapshot ${src_pbf}" >&2
      else
        exit 1
      fi
    fi
  fi

  echo "[fetch_osm] filtering and exporting data (${code})"
  admin_filters="$(run_gen_filters --taxonomy "${CONFIG_FILE%/*}/taxonomy.yaml" --layer admin)"
  poi_filters="$(run_gen_filters --taxonomy "${CONFIG_FILE%/*}/taxonomy.yaml" --layer poi)"

  # Clean intermediate files before running osmium to avoid stale data
  rm -f "${admin_pbf}" "${pois_pbf}" "${admin_geojsonseq}" "${pois_geojsonseq}"

  # shellcheck disable=SC2086
  osmium tags-filter "${src_pbf}" ${admin_filters} -o "${admin_pbf}" --overwrite
  # shellcheck disable=SC2086
  osmium tags-filter "${src_pbf}" ${poi_filters} -o "${pois_pbf}" --overwrite
  osmium export "${admin_pbf}" -f geojsonseq -o "${admin_geojsonseq}" --overwrite
  osmium export "${pois_pbf}" -f geojsonseq -o "${pois_geojsonseq}" --overwrite

  src_sha256="$(calc_sha256 "${src_pbf}")"
  src_size="$(wc -c < "${src_pbf}" | tr -d ' ')"
  admin_sha256="$(calc_sha256 "${admin_geojsonseq}")"
  pois_sha256="$(calc_sha256 "${pois_geojsonseq}")"
  updated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  cat > "${manifest_file}" <<EOF
{
  "job": "fetch-osm",
  "country_code": "${code}",
  "url": "${url}",
  "source_pbf": "raw/osm/${lc_code}-latest.osm.pbf",
  "source_sha256": "${src_sha256}",
  "source_size_bytes": ${src_size},
  "admin_geojsonseq": "raw/osm/${lc_code}-admin.geojsonseq",
  "admin_geojsonseq_sha256": "${admin_sha256}",
  "pois_geojsonseq": "raw/osm/${lc_code}-pois.geojsonseq",
  "pois_geojsonseq_sha256": "${pois_sha256}",
  "etag": "${manifest_etag}",
  "updated_at": "${updated_at}"
}
EOF

done < <(run_countries --config "${CONFIG_FILE}" --base-url "${GEOFABRIK_BASE_URL}" ${filter_args[@]+"${filter_args[@]}"})

echo "[fetch_osm] complete"
