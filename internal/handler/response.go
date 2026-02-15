package handler

import (
	"encoding/json"
	"net/http"
)

const (
	typeItem  = "item"
	typeItems = "items"
	typeError = "error"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"_type": typeError,
		"error": message,
	})
}

func itemPayload(data map[string]interface{}) map[string]interface{} {
	payload := make(map[string]interface{}, len(data)+1)
	for k, v := range data {
		payload[k] = v
	}
	payload["_type"] = typeItem
	return payload
}
