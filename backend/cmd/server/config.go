package main

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"weather-retailer/backend/internal/weather"
)

type config struct {
	WindInsights  weather.WindInsightConfig
	Address       string
	OpenMeteoURL  string
	ClientTimeout time.Duration
	CacheTTL      time.Duration
	AllowedOrigin string
}

func loadConfig() (config, error) {
	cfg := config{

		Address: envOrDefault("SERVER_ADDRESS", ":8080"),

		OpenMeteoURL: envOrDefault("OPEN_METEO_BASE_URL", weather.DefaultBaseURL),

		AllowedOrigin: envOrDefault("ALLOWED_FRONTEND_ORIGIN", "*"),
	}
	var err error
	cfg.ClientTimeout, err = positiveDuration("HTTP_CLIENT_TIMEOUT", "5s")
	if err != nil {
		return config{}, err
	}
	cfg.CacheTTL, err = positiveDuration("CACHE_TTL", "10m")
	if err != nil {
		return config{}, err
	}
	cfg.WindInsights = weather.DefaultWindInsightConfig()
	for _, threshold := range []struct {
		name  string
		value *float64
	}{
		{name: "WIND_SPEED_CHANGE_MS", value: &cfg.WindInsights.SpeedChange},
		{name: "WIND_LOW_SPEED_MS", value: &cfg.WindInsights.LowSpeed},
		{name: "WIND_DIRECTION_CHANGE_DEG", value: &cfg.WindInsights.DirectionChange},
		{name: "WIND_DIRECTION_CONSISTENCY", value: &cfg.WindInsights.DirectionConsistency},
		{name: "WIND_STRONG_SPEED_MS", value: &cfg.WindInsights.StrongSpeed},
		{name: "WIND_COMBINED_LOW_SOLAR_KWH_M2", value: &cfg.WindInsights.LowSolarDaily},
		{name: "WIND_COMBINED_LOW_SOLAR_W_M2", value: &cfg.WindInsights.LowSolarHourly},
	} {
		raw := envOrDefault(threshold.name, strconv.FormatFloat(*threshold.value, 'f', -1, 64))
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
			return config{}, fmt.Errorf("%s must be a finite positive number", threshold.name)
		}
		*threshold.value = value
	}
	// Celsius thresholds may be zero or negative, so only require finiteness.
	cold, err := strconv.ParseFloat(envOrDefault("WIND_COMBINED_PERSISTENT_COLD_C", strconv.FormatFloat(cfg.WindInsights.PersistentCold, 'f', -1, 64)), 64)
	if err != nil || math.IsNaN(cold) || math.IsInf(cold, 0) {
		return config{}, fmt.Errorf("WIND_COMBINED_PERSISTENT_COLD_C must be a finite number")
	}
	cfg.WindInsights.PersistentCold = cold
	for _, threshold := range []struct {
		name             string
		value            *int
		minimum, maximum int
	}{
		{name: "WIND_MIN_LOW_HOURS", value: &cfg.WindInsights.MinimumLowHours, minimum: 12, maximum: 72},
		{name: "WIND_MIN_LOW_DAYS", value: &cfg.WindInsights.MinimumLowDays, minimum: 2, maximum: 14},
	} {
		value, err := strconv.Atoi(envOrDefault(threshold.name, strconv.Itoa(*threshold.value)))
		if err != nil || value < threshold.minimum || value > threshold.maximum {
			return config{}, fmt.Errorf("%s must be an integer between %d and %d", threshold.name, threshold.minimum, threshold.maximum)
		}
		*threshold.value = value
	}
	if cfg.WindInsights.DirectionChange > 180 || cfg.WindInsights.DirectionConsistency > 1 || cfg.WindInsights.StrongSpeed <= cfg.WindInsights.LowSpeed {
		return config{}, fmt.Errorf("wind direction change must be <= 180, consistency <= 1, and strong speed > low speed")
	}
	u, err := url.Parse(cfg.OpenMeteoURL)
	if err != nil {
		return config{}, fmt.Errorf("OPEN_METEO_BASE_URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return config{}, fmt.Errorf("OPEN_METEO_BASE_URL must be an absolute HTTP or HTTPS URL")
	}
	if strings.ContainsAny(cfg.AllowedOrigin, "\r\n") {
		return config{}, fmt.Errorf("ALLOWED_FRONTEND_ORIGIN must not contain newlines")
	}
	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func positiveDuration(name, fallback string) (time.Duration, error) {
	value, err := time.ParseDuration(envOrDefault(name, fallback))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return value, nil
}
