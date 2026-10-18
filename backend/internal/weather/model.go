package weather

import (
	"time"
)

// Wind fields use 120 m forecasts; speed is m/s and direction is degrees.
type Point struct {
	UnixTime        *int64   `json:"unixTime,omitempty"`
	WindSpeed       *float64 `json:"windSpeed"`
	WindDirection   *float64 `json:"windDirection"`
	WindPotential   *string  `json:"WindPotential"`
	Time            string   `json:"time"`
	Temperature     *float64 `json:"temperature"`
	SolarIrradiance *float64 `json:"solarIrradiance"`
}

// Daily wind statistics use the same 120 m hourly measurements.
type DailySummary struct {
	WindDirectionConsistency float64  `json:"windDirectionConsistency"`
	AvgWindDirection         *float64 `json:"avgWindDirection"`
	WindDirectionSampleCount int      `json:"windDirectionSampleCount"`
	MinWindSpeed             *float64 `json:"minWindSpeed"`
	AvgWindSpeed             *float64 `json:"avgWindSpeed"`
	MaxWindSpeed             *float64 `json:"maxWindSpeed"`
	WindSampleCount          int      `json:"windSampleCount"`
	Date                     string   `json:"date"`
	AvgTemperature           *float64 `json:"avgTemperature"`
	MinTemperature           *float64 `json:"minTemperature"`
	MaxTemperature           *float64 `json:"maxTemperature"`
	SolarEnergyKWhM2         float64  `json:"solarEnergyKWhM2"`
	PeakSolarIrradiance      *float64 `json:"peakSolarIrradiance"`
	PeakSolarTime            string   `json:"peakSolarTime,omitempty"`
	SolarSampleCount         int      `json:"solarSampleCount"`
	HourCount                int      `json:"hourCount"`
	ExpectedHourCount        int      `json:"expectedHourCount"`
	CompleteTemperature      bool     `json:"completeTemperature"`
	CompleteWind             bool     `json:"completeWind"`
	CompleteSolar            bool     `json:"completeSolar"`
	CompleteDay              bool     `json:"completeDay"`
	TemperatureSampleCount   int      `json:"temperatureSampleCount"`
}
type Signal struct {
	Code     string         `json:"code,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	Kind     string         `json:"kind"`
	Severity string         `json:"severity"`
	Title    string         `json:"title"`
	Detail   string         `json:"detail"`
}
type Forecast struct {
	Latitude       float64        `json:"latitude"`
	Longitude      float64        `json:"longitude"`
	Timezone       string         `json:"timezone"`
	Hourly         []Point        `json:"hourly"`
	Daily          []DailySummary `json:"daily"`
	Signals        []Signal       `json:"signals"`
	OutlookSignals []Signal       `json:"outlookSignals"`
	FetchedAt      time.Time      `json:"fetchedAt"`
}
