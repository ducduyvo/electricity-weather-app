package weather

import (
	"context"
)

type Provider interface {
	Forecast(context.Context, float64, float64) (Forecast, error)
}
