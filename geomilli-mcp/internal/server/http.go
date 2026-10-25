package server

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewHTTPHandler creates the HTTP handler for the MCP server, exposing /healthz and /mcp.
// The /mcp path runs the MCP StreamableHTTPHandler in stateless mode, protected by
// bearer authentication and Origin allowlist middleware.
func NewHTTPHandler(s *mcp.Server, cfg *config.Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// 1. Health check endpoint (exempt from auth)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"version": version.Version,
		})
	})

	// 2. Streamable HTTP handler for MCP in stateless mode
	mcpStreamable := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return s
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true,
			Logger:    logger,
		},
	)

	// Wrap /mcp with auth and origin middleware
	var mcpHandler http.Handler = mcpStreamable
	if cfg.AuthToken != "" {
		mcpHandler = AuthMiddleware(cfg.AuthToken, mcpHandler)
	}
	if len(cfg.AllowedOrigins) > 0 {
		mcpHandler = OriginMiddleware(cfg.AllowedOrigins, mcpHandler)
	}

	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)

	return mux
}

// AuthMiddleware validates incoming Bearer tokens using constant-time comparison.
func AuthMiddleware(expectedToken string, next http.Handler) http.Handler {
	expectedBytes := []byte(expectedToken)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeHTTPError(w, http.StatusUnauthorized, "unauthorized", "missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeHTTPError(w, http.StatusUnauthorized, "unauthorized", "invalid Authorization scheme, expected Bearer <token>")
			return
		}

		tokenBytes := []byte(strings.TrimSpace(parts[1]))
		if subtle.ConstantTimeCompare(tokenBytes, expectedBytes) != 1 {
			writeHTTPError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OriginMiddleware enforces Origin header allowlisting against DNS-rebinding attacks.
func OriginMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	allowedMap := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed == "*" {
			allowAll = true
		}
		if trimmed != "" {
			allowedMap[strings.ToLower(trimmed)] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Non-browser HTTP clients (cURL, CLI agents, SDKs) do not send Origin header.
		// If Origin is present (browsers), validate it against the allowlist.
		if origin != "" && !allowAll {
			lower := strings.ToLower(strings.TrimSpace(origin))
			if _, ok := allowedMap[lower]; !ok {
				writeHTTPError(w, http.StatusForbidden, "forbidden", "origin not allowed: "+origin)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func writeHTTPError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   code,
		"message": message,
		"status":  status,
	})
}
