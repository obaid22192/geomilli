package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLatLng(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedLat float64
		expectedLng float64
		expectErr   bool
		errCode     ErrorCode
	}{
		{
			name:        "valid coordinates",
			url:         "/test?lat=50.8503&lng=4.3517",
			expectedLat: 50.8503,
			expectedLng: 4.3517,
			expectErr:   false,
		},
		{
			name:      "missing both lat and lng",
			url:       "/test",
			expectErr: true,
			errCode:   ErrMissingParameter,
		},
		{
			name:      "missing lat only",
			url:       "/test?lng=4.3517",
			expectErr: true,
			errCode:   ErrMissingParameter,
		},
		{
			name:      "missing lng only",
			url:       "/test?lat=50.8503",
			expectErr: true,
			errCode:   ErrMissingParameter,
		},
		{
			name:      "empty lat",
			url:       "/test?lat=&lng=4.3517",
			expectErr: true,
			errCode:   ErrMissingParameter,
		},
		{
			name:      "invalid lat string",
			url:       "/test?lat=abc&lng=4.3517",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "invalid lng string",
			url:       "/test?lat=50.8503&lng=xyz",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "lat out of range high",
			url:       "/test?lat=90.0001&lng=4.3517",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "lat out of range low",
			url:       "/test?lat=-90.0001&lng=4.3517",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "lng out of range high",
			url:       "/test?lat=50.8503&lng=180.0001",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "lng out of range low",
			url:       "/test?lat=50.8503&lng=-180.0001",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:        "exact bounds -90, -180",
			url:         "/test?lat=-90&lng=-180",
			expectedLat: -90,
			expectedLng: -180,
			expectErr:   false,
		},
		{
			name:        "exact bounds 90, 180",
			url:         "/test?lat=90&lng=180",
			expectedLat: 90,
			expectedLng: 180,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			lat, lng, err := parseLatLng(req)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				pErr, ok := err.(*ParamError)
				if !ok {
					t.Fatalf("expected *ParamError, got %T", err)
				}
				if pErr.Code != tt.errCode {
					t.Errorf("expected error code %s, got %s", tt.errCode, pErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if lat != tt.expectedLat || lng != tt.expectedLng {
					t.Errorf("expected (%f, %f), got (%f, %f)", tt.expectedLat, tt.expectedLng, lat, lng)
				}
			}
		})
	}
}

func TestParseOptionalLatLng(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedLat float64
		expectedLng float64
		expectedOK  bool
	}{
		{
			name:        "both valid",
			url:         "/test?lat=50.85&lng=4.35",
			expectedLat: 50.85,
			expectedLng: 4.35,
			expectedOK:  true,
		},
		{
			name:       "neither present",
			url:        "/test",
			expectedOK: false,
		},
		{
			name:       "only lat",
			url:        "/test?lat=50.85",
			expectedOK: false,
		},
		{
			name:       "only lng",
			url:        "/test?lng=4.35",
			expectedOK: false,
		},
		{
			name:       "invalid lat",
			url:        "/test?lat=abc&lng=4.35",
			expectedOK: false,
		},
		{
			name:       "out of range lat",
			url:        "/test?lat=95.0&lng=4.35",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			lat, lng, ok := parseOptionalLatLng(req)
			if ok != tt.expectedOK {
				t.Fatalf("expected ok=%v, got %v", tt.expectedOK, ok)
			}
			if ok && (lat != tt.expectedLat || lng != tt.expectedLng) {
				t.Errorf("expected (%f, %f), got (%f, %f)", tt.expectedLat, tt.expectedLng, lat, lng)
			}
		})
	}
}

