package weather

import (
	"math"
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
		directionSin, directionCos  float64
		directionCount              int
		windSum, minWind, maxWind   float64
		windCount                   int
		tempSum                     float64
		tempSampleCount             int
		minTemp, maxTemp            float64
		solarEnergy                 float64
		hourCount, solarSampleCount int
		peakIrradiance              *float64
		peakTime                    string
		expectedHourlySlots         map[int64]struct{}
		observedHourlySlots         map[int64]struct{}
		observedSolarSlots          map[int64]struct{}
		expectedHourCount           int
	}

	aggByDate := map[string]*dailyAgg{}
	dates := []string{}

	for _, point := range points {
		timestamp, err := forecastPointTime(point, location)
		if err != nil {
			continue
		}

		date := timestamp.Format("2006-01-02")
		agg := aggByDate[date]

		if agg == nil {
			dayStart, _ := time.ParseInLocation("2006-01-02", date, location)
			dayEnd := dayStart.AddDate(0, 0, 1)
			agg = &dailyAgg{
				expectedHourlySlots: make(map[int64]struct{}),
				observedHourlySlots: make(map[int64]struct{}),
				observedSolarSlots:  make(map[int64]struct{}),
				expectedHourCount:   int(dayEnd.Sub(dayStart) / time.Hour),
			}
			// The repeated wall-clock hour on a DST fall-back day has two slots.
			for hour := dayStart; hour.Before(dayEnd); hour = hour.Add(time.Hour) {
				agg.expectedHourlySlots[hour.Unix()] = struct{}{}
			}
			aggByDate[date] = agg
			dates = append(dates, date)
		}
		// Accept each exact hourly instant once, including distinct repeated DST hours.
		slotKey := timestamp.Unix()
		if _, expected := agg.expectedHourlySlots[slotKey]; !expected {
			continue
		}
		if _, observed := agg.observedHourlySlots[slotKey]; observed {
			continue
		}
		agg.observedHourlySlots[slotKey] = struct{}{}
		agg.hourCount++

		if speed := validWindValue(point.WindSpeed, false); speed != nil {
			agg.windSum += *speed
			if agg.windCount == 0 || *speed < agg.minWind {
				agg.minWind = *speed
			}
			if agg.windCount == 0 || *speed > agg.maxWind {
				agg.maxWind = *speed
			}
			agg.windCount++
			// Equal-weight circular average of wind-from bearings; calm winds have no direction.
			if direction := validWindValue(point.WindDirection, true); direction != nil && *speed > 0 {
				radians := *direction * math.Pi / 180
				agg.directionSin += math.Sin(radians)
				agg.directionCos += math.Cos(radians)
				agg.directionCount++
			}
		}

		if validTemperature(point.Temperature) {
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

	}

	// Radiation is the mean of the preceding hour. Assign its energy to
	// that interval's local day, independently of instantaneous measurements.
	for _, point := range points {
		end, err := forecastPointTime(point, location)
		if err != nil {
			continue
		}
		start := end.Add(-time.Hour)
		agg := aggByDate[start.Format("2006-01-02")]
		if agg == nil {
			continue
		}
		slotKey := start.Unix()
		if _, expected := agg.expectedHourlySlots[slotKey]; !expected {
			continue
		}
		if _, observed := agg.observedSolarSlots[slotKey]; observed {
			continue
		}
		if !validSolarIrradiance(point.SolarIrradiance) {
			continue
		}
		agg.observedSolarSlots[slotKey] = struct{}{}
		irradiance := *point.SolarIrradiance
		agg.solarSampleCount++
		agg.solarEnergy += irradiance / 1000
		if agg.peakIrradiance == nil || irradiance > *agg.peakIrradiance {
			agg.peakIrradiance = &irradiance
			agg.peakTime = point.Time
		}
	}

	sort.Strings(dates)
	summaries := []DailySummary{}

	for _, date := range dates {
		agg := aggByDate[date]
		summary := DailySummary{
			Date:                     date,
			WindDirectionSampleCount: agg.directionCount,
			WindSampleCount:          agg.windCount,
			SolarEnergyKWhM2:         agg.solarEnergy,
			PeakSolarIrradiance:      agg.peakIrradiance,
			PeakSolarTime:            agg.peakTime,
			SolarSampleCount:         agg.solarSampleCount,
			HourCount:                agg.hourCount,
			ExpectedHourCount:        agg.expectedHourCount,
			CompleteDay:              agg.hourCount == agg.expectedHourCount,
			CompleteTemperature:      agg.expectedHourCount > 0 && agg.tempSampleCount == agg.expectedHourCount,
			CompleteWind:             agg.expectedHourCount > 0 && agg.windCount == agg.expectedHourCount,
			CompleteSolar:            agg.expectedHourCount > 0 && agg.solarSampleCount == agg.expectedHourCount,
			TemperatureSampleCount:   agg.tempSampleCount,
		}
		summary.AvgWindDirection, summary.WindDirectionConsistency = avgWindDirection(
			agg.directionSin,
			agg.directionCos,
			agg.directionCount,
		)
		if agg.windCount > 0 {
			avgWind := agg.windSum / float64(agg.windCount)
			minWind := agg.minWind
			maxWind := agg.maxWind
			summary.MinWindSpeed = &minWind
			summary.AvgWindSpeed = &avgWind
			summary.MaxWindSpeed = &maxWind
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
