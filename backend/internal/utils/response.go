package utils

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}, message string) {
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":    data,
		"error":   nil,
		"message": message,
	})
}
func WriteError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":    nil,
		"error":   message,
		"message": "",
	})
}
