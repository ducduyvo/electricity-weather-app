package weather

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

// publicError translates domain categories at the HTTP boundary. Never expose
// err.Error(): it may contain upstream bodies, URLs or other internal details.
func publicError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrInvalidCoordinates):
		return http.StatusBadRequest, "invalid latitude or longitude"
	case errors.Is(err, ErrInvalidLocation):
		return http.StatusUnprocessableEntity, "forecast unavailable for this location"
	case errors.Is(err, ErrProviderResponse):
		return http.StatusBadGateway, "invalid response from weather provider"
	case errors.Is(err, ErrProviderUnavailable):
		return http.StatusServiceUnavailable, "weather provider unavailable"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func writeErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	status, message := publicError(err)
	if status >= 500 {
		slog.ErrorContext(r.Context(), "weather request failed", "error", err)
	}
	writeJSONError(w, status, message)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	// Encode before writing headers so encoding failures can still return a 500.
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error("encode JSON response", "error", err)
		status = http.StatusInternalServerError
		body = []byte(`{"error":"internal server error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}
