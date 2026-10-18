package weather

import (
	"math"
	"testing"
	"time"
)

func TestMeasurementValidity(t *testing.T) {
	for _, value := range []*float64{nil, f(math.NaN()), f(math.Inf(1)), f(math.Inf(-1))} {
		if validTemperature(value) || validSolarIrradiance(value) || validWindValue(value, false) != nil || validWindValue(value, true) != nil {
			t.Fatal("invalid measurement accepted")
		}
	}
	if !validTemperature(f(-5)) || !validTemperature(f(0)) || !validSolarIrradiance(f(0)) || validSolarIrradiance(f(-1)) {
		t.Fatal("temperature/solar range")
	}
	if validWindValue(f(0), false) == nil || validWindValue(f(-1), false) != nil || validWindValue(f(360), true) == nil || validWindValue(f(361), true) != nil {
		t.Fatal("wind range")
	}
}

func TestSummaryExcludesInvalidMeasurements(t *testing.T) {
	points := windHours(5, 5, 5, 5)
	for i := range points {
		points[i].Temperature = f(-5)
		points[i].SolarIrradiance = f(0)
	}
	for i, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		points[i].Temperature = f(value)
		points[i].SolarIrradiance = f(value)
		points[i].WindSpeed = f(value)
	}
	day := summarize(points)[0]
	if day.TemperatureSampleCount != 21 || day.SolarSampleCount != 21 || day.WindSampleCount != 21 {
		t.Fatalf("invalid readings counted: %+v", day)
	}
	if *day.AvgTemperature != -5 || *day.MinTemperature != -5 || *day.MaxTemperature != -5 || day.SolarEnergyKWhM2 != 0 || *day.AvgWindSpeed != 5 {
		t.Fatal("invalid readings changed aggregates")
	}
	if completeTemperature(day) || completeSolar(day) {
		t.Fatal("invalid readings counted as complete")
	}
}

func TestDailyInsightsRejectInvalidAggregates(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		day := windDays(5)[0]
		day.AvgTemperature = f(value)
		day.TemperatureSampleCount = 24
		day.SolarSampleCount = 24
		day.SolarEnergyKWhM2 = value
		day.AvgWindSpeed = f(value)
		if completeTemperature(day) || completeSolar(day) {
			t.Fatal("invalid aggregate marked complete")
		}
		if _, valid := windAverage(day); valid {
			t.Fatal("invalid wind average accepted")
		}
		day.PeakSolarIrradiance = f(value)
		day.PeakSolarTime = "2026-10-10T12:00"
		if solarPeakSignal(day) != nil {
			t.Fatal("invalid solar peak accepted")
		}
	}
}

func TestHourlyCombinedInsightExcludesInvalidMeasurements(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for _, field := range []string{"temperature", "solar"} {
		for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			points := windHours(2, 2)
			for i := range points {
				points[i].Temperature = f(12 - float64(i/6)*4)
				points[i].SolarIrradiance = f(50)
			}
			if field == "temperature" {
				points[0].Temperature = f(value)
			} else {
				points[0].SolarIrradiance = f(value)
			}
			items := hourlyWindInsights(points, "UTC", now, DefaultWindInsightConfig())
			if hasWindTitle(items, "Cold, low-solar and low-wind conditions") {
				t.Fatal("invalid measurement used for combined insight")
			}
			if !hasWindTitle(items, "Prolonged low-wind period") {
				t.Fatal("unrelated wind insight changed")
			}
		}
	}
}
