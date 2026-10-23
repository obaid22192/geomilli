package meili_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/travayo/auto-complete/internal/meili"
)

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func TestValidateGoldenQueries(t *testing.T) {
	oldDefaultClient := http.DefaultClient
	defer func() { http.DefaultClient = oldDefaultClient }()

	http.DefaultClient = &http.Client{
		Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			_ = json.NewDecoder(req.Body).Decode(&body)
			query, _ := body["q"].(string)

			var hits []map[string]interface{}
			if query == "brussels" {
				hits = []map[string]interface{}{
					{"name_display": "Brussels", "country_code": "BE"},
				}
			} else if query == "atomium" {
				hits = []map[string]interface{}{
					{"name_display": "Atomium", "class": "TouristAttraction"},
				}
			}

			out, _ := json.Marshal(map[string]interface{}{"hits": hits})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(out)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	indexer := meili.New(meili.Config{
		Host:   "http://mock-meili:7700",
		APIKey: "testKey",
	})

	checks := []meili.QueryExpectation{
		{Query: "brussels", ExpectedTokens: []string{"brussels", "be"}},
		{Query: "atomium", ExpectedTokens: []string{"atomium"}},
	}

	err := indexer.ValidateGoldenQueries("places", checks, 5)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Test missing token failure
	failingChecks := []meili.QueryExpectation{
		{Query: "brussels", ExpectedTokens: []string{"nonexistent_token_xyz"}},
	}
	errFail := indexer.ValidateGoldenQueries("places", failingChecks, 5)
	if errFail == nil {
		t.Fatalf("expected error for missing token, got nil")
	}
}
