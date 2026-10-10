package weather

import (
	"errors"
	"math"
)

var ErrInvalidCoordinates = errors.New("invalid coordinates")

func Validate(lat, lon float64) error {
	if math.IsNaN(lat) ||
		math.IsNaN(lon) ||
		math.IsInf(lat, 0) ||
		math.IsInf(lon, 0) ||
		lat < -90 ||
		lat > 90 ||
		lon < -180 ||
		lon > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}
