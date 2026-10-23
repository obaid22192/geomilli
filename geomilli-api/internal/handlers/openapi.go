package handlers

import (
	_ "embed"
	"net/http"
)

// openAPISpec defines the OpenAPI 3.1 specification for Geomilli API.
const openAPISpec = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Geomilli OpenStreetMap & Geospatial API",
    "version": "1.0.0",
    "description": "High-performance, sub-10ms OpenStreetMap geospatial search, autocomplete, reverse geocoding, and POI intelligence API powered by Meilisearch."
  },
  "servers": [
    {
      "url": "http://localhost:8080",
      "description": "Local Geomilli API"
    }
  ],
  "paths": {
    "/v1/autocomplete": {
      "get": {
        "summary": "Prefix autosuggest",
        "description": "Sub-10ms prefix autosuggestion with multilingual support and geo-proximity boost.",
        "parameters": [
          {"name": "q", "in": "query", "schema": {"type": "string"}, "description": "Prefix query string"},
          {"name": "lang", "in": "query", "schema": {"type": "string"}, "description": "Language code (e.g. en, fr, nl)"},
          {"name": "country", "in": "query", "schema": {"type": "string"}, "description": "Country ISO code (e.g. BE)"},
          {"name": "class", "in": "query", "schema": {"type": "string"}, "description": "Taxonomy class (e.g. City, Airport, Hotel)"},
          {"name": "subclass", "in": "query", "schema": {"type": "string"}, "description": "Taxonomy subclass"},
          {"name": "lat", "in": "query", "schema": {"type": "number"}, "description": "Latitude for proximity sorting"},
          {"name": "lng", "in": "query", "schema": {"type": "number"}, "description": "Longitude for proximity sorting"},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 10}, "description": "Max results"}
        ],
        "responses": {
          "200": {"description": "Autocomplete results"}
        }
      }
    },
    "/v1/search": {
      "get": {
        "summary": "Full-text & faceted search",
        "parameters": [
          {"name": "q", "in": "query", "schema": {"type": "string"}},
          {"name": "facets", "in": "query", "schema": {"type": "string"}, "description": "Comma-separated facet fields or *"},
          {"name": "filter", "in": "query", "schema": {"type": "string"}},
          {"name": "sort", "in": "query", "schema": {"type": "string"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 20}},
          {"name": "offset", "in": "query", "schema": {"type": "integer", "default": 0}}
        ],
        "responses": {
          "200": {"description": "Search results with facet distributions"}
        }
      }
    },
    "/v1/reverse": {
      "get": {
        "summary": "Reverse geocoding",
        "parameters": [
          {"name": "lat", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "lng", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "radius_meters", "in": "query", "schema": {"type": "integer", "default": 50000}}
        ],
        "responses": {
          "200": {"description": "Enclosing admin hierarchy and nearest places"}
        }
      }
    },
    "/v1/nearby": {
      "get": {
        "summary": "Nearby POIs (Geo-Radius)",
        "parameters": [
          {"name": "lat", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "lng", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "radius_meters", "in": "query", "schema": {"type": "integer", "default": 5000}},
          {"name": "class", "in": "query", "schema": {"type": "string"}},
          {"name": "subclass", "in": "query", "schema": {"type": "string"}},
          {"name": "cuisine", "in": "query", "schema": {"type": "string"}},
          {"name": "min_stars", "in": "query", "schema": {"type": "integer"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 20}},
          {"name": "offset", "in": "query", "schema": {"type": "integer", "default": 0}}
        ],
        "responses": {
          "200": {"description": "Nearby places sorted by distance"}
        }
      }
    },
    "/v1/bbox": {
      "get": {
        "summary": "Map viewport bounding box",
        "parameters": [
          {"name": "min_lat", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "min_lng", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "max_lat", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "max_lng", "in": "query", "required": true, "schema": {"type": "number"}},
          {"name": "class", "in": "query", "schema": {"type": "string"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 100}}
        ],
        "responses": {
          "200": {"description": "Places located inside the bounding box viewport"}
        }
      }
    },
    "/v1/places/{id}": {
      "get": {
        "summary": "Place entity lookup by ID",
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}},
          {"name": "include_nearby", "in": "query", "schema": {"type": "boolean", "default": false}}
        ],
        "responses": {
          "200": {"description": "Place details"},
          "404": {"description": "Place not found"}
        }
      }
    },
    "/v1/places/wikidata/{qid}": {
      "get": {
        "summary": "Place lookup by Wikidata QID",
        "parameters": [
          {"name": "qid", "in": "path", "required": true, "schema": {"type": "string"}},
          {"name": "include_nearby", "in": "query", "schema": {"type": "boolean", "default": false}}
        ],
        "responses": {
          "200": {"description": "Place details"},
          "404": {"description": "Place not found"}
        }
      }
    },
    "/v1/airports": {
      "get": {
        "summary": "Search airports",
        "parameters": [
          {"name": "q", "in": "query", "schema": {"type": "string"}},
          {"name": "country", "in": "query", "schema": {"type": "string"}},
          {"name": "lat", "in": "query", "schema": {"type": "number"}},
          {"name": "lng", "in": "query", "schema": {"type": "number"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 20}}
        ],
        "responses": {
          "200": {"description": "Airport search results"}
        }
      }
    },
    "/v1/airports/{code}": {
      "get": {
        "summary": "Airport lookup by IATA or ICAO code",
        "parameters": [
          {"name": "code", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {
          "200": {"description": "Airport details"},
          "404": {"description": "Airport not found"}
        }
      }
    },
    "/v1/countries": {
      "get": {
        "summary": "List indexed countries and place counts",
        "responses": {
          "200": {"description": "List of indexed countries"}
        }
      }
    },
    "/v1/countries/{code}/regions": {
      "get": {
        "summary": "List regions/provinces for a country",
        "parameters": [
          {"name": "code", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {
          "200": {"description": "List of country regions"}
        }
      }
    },
    "/v1/countries/{code}/cities": {
      "get": {
        "summary": "List cities for a country sorted by population",
        "parameters": [
          {"name": "code", "in": "path", "required": true, "schema": {"type": "string"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "default": 50}},
          {"name": "offset", "in": "query", "schema": {"type": "integer", "default": 0}}
        ],
        "responses": {
          "200": {"description": "List of country cities"}
        }
      }
    },
    "/v1/taxonomy": {
      "get": {
        "summary": "Taxonomy classes and matching rules",
        "responses": {
          "200": {"description": "Taxonomy schema"}
        }
      }
    },
    "/v1/languages": {
      "get": {
        "summary": "Supported multilingual languages",
        "responses": {
          "200": {"description": "Languages list"}
        }
      }
    },
    "/v1/importer/status": {
      "get": {
        "summary": "Harvester daemon live status and resource usage",
        "responses": {
          "200": {"description": "Importer status"}
        }
      }
    },
    "/v1/importer/countries": {
      "get": {
        "summary": "Available and indexed Geofabrik countries",
        "responses": {
          "200": {"description": "Countries catalogue"}
        }
      }
    },
    "/v1/importer/jobs": {
      "post": {
        "summary": "Trigger dynamic country import job",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "countries": {"type": "array", "items": {"type": "string"}},
                  "online_enrich": {"type": "boolean"},
                  "mode": {"type": "string", "enum": ["append", "replace"]}
                },
                "required": ["countries"]
              }
            }
          }
        },
        "responses": {
          "202": {"description": "Job queued"}
        }
      }
    },
    "/v1/importer/events": {
      "get": {
        "summary": "Server-Sent Events progress stream from harvester",
        "responses": {
          "200": {"description": "SSE text/event-stream"}
        }
      }
    },
    "/v1/metrics": {
      "get": {
        "summary": "Real-time ring-buffer telemetry (QPS, p50/p95/p99, memory, top queries)",
        "responses": {
          "200": {"description": "Telemetry snapshot"}
        }
      }
    },
    "/v1/health": {
      "get": {
        "summary": "API and Meilisearch health check",
        "responses": {
          "200": {"description": "Health status"}
        }
      }
    },
    "/v1/openapi.json": {
      "get": {
        "summary": "OpenAPI 3.1 schema specification",
        "responses": {
          "200": {"description": "OpenAPI specification JSON"}
        }
      }
    }
  }
}`

// GetOpenAPI returns the full OpenAPI 3.1 specification JSON.
func (h *Handler) GetOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPISpec))
}
