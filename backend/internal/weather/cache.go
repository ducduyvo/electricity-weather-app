package weather

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type cacheItem struct {
	f       Forecast
	expires time.Time
}
type Cached struct {
	Next Provider
	TTL  time.Duration
	mu   sync.Mutex
	data map[string]cacheItem
}

func (c *Cached) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	// Coordinates are rounded to three decimal places; nearby points share an entry.
	// The upstream provider formats coordinates to four decimal places.
	key := fmt.Sprintf("%.3f,%.3f", lat, lon)
	c.mu.Lock()
	if c.data == nil {
		c.data = map[string]cacheItem{}
	}
	// Lazy eviction needs no background goroutine.
	now := time.Now()
	for k, v := range c.data {
		if !now.Before(v.expires) {
			delete(c.data, k)
		}
	}
	if v, ok := c.data[key]; ok {
		c.mu.Unlock()
		return v.f, nil
	}
	c.mu.Unlock()
	f, e := c.Next.Forecast(ctx, lat, lon)
	if e != nil {
		return Forecast{}, e
	}
	c.mu.Lock()
	c.data[key] = cacheItem{f: f, expires: time.Now().Add(c.TTL)}
	c.mu.Unlock()
	return f, nil
}
