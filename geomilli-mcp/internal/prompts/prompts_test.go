package prompts

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestPromptServerAndClient(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()

	deps := tools.Deps{
		Config: &config.Config{},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{Name: "test-geomilli-prompts", Version: "1.0.0"},
		nil,
	)
	Register(mcpServer, deps)

	st, ct := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	serverSession, err := mcpServer.Connect(ctx, st, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect server session: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, ct, nil)
	if err != nil {
		serverSession.Close()
		cancel()
		t.Fatalf("failed to connect client session: %v", err)
	}

	cleanup := func() {
		clientSession.Close()
		serverSession.Close()
		cancel()
	}

	return clientSession, cleanup
}

func TestPrompts_ListAndGet(t *testing.T) {
	cs, cleanup := setupTestPromptServerAndClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Verify all 3 prompts are listed
	expectedPrompts := map[string]bool{
		"explore_area":          false,
		"describe_location":     false,
		"plan_airport_transfer": false,
	}

	for p, err := range cs.Prompts(ctx, nil) {
		if err != nil {
			t.Fatalf("unexpected error iterating prompts: %v", err)
		}
		if _, ok := expectedPrompts[p.Name]; ok {
			expectedPrompts[p.Name] = true
		}
	}

	for name, found := range expectedPrompts {
		if !found {
			t.Errorf("expected prompt %q to be registered", name)
		}
	}

	// 2. Test explore_area prompt
	t.Run("explore_area_success", func(t *testing.T) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "explore_area",
			Arguments: map[string]string{
				"place":     "Montmartre, Paris",
				"interests": "art, cafes",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error getting explore_area: %v", err)
		}
		if len(res.Messages) == 0 {
			t.Fatalf("expected at least 1 message in result")
		}
		textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent, got %T", res.Messages[0].Content)
		}
		if !strings.Contains(textContent.Text, "Montmartre, Paris") {
			t.Errorf("expected text to mention Montmartre, Paris")
		}
		if !strings.Contains(textContent.Text, "art, cafes") {
			t.Errorf("expected text to mention art, cafes")
		}
		if !strings.Contains(textContent.Text, "geocode") || !strings.Contains(textContent.Text, "find_nearby") {
			t.Errorf("expected workflow instructions mentioning geocode and find_nearby")
		}
	})

	t.Run("explore_area_missing_place", func(t *testing.T) {
		_, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "explore_area",
			Arguments: map[string]string{},
		})
		if err == nil {
			t.Errorf("expected error when required argument 'place' is missing, got nil")
		}
	})

	// 3. Test describe_location prompt
	t.Run("describe_location_success", func(t *testing.T) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "describe_location",
			Arguments: map[string]string{
				"lat": "50.8503",
				"lng": "4.3517",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error getting describe_location: %v", err)
		}
		if len(res.Messages) == 0 {
			t.Fatalf("expected message in describe_location result")
		}
		textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent")
		}
		if !strings.Contains(textContent.Text, "50.8503") || !strings.Contains(textContent.Text, "4.3517") {
			t.Errorf("expected coordinates in text, got: %s", textContent.Text)
		}
		if !strings.Contains(textContent.Text, "reverse_geocode") {
			t.Errorf("expected instructions to call reverse_geocode")
		}
	})

	t.Run("describe_location_missing_coords", func(t *testing.T) {
		_, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "describe_location",
			Arguments: map[string]string{
				"lat": "50.8503",
			},
		})
		if err == nil {
			t.Errorf("expected error when lng is missing")
		}
	})

	// 4. Test plan_airport_transfer prompt
	t.Run("plan_airport_transfer_success", func(t *testing.T) {
		res, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "plan_airport_transfer",
			Arguments: map[string]string{
				"city": "Tokyo",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error getting plan_airport_transfer: %v", err)
		}
		if len(res.Messages) == 0 {
			t.Fatalf("expected message in plan_airport_transfer result")
		}
		textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent")
		}
		if !strings.Contains(textContent.Text, "Tokyo") {
			t.Errorf("expected city Tokyo in prompt text")
		}
		if !strings.Contains(textContent.Text, "find_airports") {
			t.Errorf("expected instruction to call find_airports")
		}
	})

	t.Run("plan_airport_transfer_missing_city", func(t *testing.T) {
		_, err := cs.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      "plan_airport_transfer",
			Arguments: map[string]string{},
		})
		if err == nil {
			t.Errorf("expected error when city is missing")
		}
	})
}
