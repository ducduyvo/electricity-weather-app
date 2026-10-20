package weather

import (
	"fmt"
	"math"
	"time"
)

// These thresholds describe forecast conditions, not calibrated portfolio demand.
const (
	temperatureChangeThreshold = 3.0
	solarChangeThreshold       = 1.0 // Daily solar-energy change in kWh/m²; descriptive heuristic.
	heatingReference           = 18.0
	coolingReference           = 22.0
	strongSolarPeak            = 500.0
)

func signals(days []DailySummary) []Signal {
	out := []Signal{}

	if len(days) == 0 {
		return []Signal{{
			Kind:     "weather",
			Severity: "info",
			Title:    "Insufficient forecast data",
			Detail:   "No daily forecast is available for weather-derived insights.",
		}}
	}

	// Prefer tomorrow's solar period; with one day, describe the available date.
	solarInsightDay := days[0]
	if len(days) >= 2 {
		solarInsightDay = days[1]
	}

	if solarPeakInsight := solarPeakSignal(solarInsightDay); solarPeakInsight != nil {
		out = append(out, *solarPeakInsight)
	}

	if len(days) < 2 {
		return append(out, Signal{
			Kind:     "weather",
			Severity: "info",
			Title:    "Insufficient forecast data",
			Detail:   "Two forecast days are needed to compare tomorrow with today.",
		})
	}

	today, tomorrow := days[0], days[1]
	temperatureAvailable := completeTemperature(today) && completeTemperature(tomorrow)
	solarAvailable := completeSolar(today) && completeSolar(tomorrow)

	var delta, solarChange float64

	if temperatureAvailable {
		delta = *tomorrow.AvgTemperature - *today.AvgTemperature

		if temperature := temperatureSignal(delta, *today.AvgTemperature, *tomorrow.AvgTemperature); temperature != nil {
			out = append(out, *temperature)
		}
	}

	if solarAvailable {
		solarChange = tomorrow.SolarEnergyKWhM2 - today.SolarEnergyKWhM2

		if solarChange <= -solarChangeThreshold {
			out = append(out, Signal{
				Kind:     "solar",
				Severity: "warning",
				Code:     "solar_daily_decreasing",
				Params: map[string]any{
					"change":   solarChange,
					"date":     tomorrow.Date,
					"baseline": today.Date,
				},
				Title: "Lower solar availability",
				Detail: fmt.Sprintf(
					"Solar-energy proxy on %s falls %.2f kWh/m² versus %s. Lower irradiance may reduce favorable conditions for solar PV generation.",
					tomorrow.Date,
					-solarChange,
					today.Date,
				),
			})
		} else if solarChange >= solarChangeThreshold {
			out = append(out, Signal{
				Kind:     "solar",
				Severity: "info",
				Code:     "solar_daily_increasing",
				Params: map[string]any{
					"change":   solarChange,
					"date":     tomorrow.Date,
					"baseline": today.Date,
				},
				Title: "Higher solar availability",
				Detail: fmt.Sprintf(
					"Solar-energy proxy on %s rises %.2f kWh/m² versus %s. Higher irradiance may improve conditions for solar PV generation.",
					tomorrow.Date,
					solarChange,
					today.Date,
				),
			})
		}
	}

	if temperatureAvailable && solarAvailable {
		if delta <= -temperatureChangeThreshold &&
			*tomorrow.AvgTemperature < heatingReference &&
			solarChange <= -solarChangeThreshold {
			out = append(out, Signal{
				Kind:     "combined",
				Severity: "warning",
				Title:    "Potentially tighter weather-driven conditions",
				Detail:   "Colder conditions may increase heating-driven electricity demand while lower solar availability may weaken solar PV generation conditions. This is a qualitative weather outlook, not a price forecast.",
			})
		} else if *tomorrow.AvgTemperature >= heatingReference &&
			*tomorrow.AvgTemperature <= coolingReference &&
			solarChange >= solarChangeThreshold {
			out = append(out, Signal{
				Kind:     "combined",
				Severity: "info",
				Title:    "Potentially more favorable weather-driven conditions",
				Detail:   "Mild temperatures and higher solar availability may support more favorable weather-driven supply and demand conditions. Portfolio demand, generation, and prices are not predicted.",
			})
		}
	}

	if !temperatureAvailable || !solarAvailable {
		out = append(out, Signal{
			Kind:     "weather",
			Severity: "info",
			Title:    "Limited comparison data",
			Detail:   "Some day-on-day insights are unavailable because temperature or solar readings are missing. Available readings still support the displayed insights.",
		})
	} else if math.Abs(delta) < temperatureChangeThreshold &&
		math.Abs(solarChange) < solarChangeThreshold {
		out = append(out, Signal{
			Kind:     "weather",
			Severity: "info",
			Title:    "Stable temperature and solar outlook",
			Detail:   "No large day-on-day temperature or solar change is detected in the available forecast. Wind stability is not assessed here. This does not imply stable electricity demand, generation, or prices.",
		})
	}

	return out
}

