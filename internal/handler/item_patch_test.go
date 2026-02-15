package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

func TestPatchRequiresPrimaryKeyInBody(t *testing.T) {
	h, err := New(nil, []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field:   "itemId",
				Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
			},
			Schema: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"itemId": map[string]interface{}{
						"type":    "string",
						"pattern": "^[A-Za-z_][A-Za-z0-9._-]*$",
					},
					"name": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"itemId"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/v1/items/data/a1/_item", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPatchRequiresRangeKeyInBody(t *testing.T) {
	h, err := New(nil, []config.TableConfig{
		{
			Name: "orders",
			PrimaryKey: config.KeyConfig{
				Field:   "orderId",
				Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
			},
			RangeKey: &config.KeyConfig{
				Field:   "lineId",
				Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
			},
			Schema: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"orderId": map[string]interface{}{
						"type":    "string",
						"pattern": "^[A-Za-z_][A-Za-z0-9._-]*$",
					},
					"lineId": map[string]interface{}{
						"type":    "string",
						"pattern": "^[A-Za-z_][A-Za-z0-9._-]*$",
					},
				},
				"required": []interface{}{"orderId", "lineId"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/v1/orders/data/order1/line1/_item", bytes.NewReader([]byte(`{"orderId":"order1"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
