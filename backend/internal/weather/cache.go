package weather

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheItem struct {
	f       Forecast
	expires time.Time
}
type Cached struct {
	Next  Provider
	TTL   time.Duration
	mu    sync.Mutex
	data  map[string]cacheItem
	loads singleflight.Group
}

func (c *Cached) Forecast(ctx context.Context, lat, lon float64) (Forecast, error) {
	// Coordinates are rounded to three decimal places; nearby points share an entry.
	// The upstream provider formats coordinates to four decimal places.
	key := fmt.Sprintf("%.3f,%.3f", lat, lon)
	c.mu.Lock()
	if c.data == nil {
		c.data = map[string]cacheItem{}
	}
	now := time.Now()
	// Remove entries whose TTL has elapsed when a request arrives.
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
	result := c.loads.DoChan(key, func() (any, error) {
		// Another load may have filled the cache since our initial lookup.
		c.mu.Lock()
		if v, ok := c.data[key]; ok && time.Now().Before(v.expires) {
			c.mu.Unlock()
			return v.f, nil
		}
		c.mu.Unlock()

		// The shared fetch outlives individual callers, but remains bounded.
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		f, err := c.Next.Forecast(fetchCtx, lat, lon)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.data[key] = cacheItem{f: f, expires: time.Now().Add(c.TTL)}
		c.mu.Unlock()
		return f, nil
	})
	select {
	case <-ctx.Done():
		return Forecast{}, ctx.Err()
	case r := <-result:
		if r.Err != nil {
			return Forecast{}, r.Err
		}
		return r.Val.(Forecast), nil
	}
}
