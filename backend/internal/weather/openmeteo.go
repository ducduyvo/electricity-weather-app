package weather

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const DefaultBaseURL = "https://api.open-meteo.com/v1/forecast"

type OpenMeteo struct {
	WindInsights *WindInsightConfig
	Client       *http.Client
	BaseURL      string
}
type omResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Hourly    struct {
		Time          []int64    `json:"time"`
		Temperature   []*float64 `json:"temperature_2m"`
		WindSpeed     []*float64 `json:"wind_speed_120m"`
		WindDirection []*float64 `json:"wind_direction_120m"`
		Solar         []*float64 `json:"shortwave_radiation"`
	} `json:"hourly"`
}

func (o OpenMeteo) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	if err := Validate(lat, lon); err != nil {
		return Forecast{}, err
	}
	base := o.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return Forecast{}, &ProviderError{Operation: "parse Open-Meteo URL", Kind: ErrProviderConfiguration, Cause: err}
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("hourly", "temperature_2m,shortwave_radiation,wind_speed_120m,wind_direction_120m")
	q.Set("forecast_days", "15")
	q.Set("timezone", "auto")
	q.Set("timeformat", "unixtime")
	q.Set("wind_speed_unit", "ms")

	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Forecast{}, &ProviderError{Operation: "create Open-Meteo request", Kind: ErrProviderConfiguration, Cause: err}
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return Forecast{}, &ProviderError{Operation: "fetch Open-Meteo forecast", Kind: ErrProviderUnavailable, Cause: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		kind := ErrProviderResponse
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			kind = ErrProviderUnavailable
		}
		return Forecast{}, &ProviderError{Operation: "fetch Open-Meteo forecast", Kind: kind, StatusCode: resp.StatusCode}
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Forecast{}, &ProviderError{Operation: "read Open-Meteo forecast", Kind: ErrProviderResponse, Cause: err}
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	slog.Info("Open-Meteo response received",
		slog.Int("status", resp.StatusCode),
		slog.Int("bytes", len(bodyBytes)),
		slog.String("body", string(bodyBytes)),
	)
	var raw omResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Forecast{}, &ProviderError{Operation: "decode Open-Meteo forecast", Kind: ErrProviderResponse, Cause: err}
	}
	timezone := raw.Timezone
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Forecast{}, &ProviderError{Operation: "load Open-Meteo timezone", Kind: ErrProviderResponse, Cause: err}
	}
	n := len(raw.Hourly.Time)
	points := make([]Point, 0, n)
	// Request an extra day so DST changes cannot shorten the final local day.
	// Keep only the original 14 local calendar days in the dashboard response.
	var forecastEnd time.Time
	if n > 0 {
		first := time.Unix(raw.Hourly.Time[0], 0).In(location)
		forecastStart := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location)
		forecastEnd = forecastStart.AddDate(0, 0, 14)
	}
	for i := 0; i < n; i++ {
		instant := time.Unix(raw.Hourly.Time[i], 0).In(location)
		if instant.After(forecastEnd) {
			continue
		}
		var t, s *float64
		if i < len(raw.Hourly.Temperature) {
			t = raw.Hourly.Temperature[i]
		}
		if i < len(raw.Hourly.Solar) {
			s = raw.Hourly.Solar[i]
		}
		var windSpeed, windDirection *float64
		if i < len(raw.Hourly.WindSpeed) {
			windSpeed = validWindValue(raw.Hourly.WindSpeed[i], false)
		}
		if i < len(raw.Hourly.WindDirection) {
			windDirection = validWindValue(raw.Hourly.WindDirection[i], true)
		}
		unixTime := raw.Hourly.Time[i]
		points = append(points, Point{
			UnixTime:        &unixTime,
			WindSpeed:       windSpeed,
			WindDirection:   windDirection,
			WindPotential:   windPotential(windSpeed),
			Time:            instant.Format("2006-01-02T15:04"),
			Temperature:     t,
			SolarIrradiance: s,
		})
	}
	// The boundary midnight supplies the final solar interval only.
	visiblePoints := points
	if len(points) > 0 && *points[len(points)-1].UnixTime == forecastEnd.Unix() {
		visiblePoints = points[:len(points)-1]
	}
	f := Forecast{
		Latitude:  raw.Latitude,
		Longitude: raw.Longitude,
		Timezone:  timezone,
		Hourly:    visiblePoints,
		FetchedAt: time.Now().UTC(),
	}
	f.Daily = summarize(points, f.Timezone)
	if len(f.Daily) > 0 && f.Daily[len(f.Daily)-1].Date == forecastEnd.Format("2006-01-02") {
		f.Daily = f.Daily[:len(f.Daily)-1]
	}
	f.Signals = signals(f.Daily)
	f.OutlookSignals = outlookSignals(f.Daily)
	windCfg := DefaultWindInsightConfig()
	if o.WindInsights != nil {
		windCfg = *o.WindInsights
	}
	f.Signals = mergeWindInsights(f.Signals, hourlyWindInsights(visiblePoints, f.Timezone, f.FetchedAt, windCfg))
	f.OutlookSignals = mergeWindInsights(f.OutlookSignals, dailyWindInsights(f.Daily, windCfg))
	return f, nil
}
