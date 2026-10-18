package weather

import "context"

type fakeProvider func(context.Context, float64, float64) (Forecast, error)

func (p fakeProvider) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	return p(ctx, lat, lon)
}
