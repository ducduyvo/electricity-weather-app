package weatherhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"weather-retailer/backend/internal/weather"
)

// publicError translates domain categories at the HTTP boundary. Never expose
// err.Error(): it may contain upstream bodies, URLs or other internal details.
func publicError(err error) (int, string) {
	switch {
	case errors.Is(err, weather.ErrInvalidCoordinates):
		return http.StatusBadRequest, "invalid latitude or longitude"
	case errors.Is(err, weather.ErrInvalidLocation):
		return http.StatusUnprocessableEntity, "forecast unavailable for this location"
	case errors.Is(err, weather.ErrProviderResponse):
		return http.StatusBadGateway, "invalid response from weather provider"
	case errors.Is(err, weather.ErrProviderUnavailable):
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
