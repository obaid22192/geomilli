package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetServiceStatusInput is empty since get_service_status requires no parameters.
type GetServiceStatusInput struct{}

// ServiceHealthSummary models API and Meilisearch health status.
type ServiceHealthSummary struct {
	Status      string         `json:"status" jsonschema:"API health status: ok, degraded, down, or unreachable"`
	Version     string         `json:"version,omitempty" jsonschema:"Geomilli API service version"`
	Timestamp   string         `json:"timestamp,omitempty" jsonschema:"Timestamp of the health check probe"`
	Meilisearch map[string]any `json:"meilisearch,omitempty" jsonschema:"Meilisearch status and connectivity details"`
	Error       string         `json:"error,omitempty" jsonschema:"Error message if the health probe failed"`
}

// ServiceImporterSummary models Harvester ingestion daemon operational status.
type ServiceImporterSummary struct {
	State     string                        `json:"state" jsonschema:"Harvester daemon operational state (e.g. idle, running, stopped, unreachable)"`
	ActiveJob *apiclient.ActiveJob          `json:"active_job,omitempty" jsonschema:"Currently active country ingestion job, if any"`
	Resources *apiclient.HarvesterResources `json:"resources,omitempty" jsonschema:"Harvester daemon memory and disk cache stats"`
	Error     string                        `json:"error,omitempty" jsonschema:"Error message if importer probe failed"`
}

// ServiceMetricsSummary models API telemetry performance and document volume.
type ServiceMetricsSummary struct {
	TotalRequests int64   `json:"total_requests" jsonschema:"Total HTTP requests handled since startup"`
	ActiveQPS     float64 `json:"active_qps" jsonschema:"Current queries per second"`
	LatencyP50Ms  float64 `json:"latency_p50_ms" jsonschema:"Median request latency in milliseconds"`
	LatencyP95Ms  float64 `json:"latency_p95_ms" jsonschema:"95th percentile request latency in milliseconds"`
	IndexDocs     int64   `json:"index_docs" jsonschema:"Total indexed place documents in Meilisearch"`
	Error         string  `json:"error,omitempty" jsonschema:"Error message if metrics probe failed"`
}

// GetServiceStatusOutput models the consolidated cluster operational status.
type GetServiceStatusOutput struct {
	OverallStatus string                 `json:"overall_status" jsonschema:"Synthesized health status: ok, degraded, or unhealthy"`
	Health        ServiceHealthSummary   `json:"health" jsonschema:"API service and search engine health probe"`
	Importer      ServiceImporterSummary `json:"importer" jsonschema:"Harvester ingestion daemon status"`
	Metrics       ServiceMetricsSummary  `json:"metrics" jsonschema:"API telemetry and indexing performance metrics"`
}

// ListImportQueueInput is empty since list_import_queue requires no parameters.
type ListImportQueueInput struct{}

// ListImportQueueOutput models the Harvester ingestion queue status.
type ListImportQueueOutput struct {
	Concurrency   int                    `json:"concurrency" jsonschema:"Max concurrent country import workers configured"`
	ActiveWorkers []apiclient.WorkerSlot `json:"active_workers" jsonschema:"Currently active worker tasks"`
	QueuedTasks   []apiclient.QueuedTask `json:"queued_tasks" jsonschema:"Queued country import tasks waiting for execution"`
	CountryLocks  map[string]string      `json:"country_locks,omitempty" jsonschema:"Active country locks held by workers"`
}

// ListImportableCountriesInput defines optional filtering parameters for importable countries.
type ListImportableCountriesInput struct {
	OnlyIndexed bool   `json:"only_indexed,omitempty" jsonschema:"Optional filter: if true, returns only countries that are already indexed in the database"`
	Continent   string `json:"continent,omitempty" jsonschema:"Optional filter: continent name to filter by (e.g. 'europe', 'north-america', 'asia')"`
}

// ListImportableCountriesOutput models the list of countries available for import.
type ListImportableCountriesOutput struct {
	TotalCountries int                     `json:"total_countries" jsonschema:"Total matching countries returned"`
	Countries      []apiclient.CountryItem `json:"countries" jsonschema:"List of countries with ingestion metadata and sync schedules"`
}

