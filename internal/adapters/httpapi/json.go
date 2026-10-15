package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	// The status line is already sent, so an encoding failure cannot change the response.
	_ = json.NewEncoder(w).Encode(body)
}

// decodeJSON reads exactly one JSON object and rejects unknown fields so client typos surface early.
func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return requestError{code: "invalid_json", message: "request body must be a valid JSON object"}
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return requestError{code: "invalid_json", message: "request body must contain a single JSON object"}
	}
	return nil
}
