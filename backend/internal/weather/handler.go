package weather

import (
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
		writeErrorResponse(w, r, ErrInvalidCoordinates)
		return
	}
	forecast, err := h.provider.Forecast(r.Context(), lat, lon)
	if err != nil {
		writeErrorResponse(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, forecast)
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
