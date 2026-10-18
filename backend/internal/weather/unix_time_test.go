package weather

import (
	"testing"
	"time"
)

func TestUnixTimestampRepeatedHourCoverage(t *testing.T) {
	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 25, 0, 0, 0, 0, location)
	var points []Point
	for instant := start; instant.Before(start.AddDate(0, 0, 1)); instant = instant.Add(time.Hour) {
		unix := instant.Unix()
		points = append(points, Point{UnixTime: &unix, Time: instant.Format("2006-01-02T15:04"), Temperature: f(5), WindSpeed: f(5), SolarIrradiance: f(100)})
	}
	day := summarize(points, "Europe/Helsinki")[0]
	if !day.CompleteDay || day.HourCount != 25 || day.ExpectedHourCount != 25 {
		t.Fatalf("incorrect repeated-hour coverage: %+v", day)
	}
	points[4] = points[3]
	day = summarize(points, "Europe/Helsinki")[0]
	if day.CompleteDay || day.HourCount != 24 {
		t.Fatalf("duplicate instant counted: %+v", day)
	}
}
