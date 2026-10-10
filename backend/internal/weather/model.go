package weather

import (
	"time"
)

type Point struct {
	Time            string   `json:"time"`
	Temperature     *float64 `json:"temperature"`
	SolarIrradiance *float64 `json:"solarIrradiance"`
}
type DailySummary struct {
	Date                   string   `json:"date"`
	AvgTemperature         *float64 `json:"avgTemperature"`
	MinTemperature         *float64 `json:"minTemperature"`
	MaxTemperature         *float64 `json:"maxTemperature"`
	SolarEnergyKWhM2       float64  `json:"solarEnergyKWhM2"`
	PeakSolarIrradiance    *float64 `json:"peakSolarIrradiance"`
	PeakSolarTime          string   `json:"peakSolarTime,omitempty"`
	SolarSampleCount       int      `json:"solarSampleCount"`
	HourCount              int      `json:"hourCount"`
	ExpectedHourCount      int      `json:"expectedHourCount"`
	CompleteDay            bool     `json:"completeDay"`
	TemperatureSampleCount int      `json:"temperatureSampleCount"`
}
type Signal struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
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
