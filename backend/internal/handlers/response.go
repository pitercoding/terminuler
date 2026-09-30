package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse is the JSON body returned for every error, so clients can
// always read the message from the "error" field.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON writes body as JSON with the given status code. Headers are
// already sent when encoding fails, so the error can only be logged.
func writeJSON(
	w http.ResponseWriter,
	status int,
	body any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// WriteError exposes writeError to middlewares outside this package, such
// as the admin authorization, so every error keeps the same JSON format.
func WriteError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeError(w, status, message)
}

// writeError writes an error response in the format {"error": message}.
func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}
