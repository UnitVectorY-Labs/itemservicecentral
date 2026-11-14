package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

func TestItemRequestsRejectAmbiguousJSON(t *testing.T) {
	h, err := New(nil, []config.TableConfig{{
		Name:       "items",
		PrimaryKey: config.KeyConfig{Field: "itemId"},
		Schema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{"itemId": map[string]any{"type": "string"}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.SetupRoutes(mux)
	for _, method := range []string{http.MethodPut, http.MethodPatch} {
		for name, body := range map[string]string{
			"duplicate key":        `{"itemId":"a1","itemId":"a1"}`,
			"nested duplicate key": `{"itemId":"a1","nested":{"x":1,"x":2}}`,
			"invalid UTF-8":        "{\"itemId\":\"a1\",\"name\":\"\xff\"}",
		} {
			t.Run(method+"/"+name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest(method, "/v1/items/data/a1/_item", strings.NewReader(body)))
				if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid JSON body") {
					t.Fatalf("got status %d and body %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestEmptyListResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, listResponse{Type: typeItems, Items: []map[string]any{}})
	if got := rec.Body.String(); got != "{\"_type\":\"items\",\"items\":[],\"_meta\":{}}\n" {
		t.Fatalf("unexpected empty list response: %s", got)
	}
}
