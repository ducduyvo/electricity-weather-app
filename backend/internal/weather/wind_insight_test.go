package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func hasWindTitle(items []Signal, title string) bool {
	for _, item := range items {
		if item.Title == title {
			return true
		}
	}
	return false
}

func windDays(speeds ...float64) []DailySummary {
	days := make([]DailySummary, len(speeds))
	start := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for i, speed := range speeds {
		days[i] = DailySummary{Date: start.AddDate(0, 0, i).Format("2006-01-02"), CompleteDay: true, HourCount: 24, WindSampleCount: 24, AvgWindSpeed: f(speed), MaxWindSpeed: f(speed), WindDirectionSampleCount: 24, AvgWindDirection: f(45), WindDirectionConsistency: 1}
	}
	return days
}

func windHours(speeds ...float64) []Point {
	var points []Point
	start := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for _, speed := range speeds {
		for hour := 0; hour < 6; hour++ {
			points = append(points, Point{Time: start.Add(time.Duration(len(points)) * time.Hour).Format("2006-01-02T15:04"), WindSpeed: f(speed), WindDirection: f(45)})
		}
	}
	return points
}

// Test focus on wind potential boundaries, including classification of wind speeds into categories and handling of invalid or missing values.

func TestWindPotentialBoundaries(t *testing.T) {
	for _, tt := range []struct {
		speed float64
		want  string
	}{
		{0, "very_low"}, {2.99, "very_low"}, {3, "low"}, {5.99, "low"}, {6, "moderate"}, {9.99, "moderate"}, {10, "high"}, {15, "high"}, {15.01, "strong_uncertain"}, {50, "strong_uncertain"},
	} {
		if got := windPotential(f(tt.speed)); got == nil || *got != tt.want {
			t.Fatalf("speed %v: %v, want %s", tt.speed, got, tt.want)
		}
	}
	for _, speed := range []*float64{nil, f(-1), f(math.NaN()), f(math.Inf(1))} {
		if windPotential(speed) != nil {
			t.Fatal("invalid wind classified")
		}
	}
}

func TestOpenMeteoWindMappingAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hourly") != "temperature_2m,shortwave_radiation,wind_speed_120m,wind_direction_120m" || r.URL.Query().Get("wind_speed_unit") != "ms" || r.URL.Query().Get("forecast_days") != "15" {
			t.Errorf("request %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"timezone":"Europe/Helsinki","hourly":{"time":[1791579600,1791583200,1791586800,1791590400],"temperature_2m":[10,11],"shortwave_radiation":[0,100],"wind_speed_120m":[0,12,null],"wind_direction_120m":[360,90,null]}}`))
	}))
	defer server.Close()
	forecast, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 60, 24)
	if err != nil {
		t.Fatal(err)
	}
	if *forecast.Hourly[0].WindSpeed != 0 || *forecast.Hourly[0].WindDirection != 360 || *forecast.Hourly[1].WindSpeed != 12 || *forecast.Hourly[1].WindDirection != 90 || *forecast.Hourly[1].WindPotential != "high" {
		t.Fatalf("wind mapping %+v", forecast.Hourly)
	}
	if *forecast.Hourly[0].Temperature != 10 || *forecast.Hourly[1].SolarIrradiance != 100 {
		t.Fatal("existing mapping changed")
	}
	for _, point := range forecast.Hourly[2:] {
		if point.WindSpeed != nil || point.WindDirection != nil || point.WindPotential != nil {
			t.Fatal("missing wind fabricated")
		}
	}
	day := forecast.Daily[0]
	if *day.AvgWindSpeed != 6 || *day.MinWindSpeed != 0 || *day.MaxWindSpeed != 12 || day.WindSampleCount != 2 || day.CompleteDay {
		t.Fatalf("summary %+v", day)
	}
	body, err := json.Marshal(forecast)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"windSpeed":0`, `"windDirection":360`, `"WindPotential":"high"`, `"windSpeed":null`, `"windDirection":null`, `"WindPotential":null`, `"avgWindSpeed":6`, `"minWindSpeed":0`, `"maxWindSpeed":12`, `"windSampleCount":2`} {
		if !strings.Contains(string(body), field) {
			t.Fatalf("missing JSON %s", field)
		}
	}
}

func TestMissingAndInvalidWind(t *testing.T) {
	for _, direction := range []*float64{nil, f(-1), f(361), f(math.NaN()), f(math.Inf(1))} {
		if validWindValue(direction, true) != nil {
			t.Fatal("invalid direction accepted")
		}
	}
	points := []Point{{Time: "2026-10-10T00:00"}, {Time: "2026-10-10T01:00", WindSpeed: f(-1)}}
	day := summarize(points)[0]
	if day.AvgWindSpeed != nil || day.MinWindSpeed != nil || day.MaxWindSpeed != nil || day.WindSampleCount != 0 {
		t.Fatalf("missing wind became zero %+v", day)
	}
}

func TestWindSummaryFullDayAndDuplicate(t *testing.T) {
	points := hourlyPoints(1)
	for i := range points {
		points[i].WindSpeed = f(float64(i))
	}
	points = append(points, points[0])
	day := summarize(points)[0]
	if !day.CompleteDay || day.WindSampleCount != 24 || *day.AvgWindSpeed != 11.5 || *day.MinWindSpeed != 0 || *day.MaxWindSpeed != 23 {
		t.Fatalf("wind coverage %+v", day)
	}
}

func TestWindSummaryMissingSamplesOnCompleteDay(t *testing.T) {
	points := hourlyPoints(1)
	for i := range points {
		points[i].WindSpeed = f(10)
	}
	points[0].WindSpeed = f(0)
	points[1].WindSpeed = nil
	day := summarize(points, "Europe/Helsinki")[0]
	if !day.CompleteDay || day.WindSampleCount != 23 || *day.AvgWindSpeed != 220.0/23 || *day.MinWindSpeed != 0 || *day.MaxWindSpeed != 10 {
		t.Fatalf("missing sample treated as zero: %+v", day)
	}
}

func TestWindMinimumIgnoresMissingAndInvalidValues(t *testing.T) {
	points := hourlyPoints(1)[:4]
	points[0].WindSpeed = nil
	points[1].WindSpeed = f(-1)
	points[2].WindSpeed = f(7)
	points[3].WindSpeed = f(5)
	day := summarize(points)[0]
	if *day.MinWindSpeed != 5 || *day.MaxWindSpeed != 7 || *day.AvgWindSpeed != 6 || day.WindSampleCount != 2 {
		t.Fatalf("minimum fabricated: %+v", day)
	}
}

func TestDailyWindDirection(t *testing.T) {
	for _, tt := range []struct {
		name       string
		directions []*float64
		speeds     []*float64
		want       *float64
		count      int
	}{
		{name: "north wraparound", directions: []*float64{f(350), f(10)}, speeds: []*float64{f(5), f(5)}, want: f(0), count: 2},
		{name: "east", directions: []*float64{f(90), f(90)}, speeds: []*float64{f(5), f(7)}, want: f(90), count: 2},
		{name: "opposing", directions: []*float64{f(0), f(180)}, speeds: []*float64{f(5), f(5)}, count: 2},
		{name: "missing and calm", directions: []*float64{nil, f(90)}, speeds: []*float64{f(5), f(0)}},
		{name: "partial valid", directions: []*float64{f(45), f(400)}, speeds: []*float64{f(5), f(5)}, want: f(45), count: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			summary := summarize([]Point{
				{Time: "2026-10-10T00:00", WindSpeed: tt.speeds[0], WindDirection: tt.directions[0]},
				{Time: "2026-10-10T01:00", WindSpeed: tt.speeds[1], WindDirection: tt.directions[1]},
			})[0]
			if summary.WindDirectionSampleCount != tt.count {
				t.Fatalf("count = %d, want %d", summary.WindDirectionSampleCount, tt.count)
			}
			if tt.want == nil {
				if summary.AvgWindDirection != nil {
					t.Fatal("unavailable direction should be nil")
				}
				return
			}
			if summary.AvgWindDirection == nil {
				t.Fatal("missing direction")
			}
			difference := math.Abs(*summary.AvgWindDirection - *tt.want)
			if math.Min(difference, 360-difference) > 1e-8 {
				t.Fatalf("direction = %v, want %v", *summary.AvgWindDirection, *tt.want)
			}
		})
	}
}

// Test focus on wind trends and low-wind periods, including daily and hourly insights, as well as wind direction consistency and combined conditions.

func TestDailyWindTrends(t *testing.T) {
	cfg := DefaultWindInsightConfig()
	for _, tt := range []struct {
		name   string
		speeds []float64
		title  string
	}{
		{name: "increase", speeds: []float64{4, 5, 7}, title: "Wind conditions improving"},
		{name: "decrease", speeds: []float64{9, 7, 6}, title: "Wind conditions weakening"},
		{name: "small", speeds: []float64{4, 4.5, 5}},
		{name: "isolated", speeds: []float64{4, 9, 4}},
		{name: "strong uncertain", speeds: []float64{12, 14, 16}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			items := dailyWindInsights(windDays(tt.speeds...), cfg)
			if tt.title != "" {
				if !hasWindTitle(items, tt.title) {
					t.Fatalf("missing %s: %+v", tt.title, items)
				}
			} else if len(items) != 0 {
				t.Fatalf("unexpected %+v", items)
			}
		})
	}
	for _, missing := range []string{"partial", "gap", "null"} {
		days := windDays(4, 5, 8)
		switch missing {
		case "partial":
			days[1].WindSampleCount = 23
		case "gap":
			days[1].Date = "2026-10-14"
		case "null":
			days[1].AvgWindSpeed = nil
		}
		if len(dailyWindInsights(days, cfg)) != 0 {
			t.Fatalf("bridged %s", missing)
		}
	}
	days := windDays(4, 5, 8)
	cfg.SpeedChange = 5
	if len(dailyWindInsights(days, cfg)) != 0 {
		t.Fatal("configured change ignored")
	}
	days = windDays(4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 7)
	if !hasWindTitle(dailyWindInsights(days, DefaultWindInsightConfig()), "Wind conditions improving") {
		t.Fatal("late 14-day trend missing")
	}
	days = append(windDays(4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4), windDays(20)...)
	if len(dailyWindInsights(days, DefaultWindInsightConfig())) != 0 {
		t.Fatal("included day 15")
	}
}

func TestHourlyWindTrendsAndLowPeriods(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	cfg := DefaultWindInsightConfig()
	for _, tt := range []struct {
		speeds []float64
		title  string
	}{
		{speeds: []float64{4, 5, 7}, title: "Wind conditions improving"},
		{speeds: []float64{9, 7, 6}, title: "Wind conditions weakening"},
		{speeds: []float64{2, 2}, title: "Prolonged low-wind period"},
	} {
		if !hasWindTitle(hourlyWindInsights(windHours(tt.speeds...), "UTC", now, cfg), tt.title) {
			t.Fatalf("missing %s", tt.title)
		}
	}
	points := windHours(4, 5, 7)
	points[7].WindSpeed = nil
	if hasWindTitle(hourlyWindInsights(points, "UTC", now, cfg), "Wind conditions improving") {
		t.Fatal("bridged missing hour")
	}
	points = windHours(2, 2)
	points[7].WindSpeed = f(8)
	if hasWindTitle(hourlyWindInsights(points, "UTC", now, cfg), "Prolonged low-wind period") {
		t.Fatal("low hours not consecutive")
	}
	if len(hourlyWindInsights(windHours(2), "UTC", now, cfg)) != 0 {
		t.Fatal("isolated low window")
	}
	if len(hourlyWindInsights(windHours(4, 5, 7), "UTC", now.Add(72*time.Hour), cfg)) != 0 {
		t.Fatal("included past forecast")
	}
	if hasWindTitle(dailyWindInsights(windDays(2, 2), cfg), "Prolonged low-wind period") == false {
		t.Fatal("daily low missing")
	}
	days := windDays(2, 2)
	days[1].CompleteDay = false
	if len(dailyWindInsights(days, cfg)) != 0 {
		t.Fatal("partial low period")
	}
}

func TestWindDirectionConsistencyAndCircularDifference(t *testing.T) {
	cfg := DefaultWindInsightConfig()
	if angleDifference(350, 10) != 20 || angleDifference(10, 350) != 20 {
		t.Fatal("angle wrap")
	}
	days := windDays(5, 5)
	days[0].AvgWindDirection = f(350)
	days[1].AvgWindDirection = f(10)
	if len(dailyWindInsights(days, cfg)) != 0 {
		t.Fatal("small wrapped shift")
	}
	days[1].AvgWindDirection = f(80)
	if !hasWindTitle(dailyWindInsights(days, cfg), "Wind direction shift") {
		t.Fatal("shift missing")
	}
	for _, invalid := range []string{"variable", "calm", "missing"} {
		testDays := append([]DailySummary(nil), days...)
		switch invalid {
		case "variable":
			testDays[1].WindDirectionConsistency = 0.3
		case "calm":
			testDays[1].AvgWindSpeed = f(0)
		case "missing":
			testDays[1].WindDirectionSampleCount = 23
		}
		if hasWindTitle(dailyWindInsights(testDays, cfg), "Wind direction shift") {
			t.Fatalf("shift with %s", invalid)
		}
	}
	points := windHours(5, 5)
	for i := 6; i < 12; i++ {
		points[i].WindDirection = f(120)
	}
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	if !hasWindTitle(hourlyWindInsights(points, "UTC", now, cfg), "Wind direction shift") {
		t.Fatal("hourly shift missing")
	}
	for i := 6; i < 12; i++ {
		points[i].WindDirection = f(float64(i%2) * 180)
	}
	if len(hourlyWindInsights(points, "UTC", now, cfg)) != 0 {
		t.Fatal("variable hourly directions")
	}
}

func TestCombinedWindConditions(t *testing.T) {
	cfg := DefaultWindInsightConfig()
	days := windDays(2, 2, 2)
	for i := range days {
		days[i].AvgTemperature = f(12 - float64(i)*2)
		days[i].TemperatureSampleCount = 24
		days[i].SolarSampleCount = 24
		days[i].SolarEnergyKWhM2 = 0.5
	}
	title := "Cold, low-solar and low-wind conditions"
	if !hasWindTitle(dailyWindInsights(days, cfg), title) {
		t.Fatal("combined daily missing")
	}
	days[1].SolarSampleCount = 23
	if hasWindTitle(dailyWindInsights(days, cfg), title) {
		t.Fatal("combined with missing solar")
	}
	points := windHours(2, 2, 2, 2)
	for i := range points {
		points[i].Temperature = f(12 - float64(i/6)*4)
		points[i].SolarIrradiance = f(50)
	}
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	items := hourlyWindInsights(points, "UTC", now, cfg)
	if !hasWindTitle(items, title) {
		t.Fatal("combined hourly missing")
	}
	existing := []Signal{{Kind: "temperature", Title: "Keep"}, {Kind: "combined", Title: "Old"}}
	merged := mergeWindInsights(existing, items)
	if !hasWindTitle(merged, "Old") || !hasWindTitle(merged, "Keep") {
		t.Fatal("combined merge")
	}
	for _, s := range items {
		if s.Kind == "combined" && !strings.Contains(s.Detail, "m/s") {
			t.Fatal("units missing")
		}
	}
}

func TestDailyWindConsistency(t *testing.T) {
	points := windHours(5)
	for i := range points {
		points[i].WindDirection = f(float64(i%2) * 180)
	}
	day := summarize(points)[0]
	if math.Abs(day.WindDirectionConsistency) > 1e-8 || day.AvgWindDirection != nil {
		t.Fatal(fmt.Sprintf("inconsistent direction %+v", day))
	}
}

func TestWindInsightsOpenMeteoIntegration(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Hour)
	points := windHours(9, 7, 6)
	timestamps := make([]int64, len(points))
	speeds := make([]float64, len(points))
	directions := make([]float64, len(points))
	for i, p := range points {
		timestamps[i] = start.Add(time.Duration(i) * time.Hour).Unix()
		speeds[i] = *p.WindSpeed
		directions[i] = *p.WindDirection
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]any{"timezone": "UTC", "hourly": map[string]any{"time": timestamps, "wind_speed_120m": speeds, "wind_direction_120m": directions}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	forecast, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasWindTitle(forecast.Signals, "Wind conditions weakening") {
		t.Fatalf("wind not wired: %+v", forecast.Signals)
	}
	original := signals(forecast.Daily)
	if !reflect.DeepEqual(original, forecast.Signals[:len(original)]) {
		t.Fatal("temperature/solar signals changed")
	}
	encoded, err := json.Marshal(forecast)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"kind":"wind"`) || !strings.Contains(string(encoded), `"windDirectionConsistency":1`) {
		t.Fatalf("missing JSON fields: %s", encoded)
	}
}

