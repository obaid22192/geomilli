package meili

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"testing"

	"github.com/geomilli/geomilli-api/internal/config"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHaversineDistance(t *testing.T) {
	// Brussels Grand Place (50.8467, 4.3525) to Brussels Airport (50.9010, 4.4844) is ~11km
	d := HaversineDistance(50.8467, 4.3525, 50.9010, 4.4844)
	if d < 10000 || d > 13000 {
		t.Fatalf("unexpected distance between Brussels Grand Place and Airport: %f meters", d)
	}

	// Distance to self should be 0
	d0 := HaversineDistance(50.8467, 4.3525, 50.8467, 4.3525)
	if math.Abs(d0) > 0.001 {
		t.Fatalf("expected 0 for self distance, got %f", d0)
	}
}

func TestClientSearch(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/indexes/places/search" {
				body := `{"hits":[{"id":"node:1","class":"City","name_display":"Brussels","_geo":{"lat":50.85,"lng":4.35}}],"query":"bru","processingTimeMs":2,"totalHits":1}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &config.Config{
		MeiliHost:  "http://mock-meili:7700",
		MeiliIndex: "places",
	}
	client := NewClientWithHTTPClient(cfg, httpClient)

	result, err := client.SearchPlaces(context.Background(), SearchRequest{
		Q:     "bru",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(result.Hits))
	}
	if result.Hits[0].NameDisplay != "Brussels" {
		t.Fatalf("expected Brussels, got %s", result.Hits[0].NameDisplay)
	}
}

func TestClientGetDocument(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/indexes/places/documents/node:123" {
				body := `{"id":"node:123","name_display":"Atomium","class":"TouristAttraction"}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	cfg := &config.Config{
		MeiliHost:  "http://mock-meili:7700",
		MeiliIndex: "places",
	}
	client := NewClientWithHTTPClient(cfg, httpClient)

	doc, err := client.GetDocument(context.Background(), "node:123")
	if err != nil {
		t.Fatalf("failed to get document: %v", err)
	}
	if doc == nil || doc.NameDisplay != "Atomium" {
		t.Fatalf("expected Atomium, got %+v", doc)
	}

	docMissing, err := client.GetDocument(context.Background(), "missing:999")
	if err != nil {
		t.Fatalf("unexpected error for missing doc: %v", err)
	}
	if docMissing != nil {
		t.Fatalf("expected nil for missing doc")
	}
}
