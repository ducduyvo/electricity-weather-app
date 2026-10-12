package weather

import (
	"math"
)

func Validate(lat, lon float64) error {
	if math.IsNaN(lat) ||
		math.IsNaN(lon) ||
		math.IsInf(lat, 0) ||
		math.IsInf(lon, 0) ||
		lat < -90 ||
		lat > 90 ||
		lon < -180 ||
		lon > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}

func validTemperature(value *float64) bool {
	return value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0)
}

func validSolarIrradiance(value *float64) bool {
	return value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0) && *value >= 0
}

func validWindValue(value *float64, direction bool) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || (direction && *value > 360) {
		return nil
	}
	return value
}

// Compare daily means only when the entire day has temperature readings.
func completeTemperature(day DailySummary) bool {
	return day.CompleteDay && day.HourCount > 0 &&
		day.TemperatureSampleCount == day.HourCount && validTemperature(day.AvgTemperature)
}

func completeSolar(day DailySummary) bool {
	return day.HourCount > 0 && day.SolarSampleCount == day.HourCount && day.CompleteDay &&
		validSolarIrradiance(&day.SolarEnergyKWhM2)
}

// Compare daily means only when the entire day has wind readings.
func windMean(day DailySummary) (float64, bool) {
	if !day.CompleteDay || day.HourCount == 0 || day.WindSampleCount != day.HourCount || validWindValue(day.MeanWindSpeed, false) == nil {
		return 0, false
	}
	return *day.MeanWindSpeed, true
}
