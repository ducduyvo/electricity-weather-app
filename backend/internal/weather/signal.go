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
				Title:    "Lower solar availability",
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
				Title:    "Higher solar availability",
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
			Severity: "normal",
			Title:    "Stable weather outlook",
			Detail:   "No large day-on-day temperature or solar change is detected in the available forecast. This does not imply stable electricity demand, generation, or prices.",
		})
	}

	return out
}

func temperatureSignal(delta, today, tomorrow float64) *Signal {
	if delta <= -temperatureChangeThreshold {
		detail := fmt.Sprintf("Average temperature is expected to fall %.1f°C tomorrow. ", -delta)

		switch {
		case tomorrow < heatingReference:
			detail += "Colder conditions may increase heating-driven electricity demand."
		case today > coolingReference:
			detail += "Cooling-driven electricity demand conditions may ease."
		default:
			detail += "The electricity-demand implication depends on heating and cooling needs."
		}

		return &Signal{
			Kind:     "temperature",
			Severity: "warning",
			Title:    "Colder tomorrow",
			Detail:   detail,
		}
	}

	if delta >= temperatureChangeThreshold {
		detail := fmt.Sprintf("Average temperature is expected to rise %.1f°C tomorrow. ", delta)

		severity := "info"
		switch {
		case tomorrow > coolingReference:
			detail += "Warmer conditions may increase cooling-driven electricity demand."
			if tomorrow >= coolingReference+5.0 {
				severity = "warning" // severe heat spike
			}
		case today < heatingReference:
			detail += "Heating-driven electricity demand conditions may ease."
		default:
			detail += "The electricity-demand implication depends on heating and cooling needs."
		}

		return &Signal{
			Kind:     "temperature",
			Severity: severity,
			Title:    "Warmer tomorrow",
			Detail:   detail,
		}
	}

	return nil
}

func solarPeakSignal(day DailySummary) *Signal {
	if !validSolarIrradiance(day.PeakSolarIrradiance) || *day.PeakSolarIrradiance <= 0 {
		return nil
	}

	peakTime, err := time.Parse("2006-01-02T15:04", day.PeakSolarTime)
	if err != nil {
		return nil
	}

	title := "Solar period expected"
	conditions := "indicating potential solar PV generation conditions"

	if *day.PeakSolarIrradiance >= strongSolarPeak {
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
		Kind:     "solar",
		Severity: "info",
		Title:    title,
		Detail:   detail,
	}
}
