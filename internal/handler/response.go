package handler

import (
	"encoding/json"
	"maps"
	"net/http"
)

const (
	typeItem  = "item"
	typeItems = "items"
	typeError = "error"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"_type":  typeError,
		"_error": message,
	})
}

func itemPayload(data map[string]any) map[string]any {
	payload := make(map[string]any, len(data)+1)
	maps.Copy(payload, data)
	payload["_type"] = typeItem
	return payload
}
