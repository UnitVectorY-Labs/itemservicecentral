package model

import "strings"

// StripKeys removes the pk and rk field names from data for storage.
// Returns a new map without the specified keys.
func StripKeys(data map[string]interface{}, pkField string, rkField string) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		if k == pkField {
			continue
		}
		if rkField != "" && k == rkField {
			continue
		}
		result[k] = v
	}
	return result
}

// InjectKeys adds pk and rk values back into the data with the configured field names.
// Returns a new map with keys added.
func InjectKeys(data map[string]interface{}, pkField string, pkValue string, rkField string, rkValue string) map[string]interface{} {
	result := make(map[string]interface{}, len(data)+2)
	for k, v := range data {
		result[k] = v
	}
	result[pkField] = pkValue
	if rkField != "" {
		result[rkField] = rkValue
	}
	return result
}

// MergePatch applies RFC 7396 JSON Merge Patch to a target document.
// For each key in patch:
//   - if the patch value is null, remove the key from target
//   - if the patch value is an object and target has an object for same key, recurse
//   - otherwise, set the key in target to the patch value
//
// Returns the merged document (modifies target in place).
func MergePatch(target, patch map[string]interface{}) map[string]interface{} {
	for k, patchVal := range patch {
		if patchVal == nil {
			delete(target, k)
			continue
		}
		if patchObj, ok := patchVal.(map[string]interface{}); ok {
			if targetObj, ok := target[k].(map[string]interface{}); ok {
				target[k] = MergePatch(targetObj, patchObj)
				continue
			}
		}
		target[k] = patchVal
	}
	return target
}

// ProjectFields returns a new map containing only the specified fields.
// Always includes pkField and rkField (if non-empty).
func ProjectFields(data map[string]interface{}, fields []string, pkField string, rkField string) map[string]interface{} {
	allowed := make(map[string]bool, len(fields)+2)
	for _, f := range fields {
		allowed[f] = true
	}
	allowed[pkField] = true
	if rkField != "" {
		allowed[rkField] = true
	}

	result := make(map[string]interface{})
	for k, v := range data {
		if allowed[k] {
			result[k] = v
		}
	}
	return result
}

// ParseFieldsParam parses a comma-separated fields string into a slice.
func ParseFieldsParam(fieldsParam string) []string {
	if fieldsParam == "" {
		return nil
	}
	parts := strings.Split(fieldsParam, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
