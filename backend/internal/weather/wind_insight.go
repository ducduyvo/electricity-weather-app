package weather

import (
	"fmt"
	"math"
	"time"
)

// WindInsightConfig contains descriptive heuristics, not turbine operating limits.
type WindInsightConfig struct {
	SpeedChange          float64
	LowSpeed             float64
	DirectionChange      float64
	DirectionConsistency float64
	StrongSpeed          float64
	LowSolarDaily        float64
	LowSolarHourly       float64
	MinimumLowHours      int
	MinimumLowDays       int
	PersistentCold       float64 // Demo threshold for every temperature, °C.
}

func DefaultWindInsightConfig() WindInsightConfig {
	return WindInsightConfig{
		SpeedChange:          3,
		LowSpeed:             3,
		DirectionChange:      60,
		DirectionConsistency: 0.8,
		StrongSpeed:          15,
		LowSolarDaily:        1,
		LowSolarHourly:       100,
		MinimumLowHours:      12,
		MinimumLowDays:       2,
		PersistentCold:       10,
	}
}

type windWindow struct {
	startTimestamp string
	endTimestamp   string
	startTime      time.Time
	avgSpeed       float64
	maxSpeed       float64
	avgDirection   *float64
	consistency    float64 // Mean direction-vector length: 0 varies/cancels, 1 is aligned.
	avgTemperature *float64
	solar          *float64 // Six-hour average irradiance (W/m²) or daily energy (kWh/m²).
	maxTemp        *float64
}

func dailyWindInsights(days []DailySummary, cfg WindInsightConfig) []Signal {
	if len(days) > 14 {
		days = days[:14]
	}
	dates := make([]time.Time, len(days))
	for i, day := range days {
		dates[i], _ = time.Parse("2006-01-02", day.Date)
	}

	measurement := func(day DailySummary) (float64, bool) {
		avgSpeed, valid := windAverage(day)
		return avgSpeed, valid && day.MaxWindSpeed != nil && *day.MaxWindSpeed <= cfg.StrongSpeed
	}

	var out []Signal

	for _, trendDirection := range []float64{decreasingTrend, increasingTrend} {
		startDay, endDay, found := trendPeriod(days, dates, measurement, trendDirection, func(a, b float64) bool { return trendDirection*(b-a) >= cfg.SpeedChange })
		if found {
			out = append(out, windTrendSignal(startDay.Date, endDay.Date, *startDay.AvgWindSpeed, *endDay.AvgWindSpeed))
		}
	}

	windows := make([]windWindow, len(days))
	for i, day := range days {
		speed, valid := windAverage(day)
		windows[i] = windWindow{
			startTimestamp: day.Date,
			endTimestamp:   day.Date,
			startTime:      dates[i],
			avgSpeed:       math.NaN(),
		}
		if !valid || dates[i].IsZero() {
			continue
		}
		windows[i].avgSpeed = speed
		if validWindValue(day.MaxWindSpeed, false) == nil {
			windows[i].avgSpeed = math.NaN()
			continue
		}
		windows[i].maxSpeed = *day.MaxWindSpeed
		windows[i].avgDirection = day.AvgWindDirection
		// Consistency measures alignment; sample count checks coverage separately.
		windows[i].consistency = day.WindDirectionConsistency
		if day.WindDirectionSampleCount != day.HourCount {
			windows[i].avgDirection = nil
		}
		if completeTemperature(day) {
			windows[i].avgTemperature = day.AvgTemperature
			windows[i].maxTemp = day.MaxTemperature
		}
		if completeSolar(day) {
			v := day.SolarEnergyKWhM2
			windows[i].solar = &v
		}
	}
	return append(out, windPeriodInsights(windows, 24*time.Hour, cfg.MinimumLowDays, cfg.LowSolarDaily, cfg)...)
}

