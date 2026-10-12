package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeProvider func(context.Context, float64, float64) (Forecast, error)

func (p fakeProvider) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	return p(ctx, lat, lon)
}

func TestHandlerWeather(t *testing.T) {
	tests := []struct {
		name, query string
		fail        bool
		status      int
	}{
		{"valid", "lat=60.17&lon=24.94", false, 200},
		{"boundary", "lat=-90&lon=180", false, 200},
		{"missing", "", false, 400},
		{"malformed latitude", "lat=abc&lon=24", false, 400},
		{"malformed longitude", "lat=60&lon=abc", false, 400},
		{"latitude range", "lat=91&lon=24", false, 400},
		{"longitude range", "lat=60&lon=-181", false, 400},
		{"NaN latitude", "lat=NaN&lon=24", false, 400},
		{"NaN longitude", "lat=60&lon=NaN", false, 400},
		{"infinite latitude", "lat=Inf&lon=24", false, 400},
		{"infinite longitude", "lat=60&lon=-Inf", false, 400},
		{"provider failure", "lat=60&lon=24", true, 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := NewHandler(fakeProvider(func(ctx context.Context, lat, lon float64) (Forecast, error) {
				called = true
				if tt.fail {
					return Forecast{}, &ProviderError{Operation: "fetch forecast", Kind: ErrProviderUnavailable, Cause: errors.New("private upstream detail")}
				}
				return Forecast{
					Latitude:  lat,
					Longitude: lon,
					Timezone:  "Europe/Helsinki",
				}, nil
			}))
			recorder := httptest.NewRecorder()
			handler.Weather(recorder, httptest.NewRequest(http.MethodGet, "/api/weather?"+tt.query, nil))
			if recorder.Code != tt.status {
				t.Fatalf("status %d, want %d", recorder.Code, tt.status)
			}
			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatal("expected JSON content type")
			}
			if tt.status == 200 {
				var got Forecast
				if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.Timezone != "Europe/Helsinki" {
					t.Fatalf("unexpected forecast: %+v", got)
				}
			} else {
				var got struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				want := "invalid latitude or longitude"
				if tt.fail {
					want = "weather provider unavailable"
				}
				if got.Error != want {
					t.Fatalf("public error %q, want %q", got.Error, want)
				}
			}
			if called != (tt.status != 400) {
				t.Fatal("unexpected provider call")
			}
		})
	}
}

func TestHandlerPassesRequestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := NewHandler(fakeProvider(func(received context.Context, lat, lon float64) (Forecast, error) {
		if received != ctx {
			t.Fatal("provider did not receive request context")
		}
		if lat != 60.17 || lon != 24.94 {
			t.Fatal("wrong coordinates")
		}
		return Forecast{}, nil
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/weather?lat=60.17&lon=24.94", nil).WithContext(ctx)
	handler.Weather(httptest.NewRecorder(), req)
}

func TestWeatherRouteRejectsPost(t *testing.T) {
	handler := NewHandler(fakeProvider(func(context.Context, float64, float64) (Forecast, error) {
		t.Fatal("provider called for POST")
		return Forecast{}, nil
	}))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/weather", handler.Weather)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/weather", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", recorder.Code)
	}
}

func TestHandlerErrorCategories(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{
		{ErrInvalidCoordinates, 400},
		{fmt.Errorf("private wrapper: %w", ErrProviderUnavailable), 503}, {ErrInvalidLocation, 422}, {ErrProviderResponse, 502},
		{ErrProviderUnavailable, 503}, {ErrProviderConfiguration, 500}, {errors.New("private details"), 500},
	} {
		handler := NewHandler(fakeProvider(func(context.Context, float64, float64) (Forecast, error) {
			return Forecast{}, &ProviderError{Operation: "private operation", Kind: tt.err, Cause: errors.New("private cause")}
		}))
		recorder := httptest.NewRecorder()
		handler.Weather(recorder, httptest.NewRequest("GET", "/api/weather?lat=60&lon=24", nil))
		if recorder.Code != tt.status {
			t.Fatalf("%v: status %d, want %d", tt.err, recorder.Code, tt.status)
		}
		var body map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if recorder.Header().Get("Content-Type") != "application/json" || strings.Contains(body["error"], "private") {
			t.Fatal("unsafe error response")
		}
	}
}
