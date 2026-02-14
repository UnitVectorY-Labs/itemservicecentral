package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/model"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/validate"
)

// handleGetItem handles GET for a single item.
func (h *Handler) handleGetItem(th *tableHandler) http.HandlerFunc {
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

		var rkPtr *string
		rkField := ""
		rkValue := ""
		if th.config.RangeKey != nil {
			rk := r.PathValue("rk")
			if err := validate.ValidateKeyValue(rk); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := validate.ValidateKeyPattern(rk, th.config.RangeKey.Pattern); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			rkPtr = &rk
			rkField = th.config.RangeKey.Field
			rkValue = rk
		}

		data, err := h.store.GetItem(r.Context(), th.config.Name, pk, rkPtr)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get item")
			return
		}
		if data == nil {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}

		data = model.InjectKeys(data, th.config.PrimaryKey.Field, pk, rkField, rkValue)
		data = applyProjection(r, data, th)

		writeJSON(w, http.StatusOK, data)
	}
}

// handlePutItem handles PUT for a single item.
func (h *Handler) handlePutItem(th *tableHandler) http.HandlerFunc {
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

		var rkPtr *string
		rkField := ""
		rkValue := ""
		if th.config.RangeKey != nil {
			rk := r.PathValue("rk")
			if err := validate.ValidateKeyValue(rk); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := validate.ValidateKeyPattern(rk, th.config.RangeKey.Pattern); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			rkPtr = &rk
			rkField = th.config.RangeKey.Field
			rkValue = rk
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read request body")
			return
		}

		var doc map[string]interface{}
		if err := json.Unmarshal(body, &doc); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if err := validate.ValidateJSONKeys(doc); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Verify pk in body matches URL
		if bodyPK, ok := doc[th.config.PrimaryKey.Field]; ok {
			if s, ok := bodyPK.(string); !ok || s != pk {
				writeError(w, http.StatusBadRequest, "pk in body does not match URL")
				return
			}
		}

		// Verify rk in body matches URL
		if th.config.RangeKey != nil {
			if bodyRK, ok := doc[th.config.RangeKey.Field]; ok {
				if s, ok := bodyRK.(string); !ok || s != rkValue {
					writeError(w, http.StatusBadRequest, "rk in body does not match URL")
					return
				}
			}
		}

		if err := th.validator.Validate(doc); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		stripped := model.StripKeys(doc, th.config.PrimaryKey.Field, rkField)
		if err := h.store.PutItem(r.Context(), th.config.Name, pk, rkPtr, stripped); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to put item")
			return
		}

		result := model.InjectKeys(stripped, th.config.PrimaryKey.Field, pk, rkField, rkValue)
		writeJSON(w, http.StatusOK, result)
	}
}

// handlePatchItem handles PATCH (JSON Merge Patch) for a single item.
func (h *Handler) handlePatchItem(th *tableHandler) http.HandlerFunc {
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

		var rkPtr *string
		rkField := ""
		rkValue := ""
		if th.config.RangeKey != nil {
			rk := r.PathValue("rk")
			if err := validate.ValidateKeyValue(rk); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := validate.ValidateKeyPattern(rk, th.config.RangeKey.Pattern); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			rkPtr = &rk
			rkField = th.config.RangeKey.Field
			rkValue = rk
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read request body")
			return
		}

		var patch map[string]interface{}
		if err := json.Unmarshal(body, &patch); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if err := validate.ValidateJSONKeys(patch); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Verify pk in patch matches URL
		if bodyPK, ok := patch[th.config.PrimaryKey.Field]; ok {
			if s, ok := bodyPK.(string); !ok || s != pk {
				writeError(w, http.StatusBadRequest, "pk in body does not match URL")
				return
			}
		}

		// Verify rk in patch matches URL
		if th.config.RangeKey != nil {
			if bodyRK, ok := patch[th.config.RangeKey.Field]; ok {
				if s, ok := bodyRK.(string); !ok || s != rkValue {
					writeError(w, http.StatusBadRequest, "rk in body does not match URL")
					return
				}
			}
		}

		existing, err := h.store.GetItem(r.Context(), th.config.Name, pk, rkPtr)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get item")
			return
		}
		if existing == nil {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}

		merged := model.MergePatch(existing, patch)
		mergedWithKeys := model.InjectKeys(merged, th.config.PrimaryKey.Field, pk, rkField, rkValue)

		if err := th.validator.Validate(mergedWithKeys); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		stripped := model.StripKeys(mergedWithKeys, th.config.PrimaryKey.Field, rkField)
		if err := h.store.PutItem(r.Context(), th.config.Name, pk, rkPtr, stripped); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to put item")
			return
		}

		writeJSON(w, http.StatusOK, mergedWithKeys)
	}
}

// handleDeleteItem handles DELETE for a single item.
func (h *Handler) handleDeleteItem(th *tableHandler) http.HandlerFunc {
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

		var rkPtr *string
		if th.config.RangeKey != nil {
			rk := r.PathValue("rk")
			if err := validate.ValidateKeyValue(rk); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := validate.ValidateKeyPattern(rk, th.config.RangeKey.Pattern); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			rkPtr = &rk
		}

		if err := h.store.DeleteItem(r.Context(), th.config.Name, pk, rkPtr); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete item")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// applyProjection applies field projection based on the fields query parameter.
func applyProjection(r *http.Request, data map[string]interface{}, th *tableHandler) map[string]interface{} {
	fieldsParam := r.URL.Query().Get("fields")
	fields := model.ParseFieldsParam(fieldsParam)

	if len(fields) == 0 {
		return data
	}

	rkField := ""
	if th.config.RangeKey != nil {
		rkField = th.config.RangeKey.Field
	}
	return model.ProjectFields(data, fields, th.config.PrimaryKey.Field, rkField)
}