// Hourly timestamps retain the forecast location's timezone. Gaps, duplicate
// instants and missing speeds break windows rather than bridging absent data.
func hourlyWindInsights(points []Point, timezone string, now time.Time, cfg WindInsightConfig) []Signal {
	const (
		windWindowHours    = 6                           // Number of consecutive hours to form a wind window for trend analysis.
		windWindowDuration = windWindowHours * time.Hour // Duration of a wind window for trend analysis.
	)

	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	localNow := now.In(location)

	forecastStart := now.Add(-time.Duration(localNow.Minute())*time.Minute - time.Duration(localNow.Second())*time.Second - time.Duration(localNow.Nanosecond()))
	forecastEnd := forecastStart.Add(72 * time.Hour)

	var windows []windWindow
	var block []Point
	var previous time.Time
	var blockStart time.Time

	resetBlock := func() { block = nil }

	for _, point := range points {
		timestamp, err := forecastPointTime(point, location)
		if err != nil {
			resetBlock()
			previous = time.Time{}
			continue
		}
		if timestamp.Before(forecastStart) || !timestamp.Before(forecastEnd) {
			continue
		}
		if validWindValue(point.WindSpeed, false) == nil {
			resetBlock()
			previous = timestamp
			continue
		}
		if !previous.IsZero() && timestamp.Sub(previous) != time.Hour {
			resetBlock()
		}
		previous = timestamp
		if len(block) == 0 {
			blockStart = timestamp
		}
		block = append(block, point)

		if len(block) != windWindowHours {
			continue
		}

		start := blockStart
		w := windWindow{
			startTimestamp: block[0].Time,
			endTimestamp:   block[len(block)-1].Time,
			startTime:      start,
		}

		var sinSum, cosSum, avgTemperature, avgSolarIrradiance float64
		var temperatureSum, solarIrradianceSum float64
		validWindDirectionCount, validTemperatureCount, validSolarReadingCount := 0, 0, 0
		maxTemp := math.Inf(-1)

		for _, p := range block {
			w.avgSpeed += *p.WindSpeed / float64(windWindowHours)
			w.maxSpeed = math.Max(w.maxSpeed, *p.WindSpeed)
			if d := validWindValue(p.WindDirection, true); d != nil && *p.WindSpeed > 0 {
				sinSum += math.Sin(*d * math.Pi / 180)
				cosSum += math.Cos(*d * math.Pi / 180)
				validWindDirectionCount++
			}
			if validTemperature(p.Temperature) {
				temperatureSum += *p.Temperature
				avgTemperature = temperatureSum / float64(windWindowHours)
				validTemperatureCount++
				maxTemp = math.Max(maxTemp, *p.Temperature)
			}
			if validSolarIrradiance(p.SolarIrradiance) {
				solarIrradianceSum += *p.SolarIrradiance
				avgSolarIrradiance = solarIrradianceSum / float64(windWindowHours)
				validSolarReadingCount++
			}
		}

		if validWindDirectionCount == windWindowHours {
			w.avgDirection, w.consistency = avgWindDirection(sinSum, cosSum, validWindDirectionCount)
		}
		if validTemperatureCount == windWindowHours {
			w.avgTemperature = &avgTemperature
			w.maxTemp = &maxTemp
		}
		if validSolarReadingCount == windWindowHours {
			w.solar = &avgSolarIrradiance
		}
		windows = append(windows, w)
		resetBlock()
	}

	var out []Signal

	for _, trendDirection := range []float64{decreasingTrend, increasingTrend} {
		bestEffectiveChange := 0.0
		var bestStart, bestEnd windWindow
		for i, start := range windows {
			if start.maxSpeed > cfg.StrongSpeed {
				continue
			}
			speedChangeCount := 0
			for j := i + 1; j < len(windows); j++ {
				current, prev := windows[j], windows[j-1]
				if current.startTime.Sub(prev.startTime) != windWindowDuration || current.maxSpeed > cfg.StrongSpeed {
					break
				}
				delta := current.avgSpeed - prev.avgSpeed
				if trendDirection*delta < 0 {
					break
				}
				if delta != 0 {
					speedChangeCount++
				}
				totalSpeedChange := current.avgSpeed - start.avgSpeed
				effectiveChange := trendDirection * totalSpeedChange
				if speedChangeCount >= 2 && effectiveChange >= cfg.SpeedChange && effectiveChange > bestEffectiveChange {
					bestEffectiveChange = effectiveChange
					bestStart = start
					bestEnd = current
				}
			}
		}
		if bestEffectiveChange > 0 {
			out = append(out, windTrendSignal(bestStart.startTimestamp, bestEnd.endTimestamp, bestStart.avgSpeed, bestEnd.avgSpeed))
		}
	}

	minimumLowWindWindows := (cfg.MinimumLowHours + windWindowHours - 1) / windWindowHours
	return append(out, windPeriodInsights(windows, windWindowDuration, minimumLowWindWindows, cfg.LowSolarHourly, cfg)...)
}

