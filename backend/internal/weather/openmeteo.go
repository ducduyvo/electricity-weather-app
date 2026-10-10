package weather

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const DefaultBaseURL = "https://api.open-meteo.com/v1/forecast"

type OpenMeteo struct {
	Client  *http.Client
	BaseURL string
}
type omResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Hourly    struct {
		Time        []string   `json:"time"`
		Temperature []*float64 `json:"temperature_2m"`
		Solar       []*float64 `json:"shortwave_radiation"`
	} `json:"hourly"`
}

func (o OpenMeteo) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	base := o.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return Forecast{}, fmt.Errorf("parse Open-Meteo URL: %w", err)
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("hourly", "temperature_2m,shortwave_radiation")
	q.Set("forecast_days", "14")
	q.Set("timezone", "auto")

	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Forecast{}, fmt.Errorf("create Open-Meteo request: %w", err)
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return Forecast{}, fmt.Errorf("fetch Open-Meteo forecast: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Forecast{}, fmt.Errorf("weather provider status %d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	slog.Info("Open-Meteo response received",
		slog.Int("status", resp.StatusCode),
		slog.Int("bytes", len(bodyBytes)),
		slog.String("body", string(bodyBytes)),
	)
	var raw omResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Forecast{}, fmt.Errorf("decode Open-Meteo forecast: %w", err)
	}
	n := len(raw.Hourly.Time)
	points := make([]Point, 0, n)
	for i := 0; i < n; i++ {
		var t, s *float64
		if i < len(raw.Hourly.Temperature) {
			t = raw.Hourly.Temperature[i]
		}
		if i < len(raw.Hourly.Solar) {
			s = raw.Hourly.Solar[i]
		}
		points = append(points, Point{
			Time:            raw.Hourly.Time[i],
			Temperature:     t,
			SolarIrradiance: s,
		})
	}
	f := Forecast{
		Latitude:  raw.Latitude,
		Longitude: raw.Longitude,
		Timezone:  raw.Timezone,
		Hourly:    points,
		FetchedAt: time.Now().UTC(),
	}
	f.Daily = summarize(points, f.Timezone)
	f.Signals = signals(f.Daily)
	f.OutlookSignals = outlookSignals(f.Daily)
	return f, nil
}
