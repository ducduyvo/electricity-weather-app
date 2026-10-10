package main

import (
	"testing"
	"time"

	"weather-retailer/backend/internal/weather"
)

func TestLoadConfigDefaults(t *testing.T) {
	for _, name := range []string{"SERVER_ADDRESS", "OPEN_METEO_BASE_URL", "HTTP_CLIENT_TIMEOUT", "CACHE_TTL", "ALLOWED_FRONTEND_ORIGIN"} {
		t.Setenv(name, "")
	}
	got, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != ":8080" ||
		got.OpenMeteoURL != weather.DefaultBaseURL ||
		got.ClientTimeout != 5*time.Second ||
		got.CacheTTL != 10*time.Minute ||
		got.AllowedOrigin != "*" {
		t.Fatalf("defaults %+v", got)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", ":9090")
	t.Setenv("OPEN_METEO_BASE_URL", "http://localhost:9000/forecast")
	t.Setenv("HTTP_CLIENT_TIMEOUT", "2s")
	t.Setenv("CACHE_TTL", "1m")
	t.Setenv("ALLOWED_FRONTEND_ORIGIN", "http://localhost:5173")
	got, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != ":9090" ||
		got.OpenMeteoURL != "http://localhost:9000/forecast" ||
		got.ClientTimeout != 2*time.Second ||
		got.CacheTTL != time.Minute ||
		got.AllowedOrigin != "http://localhost:5173" {
		t.Fatalf("overrides %+v", got)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	for _, tt := range []struct{ name, value string }{
		{"HTTP_CLIENT_TIMEOUT", "abc"}, {"HTTP_CLIENT_TIMEOUT", "0s"},
		{"CACHE_TTL", "-1m"}, {"OPEN_METEO_BASE_URL", "http://%zz"},
		{"OPEN_METEO_BASE_URL", "/forecast"}, {"OPEN_METEO_BASE_URL", "ftp://example.com"},
		{"ALLOWED_FRONTEND_ORIGIN", "bad\norigin"},
	} {
		t.Run(tt.name+tt.value, func(t *testing.T) {
			for _, name := range []string{"HTTP_CLIENT_TIMEOUT", "CACHE_TTL", "OPEN_METEO_BASE_URL", "ALLOWED_FRONTEND_ORIGIN"} {
				t.Setenv(name, "")
			}
			t.Setenv(tt.name, tt.value)
			if _, err := loadConfig(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}
