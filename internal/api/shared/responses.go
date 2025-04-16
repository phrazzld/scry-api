package shared

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse defines the standard error response structure.
type ErrorResponse struct {
	Error string `json:"error"`
}

// RespondWithJSON writes a JSON response with the given status code and data.
func RespondWithJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// RespondWithError writes a JSON error response with the given status code and message.
func RespondWithError(w http.ResponseWriter, r *http.Request, status int, message string) {
	RespondWithJSON(w, r, status, ErrorResponse{Error: message})
}
