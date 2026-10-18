package weather

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func findSignal(items []Signal, title string) *Signal {
	for _, item := range items {
		if item.Title == title {
			return &item
		}
	}
	return nil
}

func TestSolarPeakDetection(t *testing.T) {
	got := summarize([]Point{
		{
			Time:            "2026-10-07T12:00",
			Temperature:     nil,
			SolarIrradiance: f(400),
		},
		{
			Time:            "2026-10-07T13:00",
			Temperature:     nil,
			SolarIrradiance: f(510),
		},
		{
			Time:            "2026-10-07T14:00",
			Temperature:     nil,
			SolarIrradiance: f(510),
		},
		{
			Time:            "2026-10-07T15:00",
			Temperature:     nil,
			SolarIrradiance: nil,
		},
	})[0]
	if *got.PeakSolarIrradiance != 510 ||
		got.PeakSolarTime != "2026-10-07T13:00" ||
		got.SolarSampleCount != 3 ||
		got.HourCount != 4 {
		t.Fatalf("peak summary %+v", got)
	}
	peak := findSignal(signals([]DailySummary{got}), "Strong solar period expected")
	if peak == nil ||
		!strings.Contains(peak.Detail, "13:00") ||
		!strings.Contains(peak.Detail, "510 W/m²") ||
		!strings.Contains(peak.Detail, "missing") {
		t.Fatalf("peak signal %+v", peak)
	}
}

func TestTemperatureDemandConditions(t *testing.T) {
	for _, tt := range []struct {
		name            string
		today, tomorrow float64
		want, absent    string
	}{
		{"heating increases", 10, 4.8, "fall 5.2°C", "cooling-driven"},
		{"cooling increases", 22, 27, "increase cooling-driven", "may ease"},
		{"heating eases", 10, 15, "Heating-driven electricity demand conditions may ease", "increase cooling"},
		{"hot weather cools", 30, 25, "Cooling-driven electricity demand conditions may ease", "increase heating"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := temperatureSignal(tt.tomorrow-tt.today, tt.today, tt.tomorrow)
			if got == nil || !strings.Contains(got.Detail, tt.want) || strings.Contains(got.Detail, tt.absent) {
				t.Fatalf("temperature signal %+v", got)
			}
		})
	}
}

func TestCombinedOutlooks(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		today, tomorrow, solar float64
		title                  string
	}{
		{"cold and low solar", 10, 5, 2, "Potentially tighter weather-driven conditions"},
		{"mild and high solar", 15, 20, 5, "Potentially more favorable weather-driven conditions"},
		{"hot but cooling", 30, 25, 2, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := signals([]DailySummary{
				{
					AvgTemperature:         f(tt.today),
					SolarEnergyKWhM2:       4,
					SolarSampleCount:       24,
					HourCount:              24,
					CompleteDay:            true,
					TemperatureSampleCount: 24,
				},
				{
					AvgTemperature:         f(tt.tomorrow),
					SolarEnergyKWhM2:       tt.solar,
					SolarSampleCount:       24,
					HourCount:              24,
					CompleteDay:            true,
					TemperatureSampleCount: 24,
				},
			})
			if tt.title != "" {
				if findSignal(got, tt.title) == nil {
					t.Fatalf("combined missing: %+v", got)
				}
			} else {
				for _, item := range got {
					if item.Kind == "combined" {
						t.Fatalf("unexpected combined: %+v", item)
					}
				}
			}
		})
	}
}

func TestMissingSolarDoesNotImplyDrop(t *testing.T) {
	for _, solar := range []*float64{nil, f(-1)} {
		daily := summarize([]Point{
			{
				Time:            "2026-10-07T12:00",
				Temperature:     f(10),
				SolarIrradiance: f(500),
			},
			{
				Time:            "2026-10-08T12:00",
				Temperature:     f(5),
				SolarIrradiance: solar,
			},
		})
		got := signals(daily)
		if daily[1].PeakSolarIrradiance != nil || daily[1].SolarSampleCount != 0 {
			t.Fatal("missing/invalid irradiance treated as valid")
		}
		if findSignal(got, "Lower solar availability") != nil ||
			findSignal(got, "Potentially tighter weather-driven conditions") != nil {
			t.Fatalf("missing solar produced comparison %+v", got)
		}
		if findSignal(got, "Colder tomorrow") != nil || findSignal(got, "Limited comparison data") == nil {
			t.Fatalf("partial temperature compared %+v", got)
		}
	}
}