func TestHourlyWindInsightFractionalTimezone(t *testing.T) {
	// 06:05 UTC is 11:50 in Kathmandu: the current local-hour slot is 11:00.
	now := time.Date(2026, 10, 10, 6, 5, 0, 0, time.UTC)
	points := windHours(4, 5, 7)
	for i := range points {
		points[i].Time = time.Date(2026, 10, 10, 11+i, 0, 0, 0, time.UTC).Format("2006-01-02T15:04")
	}
	if !hasWindTitle(hourlyWindInsights(points, "Asia/Kathmandu", now, DefaultWindInsightConfig()), "Wind conditions improving") {
		t.Fatal("current local hour incorrectly excluded")
	}
}

func TestWindInsightsReportBothDatedTrendDirections(t *testing.T) {
	cfg := DefaultWindInsightConfig()
	speeds := []float64{9, 7, 5, 7, 9}
	cases := []struct {
		name  string
		items []Signal
	}{
		{name: "daily", items: dailyWindInsights(windDays(speeds...), cfg)},
		{name: "hourly", items: hourlyWindInsights(windHours(speeds...), "UTC", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), cfg)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			counts := map[string]int{}
			details := map[string]string{}
			for _, item := range tt.items {
				counts[item.Title]++
				details[item.Title] = item.Detail
			}
			for _, title := range []string{"Wind conditions weakening", "Wind conditions improving"} {
				if counts[title] != 1 {
					t.Fatalf("%s count=%d", title, counts[title])
				}
			}
			if details["Wind conditions weakening"] == details["Wind conditions improving"] {
				t.Fatal("trend periods not distinguished")
			}
			if !strings.Contains(details["Wind conditions weakening"], "changes by -4.0 m/s") || !strings.Contains(details["Wind conditions improving"], "changes by +4.0 m/s") {
				t.Fatal("incorrect values")
			}
		})
	}
}

