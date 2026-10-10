package weather

import (
	"sort"
	"time"
	_ "time/tzdata"
)

func summarize(points []Point, timezone ...string) []DailySummary {
	location := time.UTC

	if len(timezone) > 0 {
		if parsedLocation, err := time.LoadLocation(timezone[0]); err == nil {
			location = parsedLocation
		}
	}

	type dailyAgg struct {
		tempSum                     float64
		tempSampleCount             int
		minTemp, maxTemp            float64
		solarEnergy                 float64
		hourCount, solarSampleCount int
		peakIrradiance              *float64
		peakTime                    string
		expectedSlots               map[string]int
		seenSlots                   map[string]int
		expectedHourCount           int
	}

	aggByDate := map[string]*dailyAgg{}
	dates := []string{}

	for _, point := range points {
		timestamp, err := time.ParseInLocation("2006-01-02T15:04", point.Time, location)

		if err != nil {
			continue
		}

		date := timestamp.Format("2006-01-02")
		agg := aggByDate[date]

		if agg == nil {
			dayStart, _ := time.ParseInLocation("2006-01-02", date, location)
			dayEnd := dayStart.AddDate(0, 0, 1)
			agg = &dailyAgg{
				expectedSlots:     make(map[string]int),
				seenSlots:         make(map[string]int),
				expectedHourCount: int(dayEnd.Sub(dayStart) / time.Hour),
			}
			// The repeated wall-clock hour on a DST fall-back day has two slots.
			for hour := dayStart; hour.Before(dayEnd); hour = hour.Add(time.Hour) {
				agg.expectedSlots[hour.Format("2006-01-02T15:04")]++
			}
			aggByDate[date] = agg
			dates = append(dates, date)
		}
		// Count each expected local-hour slot once (twice for a repeated DST hour).
		// Duplicates and timestamps outside the hourly grid cannot fill missing hours.
		if agg.seenSlots[point.Time] >= agg.expectedSlots[point.Time] {
			continue
		}

		agg.seenSlots[point.Time]++
		agg.hourCount++

		if point.Temperature != nil {
			temp := *point.Temperature
			if agg.tempSampleCount == 0 || temp < agg.minTemp {
				agg.minTemp = temp
			}
			if agg.tempSampleCount == 0 || temp > agg.maxTemp {
				agg.maxTemp = temp
			}
			agg.tempSum += temp
			agg.tempSampleCount++
		}

		if point.SolarIrradiance != nil && *point.SolarIrradiance >= 0 {
			irradiance := *point.SolarIrradiance
			agg.solarSampleCount++
			agg.solarEnergy += irradiance / 1000
			if agg.peakIrradiance == nil || irradiance > *agg.peakIrradiance {
				agg.peakIrradiance = &irradiance
				agg.peakTime = point.Time
			}
		}
	}

	sort.Strings(dates)
	summaries := []DailySummary{}

	for _, date := range dates {
		agg := aggByDate[date]
		summary := DailySummary{
			Date:                   date,
			SolarEnergyKWhM2:       agg.solarEnergy,
			PeakSolarIrradiance:    agg.peakIrradiance,
			PeakSolarTime:          agg.peakTime,
			SolarSampleCount:       agg.solarSampleCount,
			HourCount:              agg.hourCount,
			ExpectedHourCount:      agg.expectedHourCount,
			CompleteDay:            agg.hourCount == agg.expectedHourCount,
			TemperatureSampleCount: agg.tempSampleCount,
		}
		if agg.tempSampleCount > 0 {
			avgTemp := agg.tempSum / float64(agg.tempSampleCount)
			minTemp := agg.minTemp
			maxTemp := agg.maxTemp
			summary.AvgTemperature = &avgTemp
			summary.MinTemperature = &minTemp
			summary.MaxTemperature = &maxTemp
		}
		summaries = append(summaries, summary)
	}
	return summaries
}