// windowInterval is the expected spacing between windows. Daily windows use UTC
// calendar dates, so 24 hours checks consecutive dates, not local day length.
func windPeriodInsights(windows []windWindow, windowInterval time.Duration, minLowWindWindows int, lowSolar float64, cfg WindInsightConfig) []Signal {
	var out []Signal
	longestLowWindStartIndex, longestLowWindWindowCount := -1, 0
	currentLowWindWindowCount := 0

	// Find the longest consecutive low-wind period.
	for i, w := range windows {
		isLowWind := !(math.IsNaN(w.avgSpeed) || w.avgSpeed >= cfg.LowSpeed || w.maxSpeed >= cfg.LowSpeed)
		isContinuous := i > 0 && w.startTime.Sub(windows[i-1].startTime) == windowInterval
		if !isLowWind {
			currentLowWindWindowCount = 0
			continue
		}
		if !isContinuous {
			currentLowWindWindowCount = 1
		} else {
			currentLowWindWindowCount++
		}
		if currentLowWindWindowCount >= minLowWindWindows && currentLowWindWindowCount > longestLowWindWindowCount {
			longestLowWindStartIndex, longestLowWindWindowCount = i-currentLowWindWindowCount+1, currentLowWindWindowCount
		}
	}

	if longestLowWindStartIndex >= 0 {
		startWindow := windows[longestLowWindStartIndex]
		endWindow := windows[longestLowWindStartIndex+longestLowWindWindowCount-1]
		duration := longestLowWindWindowCount * int(windowInterval/time.Hour)

		unit := "hours"
		if windowInterval == 24*time.Hour {
			duration = longestLowWindWindowCount
			unit = "days"
		}
		out = append(out, Signal{
			Kind:     "wind",
			Severity: "warning",
			Title:    "Prolonged low-wind period",
			Detail: fmt.Sprintf(
				"Average wind at 120 m remains below %.1f m/s for %d forecast %s from %s through %s. "+
					"Low wind may limit favorable wind generation conditions.",
				cfg.LowSpeed,
				duration,
				unit,
				startWindow.startTimestamp,
				endWindow.endTimestamp,
			),
		})

		hasValidTemps := startWindow.avgTemperature != nil && endWindow.avgTemperature != nil
		tempFalling := hasValidTemps && *endWindow.avgTemperature < heatingReference && (*endWindow.avgTemperature-*startWindow.avgTemperature) <= -temperatureChangeThreshold
		persistentCold, complete := true, true
		for _, w := range windows[longestLowWindStartIndex : longestLowWindStartIndex+longestLowWindWindowCount] {
			if !validTemperature(w.avgTemperature) {
				complete = false
			}
			if !validTemperature(w.maxTemp) || *w.maxTemp >= cfg.PersistentCold {
				persistentCold = false
			}
			if !validSolarIrradiance(w.solar) || *w.solar > lowSolar {
				complete = false
			}
		}
		if complete && (tempFalling || persistentCold) {
			title := "Cold, low-solar and low-wind conditions"
			temperatureDetail := fmt.Sprintf("average temperature falls from %.1f to %.1f°C", *startWindow.avgTemperature, *endWindow.avgTemperature)
			if !tempFalling {
				title = "Persistent cold, low-solar and low-wind conditions"
				temperatureDetail = fmt.Sprintf("all measured temperatures remain below %.1f°C", cfg.PersistentCold)
			}
			solarDetail := fmt.Sprintf("daily solar energy stays at or below %.1f kWh/m²", lowSolar)
			if windowInterval != 24*time.Hour {
				solarDetail = fmt.Sprintf("six-hour solar averages stay at or below %.0f W/m² (including nighttime)", lowSolar)
			}
			// The combined description replaces low wind for this exact period.
			out[len(out)-1] = Signal{
				Kind: "combined", Severity: "warning", Title: title,
				Params: map[string]any{"start": startWindow.startTimestamp, "end": endWindow.endTimestamp, "worsening": tempFalling},
				Detail: fmt.Sprintf("From %s through %s, %s, %s, and wind remains below %.1f m/s for %d forecast %s. Heating-related demand may be elevated while renewable generation conditions are weak; actual demand, production and prices are not predicted.", startWindow.startTimestamp, endWindow.endTimestamp, temperatureDetail, solarDetail, cfg.LowSpeed, duration, unit),
			}
		}
	}

	bestShift := 0.0
	var shiftStart, shiftEnd windWindow
	for i := 1; i < len(windows); i++ {
		prev, curr := windows[i-1], windows[i]
		if curr.startTime.Sub(prev.startTime) != windowInterval {
			continue
		}
		if prev.avgDirection == nil || curr.avgDirection == nil {
			continue
		}
		if math.IsNaN(prev.avgSpeed) || math.IsNaN(curr.avgSpeed) || prev.avgSpeed < cfg.LowSpeed || curr.avgSpeed < cfg.LowSpeed {
			continue
		}
		if prev.consistency < cfg.DirectionConsistency || curr.consistency < cfg.DirectionConsistency {
			continue
		}
		shift := angleDifference(*prev.avgDirection, *curr.avgDirection)
		if shift >= cfg.DirectionChange && shift > bestShift {
			bestShift = shift
			shiftStart, shiftEnd = prev, curr
		}
	}
	if bestShift > 0 {
		prevDirection := *shiftStart.avgDirection
		currDirection := *shiftEnd.avgDirection

		prevDirectionName := windDirectionName(prevDirection)
		currDirectionName := windDirectionName(currDirection)
		out = append(out, Signal{
			Kind:     "wind",
			Severity: "info",
			Title:    "Wind direction shift",
			Detail: fmt.Sprintf(
				"The prevailing wind direction at 120 m shifts by %.0f°, "+
					"from %s (%.0f°) in the forecast period starting at %s "+
					"to %s (%.0f°) in the period starting at %s. "+
					"This may influence local wind generation depending on "+
					"turbine layout and site conditions.",
				bestShift,
				prevDirectionName,
				prevDirection,
				shiftStart.startTimestamp,
				currDirectionName,
				currDirection,
				shiftEnd.startTimestamp,
			),
		})
	}
	return out
}

