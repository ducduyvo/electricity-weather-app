package weather

import (
	"math"
	"testing"
	"time"
)

func TestMetricCompleteness(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Helsinki")
	for _, date := range []string{"2026-10-14", "2026-03-29", "2026-10-25"} {
		t.Run(date, func(t *testing.T) {
			start, _ := time.ParseInLocation("2006-01-02", date, location)
			points := solarIntervalPoints(start)
			day := summarize(points, "Europe/Helsinki")[0]
			if !day.CompleteDay || !day.CompleteTemperature || !day.CompleteWind || !day.CompleteSolar {
				t.Fatalf("full data marked incomplete: %+v", day)
			}
			points[2].Temperature = nil
			points[3].WindSpeed = nil
			points[4].SolarIrradiance = nil
			day = summarize(points, "Europe/Helsinki")[0]
			if !day.CompleteDay || day.CompleteTemperature || day.CompleteWind || day.CompleteSolar {
				t.Fatalf("metric completeness mixed with coverage: %+v", day)
			}
			partial := summarize(points[:len(points)-2], "Europe/Helsinki")[0]
			if partial.CompleteDay || partial.CompleteTemperature || partial.CompleteWind || partial.CompleteSolar {
				t.Fatal("partial day marked complete")
			}
		})
	}
}

func TestValidSolarAfterInvalidDuplicate(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	for _, invalid := range []*float64{nil, f(-1), f(math.NaN()), f(math.Inf(1))} {
		points := solarIntervalPoints(start)
		good := points[1]
		points[1].SolarIrradiance = invalid
		points = append(points, good, good)
		day := summarize(points, "UTC")[0]
		if !day.CompleteSolar || day.SolarSampleCount != 24 || math.Abs(day.SolarEnergyKWhM2-2.8) > 1e-9 {
			t.Fatalf("valid replacement rejected or duplicated: %+v", day)
		}
	}
}
