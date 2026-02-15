package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

func TestSwaggerEndpointsEnabled(t *testing.T) {
	h, err := NewWithOptions(nil, []config.TableConfig{testSwaggerTable()}, Options{
		SwaggerEnabled: true,
		JWTEnabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	swaggerReq := httptest.NewRequest(http.MethodGet, "/v1/items/_swagger", nil)
	swaggerRec := httptest.NewRecorder()
	mux.ServeHTTP(swaggerRec, swaggerReq)

	if swaggerRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for swagger UI, got %d", swaggerRec.Code)
	}
	if ct := swaggerRec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	swaggerBody, err := io.ReadAll(swaggerRec.Result().Body)
	if err != nil {
		t.Fatalf("failed to read swagger body: %v", err)
	}
	if !strings.Contains(string(swaggerBody), "SwaggerUIBundle") {
		t.Fatalf("expected swagger UI HTML content")
	}

	openAPIReq := httptest.NewRequest(http.MethodGet, "/v1/items/_openapi", nil)
	openAPIRec := httptest.NewRecorder()
	mux.ServeHTTP(openAPIRec, openAPIReq)

	if openAPIRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for openapi, got %d", openAPIRec.Code)
	}
	if ct := openAPIRec.Header().Get("Content-Type"); !strings.Contains(ct, "application/yaml") {
		t.Fatalf("expected application/yaml content type, got %q", ct)
	}
	openAPIBody, err := io.ReadAll(openAPIRec.Result().Body)
	if err != nil {
		t.Fatalf("failed to read openapi body: %v", err)
	}
	bodyText := string(openAPIBody)
	if !strings.Contains(bodyText, "openapi: 3.0.3") {
		t.Fatalf("expected openapi version in response")
	}
	if !strings.Contains(bodyText, "bearerAuth") {
		t.Fatalf("expected bearer auth to be present when JWT is enabled")
	}
}

func TestSwaggerEndpointsDisabled(t *testing.T) {
	h, err := New(nil, []config.TableConfig{testSwaggerTable()})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/items/_swagger", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when swagger is disabled, got %d", rec.Code)
	}
}

func TestIsSwaggerRequestPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "/v1/items/_swagger", want: true},
		{path: "/v1/items/_openapi", want: true},
		{path: "/v1/items/data/abc/_item", want: false},
		{path: "/healthz", want: false},
	}

	for _, tc := range cases {
		got := IsSwaggerRequestPath(tc.path)
		if got != tc.want {
			t.Fatalf("path %q: expected %v, got %v", tc.path, tc.want, got)
		}
	}
}

func testSwaggerTable() config.TableConfig {
	return config.TableConfig{
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
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"itemId", "name"},
		},
	}
}
