package weather

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func solarIntervalPoints(start time.Time) []Point {
	var points []Point
	for instant := start; !instant.After(start.AddDate(0, 0, 1)); instant = instant.Add(time.Hour) {
		unix := instant.Unix()
		solar := 100.0
		if instant.Equal(start) {
			solar = 9000
		} // Prior day's final interval.
		if instant.Equal(start.AddDate(0, 0, 1)) {
			solar = 500
		}
		points = append(points, Point{UnixTime: &unix, Time: instant.Format("2006-01-02T15:04"), Temperature: f(5), WindSpeed: f(7), SolarIrradiance: f(solar)})
	}
	return points
}

func TestSolarIntervalsLocalDays(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Helsinki")
	for _, date := range []string{"2026-10-14", "2026-03-29", "2026-10-25"} {
		t.Run(date, func(t *testing.T) {
			start, _ := time.ParseInLocation("2006-01-02", date, location)
			points := solarIntervalPoints(start)
			hours := int(start.AddDate(0, 0, 1).Sub(start) / time.Hour)
			day := summarize(points, "Europe/Helsinki")[0]
			if day.SolarSampleCount != hours || !completeSolar(day) || math.Abs(day.SolarEnergyKWhM2-(float64(hours-1)*0.1+0.5)) > 1e-9 {
				t.Fatalf("incorrect interval total: %+v", day)
			}
			if day.HourCount != hours || *day.AvgTemperature != 5 || *day.AvgWindSpeed != 7 {
				t.Fatal("instantaneous measurements changed")
			}
			if day.PeakSolarTime != points[len(points)-1].Time || *day.PeakSolarIrradiance != 500 {
				t.Fatal("wrong peak interval")
			}
			missing := summarize(points[:len(points)-1], "Europe/Helsinki")[0]
			if !missing.CompleteDay || missing.SolarSampleCount != hours-1 || completeSolar(missing) {
				t.Fatal("missing final interval treated as complete")
			}
			points[2].SolarIrradiance = nil
			day = summarize(points, "Europe/Helsinki")[0]
			if day.SolarSampleCount != hours-1 || completeSolar(day) {
				t.Fatal("missing solar measurement counted")
			}
			// A truncated first day can have as many valid solar intervals as
			// timestamps, but must still be classified as incomplete.
			points = solarIntervalPoints(start)
			partial := summarize(points[3:len(points)-1], "Europe/Helsinki")[0]
			if partial.CompleteDay || partial.ExpectedHourCount != hours || partial.HourCount != hours-3 || partial.SolarSampleCount != hours-3 || completeSolar(partial) {
				t.Fatalf("truncated day classified incorrectly: %+v", partial)
			}
			if math.Abs(partial.SolarEnergyKWhM2-float64(hours-3)*0.1) > 1e-9 {
				t.Fatal("missing intervals invented energy")
			}
			points = solarIntervalPoints(start)
			points = append(points, points[1])
			if summarize(points, "Europe/Helsinki")[0].SolarSampleCount != hours {
				t.Fatal("duplicate interval counted")
			}
		})
	}
}

func TestOpenMeteoFinalMidnightSolarInterval(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 14)
	raw := omResponse{Timezone: "UTC"}
	for instant := start; !instant.After(end); instant = instant.Add(time.Hour) {
		raw.Hourly.Time = append(raw.Hourly.Time, instant.Unix())
		raw.Hourly.Temperature = append(raw.Hourly.Temperature, f(5))
		solar := 100.0
		if instant.Equal(start) {
			solar = 9000
		}
		if instant.Equal(end) {
			solar = 500
		}
		raw.Hourly.Solar = append(raw.Hourly.Solar, f(solar))
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(raw) }))
	defer server.Close()
	forecast, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 60, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(forecast.Daily) != 14 || len(forecast.Hourly) != 14*24 {
		t.Fatal("extra boundary point leaked into forecast")
	}
	last := forecast.Daily[13]
	if !completeSolar(last) || math.Abs(last.SolarEnergyKWhM2-2.8) > 1e-9 || last.PeakSolarTime != end.Format("2006-01-02T15:04") {
		t.Fatalf("final solar interval omitted: %+v", last)
	}
}
