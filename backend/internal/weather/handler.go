package weather

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// Handler translates HTTP requests and responses; calculations belong to the provider.
type Handler struct{ provider Provider }

func NewHandler(provider Provider) *Handler { return &Handler{provider: provider} }

func (h *Handler) Weather(w http.ResponseWriter, r *http.Request) {
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if latErr != nil || lonErr != nil || Validate(lat, lon) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid latitude or longitude")
		return
	}
	forecast, err := h.provider.Forecast(r.Context(), lat, lon)
	if err != nil {
		slog.ErrorContext(r.Context(), "weather forecast failed", "error", err)
		writeJSONError(w, http.StatusServiceUnavailable, "weather provider unavailable")
		return
	}
	writeJSON(w, http.StatusOK, forecast)
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{message})
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
