package handler

import (
	"net/http"
	"strconv"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/model"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/validate"
)

// listResponse is the JSON envelope for list endpoints.
type listResponse struct {
	Items      []map[string]interface{} `json:"items"`
	NextCursor string                   `json:"nextCursor,omitempty"`
}

// handleListItems handles GET /v1/{table}/data/{pk}/_items.
func (h *Handler) handleListItems(th *tableHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pk := r.PathValue("pk")
		if err := validate.ValidateKeyValue(pk); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validate.ValidateKeyPattern(pk, th.config.PrimaryKey.Pattern); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		opts := parseListOptions(r)
		hasRK := th.config.RangeKey != nil

		result, err := h.store.ListItems(r.Context(), th.config.Name, pk, hasRK, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list items")
			return
		}

		rkField := ""
		if th.config.RangeKey != nil {
			rkField = th.config.RangeKey.Field
		}

		items := projectItems(r, result.Items, th, th.config.PrimaryKey.Field, rkField)
		writeJSON(w, http.StatusOK, listResponse{
			Items:      items,
			NextCursor: result.NextCursor,
		})
	}
}

// handleScanTable handles GET /v1/{table}/_items.
func (h *Handler) handleScanTable(th *tableHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := parseListOptions(r)
		hasRK := th.config.RangeKey != nil

		result, err := h.store.ScanTable(r.Context(), th.config.Name, hasRK, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan table")
			return
		}

		rkField := ""
		if th.config.RangeKey != nil {
			rkField = th.config.RangeKey.Field
		}

		items := projectItems(r, result.Items, th, th.config.PrimaryKey.Field, rkField)
		writeJSON(w, http.StatusOK, listResponse{
			Items:      items,
			NextCursor: result.NextCursor,
		})
	}
}

// handleQueryIndex handles GET /v1/{table}/_index/{indexName}/{indexPk}/_items.
func (h *Handler) handleQueryIndex(th *tableHandler, idx config.IndexConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indexPk := r.PathValue("indexPk")
		if err := validate.ValidateKeyValue(indexPk); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		opts := parseListOptions(r)
		iqc := database.IndexQueryConfig{
			PKField: idx.PrimaryKey.Field,
		}
		if idx.RangeKey != nil {
			iqc.RKField = idx.RangeKey.Field
		}

		result, err := h.store.QueryIndex(r.Context(), th.config.Name, iqc, indexPk, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query index")
			return
		}

		rkField := ""
		if th.config.RangeKey != nil {
			rkField = th.config.RangeKey.Field
		}

		items := projectIndexItems(r, result.Items, th, th.config.PrimaryKey.Field, rkField, idx)
		writeJSON(w, http.StatusOK, listResponse{
			Items:      items,
			NextCursor: result.NextCursor,
		})
	}
}

// handleScanIndex handles GET /v1/{table}/_index/{indexName}/_items.
func (h *Handler) handleScanIndex(th *tableHandler, idx config.IndexConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := parseListOptions(r)
		iqc := database.IndexQueryConfig{
			PKField: idx.PrimaryKey.Field,
		}
		if idx.RangeKey != nil {
			iqc.RKField = idx.RangeKey.Field
		}

		result, err := h.store.ScanIndex(r.Context(), th.config.Name, iqc, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan index")
			return
		}

		rkField := ""
		if th.config.RangeKey != nil {
			rkField = th.config.RangeKey.Field
		}

		items := projectIndexItems(r, result.Items, th, th.config.PrimaryKey.Field, rkField, idx)
		writeJSON(w, http.StatusOK, listResponse{
			Items:      items,
			NextCursor: result.NextCursor,
		})
	}
}

// handleGetIndexItem handles GET /v1/{table}/_index/{indexName}/{indexPk}/{indexRk}/_item.
func (h *Handler) handleGetIndexItem(th *tableHandler, idx config.IndexConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indexPk := r.PathValue("indexPk")
		if err := validate.ValidateKeyValue(indexPk); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		indexRk := r.PathValue("indexRk")
		if err := validate.ValidateKeyValue(indexRk); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		iqc := database.IndexQueryConfig{
			PKField: idx.PrimaryKey.Field,
			RKField: idx.RangeKey.Field,
		}

		result, err := h.store.GetItemByIndex(r.Context(), th.config.Name, iqc, indexPk, indexRk)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get item by index")
			return
		}
		if result == nil {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}

		rkField := ""
		rkValue := ""
		if th.config.RangeKey != nil {
			rkField = th.config.RangeKey.Field
			rkValue = result.RK
		}

		data := model.InjectKeys(result.Data, th.config.PrimaryKey.Field, result.PK, rkField, rkValue)
		data = applyIndexProjection(r, data, th, idx)

		writeJSON(w, http.StatusOK, data)
	}
}

// parseListOptions extracts pagination and filter params from the request.
func parseListOptions(r *http.Request) database.ListOptions {
	q := r.URL.Query()
	opts := database.ListOptions{
		Cursor:       q.Get("cursor"),
		RKBeginsWith: q.Get("rkBeginsWith"),
		RKGt:         q.Get("rkGt"),
		RKGte:        q.Get("rkGte"),
		RKLt:         q.Get("rkLt"),
		RKLte:        q.Get("rkLte"),
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	return opts
}

// projectItems injects keys and applies projection to each item in the list.
func projectItems(r *http.Request, items []database.ItemResult, th *tableHandler, pkField, rkField string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		rkValue := ""
		if rkField != "" {
			rkValue = item.RK
		}
		data := model.InjectKeys(item.Data, pkField, item.PK, rkField, rkValue)
		result[i] = applyProjection(r, data, th)
	}
	return result
}

// projectIndexItems injects keys and applies index projection then field projection.
func projectIndexItems(r *http.Request, items []database.ItemResult, th *tableHandler, pkField, rkField string, idx config.IndexConfig) []map[string]interface{} {
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		rkValue := ""
		if rkField != "" {
			rkValue = item.RK
		}
		data := model.InjectKeys(item.Data, pkField, item.PK, rkField, rkValue)
		result[i] = applyIndexProjection(r, data, th, idx)
	}
	return result
}

// applyIndexProjection applies index projection first, then field-level projection.
func applyIndexProjection(r *http.Request, data map[string]interface{}, th *tableHandler, idx config.IndexConfig) map[string]interface{} {
	rkField := ""
	if th.config.RangeKey != nil {
		rkField = th.config.RangeKey.Field
	}

	// Apply index projection if configured
	if idx.Projection != nil {
		switch idx.Projection.Type {
		case "KEYS_ONLY":
			keyFields := []string{idx.PrimaryKey.Field}
			if idx.RangeKey != nil {
				keyFields = append(keyFields, idx.RangeKey.Field)
			}
			data = model.ProjectFields(data, keyFields, th.config.PrimaryKey.Field, rkField)
		case "INCLUDE":
			fields := make([]string, 0, len(idx.Projection.NonKeyAttributes)+2)
			fields = append(fields, idx.PrimaryKey.Field)
			if idx.RangeKey != nil {
				fields = append(fields, idx.RangeKey.Field)
			}
			fields = append(fields, idx.Projection.NonKeyAttributes...)
			data = model.ProjectFields(data, fields, th.config.PrimaryKey.Field, rkField)
		}
		// "ALL" or nil: return all fields
	}

	// Then apply request-level projection
	return applyProjection(r, data, th)
}
