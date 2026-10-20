package weather

import (
	"strings"
	"testing"
	"time"
)

const persistentTitle = "Persistent cold, low-solar and low-wind conditions"
const worseningTitle = "Cold, low-solar and low-wind conditions"

func coldDays() []DailySummary {
	days := windDays(2, 2, 2)
	for i := range days {
		days[i].AvgTemperature = f(5)
		days[i].MaxTemperature = f(6)
		days[i].TemperatureSampleCount = 24
		days[i].SolarSampleCount = 24
		days[i].SolarEnergyKWhM2 = 1
	}
	return days
}

func TestPersistentCombinedDailyBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func([]DailySummary, *WindInsightConfig)
		want   string
	}{
		{"persistent at solar boundary", func(_ []DailySummary, _ *WindInsightConfig) {}, persistentTitle},
		{"worsening wins", func(d []DailySummary, _ *WindInsightConfig) { d[0].AvgTemperature = f(8); d[0].MaxTemperature = f(8) }, worseningTitle},
		{"cold boundary strict", func(d []DailySummary, _ *WindInsightConfig) { d[1].MaxTemperature = f(10) }, ""},
		{"configurable cold", func(_ []DailySummary, c *WindInsightConfig) { c.PersistentCold = 6 }, ""},
		{"no low wind", func(d []DailySummary, _ *WindInsightConfig) {
			for i := range d {
				d[i].AvgWindSpeed = f(3)
				d[i].MaxWindSpeed = f(3)
			}
		}, ""},
		{"interrupted wind", func(d []DailySummary, _ *WindInsightConfig) { d[1].MaxWindSpeed = f(3) }, ""},
		{"solar above boundary", func(d []DailySummary, _ *WindInsightConfig) { d[1].SolarEnergyKWhM2 = 1.001 }, ""},
		{"missing temperature", func(d []DailySummary, _ *WindInsightConfig) { d[1].AvgTemperature = nil }, ""},
		{"missing maximum", func(d []DailySummary, _ *WindInsightConfig) { d[1].MaxTemperature = nil }, ""},
		{"incomplete temperature", func(d []DailySummary, _ *WindInsightConfig) { d[1].TemperatureSampleCount-- }, ""},
		{"incomplete solar", func(d []DailySummary, _ *WindInsightConfig) { d[1].SolarSampleCount-- }, ""},
		{"partial day", func(d []DailySummary, _ *WindInsightConfig) { d[1].CompleteDay = false }, ""},
		{"ending at eighteen", func(d []DailySummary, _ *WindInsightConfig) {
			for i := range d {
				d[i].AvgTemperature = f(21 - float64(i)*1.5)
				d[i].MaxTemperature = d[i].AvgTemperature
			}
		}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, c := coldDays(), DefaultWindInsightConfig()
			tt.mutate(d, &c)
			items := dailyWindInsights(d, c)
			count := 0
			for _, s := range items {
				if s.Kind == "combined" {
					count++
					if s.Title != tt.want {
						t.Fatalf("unexpected %+v", s)
					}
				}
			}
			if (tt.want != "" && count != 1) || (tt.want == "" && count != 0) {
				t.Fatalf("combined count %d: %+v", count, items)
			}
		})
	}
}

func TestHourlySolarEvidence(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hours  int
		solar  float64
		mutate func([]Point)
		want   bool
	}{
		{"nighttime only", 2, 0, func(_ []Point) {}, true},
		{"complete solar", 4, 100, func(_ []Point) {}, true},
		{"above threshold", 4, 100.01, func(_ []Point) {}, false},
		{"missing solar", 4, 50, func(p []Point) { p[10].SolarIrradiance = nil }, false},
		{"missing night solar", 4, 50, func(p []Point) { p[0].SolarIrradiance = nil }, false},
		{"missing night temperature", 4, 50, func(p []Point) { p[0].Temperature = nil }, false},
		{"incomplete window", 1, 50, func(_ []Point) {}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			speeds := make([]float64, tt.hours)
			for i := range speeds {
				speeds[i] = 2
			}
			p := windHours(speeds...)
			for i := range p {
				p[i].Temperature = f(5)
				p[i].SolarIrradiance = f(tt.solar)
			}
			tt.mutate(p)
			got := hourlyWindInsights(p, "UTC", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), DefaultWindInsightConfig())
			if hasWindTitle(got, persistentTitle) != tt.want {
				t.Fatalf("%+v", got)
			}
		})
	}
}

func TestCombinedPrecedenceKeepsIndependentPeriods(t *testing.T) {
	d := coldDays()
	d[0].AvgTemperature = f(8)
	d[0].MaxTemperature = f(8)
	combined := dailyWindInsights(d, DefaultWindInsightConfig())
	same := temperatureTrendSignal(d[0], d[2])
	other := same
	other.Params = map[string]any{"start": "2026-11-01", "end": "2026-11-03", "interpretation": "heating_increase"}
	solar := solarTrendSignal(d[0], d[2])
	got := mergeWindInsights([]Signal{same, other, solar, {Kind: "combined", Title: "Independent combined"}}, combined)
	if len(got) != 4 || got[0].Kind != "combined" || !hasWindTitle(got, "Independent combined") {
		t.Fatalf("%+v", got)
	}
	if hasWindTitle(got, "Prolonged low-wind period") {
		t.Fatal("redundant wind")
	}
}

func TestStableTemperatureSolarWithChangingWind(t *testing.T) {
	d := coldDays()
	d[0].AvgWindSpeed = f(4)
	d[1].AvgWindSpeed = f(10)
	s := findSignal(signals(d), "Stable temperature and solar outlook")
	if s == nil || !strings.Contains(s.Detail, "Wind stability is not assessed") {
		t.Fatalf("%+v", s)
	}
	d[1].SolarSampleCount--
	if findSignal(signals(d), "Limited comparison data") == nil {
		t.Fatal("limited data missing")
	}
	if findSignal(outlookSignals(d), "No clear multi-day trend") == nil {
		t.Fatal("no trend missing")
	}
}

func TestPersistentCombinedDSTDays(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Helsinki")
	for _, date := range []string{"2026-03-28", "2026-10-24"} {
		start, _ := time.ParseInLocation("2006-01-02", date, loc)
		var p []Point
		for stamp := start; !stamp.After(start.AddDate(0, 0, 3)); stamp = stamp.Add(time.Hour) {
			unix := stamp.Unix()
			p = append(p, Point{Time: stamp.Format(time.RFC3339), UnixTime: &unix, Temperature: f(5), SolarIrradiance: f(20), WindSpeed: f(2)})
		}
		d := summarize(p, "Europe/Helsinki")
		if !hasWindTitle(dailyWindInsights(d, DefaultWindInsightConfig()), persistentTitle) {
			t.Fatalf("DST %s: %+v", date, d)
		}
		p[12].Temperature = nil
		if hasWindTitle(dailyWindInsights(summarize(p, "Europe/Helsinki"), DefaultWindInsightConfig()), persistentTitle) {
			t.Fatal("incomplete temperature accepted")
		}
	}
}