func mergeWindInsights(existing, wind []Signal) []Signal {
	out := make([]Signal, 0, len(existing)+len(wind))
	for _, s := range wind {
		if s.Kind == "combined" {
			out = append(out, s)
		}
	}
	// Only worsening combined insights can replace a cold temperature trend.
	combinedCount := len(out)
	for _, s := range existing {
		redundant := false
		if s.Code == "temperature_trend" && s.Params["interpretation"] == "heating_increase" {
			start, end := s.Params["start"], s.Params["end"]
			if start != nil && end != nil {
				for _, combined := range out[:combinedCount] {
					worsening, _ := combined.Params["worsening"].(bool)
					if worsening && start == combined.Params["start"] && end == combined.Params["end"] {
						redundant = true
						break
					}
				}
			}
		}
		if !redundant {
			out = append(out, s)
		}
	}
	for _, s := range wind {
		if s.Kind != "combined" {
			out = append(out, s)
		}
	}
	return out
}

// These categories are illustrative weather proxies, not turbine power curves.
// Boundary values belong to the higher category, except 15 m/s remains high.
func windPotential(speed *float64) *string {
	if validWindValue(speed, false) == nil {
		return nil
	}
	var potential string
	switch {
	case *speed < 3:
		potential = "very_low"
	case *speed < 6:
		potential = "low"
	case *speed < 10:
		potential = "moderate"
	case *speed <= 15:
		potential = "high"
	default:
		potential = "strong_uncertain"
	}
	return &potential
}

// windTrendSignal returns a signal for a sustained change in average wind speed.
func windTrendSignal(start, end string, startSpeed, endSpeed float64) Signal {
	code := "wind_speed_increasing"
	title, severity := "Wind conditions improving", "info"
	implication := "Increasing wind speeds may improve conditions for wind power generation."

	if endSpeed < startSpeed {
		code = "wind_speed_decreasing"
		title, severity = "Wind conditions weakening", "warning"
		implication = "Decreasing wind speeds may reduce favorable conditions for wind power generation."
	}
	return Signal{
		Code: code,
		Params: map[string]any{
			"change": endSpeed - startSpeed,
			"start":  start,
			"end":    end,
		},
		Kind:     "wind",
		Severity: severity,
		Title:    title,
		Detail: fmt.Sprintf(
			"Average wind at 120 m changes by %+.1f m/s from %s through %s across sustained forecast periods. %s "+
				"Wind speed indicates generation conditions, not actual production.",
			endSpeed-startSpeed,
			start,
			end,
			implication,
		),
	}
}

// angleDifference returns the absolute difference between two angles in degrees, normalized to [0, 180].
func angleDifference(prev, curr float64) float64 {
	diff := math.Mod(math.Abs(curr-prev), 360)
	// Normalize to [0, 180] by taking the smaller angle between the two directions.
	if diff > 180 {
		return 360 - diff
	}
	return diff
}

func windDirectionName(degrees float64) string {
	var windDirections = [...]string{
		"northerly",
		"northeasterly",
		"easterly",
		"southeasterly",
		"southerly",
		"southwesterly",
		"westerly",
		"northwesterly",
	}

	if math.IsNaN(degrees) || math.IsInf(degrees, 0) {
		return "unknown"
	}

	normalized := math.Mod(degrees, 360)
	if normalized < 0 {
		normalized += 360
	}

	index := int((normalized+22.5)/45) % len(windDirections)
	return windDirections[index]
}
