package countries

import (
	"testing"
)

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		// Alpha-2
		{"BE", "BE", true},
		{"be", "BE", true},
		{"  fr  ", "FR", true},
		{"US", "US", true},
		{"de", "DE", true},

		// Alpha-3
		{"BEL", "BE", true},
		{"bel", "BE", true},
		{"USA", "US", true},
		{"DEU", "DE", true},
		{"FRA", "FR", true},
		{"GBR", "GB", true},
		{"NLD", "NL", true},

		// English names
		{"Belgium", "BE", true},
		{"belgium", "BE", true},
		{"France", "FR", true},
		{"germany", "DE", true},
		{"United States", "US", true},
		{"United Kingdom", "GB", true},
		{"Netherlands", "NL", true},
		{"Switzerland", "CH", true},
		{"Japan", "JP", true},

		// Aliases and localized names
		{"België", "BE", true},
		{"Belgique", "BE", true},
		{"Deutschland", "DE", true},
		{"España", "ES", true},
		{"Italia", "IT", true},
		{"Holland", "NL", true},
		{"UK", "GB", true},
		{"America", "US", true},
		{"Czechia", "CZ", true},
		{"Türkiye", "TR", true},

		// Quoted / bracketed
		{`"Belgium"`, "BE", true},
		{`'BE'`, "BE", true},

		// Invalid
		{"", "", false},
		{"   ", "", false},
		{"Atlantis", "", false},
		{"ZZ", "", false},
		{"XYZ123", "", false},
	}

	for _, tt := range tests {
		got, ok := NormalizeCode(tt.input)
		if ok != tt.ok || got != tt.expected {
			t.Errorf("NormalizeCode(%q) = (%q, %v); expected (%q, %v)", tt.input, got, ok, tt.expected, tt.ok)
		}
	}
}

func TestName(t *testing.T) {
	if got := Name("BE"); got != "Belgium" {
		t.Errorf("expected Belgium, got %q", got)
	}
	if got := Name("US"); got != "United States" {
		t.Errorf("expected United States, got %q", got)
	}
	if got := Name("XYZ"); got != "XYZ" {
		t.Errorf("expected XYZ fallback, got %q", got)
	}
}

func TestExtractFromQuery(t *testing.T) {
	tests := []struct {
		input         string
		expectedQuery string
		expectedCode  string
		expectedFound bool
	}{
		// Exact country
		{"belgium", "", "BE", true},
		{"Belgium", "", "BE", true},
		{"BE", "", "BE", true},

		// "in <country>"
		{"airports in belgium", "airports", "BE", true},
		{"hotels in France", "hotels", "FR", true},
		{"restaurants in Deutschland", "restaurants", "DE", true},
		{"cafes inside Italy", "cafes", "IT", true},

		// ", <country>"
		{"brussels, belgium", "brussels", "BE", true},
		{"paris, france", "paris", "FR", true},
		{"rome, IT", "rome", "IT", true},

		// "<country> <something>"
		{"belgium airports", "airports", "BE", true},
		{"france hotels", "hotels", "FR", true},
		{"united kingdom castles", "castles", "GB", true},

		// No country
		{"brussels", "brussels", "", false},
		{"eiffel tower", "eiffel tower", "", false},
		{"hotel central", "hotel central", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		gotQ, gotCode, gotFound := ExtractFromQuery(tt.input)
		if gotFound != tt.expectedFound || gotCode != tt.expectedCode || gotQ != tt.expectedQuery {
			t.Errorf("ExtractFromQuery(%q) = (%q, %q, %v); expected (%q, %q, %v)",
				tt.input, gotQ, gotCode, gotFound, tt.expectedQuery, tt.expectedCode, tt.expectedFound)
		}
	}
}
