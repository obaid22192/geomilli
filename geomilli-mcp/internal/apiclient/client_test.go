package apiclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_SuccessEndpoints(t *testing.T) {
	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		// Verify headers
		if req.Header.Get("X-Geomilli-Client") != "mcp" {
			t.Errorf("expected X-Geomilli-Client=mcp, got %q", req.Header.Get("X-Geomilli-Client"))
		}
		if req.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("expected Authorization header with token, got %q", req.Header.Get("Authorization"))
		}

		path := req.URL.Path
		switch path {
		case "/v1/countries":
			body := `{
				"total_countries": 2,
				"countries": [
					{"country_code": "BE", "country_name": "Belgium", "place_count": 120},
					{"country_code": "FR", "country_name": "France", "place_count": 95}
				]
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil

		case "/v1/autocomplete":
			if req.URL.Query().Get("q") != "Grand Place" {
				t.Errorf("expected q='Grand Place', got %q", req.URL.Query().Get("q"))
			}
			body := `{
				"query": "Grand Place",
				"took_ms": 12,
				"total_hits": 1,
				"results": [{
					"id": "osm_n1",
					"name_display": "Grand Place",
					"class": "TouristAttraction",
					"_geo": {"lat": 50.8467, "lng": 4.3524}
				}]
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil

		case "/v1/reverse":
			body := `{
				"lat": 50.8467,
				"lng": 4.3524,
				"radius_meters": 5000,
				"address": {"city": "Brussels", "country": "Belgium", "country_code": "BE"},
				"admin_hierarchy": ["Brussels", "Belgium"],
				"nearest_places": []
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error": "not found"}`)),
		}, nil
	})

	httpClient := &http.Client{Transport: transport}
	client := NewClient("http://mock-geomilli", "test-secret", 5*time.Second, httpClient)

	ctx := context.Background()

	// 1. Countries
	countries, err := client.ListCountries(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing countries: %v", err)
	}
	if countries.TotalCountries != 2 || len(countries.Countries) != 2 {
		t.Errorf("unexpected countries response: %+v", countries)
	}
	if countries.Countries[0].CountryCode != "BE" {
		t.Errorf("expected BE as first country, got %s", countries.Countries[0].CountryCode)
	}

	// 2. Autocomplete
	auto, err := client.Autocomplete(ctx, AutocompleteParams{
		Query: "Grand Place",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error autocomplete: %v", err)
	}
	if auto.TotalHits != 1 || len(auto.Results) != 1 {
		t.Errorf("unexpected autocomplete response: %+v", auto)
	}
	if auto.Results[0].NameDisplay != "Grand Place" {
		t.Errorf("expected Grand Place, got %s", auto.Results[0].NameDisplay)
	}

	// 3. Reverse
	rev, err := client.Reverse(ctx, ReverseParams{
		Lat: 50.8467,
		Lng: 4.3524,
	})
	if err != nil {
		t.Fatalf("unexpected error reverse: %v", err)
	}
	if rev.Address.City != "Brussels" {
		t.Errorf("expected Brussels, got %s", rev.Address.City)
	}
	if rev.RadiusMeters != 5000 {
		t.Errorf("expected RadiusMeters=5000, got %d", rev.RadiusMeters)
	}
}

func TestClient_NewErrorEnvelope(t *testing.T) {
	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		body := `{
			"error": {
				"code": "invalid_parameter",
				"message": "Parameter 'lat' must be between -90 and 90",
				"status": 400
			}
		}`
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})

	client := NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	_, err := client.Reverse(context.Background(), ReverseParams{Lat: 150, Lng: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error of type *APIError, got %T (%v)", err, err)
	}

	if apiErr.Status != 400 {
		t.Errorf("expected Status=400, got %d", apiErr.Status)
	}
	if apiErr.Code != "invalid_parameter" {
		t.Errorf("expected Code='invalid_parameter', got %q", apiErr.Code)
	}
	if apiErr.Message != "Parameter 'lat' must be between -90 and 90" {
		t.Errorf("expected message, got %q", apiErr.Message)
	}
}

func TestClient_LegacyErrorEnvelope(t *testing.T) {
	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		body := `{
			"error": true,
			"status": 404,
			"message": "Airport not found for code: XYZ"
		}`
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})

	client := NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	_, err := client.GetAirportByCode(context.Background(), "XYZ", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error of type *APIError, got %T (%v)", err, err)
	}

	if apiErr.Status != 404 {
		t.Errorf("expected Status=404, got %d", apiErr.Status)
	}
	if apiErr.Message != "Airport not found for code: XYZ" {
		t.Errorf("expected legacy message, got %q", apiErr.Message)
	}
}

func TestClient_Timeout(t *testing.T) {
	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(100 * time.Millisecond):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
			}, nil
		}
	})

	client := NewClient("http://mock-geomilli", "", 10*time.Millisecond, &http.Client{Transport: transport})
	_, err := client.ListCountries(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestClient_FallbackError(t *testing.T) {
	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(bytes.NewBufferString("502 Bad Gateway: nginx down")),
		}, nil
	})

	client := NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	_, err := client.GetHealth(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Status != 502 {
		t.Errorf("expected Status=502, got %d", apiErr.Status)
	}
	if apiErr.Message != "502 Bad Gateway: nginx down" {
		t.Errorf("expected fallback message, got %q", apiErr.Message)
	}
}
