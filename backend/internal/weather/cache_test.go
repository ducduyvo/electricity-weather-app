package weather

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCachedForecast(t *testing.T) {
	calls := 0
	fail := false
	failure := errors.New("upstream failed")
	ctx := context.WithValue(context.Background(), cacheContextKey{}, "request")
	cache := &Cached{
		TTL: time.Minute,
		Next: fakeProvider(func(received context.Context, lat, lon float64) (Forecast, error) {
			calls++
			if received.Value(cacheContextKey{}) != ctx.Value(cacheContextKey{}) {
				t.Error("context value lost")
			}
			if deadline, ok := received.Deadline(); !ok || time.Until(deadline) > 10*time.Second {
				t.Error("shared fetch must have a bounded deadline")
			}
			if fail {
				return Forecast{}, failure
			}
			return Forecast{Latitude: lat, Longitude: lon}, nil
		}),
	}
	get := func(lat, lon float64) Forecast {
		t.Helper()
		got, err := cache.Forecast(ctx, lat, lon)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	first := get(60, 24)
	if calls != 1 {
		t.Fatalf("first call count %d", calls)
	}
	second := get(60, 24)
	if calls != 1 || second.Latitude != first.Latitude {
		t.Fatal("cache miss on second call")
	}
	get(61, 24)
	get(60, 25)
	if calls != 3 {
		t.Fatal("coordinates did not have separate entries")
	}
	// Expire deterministically instead of sleeping for the TTL.
	cache.mu.Lock()
	item := cache.data["60.000,24.000"]
	item.expires = time.Now().Add(-time.Second)
	cache.data["60.000,24.000"] = item
	cache.mu.Unlock()
	get(60, 24)
	if calls != 4 {
		t.Fatal("expired entry was reused")
	}
	fail = true
	for range 2 {
		_, err := cache.Forecast(ctx, 62, 24)
		if !errors.Is(err, failure) {
			t.Fatalf("error cause lost: %v", err)
		}
	}
	if calls != 6 {
		t.Fatal("failed call was cached")
	}
	fail = false
	get(62, 24)
	get(62, 24)
	if calls != 7 {
		t.Fatal("successful retry not cached")
	}
}

type cacheContextKey struct{}

func TestCachedSharedLoadSurvivesInitiatorCancellation(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	cache := &Cached{
		TTL: time.Minute,
		Next: fakeProvider(func(ctx context.Context, lat, lon float64) (Forecast, error) {
			calls.Add(1)
			started <- struct{}{}
			select {
			case <-ctx.Done():
				return Forecast{}, ctx.Err()
			case <-release:
				return Forecast{Latitude: lat}, ctx.Err()
			}
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	go func() {
		_, err := cache.Forecast(ctx, 60, 24)
		first <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream load did not start")
	}
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("initiating caller should return cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("initiating caller did not stop waiting")
	}
	second := make(chan error, 1)
	go func() {
		got, err := cache.Forecast(context.Background(), 60, 24)
		if err == nil && got.Latitude != 60 {
			err = errors.New("incorrect shared forecast")
		}
		second <- err
	}()
	unblock()
	select {
	case err := <-second:
		if err != nil {
			t.Fatalf("shared load was affected by initiator cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second caller did not receive the forecast")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one shared fetch, got %d", calls.Load())
	}
}

func TestCachedSharedLoadAndCanceledWaiter(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	cache := &Cached{
		TTL: time.Minute,
		Next: fakeProvider(func(ctx context.Context, lat, lon float64) (Forecast, error) {
			calls.Add(1)
			if lat == 60 {
				close(started)
				<-release
			}
			return Forecast{Latitude: lat}, nil
		}),
	}
	finished := make(chan error, 1)
	go func() {
		_, err := cache.Forecast(context.Background(), 60, 24)
		finished <- err
	}()
	defer close(release)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream load did not start")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cache.Forecast(ctx, 60, 24); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting caller should honor cancellation: %v", err)
	}
	// A different location must not wait for the blocked Helsinki load.
	other := make(chan error, 1)
	go func() {
		_, err := cache.Forecast(context.Background(), 61, 24)
		other <- err
	}()
	select {
	case err := <-other:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("different location was blocked")
	}
	if calls.Load() != 2 {
		t.Fatalf("shared location triggered a duplicate load: %d calls", calls.Load())
	}
	// Release exactly once, and wait for the successful shared load to finish.
	release <- struct{}{}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func TestCachedCoordinateRounding(t *testing.T) {
	calls := 0
	cache := &Cached{
		TTL: time.Minute,
		Next: fakeProvider(func(context.Context, float64, float64) (Forecast, error) {
			calls++
			return Forecast{}, nil
		}),
	}
	for _, lat := range []float64{60.1701, 60.1702} {
		if _, err := cache.Forecast(context.Background(), lat, 24.94); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("three-decimal rounding changed")
	}
}

func TestCachedConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	cache := &Cached{
		TTL: time.Minute,
		Next: fakeProvider(func(ctx context.Context, lat, lon float64) (Forecast, error) {
			calls.Add(1)
			return Forecast{Latitude: lat, Longitude: lon}, nil
		}),
	}
	var wg sync.WaitGroup
	for i := range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lat := float64(i % 4)
			for range 10 {
				got, err := cache.Forecast(context.Background(), lat, 24)
				if err != nil || got.Latitude != lat {
					t.Errorf("bad forecast: %+v, %v", got, err)
				}
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 4 {
		t.Fatalf("expected one upstream call per location, got %d", calls.Load())
	}
	// Warmed entries must remain cache hits.
	before := calls.Load()
	for i := range 4 {
		_, _ = cache.Forecast(context.Background(), float64(i), 24)
	}
	if calls.Load() != before {
		t.Fatal("warmed entries were not cached")
	}
}