func TestZeroSolarAndInsufficientForecast(t *testing.T) {
	daily := summarize([]Point{{
		Time:            "2026-10-07T12:00",
		Temperature:     nil,
		SolarIrradiance: f(0),
	}})
	if daily[0].PeakSolarIrradiance == nil ||
		*daily[0].PeakSolarIrradiance != 0 ||
		daily[0].SolarSampleCount != 1 {
		t.Fatal("valid zero treated as missing")
	}
	if got := signals(daily); len(got) != 1 || got[0].Title != "Insufficient forecast data" {
		t.Fatalf("single-day signals %+v", got)
	}
	if got := signals(nil); len(got) != 1 || got[0].Title != "Insufficient forecast data" {
		t.Fatalf("empty signals %+v", got)
	}
}

func TestPartialSolarSuppressesComparison(t *testing.T) {
	days := []DailySummary{
		{
			AvgTemperature:         f(10),
			SolarEnergyKWhM2:       4,
			SolarSampleCount:       24,
			HourCount:              24,
			CompleteDay:            true,
			TemperatureSampleCount: 24,
		},
		{
			AvgTemperature:         f(5),
			SolarEnergyKWhM2:       1,
			SolarSampleCount:       12,
			HourCount:              24,
			CompleteDay:            true,
			TemperatureSampleCount: 24,
		},
	}
	if got := signals(days); findSignal(got, "Lower solar availability") != nil {
		t.Fatalf("partial energy compared %+v", got)
	}
}

func TestZeroBaselineAllowsSolarDelta(t *testing.T) {
	got := signals([]DailySummary{
		{
			AvgTemperature:         f(10),
			SolarEnergyKWhM2:       0,
			SolarSampleCount:       24,
			HourCount:              24,
			CompleteDay:            true,
			TemperatureSampleCount: 24,
		},
		{
			AvgTemperature:         f(10),
			SolarEnergyKWhM2:       4,
			SolarSampleCount:       24,
			HourCount:              24,
			CompleteDay:            true,
			TemperatureSampleCount: 24,
		},
	})
	if findSignal(got, "Higher solar availability") == nil || findSignal(got, "Limited comparison data") != nil {
		t.Fatalf("zero baseline %+v", got)
	}
}

func TestIncompleteTemperatureSuppressesComparisons(t *testing.T) {
	for _, invalidDay := range []int{0, 1} {
		for _, missing := range []string{"partial day", "missing reading", "missing average", "empty day"} {
			t.Run(fmt.Sprintf("day=%d/%s", invalidDay, missing), func(t *testing.T) {
				days := trendDays([]float64{10, 5}, []float64{4, 2})
				switch missing {
				case "partial day":
					days[invalidDay].CompleteDay = false
				case "missing reading":
					days[invalidDay].TemperatureSampleCount--
				case "missing average":
					days[invalidDay].AvgTemperature = nil
				case "empty day":
					days[invalidDay].HourCount = 0
					days[invalidDay].TemperatureSampleCount = 0
				}
				got := signals(days)
				for _, signal := range got {
					if signal.Kind == "temperature" || signal.Kind == "combined" || signal.Title == "Stable weather outlook" {
						t.Fatalf("incomplete temperature produced comparison: %+v", got)
					}
				}
				if findSignal(got, "Limited comparison data") == nil {
					t.Fatalf("missing data-quality signal: %+v", got)
				}
				if missing == "missing reading" || missing == "missing average" {
					if findSignal(got, "Lower solar availability") == nil {
						t.Fatalf("valid solar comparison lost: %+v", got)
					}
				}
			})
		}
	}
}

