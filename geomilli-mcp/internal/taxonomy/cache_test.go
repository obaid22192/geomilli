package taxonomy

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
)

type mockFetcher struct {
	taxonomyCalls  atomic.Int32
	languagesCalls atomic.Int32
	taxResp        *apiclient.TaxonomyConfig
	taxErr         error
	langResp       *apiclient.LanguagesConfig
	langErr        error
}

func (m *mockFetcher) GetTaxonomy(ctx context.Context) (*apiclient.TaxonomyConfig, error) {
	m.taxonomyCalls.Add(1)
	return m.taxResp, m.taxErr
}

func (m *mockFetcher) GetLanguages(ctx context.Context) (*apiclient.LanguagesConfig, error) {
	m.languagesCalls.Add(1)
	return m.langResp, m.langErr
}

func TestCache_ColdFallback(t *testing.T) {
	fetcher := &mockFetcher{
		taxErr:  errors.New("network down"),
		langErr: errors.New("network down"),
	}

	cache := NewCache(fetcher, 5*time.Minute)

	// Cold fallback classes must be present immediately
	classes := cache.GetClassNames()
	if len(classes) != 19 {
		t.Fatalf("expected 19 cold fallback classes, got %d", len(classes))
	}

	canon, ok := cache.IsValidClass("airport")
	if !ok || canon != "Airport" {
		t.Errorf("expected Airport to be valid in cold fallback, got ok=%v, canon=%s", ok, canon)
	}

	langs := cache.GetLanguageCodes()
	if len(langs) == 0 {
		t.Fatal("expected cold fallback languages, got 0")
	}
	if !cache.IsValidLanguage("en") {
		t.Errorf("expected en to be valid in cold fallback")
	}

	// Attempting refresh should fail but preserve cold fallback
	err := cache.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected refresh error, got nil")
	}

	if len(cache.GetClassNames()) != 19 {
		t.Errorf("cold fallback classes lost after failed refresh")
	}
}

func TestCache_RefreshAndTTL(t *testing.T) {
	fetcher := &mockFetcher{
		taxResp: &apiclient.TaxonomyConfig{
			Version: 1,
			Classes: []apiclient.ClassDefinition{
				{
					Name: "Hotel",
					Match: []apiclient.MatchRule{
						{Key: "tourism", Values: []string{"hotel", "motel"}},
					},
				},
				{
					Name: "Restaurant",
					Match: []apiclient.MatchRule{
						{Key: "amenity", Values: []string{"restaurant", "fast_food"}},
					},
				},
			},
		},
		langResp: &apiclient.LanguagesConfig{
			Version: 1,
			Languages: []apiclient.LanguageItem{
				{Code: "en", Name: "English", Enabled: true},
				{Code: "fr", Name: "French", Enabled: true},
				{Code: "xx", Name: "Disabled", Enabled: false},
			},
		},
	}

	// 50ms TTL for testing
	cache := NewCache(fetcher, 50*time.Millisecond)

	ctx := context.Background()
	if err := cache.Refresh(ctx); err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}

	if fetcher.taxonomyCalls.Load() != 1 {
		t.Errorf("expected 1 taxonomy call, got %d", fetcher.taxonomyCalls.Load())
	}

	classes := cache.GetClassNames()
	if len(classes) != 2 || classes[0] != "Hotel" || classes[1] != "Restaurant" {
		t.Errorf("expected [Hotel Restaurant], got %v", classes)
	}

	hotelSubs := cache.GetSubclasses("Hotel")
	if len(hotelSubs) != 2 || hotelSubs[0] != "hotel" || hotelSubs[1] != "motel" {
		t.Errorf("expected [hotel motel], got %v", hotelSubs)
	}

	langs := cache.GetLanguageCodes()
	if len(langs) != 2 || langs[0] != "en" || langs[1] != "fr" {
		t.Errorf("expected enabled languages [en fr], got %v", langs)
	}

	// Calling RefreshIfNeeded immediately should NOT trigger fetch (within TTL)
	if err := cache.RefreshIfNeeded(ctx); err != nil {
		t.Fatalf("unexpected RefreshIfNeeded error: %v", err)
	}
	if fetcher.taxonomyCalls.Load() != 1 {
		t.Errorf("expected still 1 call within TTL, got %d", fetcher.taxonomyCalls.Load())
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	if err := cache.RefreshIfNeeded(ctx); err != nil {
		t.Fatalf("unexpected RefreshIfNeeded error after expiry: %v", err)
	}
	if fetcher.taxonomyCalls.Load() != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", fetcher.taxonomyCalls.Load())
	}
}
