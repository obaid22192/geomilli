package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/server"
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/geomilli/geomilli-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// JSON logging to stderr ONLY.
	// Standard output is strictly reserved for the stdio MCP transport protocol.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger.Info("starting geomilli-mcp",
		"version", version.Version,
		"transport", cfg.Transport,
		"api_url", cfg.APIURL,
		"enable_ops_tools", cfg.EnableOpsTools,
	)

	// Context for graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize typed API client
	api := apiclient.NewClient(
		cfg.APIURL,
		cfg.APIToken,
		cfg.RequestTimeout,
		nil,
	)

	// Initialize taxonomy & language metadata cache
	taxCache := taxonomy.NewCache(api, cfg.TaxonomyTTL)

	// Pre-warm taxonomy cache (non-blocking fallback on network failure)
	warmCtx, warmCancel := context.WithTimeout(ctx, 3*time.Second)
	if err := taxCache.Refresh(warmCtx); err != nil {
		logger.Warn("taxonomy cache warm-up failed, using cold fallback", "error", err)
	}
	warmCancel()

	// Build MCP server with dependencies
	deps := tools.Deps{
		API:      api,
		Taxonomy: taxCache,
		Config:   cfg,
		Logger:   logger,
	}
	mcpServer := server.NewServer(deps)

	switch cfg.Transport {
	case "stdio":
		logger.Info("running MCP on stdio transport")
		if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("stdio transport error", "error", err)
			os.Exit(1)
		}
		logger.Info("geomilli-mcp stopped")

	case "http":
		httpHandler := server.NewHTTPHandler(mcpServer, cfg, logger)
		httpSrv := &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           httpHandler,
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		go func() {
			logger.Info("running MCP on Streamable HTTP transport", "addr", cfg.HTTPAddr, "path", "/mcp")
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("HTTP server error", "error", err)
				os.Exit(1)
			}
		}()

		// Await termination signal
		<-ctx.Done()
		logger.Info("shutting down HTTP server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP graceful shutdown error", "error", err)
			_ = httpSrv.Close()
		}
		logger.Info("geomilli-mcp stopped")

	default:
		logger.Error(fmt.Sprintf("unknown transport: %s", cfg.Transport))
		os.Exit(1)
	}
}