func TestWindTrendSignalCodesAndParameters(t *testing.T) {
	for _, tt := range []struct {
		start, end     float64
		code, severity string
	}{
		{start: 5, end: 8.27, code: "wind_speed_increasing", severity: "info"},
		{start: 9.8, end: 2.1, code: "wind_speed_decreasing", severity: "warning"},
	} {
		signal := windTrendSignal("2026-10-12T16:00", "2026-10-13T21:00", tt.start, tt.end)
		if signal.Code != tt.code || signal.Severity != tt.severity || signal.Params["change"] != tt.end-tt.start {
			t.Fatalf("incorrect signal %+v", signal)
		}
		if len(signal.Params) != 3 || signal.Title == "" || !strings.Contains(signal.Detail, "Wind speed indicates generation conditions, not actual production.") {
			t.Fatal("fallback or params changed")
		}
		body, err := json.Marshal(signal)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Code          string
			Params        map[string]any
			Title, Detail string
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Code != tt.code || decoded.Params["start"] != "2026-10-12T16:00" || decoded.Params["end"] != "2026-10-13T21:00" || decoded.Title != signal.Title || decoded.Detail != signal.Detail {
			t.Fatal("JSON contract")
		}
	}
	body, err := json.Marshal(Signal{Kind: "solar", Title: "Existing signal"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"code"`) || strings.Contains(string(body), `"params"`) {
		t.Fatal("optional fields emitted for legacy signals")
	}
}

func TestTemperatureSolarCodesAndParameters(t *testing.T) {
	for _, tt := range []struct {
		delta, today, tomorrow float64
		code, interpretation   string
	}{
		{delta: -4, today: 12, tomorrow: 8, code: "temperature_decreasing", interpretation: "heating_increase"},
		{delta: -4, today: 26, tomorrow: 22, code: "temperature_decreasing", interpretation: "cooling_ease"},
		{delta: -3, today: 22, tomorrow: 19, code: "temperature_decreasing", interpretation: "dependent"},
		{delta: 4, today: 24, tomorrow: 28, code: "temperature_increasing", interpretation: "cooling_increase"},
		{delta: 4, today: 12, tomorrow: 16, code: "temperature_increasing", interpretation: "heating_ease"},
		{delta: 3, today: 18, tomorrow: 21, code: "temperature_increasing", interpretation: "dependent"},
	} {
		item := temperatureSignal(tt.delta, tt.today, tt.tomorrow)
		if item == nil || item.Code != tt.code || item.Params["change"] != tt.delta || item.Params["interpretation"] != tt.interpretation {
			t.Fatalf("temperature metadata %+v", item)
		}
		if tt.tomorrow == 28 && item.Severity != "warning" {
			t.Fatal("heat severity changed")
		}
	}
	start := DailySummary{Date: "2026-10-12", AvgTemperature: f(12), SolarEnergyKWhM2: 4}
	end := DailySummary{Date: "2026-10-15", AvgTemperature: f(8), SolarEnergyKWhM2: 2}
	for _, item := range []Signal{temperatureTrendSignal(start, end), solarTrendSignal(start, end)} {
		if item.Code == "" || item.Params["start"] != start.Date || item.Params["end"] != end.Date {
			t.Fatal("trend dates missing")
		}
		body, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"params"`) {
			t.Fatal("metadata not serialized")
		}
	}
	for _, strong := range []bool{false, true} {
		peak := 400.0
		code := "solar_peak"
		if strong {
			peak = 600
			code = "solar_peak_strong"
		}
		item := solarPeakSignal(DailySummary{Date: start.Date, PeakSolarTime: start.Date + "T13:00", PeakSolarIrradiance: f(peak)})
		if item == nil || item.Code != code || item.Params["partial"] != true || item.Params["time"] != "13:00" || item.Params["peak"] != peak {
			t.Fatal("peak metadata")
		}
	}
	for _, change := range []float64{-2, 2} {
		days := trendDays([]float64{12, 12}, []float64{4, 4 + change})
		found := false
		for _, item := range signals(days) {
			if item.Code == "solar_daily_decreasing" || item.Code == "solar_daily_increasing" {
				found = true
				if item.Params["change"] != change || item.Params["date"] != days[1].Date || item.Params["baseline"] != days[0].Date {
					t.Fatal("daily solar metadata")
				}
			}
		}
		if !found {
			t.Fatal("daily solar code missing")
		}
	}
}

func TestSolarPeakSignalTimeFormats(t *testing.T) {
	for _, value := range []string{
		"2026-10-14T12:00", "2026-10-14T12:00:00",
		"2026-10-14T12:00:00+03:00", "2026-10-14T12:00+03:00",
		"2026-10-14T12:00:00Z", "2026-10-14T12:00Z",
		"2026-10-14T12:00:00.123+03:00", "12:00", "12:00:00",
	} {
		t.Run(value, func(t *testing.T) {
			item := solarPeakSignal(DailySummary{Date: "2026-10-14", PeakSolarTime: value, PeakSolarIrradiance: f(600)})
			if item == nil || item.Params["time"] != "12:00" {
				t.Fatalf("valid local peak time rejected or shifted: %+v", item)
			}
		})
	}
	for _, value := range []string{"", "invalid", "25:00", "2026-10-14"} {
		if item := solarPeakSignal(DailySummary{Date: "2026-10-14", PeakSolarTime: value, PeakSolarIrradiance: f(600)}); item != nil {
			t.Fatalf("invalid peak time accepted: %q", value)
		}
	}
}