func TestWindShiftMessageIncludesCorrespondingNamesAndDates(t *testing.T) {
	days := windDays(5, 5)
	days[0].AvgWindDirection = f(45)
	days[1].AvgWindDirection = f(135)
	for _, item := range dailyWindInsights(days, DefaultWindInsightConfig()) {
		if item.Title == "Wind direction shift" {
			for _, part := range []string{"northeasterly (45°)", "southeasterly (135°)", days[0].Date, days[1].Date} {
				if !strings.Contains(item.Detail, part) {
					t.Fatalf("missing %q in %s", part, item.Detail)
				}
			}
			return
		}
	}
	t.Fatal("direction shift missing")
}

func TestAvgWindDirection(t *testing.T) {
	for _, tt := range []struct {
		name       string
		directions []float64
		want       *float64
	}{
		{name: "missing"},
		{name: "east", directions: []float64{90, 90}, want: f(90)},
		{name: "north wraparound", directions: []float64{350, 10}, want: f(0)},
		{name: "opposing", directions: []float64{0, 180}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var sinSum, cosSum float64
			for _, direction := range tt.directions {
				radians := direction * math.Pi / 180
				sinSum += math.Sin(radians)
				cosSum += math.Cos(radians)
			}
			average, consistency := avgWindDirection(sinSum, cosSum, len(tt.directions))
			if tt.want == nil {
				if average != nil || consistency > 1e-6 {
					t.Fatalf("unexpected average %v, consistency %v", average, consistency)
				}
				return
			}
			if average == nil || angleDifference(*average, *tt.want) > 1e-8 {
				t.Fatalf("average %v, want %v", average, *tt.want)
			}
			if consistency < 0.98 || consistency > 1+1e-12 {
				t.Fatalf("unexpected consistency %v", consistency)
			}
		})
	}
}
