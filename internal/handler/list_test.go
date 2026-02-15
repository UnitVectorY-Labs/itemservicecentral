package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
)

func TestBuildListMetaAlwaysReturnsObject(t *testing.T) {
	meta := buildListMeta(&database.ListResult{})
	if meta.NextPageToken != "" || meta.PreviousPageToken != "" {
		t.Fatalf("expected empty list meta, got %#v", meta)
	}
}

func TestQueryIndexValidatesConfiguredPattern(t *testing.T) {
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
					"itemId": map[string]interface{}{"type": "string"},
					"status": map[string]interface{}{"type": "string"},
				},
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field:   "status",
						Pattern: "^allowed$",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/items/_index/by_status/blocked/_items", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
