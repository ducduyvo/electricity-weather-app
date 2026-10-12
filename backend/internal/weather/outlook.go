package weather

import (
	"fmt"
	"math"
	"time"
)

const (
	decreasingTrend = -1.0
	increasingTrend = 1.0
)

// outlookSignals identifies significant temperature and solar trends
// across up to 14 forecast days.
//
// A trend requires at least three consecutive complete days,
// at least two actual changes, and a significant overall change.
func outlookSignals(days []DailySummary) []Signal {
	if len(days) > 14 {
		days = days[:14]
	}

	// Pre-parse dates once upfront in UTC to avoid repeated allocations in inner loops
	parsedDates := make([]time.Time, len(days))
	for i, d := range days {
		t, err := time.Parse("2006-01-02", d.Date)
		if err == nil {
			parsedDates[i] = t
		}
	}

	var signals []Signal

	// Extract temperature only when daily measurements are complete.
	temperature := func(day DailySummary) (float64, bool) {
		if !completeTemperature(day) {
			return 0, false
		}
		return *day.AvgTemperature, true
	}

	// Extract solar energy only when daily measurements are complete.
	solar := func(day DailySummary) (float64, bool) {
		return day.SolarEnergyKWhM2, completeSolar(day)
	}

	// Detect both decreasing and increasing trends.
	for _, trendDirection := range []float64{decreasingTrend, increasingTrend} {
		// TEMPERATURE TREND
		startDay, endDay, found := trendPeriod(days, parsedDates, temperature, trendDirection, func(startTemperature, endTemperature float64) bool {
			return trendDirection*(endTemperature-startTemperature) >= temperatureChangeThreshold
		})

		if found {
			signals = append(signals, temperatureTrendSignal(startDay, endDay))
		}

		// SOLAR TREND
		startDay, endDay, found = trendPeriod(days, parsedDates, solar, trendDirection, func(startSolar, endSolar float64) bool {
			return trendDirection*(endSolar-startSolar) >= solarChangeThreshold
		})

		if found {
			signals = append(signals, solarTrendSignal(startDay, endDay))
		}
	}
	// No qualifying temperature or solar trend.
	if len(signals) == 0 {
		signals = append(signals, Signal{
			Kind:     "weather",
			Severity: "info",
			Title:    "No clear multi-day trend",
			Detail: "No qualifying temperature or solar trend was found across at least three consecutive complete forecast days. " +
				"Missing or incomplete days can limit this comparison; this is not a demand, generation, or price forecast.",
		})
	}

	return signals
}

// trendPeriod finds the longest qualifying monotonic trend.
//
// Requirements:
// - At least 3 consecutive calendar days
// - Valid measurements for every included day
// - At least 2 actual day-to-day changes
// - No movement against the requested trendDirection
// - Significant overall change
//
// Returns the start day, end day, and whether a trend was found.
func trendPeriod(days []DailySummary, parsedDates []time.Time, getMeasurement func(DailySummary) (float64, bool), trendDirection float64, isSignificantChange func(float64, float64) bool) (DailySummary, DailySummary, bool) {
	var bestStartDay DailySummary
	var bestEndDay DailySummary

	const minTrendDays = 3

	if len(days) < minTrendDays {
		return DailySummary{}, DailySummary{}, false
	}

	bestChange := 0.0
	bestTrendDays := 0
	found := false

	for startIndex, startDay := range days {
		startValue, valid := getMeasurement(startDay)
		if !valid || parsedDates[startIndex].IsZero() {
			continue
		}

		previousValue := startValue
		changeCount := 0

		for endIndex := startIndex + 1; endIndex < len(days); endIndex++ {
			endDay := days[endIndex]

			currentValue, valid := getMeasurement(endDay)
			currentDate := parsedDates[endIndex]
			previousDate := parsedDates[endIndex-1]

			if !valid || currentDate.IsZero() || !currentDate.Equal(previousDate.AddDate(0, 0, 1)) {
				break
			}

			dailyChange := currentValue - previousValue
			if trendDirection*dailyChange < 0 {
				break
			}

			if dailyChange != 0 {
				changeCount++
			}

			dayCount := endIndex - startIndex + 1
			totalChange := math.Abs(currentValue - startValue)

			// Magnitude first, shorter duration second.
			if dayCount >= minTrendDays && changeCount >= 2 && isSignificantChange(startValue, currentValue) &&
				(!found || totalChange > bestChange || (totalChange == bestChange && dayCount < bestTrendDays)) {
				bestStartDay = startDay
				bestEndDay = endDay
				bestChange = totalChange
				bestTrendDays = dayCount
				found = true
			}

			previousValue = currentValue
		}
	}

	return bestStartDay, bestEndDay, found
}

// temperatureTrendSignal generates a signal describing the temperature trend between two days of forecast data.

func temperatureTrendSignal(startDay, endDay DailySummary) Signal {
	startTemperature := *startDay.AvgTemperature
	endTemperature := *endDay.AvgTemperature
	temperatureChange := endTemperature - startTemperature

	title := "Warming trend"
	severity := "info"
	implication := "The heating/cooling demand implication depends on local conditions."

	switch {
	case temperatureChange < 0 && endTemperature < heatingReference:
		title = "Cold spell developing"
		severity = "warning"
		implication = "Colder conditions may increase heating-driven electricity demand."

	case temperatureChange < 0 && startTemperature > coolingReference:
		title = "Cooling trend"
		severity = "warning"
		implication = "Cooling-driven electricity demand conditions may ease."

	case temperatureChange < 0:
		title = "Cooling trend"
		severity = "warning"

	case temperatureChange > 0 && endTemperature > coolingReference:
		implication = "Warmer conditions may increase cooling-driven electricity demand."

	case temperatureChange > 0 && startTemperature < heatingReference:
		implication = "Heating-driven electricity demand conditions may ease."
	}

	return Signal{
		Kind:     "temperature",
		Severity: severity,
		Title:    title,
		Detail: fmt.Sprintf(
			"Daily average temperature changes by %+.1f°C from %s through %s across consecutive forecast days. %s",
			temperatureChange,
			startDay.Date,
			endDay.Date,
			implication,
		),
	}
}

// solarTrendSignal generates a signal describing the solar energy trend between two days of forecast data.

func solarTrendSignal(startDay, endDay DailySummary) Signal {
	startSolar := startDay.SolarEnergyKWhM2
	endSolar := endDay.SolarEnergyKWhM2

	solarChange := endSolar - startSolar

	title := "Solar availability improving"
	severity := "info"
	implication := "Higher irradiance may improve conditions for solar PV generation."

	if solarChange < 0 {
		title = "Solar availability weakening"
		severity = "warning"
		implication = "Lower irradiance may weaken conditions for solar PV generation."
	}

	return Signal{
		Kind:     "solar",
		Severity: severity,
		Title:    title,
		Detail: fmt.Sprintf(
			"The daily solar-energy proxy changes by %+.2f kWh/m² from %s through %s across consecutive forecast days. %s",
			solarChange,
			startDay.Date,
			endDay.Date,
			implication,
		),
	}
}