func TestParseRadius(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		defaultRadius  int
		maxRadius      int
		expectedRadius int
	}{
		{
			name:           "missing param uses default",
			url:            "/test",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 5000,
		},
		{
			name:           "valid radius_meters",
			url:            "/test?radius_meters=3000",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 3000,
		},
		{
			name:           "fallback to radius param",
			url:            "/test?radius=8000",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 8000,
		},
		{
			name:           "radius_meters takes precedence over radius",
			url:            "/test?radius_meters=4000&radius=8000",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 4000,
		},
		{
			name:           "clamped to max",
			url:            "/test?radius_meters=200000",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 100000,
		},
		{
			name:           "clamped to 1 when negative",
			url:            "/test?radius_meters=-50",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 1,
		},
		{
			name:           "clamped to 1 when zero",
			url:            "/test?radius_meters=0",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 1,
		},
		{
			name:           "non-numeric uses default",
			url:            "/test?radius_meters=foo",
			defaultRadius:  5000,
			maxRadius:      100000,
			expectedRadius: 5000,
		},
		{
			name:           "reverse default and cap",
			url:            "/test?radius_meters=150000",
			defaultRadius:  DefaultReverseRadius,
			maxRadius:      MaxReverseRadius,
			expectedRadius: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			radius := parseRadius(req, tt.defaultRadius, tt.maxRadius)
			if radius != tt.expectedRadius {
				t.Errorf("expected radius %d, got %d", tt.expectedRadius, radius)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		defaultLimit  int
		maxLimit      int
		expectedLimit int
	}{
		{
			name:          "missing uses default",
			url:           "/test",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 20,
		},
		{
			name:          "valid limit",
			url:           "/test?limit=50",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 50,
		},
		{
			name:          "clamped to max",
			url:           "/test?limit=500",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 100,
		},
		{
			name:          "clamped to 1 when negative",
			url:           "/test?limit=-5",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 1,
		},
		{
			name:          "clamped to 1 when zero",
			url:           "/test?limit=0",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 1,
		},
		{
			name:          "non-numeric uses default",
			url:           "/test?limit=abc",
			defaultLimit:  20,
			maxLimit:      100,
			expectedLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			limit := parseLimit(req, tt.defaultLimit, tt.maxLimit)
			if limit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, limit)
			}
		})
	}
}

func TestParseOffset(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedOffset int
	}{
		{
			name:           "missing uses default",
			url:            "/test",
			expectedOffset: 0,
		},
		{
			name:           "valid offset",
			url:            "/test?offset=30",
			expectedOffset: 30,
		},
		{
			name:           "negative clamped to 0",
			url:            "/test?offset=-10",
			expectedOffset: 0,
		},
		{
			name:           "non-numeric uses default",
			url:            "/test?offset=xyz",
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			offset := parseOffset(req)
			if offset != tt.expectedOffset {
				t.Errorf("expected offset %d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}

func TestParseBBox(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
		errCode   ErrorCode
	}{
		{
			name:      "valid bbox",
			url:       "/test?min_lat=50.8&min_lng=4.3&max_lat=50.9&max_lng=4.4",
			expectErr: false,
		},
		{
			name:      "missing min_lat",
			url:       "/test?min_lng=4.3&max_lat=50.9&max_lng=4.4",
			expectErr: true,
			errCode:   ErrMissingParameter,
		},
		{
			name:      "invalid float",
			url:       "/test?min_lat=bad&min_lng=4.3&max_lat=50.9&max_lng=4.4",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "out of range lat",
			url:       "/test?min_lat=-95&min_lng=4.3&max_lat=50.9&max_lng=4.4",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
		{
			name:      "out of range lng",
			url:       "/test?min_lat=50.8&min_lng=-190&max_lat=50.9&max_lng=4.4",
			expectErr: true,
			errCode:   ErrInvalidParameter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			_, _, _, _, err := parseBBox(req)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				pErr, ok := err.(*ParamError)
				if !ok {
					t.Fatalf("expected *ParamError, got %T", err)
				}
				if pErr.Code != tt.errCode {
					t.Errorf("expected error code %s, got %s", tt.errCode, pErr.Code)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
