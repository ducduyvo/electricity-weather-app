package weather

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenMeteoMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method != http.MethodGet ||
			q.Get("latitude") != "60.1700" ||
			q.Get("longitude") != "24.9400" ||
			q.Get("hourly") != "temperature_2m,shortwave_radiation" ||
			q.Get("forecast_days") != "14" ||
			q.Get("timezone") != "auto" {
			t.Error("unexpected upstream request", r.URL)
		}
		_, _ = w.Write([]byte(`{"latitude":60.17,"longitude":24.94,"timezone":"Europe/Helsinki","hourly":{"time":["2026-10-07T00:00","2026-10-07T01:00","2026-10-08T00:00"],"temperature_2m":[4,8,1],"shortwave_radiation":[0,500,250]}}`))
	}))
	defer server.Close()
	provider := OpenMeteo{Client: server.Client(), BaseURL: server.URL}
	got, err := provider.Forecast(context.Background(), 60.17, 24.94)
	if err != nil {
		t.Fatal(err)
	}
	if got.Latitude != 60.17 ||
		got.Longitude != 24.94 ||
		got.Timezone != "Europe/Helsinki" ||
		got.FetchedAt.IsZero() {
		t.Fatalf("bad metadata: %+v", got)
	}
	if len(got.Hourly) != 3 || *got.Hourly[1].Temperature != 8 || *got.Hourly[1].SolarIrradiance != 500 {
		t.Fatalf("bad hourly mapping: %+v", got.Hourly)
	}
	if len(got.Daily) != 2 || *got.Daily[0].AvgTemperature != 6 || got.Daily[0].SolarEnergyKWhM2 != 0.5 {
		t.Fatalf("bad summary: %+v", got.Daily)
	}
	if len(got.Signals) != 2 ||
		got.Signals[0].Title != "Solar period expected" ||
		got.Signals[1].Title != "Limited comparison data" {
		t.Fatalf("bad signals: %+v", got.Signals)
	}
}

func TestOpenMeteoResponses(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		body, errorText string
	}{
		{"non-200", 503, "unavailable", "status 503"},
		{"malformed JSON", 200, "{", "decode Open-Meteo forecast"},
		{"null and short arrays", 200, `{"hourly":{"time":["2026-10-07T00:00","2026-10-07T01:00"],"temperature_2m":[null],"shortwave_radiation":[null,500]}}`, ""},
		{"missing arrays", 200, `{"hourly":{"time":["2026-10-07T00:00","2026-10-07T01:00"]}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			got, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 0, 0)
			if tt.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorText) {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Hourly) != 2 ||
				got.Hourly[0].Temperature != nil ||
				got.Hourly[1].Temperature != nil ||
				got.Hourly[0].SolarIrradiance != nil {
				t.Fatalf("bad missing-value mapping: %+v", got.Hourly)
			}
			if got.Daily[0].AvgTemperature != nil {
				t.Fatal("missing temperature became zero")
			}
			if tt.name == "null and short arrays" && *got.Hourly[1].SolarIrradiance != 500 {
				t.Fatal("solar value lost")
			}
		})
	}
}

func TestOpenMeteoCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(ctx, 0, 0)
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped cancellation, got %v", err)
	}
}

func TestOpenMeteoURLFailures(t *testing.T) {
	for _, base := range []string{"http://%zz", "http://example.com/\x7f"} {
		_, err := (OpenMeteo{Client: &http.Client{}, BaseURL: base}).Forecast(context.Background(), 0, 0)
		if err == nil {
			t.Fatalf("expected URL error for %q", base)
		}
	}
	// A nil context makes request construction fail; it must not be ignored.
	_, err := (OpenMeteo{Client: &http.Client{}, BaseURL: "http://example.com"}).Forecast(nil, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "create Open-Meteo request") {
		t.Fatalf("request error %v", err)
	}
}
