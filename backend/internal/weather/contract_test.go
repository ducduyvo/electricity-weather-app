package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test fixtures stay in tests; production continues to use the real provider.
func TestForecastHTTPJSONContract(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprintf("empty=%t", empty), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("forecast_days") != "14" {
					t.Error("expected 14-day request")
				}
				raw := omResponse{Timezone: "Europe/Helsinki"}
				if !empty {
					// A partial day with all readings missing verifies false/zero/null fields.
					raw.Hourly.Time = []string{"2026-10-08T12:00"}
					raw.Hourly.Temperature = []*float64{nil}
					raw.Hourly.Solar = []*float64{nil}
				}
				if err := json.NewEncoder(w).Encode(raw); err != nil {
					t.Error(err)
				}
			}))
			defer upstream.Close()
			handler := NewHandler(OpenMeteo{Client: upstream.Client(), BaseURL: upstream.URL})
			response := httptest.NewRecorder()
			handler.Weather(response, httptest.NewRequest(http.MethodGet, "/api/weather?lat=60&lon=24", nil))
			if response.Code != 200 || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response %d: %s", response.Code, response.Body.String())
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"timezone", "fetchedAt", "hourly", "daily", "signals", "outlookSignals"} {
				if _, ok := payload[key]; !ok {
					t.Fatalf("missing forecast field %s", key)
				}
			}
			for _, key := range []string{"hourly", "daily", "signals", "outlookSignals"} {
				if string(payload[key]) == "null" {
					t.Fatalf("%s must be an array", key)
				}
			}
			var days []map[string]json.RawMessage
			_ = json.Unmarshal(payload["daily"], &days)
			if !empty {
				for _, key := range []string{"date", "minTemperature", "maxTemperature", "avgTemperature", "solarEnergyKWhM2", "peakSolarIrradiance", "solarSampleCount", "hourCount", "expectedHourCount", "completeDay", "temperatureSampleCount", "minWindSpeed", "meanWindSpeed", "maxWindSpeed", "windSampleCount", "meanWindDirection", "windDirectionSampleCount", "windDirectionConsistency"} {
					if _, ok := days[0][key]; !ok {
						t.Fatalf("missing daily field %s", key)
					}
				}
				for _, key := range []string{"minTemperature", "maxTemperature", "avgTemperature", "peakSolarIrradiance", "minWindSpeed", "meanWindSpeed", "maxWindSpeed", "meanWindDirection"} {
					if string(days[0][key]) != "null" {
						t.Fatalf("%s missing reading fabricated", key)
					}
				}
				if string(days[0]["completeDay"]) != "false" || string(days[0]["hourCount"]) != "1" || string(days[0]["expectedHourCount"]) != "24" || string(days[0]["solarSampleCount"]) != "0" {
					t.Fatalf("coverage %+v", days[0])
				}
			}
			for _, key := range []string{"signals", "outlookSignals"} {
				var items []map[string]json.RawMessage
				if err := json.Unmarshal(payload[key], &items); err != nil {
					t.Fatal(err)
				}
				if len(items) == 0 {
					t.Fatalf("missing data signal %s", key)
				}
				for _, item := range items {
					for _, field := range []string{"kind", "severity", "title", "detail"} {
						var value string
						if err := json.Unmarshal(item[field], &value); err != nil || value == "" {
							t.Fatalf("invalid signal field %s: %s", field, item[field])
						}
					}
				}
			}
		})
	}
}

func TestOutlookIncludesSecondWeekAndStopsAtFourteen(t *testing.T) {
	temperatures, solar := make([]float64, 16), make([]float64, 16)
	for i := range temperatures {
		temperatures[i] = 10
		solar[i] = 4
	}
	temperatures[11], temperatures[12], temperatures[13] = 8, 6, 4
	days := trendDays(temperatures, solar)
	cold := findSignal(outlookSignals(days), "Cold spell developing")
	if cold == nil || !strings.Contains(cold.Detail, days[13].Date) {
		t.Fatalf("second-week trend missing: %+v", cold)
	}
	for i := range temperatures {
		temperatures[i] = 10
	}
	temperatures[14], temperatures[15] = 5, 0
	if got := outlookSignals(trendDays(temperatures, solar)); findSignal(got, "Cold spell developing") != nil {
		t.Fatalf("included days beyond 14: %+v", got)
	}
}

func TestDuplicateHourCannotFillMissingCoverage(t *testing.T) {
	points := hourlyPoints(1)
	points[23] = points[22]
	day := summarize(points, "Europe/Helsinki")[0]
	if day.CompleteDay || day.HourCount != 23 || day.TemperatureSampleCount != 23 || day.SolarSampleCount != 23 {
		t.Fatalf("duplicate filled missing hour: %+v", day)
	}
}

func TestShortTermSignalsIgnoreLaterDays(t *testing.T) {
	days := trendDays([]float64{10, 5, 5}, []float64{4, 2, 2})
	before := signals(days)
	days[2].AvgTemperature = f(30)
	days[2].SolarEnergyKWhM2 = 100
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(signals(days))
	if string(a) != string(b) {
		t.Fatal("later forecast days changed short-term insights")
	}
}

var _ Provider = OpenMeteo{}
var _ Provider = (*Cached)(nil)
var _ Provider = fakeProvider(func(context.Context, float64, float64) (Forecast, error) { return Forecast{}, nil })
