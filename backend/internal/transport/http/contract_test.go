package weatherhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"weather-retailer/backend/internal/weather"
)

// Test fixtures stay in tests; production continues to use the real provider.
func TestForecastHTTPJSONContract(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprintf("empty=%t", empty), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("forecast_days") != "15" {
					t.Error("expected 15-day request")
				}
				raw := map[string]any{"timezone": "Europe/Helsinki", "hourly": map[string]any{"time": []int64{}}}
				if !empty {
					raw["hourly"] = map[string]any{"time": []int64{1791450000}, "temperature_2m": []any{nil}, "shortwave_radiation": []any{nil}}
				}
				if err := json.NewEncoder(w).Encode(raw); err != nil {
					t.Error(err)
				}
			}))
			defer upstream.Close()
			handler := NewHandler(weather.OpenMeteo{Client: upstream.Client(), BaseURL: upstream.URL})
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
				for _, key := range []string{"date", "minTemperature", "maxTemperature", "avgTemperature", "solarEnergyKWhM2", "peakSolarIrradiance", "solarSampleCount", "hourCount", "expectedHourCount", "completeDay", "temperatureSampleCount", "minWindSpeed", "avgWindSpeed", "maxWindSpeed", "windSampleCount", "avgWindDirection", "windDirectionSampleCount", "windDirectionConsistency"} {
					if _, ok := days[0][key]; !ok {
						t.Fatalf("missing daily field %s", key)
					}
				}
				for _, key := range []string{"minTemperature", "maxTemperature", "avgTemperature", "peakSolarIrradiance", "minWindSpeed", "avgWindSpeed", "maxWindSpeed", "avgWindDirection"} {
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
