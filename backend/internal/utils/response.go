package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

// envelope is the single response shape for the whole API. Success and failure
// use the same keys so a client never has to branch on the shape itself.
type envelope struct {
	Data    interface{} `json:"data"`
	Error   *apiError   `json:"error"`
	Message string      `json:"message"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Stable machine-readable error codes. Clients branch on these, not on prose.
const (
	CodeBadRequest      = "bad_request"
	CodeValidation      = "validation_failed"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodePayloadTooLarge = "payload_too_large"
	CodeRateLimited     = "rate_limited"
	CodeInternal        = "internal_error"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}, message string) {
	write(w, status, envelope{Data: data, Error: nil, Message: message})
}

// WriteError emits the same envelope with data nulled out. The message is
// always a caller-supplied string — never a raw database or driver error.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	write(w, status, envelope{
		Data:    nil,
		Error:   &apiError{Code: code, Message: message},
		Message: message,
	})
}

func write(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The status line is already on the wire; all we can do is record it.
		log.Printf("[HTTP] failed to encode response body: %v", err)
	}
}
