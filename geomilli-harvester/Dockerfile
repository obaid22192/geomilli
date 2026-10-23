# ── Stage 1: build all Go binaries ─────────────────────────────────────────────
FROM golang:1.25-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux \
    go build -o /out/harvester         ./cmd/harvester         && \
    go build -o /out/pipeline          ./cmd/pipeline          && \
    go build -o /out/transform-places  ./cmd/transform-places  && \
    go build -o /out/enrich-places     ./cmd/enrich-places     && \
    go build -o /out/index             ./cmd/index             && \
    go build -o /out/cleanup           ./cmd/cleanup           && \
    go build -o /out/countries         ./cmd/countries         && \
    go build -o /out/gen-filters       ./cmd/gen-filters

# ── Stage 2: slim runtime image ─────────────────────────────────────────────────
FROM debian:bookworm-slim

# osmium-tool, curl (fetches), ca-certificates (HTTPS), make
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    osmium-tool \
    curl \
    ca-certificates \
    bash \
    make \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy compiled binaries
COPY --from=builder /out/ /usr/local/bin/

# Copy scripts, config, and golden queries needed at runtime
COPY scripts/           scripts/
COPY config/            config/
COPY Makefile           Makefile
COPY .env.example       .env.example

RUN chmod +x scripts/*.sh

# osmium is installed locally — no Docker-in-Docker needed in Kubernetes/Compose
ENV OSMIUM_MODE=local
ENV HARVESTER_PORT=8085
ENV COUNTRIES=BE
ENV AUTO_IMPORT=true

EXPOSE 8085

# Directories created at startup by jobs
RUN mkdir -p raw/osm staging data/cache data/locks data/manifests

ENTRYPOINT ["harvester"]
CMD ["serve"]
