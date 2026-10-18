package weather

import (
	"math"
	"testing"
	"time"
)

func TestExactHourlySlots(t *testing.T) {
	for _, timezone := range []string{"Europe/Helsinki", "Asia/Kathmandu"} {
		t.Run(timezone, func(t *testing.T) {
			location, err := time.LoadLocation(timezone)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Date(2026, 10, 14, 0, 0, 0, 0, location)
			points := solarIntervalPoints(start)
			// Distinct off-grid instants cannot fill hourly slots or affect measurements.
			for _, offset := range []time.Duration{30 * time.Minute, time.Second} {
				instant := start.Add(time.Hour + offset)
				unix := instant.Unix()
				points = append(points, Point{UnixTime: &unix, Time: instant.Format("2006-01-02T15:04"), Temperature: f(99), WindSpeed: f(99), SolarIrradiance: f(9999)})
			}
			points = append(points, points[1])
			day := summarize(points, timezone)[0]
			if !day.CompleteDay || day.HourCount != 24 || day.TemperatureSampleCount != 24 || day.WindSampleCount != 24 || day.SolarSampleCount != 24 {
				t.Fatalf("incorrect slot coverage: %+v", day)
			}
			if *day.AvgTemperature != 5 || *day.AvgWindSpeed != 7 || math.Abs(day.SolarEnergyKWhM2-2.8) > 1e-9 {
				t.Fatalf("unexpected samples entered calculations: %+v", day)
			}
		})
	}
}

func TestLocalOnlyRepeatedHourCannotClaimFullCoverage(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Helsinki")
	start := time.Date(2026, 10, 25, 0, 0, 0, 0, location)
	points := solarIntervalPoints(start)
	for i := range points {
		points[i].UnixTime = nil
	}
	day := summarize(points, "Europe/Helsinki")[0]
	if day.CompleteDay || day.HourCount != 24 || day.ExpectedHourCount != 25 {
		t.Fatalf("ambiguous local strings claimed full coverage: %+v", day)
	}
}
