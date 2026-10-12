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

func TestWindInsightConfiguration(t *testing.T) {
	t.Setenv("WIND_SPEED_CHANGE_MS", "4.5")
	t.Setenv("WIND_MIN_LOW_HOURS", "18")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WindInsights.SpeedChange != 4.5 || cfg.WindInsights.MinimumLowHours != 18 {
		t.Fatalf("wind overrides: %+v", cfg.WindInsights)
	}
	for _, tt := range []struct{ name, value string }{
		{"WIND_SPEED_CHANGE_MS", "NaN"}, {"WIND_LOW_SPEED_MS", "-1"},
		{"WIND_DIRECTION_CHANGE_DEG", "181"}, {"WIND_DIRECTION_CONSISTENCY", "1.1"},
		{"WIND_STRONG_SPEED_MS", "2"}, {"WIND_MIN_LOW_HOURS", "6"},
		{"WIND_MIN_LOW_DAYS", "15"}, {"WIND_COMBINED_LOW_SOLAR_KWH_M2", "Inf"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.name, tt.value)
			if _, err := loadConfig(); err == nil {
				t.Fatal("invalid wind config accepted")
			}
		})
	}
}
