package handler

import (
	"net/http"
	"strings"

	swaggerdoc "github.com/UnitVectorY-Labs/itemservicecentral/internal/swagger"
)

// IsSwaggerRequestPath returns true when the request path targets the Swagger UI or OpenAPI endpoints.
func IsSwaggerRequestPath(path string) bool {
	if !strings.HasPrefix(path, "/v1/") {
		return false
	}
	return strings.HasSuffix(path, "/_swagger") || strings.HasSuffix(path, "/_openapi")
}

func (h *Handler) handleSwaggerUI() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(swaggerdoc.HTML())
	}
}

func (h *Handler) handleOpenAPI(tableName string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if h.openAPIDoc == nil {
			writeError(w, http.StatusNotFound, "swagger is disabled")
			return
		}

		doc, err := h.openAPIDoc.YAML(tableName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate OpenAPI")
			return
		}

		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc)
	}
}
