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
	}
}

type windWindow struct {
	startTimestamp string
	endTimestamp   string
	startTime      time.Time
	meanSpeed      float64
	maxSpeed       float64
	meanDirection  *float64
	consistency    float64
	temp           *float64
	solar          *float64
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
		meanSpeed, valid := windMean(day)
		return meanSpeed, valid && day.MaxWindSpeed != nil && *day.MaxWindSpeed <= cfg.StrongSpeed
	}

	var out []Signal

	for _, trendDirection := range []float64{decreasingTrend, increasingTrend} {
		startDay, endDay, found := trendPeriod(days, dates, measurement, trendDirection, func(a, b float64) bool { return trendDirection*(b-a) >= cfg.SpeedChange })
		if found {
			out = append(out, windTrendSignal(startDay.Date, endDay.Date, *startDay.MeanWindSpeed, *endDay.MeanWindSpeed))
		}
	}

	windows := make([]windWindow, len(days))
	for i, day := range days {
		speed, valid := windMean(day)
		windows[i] = windWindow{
			startTimestamp: day.Date,
			endTimestamp:   day.Date,
			startTime:      dates[i],
			meanSpeed:      math.NaN(),
		}
		if !valid || dates[i].IsZero() {
			continue
		}
		windows[i].meanSpeed = speed
		windows[i].meanDirection = day.MeanWindDirection
		// WindDirectionConsistency is a ratio of valid wind-direction samples to total samples, not a directional measure.
		// 1 indicates all valid samples are mostly from the same direction, 0 indicates directions vary widely or cancel each other out.
		windows[i].consistency = day.WindDirectionConsistency
		if day.WindDirectionSampleCount != day.HourCount {
			windows[i].meanDirection = nil
		}
		if completeTemperature(day) {
			windows[i].temp = day.AvgTemperature
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
	end := forecastStart.Add(72 * time.Hour)

	var windows []windWindow
	var block []Point
	var previous time.Time

	resetBlock := func() { block = nil }

	for _, point := range points {
		timestamp, err := time.ParseInLocation("2006-01-02T15:04", point.Time, location)
		if err != nil {
			resetBlock()
			previous = time.Time{}
			continue
		}
		if timestamp.Before(forecastStart) || !timestamp.Before(end) {
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
		block = append(block, point)

		if len(block) != windWindowHours {
			continue
		}

		start, _ := time.ParseInLocation("2006-01-02T15:04", block[0].Time, location)
		w := windWindow{
			startTimestamp: block[0].Time,
			endTimestamp:   block[len(block)-1].Time,
			startTime:      start,
		}

		var sinSum, cosSum, temp, solar float64
		directions, temps, solars := 0, 0, 0

		for _, p := range block {
			w.meanSpeed += *p.WindSpeed / float64(windWindowHours)
			w.maxSpeed = math.Max(w.maxSpeed, *p.WindSpeed)
			if d := validWindValue(p.WindDirection, true); d != nil && *p.WindSpeed > 0 {
				sinSum += math.Sin(*d * math.Pi / 180)
				cosSum += math.Cos(*d * math.Pi / 180)
				directions++
			}
			if validTemperature(p.Temperature) {
				temp += *p.Temperature / windWindowHours
				temps++
			}
			if validSolarIrradiance(p.SolarIrradiance) {
				solar += *p.SolarIrradiance / windWindowHours
				solars++
			}
		}

		if directions == windWindowHours {
			d := math.Mod(math.Atan2(sinSum, cosSum)*180/math.Pi+360, 360)
			w.meanDirection = &d
			length := math.Hypot(sinSum, cosSum)
			w.consistency = length / windWindowHours
		}
		if temps == windWindowHours {
			w.temp = &temp
		}
		if solars == windWindowHours {
			w.solar = &solar
		}
		windows = append(windows, w)
		resetBlock()
	}
	var out []Signal
	for _, trendDirection := range []float64{decreasingTrend, increasingTrend} {
		bestChange := 0.0
		var bestStart, bestEnd windWindow
		for i, start := range windows {
			if start.maxSpeed > cfg.StrongSpeed {
				continue
			}
			changes := 0
			for j := i + 1; j < len(windows); j++ {
				current, prev := windows[j], windows[j-1]
				if current.startTime.Sub(prev.startTime) != windWindowDuration || current.maxSpeed > cfg.StrongSpeed {
					break
				}
				delta := current.meanSpeed - prev.meanSpeed
				if trendDirection*delta < 0 {
					break
				}
				if delta != 0 {
					changes++
				}
				change := current.meanSpeed - start.meanSpeed
				if j-i >= 2 && changes >= 2 && math.Abs(change) >= cfg.SpeedChange && math.Abs(change) > math.Abs(bestChange) {
					bestChange = change
					bestStart = start
					bestEnd = current
				}
			}
		}
		if bestChange != 0 {
			out = append(out, windTrendSignal(bestStart.startTimestamp, bestEnd.endTimestamp, bestStart.meanSpeed, bestEnd.meanSpeed))
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
		isLowWind := !(math.IsNaN(w.meanSpeed) || w.meanSpeed >= cfg.LowSpeed || w.maxSpeed >= cfg.LowSpeed)
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
				"Mean wind at 120 m remains below %.1f m/s for %d forecast %s from %s through %s. "+
					"Low wind may limit favorable wind generation conditions.",
				cfg.LowSpeed,
				duration,
				unit,
				startWindow.startTimestamp,
				endWindow.endTimestamp,
			),
		})

		hasValidTemps := startWindow.temp != nil && endWindow.temp != nil
		tempFalling := hasValidTemps && *endWindow.temp < heatingReference && (*endWindow.temp-*startWindow.temp) <= -temperatureChangeThreshold
		hasCompleteLowSolarData := false

		if tempFalling {
			hasCompleteLowSolarData = true
			for _, w := range windows[longestLowWindStartIndex : longestLowWindStartIndex+longestLowWindWindowCount] {
				if w.temp == nil || w.solar == nil || *w.solar > lowSolar {
					hasCompleteLowSolarData = false
					break
				}
			}
		}
		if tempFalling && hasCompleteLowSolarData {
			out = append(out, Signal{
				Kind:     "combined",
				Severity: "warning",
				Title:    "Cold, low-solar and low-wind conditions",
				Detail: fmt.Sprintf(
					"From %s through %s, mean temperature falls from %.1f to %.1f°C alongside low solar availability and mean wind below %.1f m/s. "+
						"Heating-related demand may rise while renewable generation conditions weaken; actual demand, production and prices are not predicted.",
					startWindow.startTimestamp,
					endWindow.endTimestamp,
					*startWindow.temp,
					*endWindow.temp,
					cfg.LowSpeed,
				),
			})
		}
	}

	bestShift := 0.0
	var shiftStart, shiftEnd windWindow
	for i := 1; i < len(windows); i++ {
		prev, curr := windows[i-1], windows[i]
		if curr.startTime.Sub(prev.startTime) != windowInterval {
			continue
		}
		if prev.meanDirection == nil || curr.meanDirection == nil {
			continue
		}
		if math.IsNaN(prev.meanSpeed) || math.IsNaN(curr.meanSpeed) || prev.meanSpeed < cfg.LowSpeed || curr.meanSpeed < cfg.LowSpeed {
			continue
		}
		if prev.consistency < cfg.DirectionConsistency || curr.consistency < cfg.DirectionConsistency {
			continue
		}
		shift := angleDifference(*prev.meanDirection, *curr.meanDirection)
		if shift >= cfg.DirectionChange && shift > bestShift {
			bestShift = shift
			shiftStart, shiftEnd = prev, curr
		}
	}
	if bestShift > 0 {
		prevDirection := *shiftStart.meanDirection
		currDirection := *shiftEnd.meanDirection

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
	combined := false
	for _, s := range wind {
		if s.Kind == "combined" {
			combined = true
		}
	}
	out := make([]Signal, 0, len(existing)+len(wind))
	for _, s := range existing {
		if combined && s.Kind == "combined" {
			continue
		}
		out = append(out, s)
	}
	return append(out, wind...)
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

// windTrendSignal returns a signal for a sustained change in mean wind speed.
func windTrendSignal(start, end string, startSpeed, endSpeed float64) Signal {
	title, severity := "Wind conditions improving", "info"
	implication := "Increasing wind speeds may improve conditions for wind power generation."

	if endSpeed < startSpeed {
		title, severity = "Wind conditions weakening", "warning"
		implication = "Decreasing wind speeds may reduce favorable conditions for wind power generation."
	}
	return Signal{
		Kind:     "wind",
		Severity: severity,
		Title:    title,
		Detail: fmt.Sprintf(
			"Mean wind at 120 m changes by %+.1f m/s from %s through %s across sustained forecast periods. %s "+
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
