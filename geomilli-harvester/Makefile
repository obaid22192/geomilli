SHELL := /bin/bash

ifneq (,$(wildcard .env))
include .env
export
endif

MEILI_HOST ?= http://127.0.0.1:7700
MEILI_API_KEY ?=
HARVESTER_PORT ?= 8085
COUNTRIES ?= BE

# When pre-built binaries exist (e.g. inside Docker) use them; otherwise use go run.
define run
$(if $(wildcard /usr/local/bin/$(1)),/usr/local/bin/$(1),go run ./cmd/$(1))
endef

.PHONY: serve run refresh build test clean cleanup \
	fetch build-data index transform-places enrich-places index-places all

# Default target starts the harvester daemon
serve:
	$(call run,harvester) serve --port $(HARVESTER_PORT)

run:
	$(call run,harvester) run --stage run

refresh:
	ENRICH_ONLINE=true $(call run,harvester) run --stage run --online-enrich=true

build:
	go build -o bin/harvester ./cmd/harvester
	go build -o bin/pipeline ./cmd/pipeline
	go build -o bin/transform-places ./cmd/transform-places
	go build -o bin/enrich-places ./cmd/enrich-places
	go build -o bin/index ./cmd/index
	go build -o bin/cleanup ./cmd/cleanup
	go build -o bin/countries ./cmd/countries

test:
	go test -v ./...

fetch:
	$(call run,pipeline) --stage fetch --target places

build-data:
	$(call run,pipeline) --stage build-data --target places

index:
	$(call run,pipeline) --stage index --target places

cleanup:
	$(call run,pipeline) --stage cleanup

clean:
	rm -rf bin/ staging/*.ndjson raw/osm/*.pbf raw/osm/*.geojsonseq data/locks/*.lock

all: run
