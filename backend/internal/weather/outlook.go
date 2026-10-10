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

	var signals []Signal

	// Extract temperature only when daily measurements are complete.
	temperature := func(day DailySummary) (float64, bool) {
		if !day.CompleteDay || day.HourCount == 0 || day.TemperatureSampleCount != day.HourCount || day.AvgTemperature == nil {
			return 0, false
		}
		return *day.AvgTemperature, true
	}

	// Extract solar energy only when daily measurements are complete.
	solar := func(day DailySummary) (float64, bool) {
		return day.SolarEnergyKWhM2, completeSolar(day)
	}

	// Detect both decreasing and increasing trends.
	for _, direction := range []float64{decreasingTrend, increasingTrend} {
		// TEMPERATURE TREND
		startDay, endDay, found := trendPeriod(days, temperature, direction, func(startTemperature, endTemperature float64) bool {
			return direction*(endTemperature-startTemperature) >= temperatureChangeThreshold
		})

		if found {
			startTemperature := *startDay.AvgTemperature
			endTemperature := *endDay.AvgTemperature
			temperatureChange := endTemperature - startTemperature

			title := "Warming trend"
			severity := "info"
			implication := "The heating/cooling demand implication depends on local conditions."

			switch {
			case direction == decreasingTrend && endTemperature < heatingReference:
				title = "Cold spell developing"
				severity = "warning"
				implication = "Colder conditions may increase heating-driven electricity demand."

			case direction == decreasingTrend && startTemperature > coolingReference:
				title = "Cooling trend"
				severity = "warning"
				implication = "Cooling-driven electricity demand conditions may ease."

			case direction == decreasingTrend:
				title = "Cooling trend"
				severity = "warning"

			case direction == increasingTrend && endTemperature > coolingReference:
				implication = "Warmer conditions may increase cooling-driven electricity demand."

			case direction == increasingTrend && startTemperature < heatingReference:
				implication = "Heating-driven electricity demand conditions may ease."
			}

			signals = append(signals, Signal{
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
			})
		}

		// SOLAR TREND
		startDay, endDay, found = trendPeriod(days, solar, direction, func(startSolar, endSolar float64) bool {
			solarChange := endSolar - startSolar
			return direction*solarChange >= solarChangeThreshold
		})

		if found {
			startSolar := startDay.SolarEnergyKWhM2
			endSolar := endDay.SolarEnergyKWhM2

			solarChange := endSolar - startSolar

			title := "Solar availability improving"
			severity := "info"
			implication := "Higher irradiance may improve conditions for solar PV generation."

			if direction == decreasingTrend {
				title = "Solar availability weakening"
				severity = "warning"
				implication = "Lower irradiance may weaken conditions for solar PV generation."
			}

			signals = append(signals, Signal{
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
			})
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
// - No movement against the requested direction
// - Significant overall change
//
// Returns the start day, end day, and whether a trend was found.
func trendPeriod(days []DailySummary, getMeasurement func(DailySummary) (float64, bool), direction float64, isSignificantChange func(float64, float64) bool) (DailySummary, DailySummary, bool) {
	var bestStartDay DailySummary
	var bestEndDay DailySummary

	const minTrendDays = 3

	if len(days) < minTrendDays {
		return DailySummary{}, DailySummary{}, false
	}

	// Pre-parse dates once upfront in UTC to avoid repeated allocations in inner loops
	parsedDates := make([]time.Time, len(days))
	for i, d := range days {
		t, err := time.Parse("2006-01-02", d.Date)
		if err == nil {
			parsedDates[i] = t
		}
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
			if direction*dailyChange < 0 {
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
