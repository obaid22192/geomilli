# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary with stripped debug symbols
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/geomilli/geomilli-mcp/internal/version.Version=${VERSION}" \
    -o /build/geomilli-mcp ./cmd/geomilli-mcp

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S geomilli && adduser -S geomilli -G geomilli
USER geomilli

WORKDIR /app
COPY --from=builder /build/geomilli-mcp /app/geomilli-mcp

ENV MCP_TRANSPORT=http \
    MCP_HTTP_ADDR=0.0.0.0:8090

EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8090/healthz || exit 1

ENTRYPOINT ["/app/geomilli-mcp"]
