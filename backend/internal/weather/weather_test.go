package weather

import (
	"testing"
)

func f(v float64) *float64 { return &v }
func TestSummarize(t *testing.T) {
	p := []Point{{
		Time:            "2026-10-06T00:00",
		Temperature:     f(4),
		SolarIrradiance: f(0),
	}, {
		Time:            "2026-10-06T12:00",
		Temperature:     f(8),
		SolarIrradiance: f(500),
	}}
	d := summarize(p)
	if len(d) != 1 || *d[0].AvgTemperature != 6 || d[0].SolarEnergyKWhM2 != 0.5 {
		t.Fatalf("unexpected summary: %+v", d)
	}
}
func TestValidate(t *testing.T) {
	if Validate(91, 0) == nil {
		t.Fatal("expected invalid latitude")
	}
	if Validate(60.17, 24.94) != nil {
		t.Fatal("expected valid coords")
	}
}

func TestSummarizeValues(t *testing.T) {
	points := []Point{
		{
			Time:            "2026-10-07T00:00",
			Temperature:     f(4),
			SolarIrradiance: f(100),
		},
		{
			Time:            "2026-10-07T01:00",
			Temperature:     nil,
			SolarIrradiance: nil,
		},
		{
			Time:            "2026-10-07T02:00",
			Temperature:     f(8),
			SolarIrradiance: f(500),
		},
		{
			Time:            "2026-10-07T03:00",
			Temperature:     f(-3),
			SolarIrradiance: f(-10),
		},
		{
			Time:            "2026-10-08T00:00",
			Temperature:     nil,
			SolarIrradiance: f(200),
		},
		{
			Time:            "bad",
			Temperature:     f(99),
			SolarIrradiance: f(999),
		},
	}
	got := summarize(points)
	if len(got) != 2 {
		t.Fatalf("daily count %d", len(got))
	}
	first := got[0]
	if first.Date != "2026-10-07" ||
		*first.AvgTemperature != 3 ||
		*first.MinTemperature != -3 ||
		*first.MaxTemperature != 8 {
		t.Fatalf("temperature summary %+v", first)
	}
	if first.SolarEnergyKWhM2 != 0.7 {
		t.Fatalf("solar energy %v", first.SolarEnergyKWhM2)
	}
	second := got[1]
	if second.AvgTemperature != nil ||
		second.MinTemperature != nil ||
		second.MaxTemperature != nil ||
		second.SolarEnergyKWhM2 != 0 {
		t.Fatalf("missing temperatures %+v", second)
	}
}

func TestSignalsThresholds(t *testing.T) {
	tests := []struct {
		name                        string
		tomorrowTemp, tomorrowSolar float64
		title, kind, severity       string
	}{
		{"colder boundary", 7, 4, "Colder tomorrow", "temperature", "warning"},
		{"warmer boundary", 13, 4, "Warmer tomorrow", "temperature", "info"},
		{"lower solar boundary", 10, 3, "Lower solar availability", "solar", "warning"},
		{"higher solar boundary", 10, 5, "Higher solar availability", "solar", "info"},
		{"stable", 10, 4, "Stable temperature and solar outlook", "weather", "info"},
		{"below thresholds", 12.9, 4.49, "Stable temperature and solar outlook", "weather", "info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := signals([]DailySummary{
				{
					AvgTemperature:         f(10),
					SolarEnergyKWhM2:       4,
					SolarSampleCount:       24,
					HourCount:              24,
					CompleteDay:            true,
					TemperatureSampleCount: 24,
				},
				{
					AvgTemperature:         f(tt.tomorrowTemp),
					SolarEnergyKWhM2:       tt.tomorrowSolar,
					SolarSampleCount:       24,
					HourCount:              24,
					CompleteDay:            true,
					TemperatureSampleCount: 24,
				},
			})
			if len(got) != 1 ||
				got[0].Title != tt.title ||
				got[0].Kind != tt.kind ||
				got[0].Severity != tt.severity ||
				got[0].Detail == "" {
				t.Fatalf("signals %+v", got)
			}
		})
	}
}

func TestSignalsMissingData(t *testing.T) {
	if got := signals([]DailySummary{{}}); len(got) != 1 || got[0].Title != "Insufficient forecast data" {
		t.Fatalf("one day: %+v", got)
	}
	got := signals([]DailySummary{{}, {SolarEnergyKWhM2: 1}})
	if len(got) != 1 || got[0].Title != "Limited comparison data" {
		t.Fatalf("missing-data signals %+v", got)
	}
}
