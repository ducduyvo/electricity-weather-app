package weather

import (
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
