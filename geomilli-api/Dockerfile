# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install git and certificates
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary with stripped debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/geomilli-api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S geomilli && adduser -S geomilli -G geomilli
USER geomilli

WORKDIR /app
COPY --from=builder /build/geomilli-api /app/geomilli-api

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/v1/health || exit 1

ENTRYPOINT ["/app/geomilli-api"]
