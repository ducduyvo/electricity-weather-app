package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func hourlyPoints(dayCount int) []Point {
	points := []Point{}
	for day := range dayCount {
		for hour := range 24 {
			points = append(points, Point{
				Time:        fmt.Sprintf("2026-10-%02dT%02d:00", day+7, hour),
				Temperature: f(float64(hour)), SolarIrradiance: f(100),
			})
		}
	}
	return points
}

func TestFourteenDailySummaries(t *testing.T) {
	days := summarize(hourlyPoints(14), "Europe/Helsinki")
	if len(days) != 14 {
		t.Fatalf("got %d days", len(days))
	}
	for i, day := range days {
		if day.Date != fmt.Sprintf("2026-10-%02d", i+7) || !day.CompleteDay || day.HourCount != 24 || day.ExpectedHourCount != 24 {
			t.Fatalf("bad day %+v", day)
		}
		if *day.MinTemperature != 0 || *day.MaxTemperature != 23 || *day.AvgTemperature != 11.5 || day.TemperatureSampleCount != 24 {
			t.Fatalf("temperature %+v", day)
		}
		wantSamples := 24
		if i == 13 {
			wantSamples = 23
		} // No next-midnight interval in this fixture.
		if math.Abs(day.SolarEnergyKWhM2-float64(wantSamples)*0.1) > 1e-9 || day.SolarSampleCount != wantSamples {
			t.Fatalf("solar %+v", day)
		}
	}
}

func TestIncompleteAndMissingDays(t *testing.T) {
	points := hourlyPoints(7)[12 : 7*24-6]
	points[2].Temperature = nil
	points[3].SolarIrradiance = nil
	days := summarize(points, "Europe/Helsinki")
	if len(days) != 7 || days[0].CompleteDay || days[6].CompleteDay || days[0].HourCount != 12 || days[6].HourCount != 18 {
		t.Fatalf("partial days %+v", days)
	}
	if days[0].TemperatureSampleCount != 11 || days[0].SolarSampleCount != 12 || completeSolar(days[0]) {
		t.Fatalf("missing readings %+v", days[0])
	}
	if got := summarize(points[:2]); len(got) != 1 {
		t.Fatalf("invented days %+v", got)
	}
}

func TestLocalDayGroupingAndDST(t *testing.T) {
	days := summarize([]Point{
		{Time: "2026-10-08T00:00", Temperature: f(2)},
		{Time: "2026-10-07T23:00", Temperature: f(1)},
		{Time: "invalid timestamp", Temperature: f(99)},
	}, "Europe/Helsinki")
	if len(days) != 2 || days[0].Date != "2026-10-07" || days[1].Date != "2026-10-08" || *days[0].AvgTemperature != 1 {
		t.Fatalf("local grouping %+v", days)
	}
	for _, tt := range []struct {
		date  string
		hours int
	}{
		{"2026-03-29", 23}, {"2026-10-25", 25},
	} {
		location, err := time.LoadLocation("Europe/Helsinki")
		if err != nil {
			t.Fatal(err)
		}
		start, _ := time.ParseInLocation("2006-01-02", tt.date, location)
		points := []Point{}
		for i := range tt.hours {
			instant := start.Add(time.Duration(i) * time.Hour)
			unixTime := instant.Unix()
			points = append(points, Point{UnixTime: &unixTime, Time: instant.Format("2006-01-02T15:04"), Temperature: f(10), SolarIrradiance: f(0)})
		}
		day := summarize(points, "Europe/Helsinki")[0]
		if day.ExpectedHourCount != tt.hours || !day.CompleteDay {
			t.Fatalf("DST coverage %+v", day)
		}
	}
}

func trendDays(temperatures, solar []float64) []DailySummary {
	days := []DailySummary{}
	for i, temp := range temperatures {
		days = append(days, DailySummary{
			Date: fmt.Sprintf("2026-10-%02d", 7+i), AvgTemperature: f(temp),
			SolarEnergyKWhM2: solar[i], HourCount: 24, ExpectedHourCount: 24,
			TemperatureSampleCount: 24, SolarSampleCount: 24, CompleteDay: true,
		})
	}
	return days
}

func TestWeeklyTrends(t *testing.T) {
	for _, tt := range []struct {
		name                string
		temperatures, solar []float64
		title               string
	}{
		{"cold spell", []float64{10, 8, 6, 4}, []float64{4, 4, 4, 4}, "Cold spell developing"},
		{"warming", []float64{10, 12, 14, 16}, []float64{4, 4, 4, 4}, "Warming trend"},
		{"solar improving", []float64{10, 10, 10}, []float64{2, 3, 4}, "Solar availability improving"},
		{"solar weakening", []float64{10, 10, 10}, []float64{4, 3, 2}, "Solar availability weakening"},
		{"reversal", []float64{10, 5, 10}, []float64{4, 1, 4}, "No clear multi-day trend"},
		{"insufficient", []float64{10, 5}, []float64{4, 2}, "No clear multi-day trend"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := outlookSignals(trendDays(tt.temperatures, tt.solar))
			if findSignal(got, tt.title) == nil {
				t.Fatalf("signals %+v", got)
			}
		})
	}
}

func TestWeeklyMissingDataBreaksTrend(t *testing.T) {
	for _, kind := range []string{"partial", "missing temperature", "missing solar", "date gap"} {
		days := trendDays([]float64{10, 8, 6}, []float64{4, 3, 2})
		switch kind {
		case "partial":
			days[1].CompleteDay = false
		case "missing temperature":
			days[1].AvgTemperature = nil
		case "missing solar":
			days[1].SolarSampleCount = 23
		case "date gap":
			days[1].Date = "2026-10-09"
			days[2].Date = "2026-10-10"
		}
		got := outlookSignals(days)
		if kind != "missing solar" && findSignal(got, "Cold spell developing") != nil {
			t.Fatalf("invalid temperature trend %+v", got)
		}
		if kind != "missing temperature" && findSignal(got, "Solar availability weakening") != nil {
			t.Fatalf("invalid solar trend %+v", got)
		}
	}
}

func TestOpenMeteoFourteenDayForecast(t *testing.T) {
	points := hourlyPoints(14)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("forecast_days") != "15" || r.URL.Query().Get("timezone") != "auto" {
			t.Error("request lost extra-day local forecast")
		}
		var raw omResponse
		raw.Timezone = "Europe/Helsinki"
		for _, point := range points {
			location, _ := time.LoadLocation("Europe/Helsinki")
			timestamp, _ := time.ParseInLocation("2006-01-02T15:04", point.Time, location)
			raw.Hourly.Time = append(raw.Hourly.Time, timestamp.Unix())
			raw.Hourly.Temperature = append(raw.Hourly.Temperature, point.Temperature)
			raw.Hourly.Solar = append(raw.Hourly.Solar, point.SolarIrradiance)
		}
		_ = json.NewEncoder(w).Encode(raw)
	}))
	defer server.Close()
	got, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 60, 24)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(got.Hourly) != 336 || len(got.Daily) != 14 || len(got.OutlookSignals) == 0 {
		t.Fatalf("one-call forecast: calls=%d hours=%d days=%d outlook=%+v", calls, len(got.Hourly), len(got.Daily), got.OutlookSignals)
	}
}

func TestSolarTrendCanStartAfterZeroBaseline(t *testing.T) {
	days := trendDays([]float64{10, 10, 10, 10}, []float64{0, 1, 2, 3})
	if got := outlookSignals(days); findSignal(got, "Solar availability improving") == nil {
		t.Fatalf("valid positive-baseline run lost: %+v", got)
	}
}