func temperatureSignal(delta, today, tomorrow float64) *Signal {
	if delta <= -temperatureChangeThreshold {
		interpretation := "dependent"
		detail := fmt.Sprintf("Average temperature is expected to fall %.1f°C tomorrow. ", -delta)

		switch {
		case tomorrow < heatingReference:
			detail += "Colder conditions may increase heating-driven electricity demand."
			interpretation = "heating_increase"
		case today > coolingReference:
			detail += "Cooling-driven electricity demand conditions may ease."
			interpretation = "cooling_ease"
		default:
			detail += "The electricity-demand implication depends on heating and cooling needs."
			interpretation = "dependent"
		}

		return &Signal{
			Kind:     "temperature",
			Severity: "warning",
			Code:     "temperature_decreasing",
			Params: map[string]any{
				"change":         delta,
				"interpretation": interpretation,
			},
			Title:  "Colder tomorrow",
			Detail: detail,
		}
	}

	if delta >= temperatureChangeThreshold {
		interpretation := "dependent"
		detail := fmt.Sprintf("Average temperature is expected to rise %.1f°C tomorrow. ", delta)

		severity := "info"
		switch {
		case tomorrow > coolingReference:
			detail += "Warmer conditions may increase cooling-driven electricity demand."
			interpretation = "cooling_increase"
			if tomorrow >= coolingReference+5.0 {
				severity = "warning" // severe heat spike
			}
		case today < heatingReference:
			detail += "Heating-driven electricity demand conditions may ease."
			interpretation = "heating_ease"
		default:
			detail += "The electricity-demand implication depends on heating and cooling needs."
			interpretation = "dependent"
		}

		return &Signal{
			Kind:     "temperature",
			Severity: severity,
			Code:     "temperature_increasing",
			Params: map[string]any{
				"change":         delta,
				"interpretation": interpretation,
			},
			Title:  "Warmer tomorrow",
			Detail: detail,
		}
	}

	return nil
}

func solarPeakSignal(day DailySummary) *Signal {
	if !validSolarIrradiance(day.PeakSolarIrradiance) || *day.PeakSolarIrradiance <= 0 {
		return nil
	}

	peakTime, valid := parseSolarPeakTime(day.PeakSolarTime)
	if !valid {
		return nil
	}

	code := "solar_peak"
	title := "Solar period expected"
	conditions := "indicating potential solar PV generation conditions"

	if *day.PeakSolarIrradiance >= strongSolarPeak {
		code = "solar_peak_strong"
		title = "Strong solar period expected"
		conditions = "indicating favorable irradiance conditions for solar PV generation"
	}

	detail := fmt.Sprintf(
		"Solar irradiance on %s is forecast to peak around %s at %.0f W/m², %s. Time is local to the forecast location.",
		day.Date,
		peakTime.Format("15:04"),
		*day.PeakSolarIrradiance,
		conditions,
	)

	if !completeSolar(day) {
		detail += " Some hourly readings are missing; this is the peak among available readings."
	}

	return &Signal{
		Code: code,
		Params: map[string]any{
			"date":    day.Date,
			"time":    peakTime.Format("15:04"),
			"peak":    *day.PeakSolarIrradiance,
			"partial": !completeSolar(day),
		},
		Kind:     "solar",
		Severity: "info",
		Title:    title,
		Detail:   detail,
	}
}

// Preserve the supplied clock time: peak timestamps are local to the forecast.
func parseSolarPeakTime(value string) (time.Time, bool) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"15:04:05",
		"15:04",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
