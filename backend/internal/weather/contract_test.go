package weather

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutlookIncludesSecondWeekAndStopsAtFourteen(t *testing.T) {
	temperatures, solar := make([]float64, 16), make([]float64, 16)
	for i := range temperatures {
		temperatures[i] = 10
		solar[i] = 4
	}
	temperatures[11], temperatures[12], temperatures[13] = 8, 6, 4
	days := trendDays(temperatures, solar)
	cold := findSignal(outlookSignals(days), "Cold spell developing")
	if cold == nil || !strings.Contains(cold.Detail, days[13].Date) {
		t.Fatalf("second-week trend missing: %+v", cold)
	}
	for i := range temperatures {
		temperatures[i] = 10
	}
	temperatures[14], temperatures[15] = 5, 0
	if got := outlookSignals(trendDays(temperatures, solar)); findSignal(got, "Cold spell developing") != nil {
		t.Fatalf("included days beyond 14: %+v", got)
	}
}

func TestDuplicateHourCannotFillMissingCoverage(t *testing.T) {
	points := hourlyPoints(1)
	points[23] = points[22]
	day := summarize(points, "Europe/Helsinki")[0]
	if day.CompleteDay || day.HourCount != 23 || day.TemperatureSampleCount != 23 || day.SolarSampleCount != 22 {
		t.Fatalf("duplicate filled missing hour: %+v", day)
	}
}

func TestShortTermSignalsIgnoreLaterDays(t *testing.T) {
	days := trendDays([]float64{10, 5, 5}, []float64{4, 2, 2})
	before := signals(days)
	days[2].AvgTemperature = f(30)
	days[2].SolarEnergyKWhM2 = 100
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(signals(days))
	if string(a) != string(b) {
		t.Fatal("later forecast days changed short-term insights")
	}
}

var _ Provider = OpenMeteo{}
var _ Provider = (*Cached)(nil)