func registerOperations(s *mcp.Server, d Deps) {
	// 1. get_service_status
	RegisterTool(s, &mcp.Tool{
		Name:        "get_service_status",
		Description: "Retrieve consolidated operational health, harvester ingestion daemon status, and API telemetry metrics for the Geomilli cluster.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetServiceStatusInput) (*mcp.CallToolResult, GetServiceStatusOutput, error) {
		health, errH := d.API.GetHealth(ctx)
		importer, errI := d.API.GetImporterStatus(ctx)
		metrics, errM := d.API.GetMetrics(ctx)

		if errH != nil && errI != nil && errM != nil {
			return nil, GetServiceStatusOutput{}, FormatAPIError("get service status", errH)
		}

		var out GetServiceStatusOutput

		if errH != nil {
			out.Health = ServiceHealthSummary{
				Status: "unreachable",
				Error:  errH.Error(),
			}
		} else if health != nil {
			out.Health = ServiceHealthSummary{
				Status:      health.Status,
				Version:     health.Version,
				Timestamp:   health.Timestamp,
				Meilisearch: health.Meilisearch,
			}
		}

		if errI != nil {
			out.Importer = ServiceImporterSummary{
				State: "unreachable",
				Error: errI.Error(),
			}
		} else if importer != nil {
			out.Importer = ServiceImporterSummary{
				State:     importer.State,
				ActiveJob: importer.ActiveJob,
				Resources: &importer.Resources,
				Error:     importer.Error,
			}
		}

		if errM != nil {
			out.Metrics = ServiceMetricsSummary{
				Error: errM.Error(),
			}
		} else if metrics != nil {
			out.Metrics = ServiceMetricsSummary{
				TotalRequests: metrics.TotalRequests,
				ActiveQPS:     metrics.ActiveQPS,
				LatencyP50Ms:  metrics.Latency.P50,
				LatencyP95Ms:  metrics.Latency.P95,
				IndexDocs:     metrics.Meilisearch.IndexDocs,
			}
		}

		if errH == nil && errI == nil && errM == nil && out.Health.Status == "ok" {
			out.OverallStatus = "ok"
		} else if out.Health.Status == "down" || errH != nil {
			out.OverallStatus = "unhealthy"
		} else {
			out.OverallStatus = "degraded"
		}

		summary := fmt.Sprintf("Cluster status: %s (API: %s, Importer: %s, Active QPS: %.2f, Indexed Docs: %d)",
			out.OverallStatus, out.Health.Status, out.Importer.State, out.Metrics.ActiveQPS, out.Metrics.IndexDocs)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// 2. list_import_queue
	RegisterTool(s, &mcp.Tool{
		Name:        "list_import_queue",
		Description: "List the Harvester ingestion daemon queue, active worker slots, and pending country import tasks.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListImportQueueInput) (*mcp.CallToolResult, ListImportQueueOutput, error) {
		res, err := d.API.GetImporterQueue(ctx)
		if err != nil {
			return nil, ListImportQueueOutput{}, FormatAPIError("list import queue", err)
		}

		active := res.ActiveWorkers
		if active == nil {
			active = []apiclient.WorkerSlot{}
		}
		queued := res.QueuedTasks
		if queued == nil {
			queued = []apiclient.QueuedTask{}
		}

		out := ListImportQueueOutput{
			Concurrency:   res.Concurrency,
			ActiveWorkers: active,
			QueuedTasks:   queued,
			CountryLocks:  res.CountryLocks,
		}

		summary := fmt.Sprintf("Harvester import queue: %d active workers, %d queued tasks (concurrency limit: %d).",
			len(active), len(queued), res.Concurrency)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// 3. list_importable_countries
	RegisterTool(s, &mcp.Tool{
		Name:        "list_importable_countries",
		Description: "List all countries available for OSM data download and ingestion by the Harvester daemon, including indexing status and sync schedules.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListImportableCountriesInput) (*mcp.CallToolResult, ListImportableCountriesOutput, error) {
		res, err := d.API.GetImporterCountries(ctx)
		if err != nil {
			return nil, ListImportableCountriesOutput{}, FormatAPIError("list importable countries", err)
		}

		var matched []apiclient.CountryItem
		var totalIndexed int
		targetContinent := strings.ToLower(strings.TrimSpace(in.Continent))

		for _, c := range res.Countries {
			if c.IsIndexed {
				totalIndexed++
			}
			if in.OnlyIndexed && !c.IsIndexed {
				continue
			}
			if targetContinent != "" && strings.ToLower(c.Continent) != targetContinent {
				continue
			}
			matched = append(matched, c)
		}

		if matched == nil {
			matched = []apiclient.CountryItem{}
		}

		out := ListImportableCountriesOutput{
			TotalCountries: len(matched),
			Countries:      matched,
		}

		summary := fmt.Sprintf("Found %d importable countries (%d currently indexed in database).", len(matched), totalIndexed)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}
