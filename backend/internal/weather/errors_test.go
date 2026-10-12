package weather

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderErrorPreservesCategoryAndCause(t *testing.T) {
	err := &ProviderError{Operation: "fetch", Kind: ErrProviderUnavailable, Cause: context.Canceled}
	if !errors.Is(err, ErrProviderUnavailable) || !errors.Is(err, context.Canceled) {
		t.Fatal("lost error category or cause")
	}
	var typed *ProviderError
	if !errors.As(err, &typed) {
		t.Fatal("lost typed error")
	}
}

func TestOpenMeteoErrorCategories(t *testing.T) {
	for _, tt := range []struct {
		status int
		body   string
		kind   error
	}{
		{429, "", ErrProviderUnavailable}, {503, "", ErrProviderUnavailable}, {400, "", ErrProviderResponse}, {200, "{", ErrProviderResponse},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
			_, _ = w.Write([]byte(tt.body))
		}))
		_, err := (OpenMeteo{Client: server.Client(), BaseURL: server.URL}).Forecast(context.Background(), 0, 0)
		server.Close()
		if !errors.Is(err, tt.kind) {
			t.Fatalf("status %d: %v, want %v", tt.status, err, tt.kind)
		}
	}
	_, err := (OpenMeteo{BaseURL: "http://%zz"}).Forecast(context.Background(), 0, 0)
	if !errors.Is(err, ErrProviderConfiguration) {
		t.Fatal("unclassified configuration error")
	}
	_, err = (OpenMeteo{}).Forecast(context.Background(), 91, 0)
	if !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatal("invalid coordinates reached provider")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := server.Client()
	base := server.URL
	server.Close()
	_, err = (OpenMeteo{Client: client, BaseURL: base}).Forecast(context.Background(), 0, 0)
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal("unclassified transport failure")
	}
}
