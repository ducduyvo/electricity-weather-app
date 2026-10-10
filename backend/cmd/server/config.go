package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"weather-retailer/backend/internal/weather"
)

type config struct {
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
