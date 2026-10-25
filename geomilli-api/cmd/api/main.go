package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/handlers"
	"github.com/geomilli/geomilli-api/internal/importer"
	"github.com/geomilli/geomilli-api/internal/meili"
	"github.com/geomilli/geomilli-api/internal/metrics"
	"github.com/geomilli/geomilli-api/internal/version"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Load()

	log.Printf("==================================================")
	log.Printf(" Starting Geomilli API Server (version %s)", version.Version)
	log.Printf(" Version:       %s", version.Version)
	log.Printf(" Port:          %s", cfg.Port)
	log.Printf(" Meili Host:    %s", cfg.MeiliHost)
	log.Printf(" Meili Mode:    %s", func() string {
		if cfg.IsCloud {
			return "Cloud"
		}
		return "Local"
	}())
	log.Printf(" Meili Index:   %s", cfg.MeiliIndex)
	log.Printf(" Harvester URL: %s", cfg.HarvesterURL)
	log.Printf("==================================================")

	meiliClient := meili.NewClient(cfg)

	// Ensure the index exists in Meilisearch asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meiliClient.EnsureIndex(ctx); err != nil {
			log.Printf("[meili] note: ensure index `%s`: %v", cfg.MeiliIndex, err)
		} else {
			log.Printf("[meili] index `%s` verified/initialized", cfg.MeiliIndex)
		}
	}()

	importerClient := importer.NewClient(cfg)
	collector := metrics.NewCollector()

	handler := handlers.NewHandler(cfg, meiliClient, importerClient, collector)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Allows longer for SSE or big responses
		IdleTimeout:  120 * time.Second,
	}

	// Server run context
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on http://0.0.0.0:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Graceful shutdown handling
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Fatal error starting server: %v", err)
	case sig := <-shutdown:
		log.Printf("Signal %v received, initiating graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			_ = server.Close()
		}
		log.Println("Geomilli API stopped cleanly.")
	}
}
