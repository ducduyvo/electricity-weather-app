package weather

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	// ErrInvalidLocation means valid coordinates for which a provider cannot serve a forecast.
	ErrInvalidLocation       = errors.New("unsupported forecast location")
	ErrProviderUnavailable   = errors.New("weather provider unavailable")
	ErrProviderResponse      = errors.New("invalid weather provider response")
	ErrProviderConfiguration = errors.New("invalid weather provider configuration")
)

// ProviderError preserves a machine-readable category and the original cause.
// StatusCode is the upstream HTTP status, never a client response decision.
type ProviderError struct {
	Operation  string
	Kind       error
	Cause      error
	StatusCode int
}

func (e *ProviderError) Error() string {
	message := e.Operation + ": " + e.Kind.Error()
	if e.StatusCode != 0 {
		message += fmt.Sprintf(" (weather provider status %d)", e.StatusCode)
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *ProviderError) Unwrap() []error {
	if e.Cause == nil {
		return []error{e.Kind}
	}
	return []error{e.Kind, e.Cause}
}
