package weather

import (
	"math"
	"time"
)

// Unix timestamps distinguish repeated local hours; older callers can use local strings.
func forecastPointTime(point Point, location *time.Location) (time.Time, error) {
	if point.UnixTime != nil {
		return time.Unix(*point.UnixTime, 0).In(location), nil
	}
	return time.ParseInLocation("2006-01-02T15:04", point.Time, location)
}

// AvgWindDirection returns the circular average in degrees and consistency (0–1).
// Missing or cancelling direction vectors produce no average direction.
func avgWindDirection(sinSum, cosSum float64, sampleCount int) (*float64, float64) {
	if sampleCount <= 0 {
		return nil, 0
	}

	directionVectorLength := math.Hypot(sinSum, cosSum)
	consistency := directionVectorLength / float64(sampleCount)

	if consistency <= 1e-6 {
		return nil, consistency
	}

	avgDirection := math.Mod(math.Atan2(sinSum, cosSum)*180/math.Pi+360, 360)

	return &avgDirection, consistency
}
