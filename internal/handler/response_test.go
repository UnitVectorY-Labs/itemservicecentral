package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorIncludesType(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "bad input")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["_type"] != typeError {
		t.Fatalf("expected _type=%q, got %v", typeError, body["_type"])
	}
	if body["error"] != "bad input" {
		t.Fatalf("expected error message, got %v", body["error"])
	}
}

func TestItemPayloadIncludesType(t *testing.T) {
	payload := itemPayload(map[string]interface{}{
		"itemId": "a1",
		"name":   "test",
	})

	if payload["_type"] != typeItem {
		t.Fatalf("expected _type=%q, got %v", typeItem, payload["_type"])
	}
	if payload["itemId"] != "a1" {
		t.Fatalf("expected itemId=a1, got %v", payload["itemId"])
	}
}
