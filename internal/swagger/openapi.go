package swagger

import (
	"fmt"
	"strings"
	"sync"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"gopkg.in/yaml.v3"
)

// Provider lazily generates and caches per-table OpenAPI YAML documents.
type Provider struct {
	jwtEnabled bool
	tables     map[string]config.TableConfig

	mu    sync.RWMutex
	cache map[string][]byte
}

// NewProvider builds a Provider for all configured tables.
func NewProvider(tables []config.TableConfig, jwtEnabled bool) *Provider {
	tableMap := make(map[string]config.TableConfig, len(tables))
	for _, table := range tables {
		tableMap[table.Name] = table
	}
	return &Provider{
		jwtEnabled: jwtEnabled,
		tables:     tableMap,
		cache:      make(map[string][]byte, len(tables)),
	}
}

// YAML returns a table OpenAPI document as YAML.
func (p *Provider) YAML(tableName string) ([]byte, error) {
	p.mu.RLock()
	if doc, ok := p.cache[tableName]; ok {
		p.mu.RUnlock()
		return copyBytes(doc), nil
	}
	p.mu.RUnlock()

	table, ok := p.tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table %q is not configured", tableName)
	}

	doc, err := GenerateTableYAML(table, p.jwtEnabled)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if cached, ok := p.cache[tableName]; ok {
		p.mu.Unlock()
		return copyBytes(cached), nil
	}
	p.cache[tableName] = doc
	p.mu.Unlock()

	return copyBytes(doc), nil
}

// FindTable returns the table config with the provided name.
func FindTable(tables []config.TableConfig, name string) (config.TableConfig, bool) {
	for _, table := range tables {
		if table.Name == name {
			return table, true
		}
	}
	return config.TableConfig{}, false
}

// GenerateTableYAML creates an OpenAPI YAML document for a single table.
func GenerateTableYAML(table config.TableConfig, jwtEnabled bool) ([]byte, error) {
	doc := buildDocument(table, jwtEnabled)
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAPI document for table %q: %w", table.Name, err)
	}
	return out, nil
}

func buildDocument(table config.TableConfig, jwtEnabled bool) map[string]interface{} {
	paths := buildPaths(table, jwtEnabled)
	components := map[string]interface{}{
		"schemas": buildSchemas(table),
	}

	doc := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       fmt.Sprintf("itemservicecentral - %s", table.Name),
			"version":     "1.0.0",
			"description": fmt.Sprintf("Dynamic OpenAPI document for table %q.", table.Name),
		},
		"paths":      paths,
		"components": components,
	}

	if jwtEnabled {
		components["securitySchemes"] = map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
		}
		doc["security"] = []interface{}{
			map[string]interface{}{
				"bearerAuth": []interface{}{},
			},
		}
	}

	return doc
}

func buildSchemas(table config.TableConfig) map[string]interface{} {
	return map[string]interface{}{
		"Item": copyValue(table.Schema),
		"ListMeta": map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"nextPageToken": map[string]interface{}{"type": "string"},
				"previousPageToken": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"ListResponse": map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"items": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"$ref": "#/components/schemas/Item",
					},
				},
				"_meta": map[string]interface{}{
					"$ref": "#/components/schemas/ListMeta",
				},
			},
			"required": []interface{}{"items"},
		},
		"ErrorResponse": map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"error": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"error"},
		},
		"MergePatchDocument": map[string]interface{}{
			"type":                 "object",
			"additionalProperties": true,
			"description":          "RFC 7396 JSON Merge Patch document.",
		},
	}
}

func buildPaths(table config.TableConfig, jwtEnabled bool) map[string]interface{} {
	paths := map[string]interface{}{}
	hasRK := table.RangeKey != nil

	if hasRK {
		path := fmt.Sprintf("/v1/%s/data/{pk}/{rk}/_item", table.Name)
		paths[path] = map[string]interface{}{
			"get":    getItemOperation(table, true, jwtEnabled),
			"put":    putItemOperation(table, true, jwtEnabled),
			"patch":  patchItemOperation(table, true, jwtEnabled),
			"delete": deleteItemOperation(table, true, jwtEnabled),
		}
	} else {
		path := fmt.Sprintf("/v1/%s/data/{pk}/_item", table.Name)
		paths[path] = map[string]interface{}{
			"get":    getItemOperation(table, false, jwtEnabled),
			"put":    putItemOperation(table, false, jwtEnabled),
			"patch":  patchItemOperation(table, false, jwtEnabled),
			"delete": deleteItemOperation(table, false, jwtEnabled),
		}
	}

	partitionPath := fmt.Sprintf("/v1/%s/data/{pk}/_items", table.Name)
	paths[partitionPath] = map[string]interface{}{
		"get": listByPKOperation(table, hasRK, jwtEnabled),
	}

	if table.AllowTableScan {
		tableScanPath := fmt.Sprintf("/v1/%s/_items", table.Name)
		paths[tableScanPath] = map[string]interface{}{
			"get": scanTableOperation(table, hasRK, jwtEnabled),
		}
	}

	for _, idx := range table.Indexes {
		indexQueryPath := fmt.Sprintf("/v1/%s/_index/%s/{indexPk}/_items", table.Name, idx.Name)
		paths[indexQueryPath] = map[string]interface{}{
			"get": queryIndexOperation(table, idx, jwtEnabled),
		}

		if idx.AllowIndexScan {
			indexScanPath := fmt.Sprintf("/v1/%s/_index/%s/_items", table.Name, idx.Name)
			paths[indexScanPath] = map[string]interface{}{
				"get": scanIndexOperation(table, idx, jwtEnabled),
			}
		}

		if idx.RangeKey != nil {
			indexGetItemPath := fmt.Sprintf("/v1/%s/_index/%s/{indexPk}/{indexRk}/_item", table.Name, idx.Name)
			paths[indexGetItemPath] = map[string]interface{}{
				"get": getIndexItemOperation(table, idx, jwtEnabled),
			}
		}
	}

	return paths
}

func getItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("pk", "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, pathParam("rk", "Range key value.", table.RangeKey.Pattern))
	}
	params = append(params, fieldsQueryParam())

	return map[string]interface{}{
		"operationId": operationID(table.Name, "get", "item"),
		"summary":     "Get item",
		"parameters":  params,
		"responses":   getItemResponses(jwtEnabled),
	}
}

func putItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("pk", "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, pathParam("rk", "Range key value.", table.RangeKey.Pattern))
	}

	return map[string]interface{}{
		"operationId": operationID(table.Name, "put", "item"),
		"summary":     "Create or replace item",
		"parameters":  params,
		"requestBody": map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/Item",
					},
				},
			},
		},
		"responses": putItemResponses(jwtEnabled),
	}
}

func patchItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("pk", "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, pathParam("rk", "Range key value.", table.RangeKey.Pattern))
	}

	return map[string]interface{}{
		"operationId": operationID(table.Name, "patch", "item"),
		"summary":     "Patch item",
		"description": "Applies RFC 7396 JSON Merge Patch and validates the merged result.",
		"parameters":  params,
		"requestBody": map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/merge-patch+json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/MergePatchDocument",
					},
				},
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/MergePatchDocument",
					},
				},
			},
		},
		"responses": patchItemResponses(jwtEnabled),
	}
}

func deleteItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("pk", "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, pathParam("rk", "Range key value.", table.RangeKey.Pattern))
	}

	return map[string]interface{}{
		"operationId": operationID(table.Name, "delete", "item"),
		"summary":     "Delete item",
		"parameters":  params,
		"responses":   deleteItemResponses(jwtEnabled),
	}
}

func listByPKOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("pk", "Primary key value.", table.PrimaryKey.Pattern),
	}
	params = append(params, listQueryParams(hasRK)...)

	return map[string]interface{}{
		"operationId": operationID(table.Name, "list", "partition"),
		"summary":     "List items by primary key",
		"parameters":  params,
		"responses":   listResponses(jwtEnabled),
	}
}

func scanTableOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID(table.Name, "scan", "table"),
		"summary":     "Scan table",
		"parameters":  listQueryParams(hasRK),
		"responses":   listResponses(jwtEnabled),
	}
}

func queryIndexOperation(table config.TableConfig, idx config.IndexConfig, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("indexPk", fmt.Sprintf("Index %q primary key value.", idx.Name), idx.PrimaryKey.Pattern),
	}
	params = append(params, listQueryParams(idx.RangeKey != nil)...)

	return map[string]interface{}{
		"operationId": operationID(table.Name, "query", "index", idx.Name),
		"summary":     fmt.Sprintf("Query index %q", idx.Name),
		"parameters":  params,
		"responses":   listResponses(jwtEnabled),
	}
}

func scanIndexOperation(table config.TableConfig, idx config.IndexConfig, jwtEnabled bool) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID(table.Name, "scan", "index", idx.Name),
		"summary":     fmt.Sprintf("Scan index %q", idx.Name),
		"parameters":  listQueryParams(idx.RangeKey != nil),
		"responses":   listResponses(jwtEnabled),
	}
}

func getIndexItemOperation(table config.TableConfig, idx config.IndexConfig, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		pathParam("indexPk", fmt.Sprintf("Index %q primary key value.", idx.Name), idx.PrimaryKey.Pattern),
		pathParam("indexRk", fmt.Sprintf("Index %q range key value.", idx.Name), idx.RangeKey.Pattern),
		fieldsQueryParam(),
	}

	return map[string]interface{}{
		"operationId": operationID(table.Name, "get", "index", idx.Name, "item"),
		"summary":     fmt.Sprintf("Get item by index %q", idx.Name),
		"parameters":  params,
		"responses":   getItemResponses(jwtEnabled),
	}
}

func getItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Item found.", map[string]interface{}{"$ref": "#/components/schemas/Item"}),
		"400": jsonErrorResponse("Invalid request."),
		"404": jsonErrorResponse("Item not found."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func putItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Item stored.", map[string]interface{}{"$ref": "#/components/schemas/Item"}),
		"400": jsonErrorResponse("Invalid request body or key mismatch."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func patchItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Item patched.", map[string]interface{}{"$ref": "#/components/schemas/Item"}),
		"400": jsonErrorResponse("Invalid patch or validation failure."),
		"404": jsonErrorResponse("Item not found."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func deleteItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"204": map[string]interface{}{"description": "Item deleted."},
		"400": jsonErrorResponse("Invalid request."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func listResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Page of items.", map[string]interface{}{"$ref": "#/components/schemas/ListResponse"}),
		"400": jsonErrorResponse("Invalid request."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func withAuthError(jwtEnabled bool, responses map[string]interface{}) map[string]interface{} {
	if !jwtEnabled {
		return responses
	}
	withAuth := make(map[string]interface{}, len(responses)+1)
	for code, response := range responses {
		withAuth[code] = response
	}
	withAuth["401"] = jsonErrorResponse("Missing or invalid token.")
	return withAuth
}

func jsonResponse(description string, schema interface{}) map[string]interface{} {
	return map[string]interface{}{
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": schema,
			},
		},
	}
}

func jsonErrorResponse(description string) map[string]interface{} {
	return jsonResponse(description, map[string]interface{}{
		"$ref": "#/components/schemas/ErrorResponse",
	})
}

func listQueryParams(hasRK bool) []interface{} {
	params := []interface{}{
		queryParam("limit", "Max items per page.", map[string]interface{}{
			"type":    "integer",
			"minimum": 1,
			"default": 50,
		}),
		queryParam("pageToken", "Opaque pagination token from a previous response.", map[string]interface{}{
			"type": "string",
		}),
		fieldsQueryParam(),
	}

	if hasRK {
		params = append(params,
			queryParam("rkBeginsWith", "Filter where range key begins with the provided prefix.", map[string]interface{}{"type": "string"}),
			queryParam("rkGt", "Filter where range key is greater than this value.", map[string]interface{}{"type": "string"}),
			queryParam("rkGte", "Filter where range key is greater than or equal to this value.", map[string]interface{}{"type": "string"}),
			queryParam("rkLt", "Filter where range key is less than this value.", map[string]interface{}{"type": "string"}),
			queryParam("rkLte", "Filter where range key is less than or equal to this value.", map[string]interface{}{"type": "string"}),
		)
	}

	return params
}

func fieldsQueryParam() map[string]interface{} {
	return queryParam("fields", "Comma-separated field names to project in the response.", map[string]interface{}{
		"type": "string",
	})
}

func pathParam(name, description, pattern string) map[string]interface{} {
	schema := map[string]interface{}{
		"type": "string",
	}
	if pattern != "" {
		schema["pattern"] = pattern
	}
	return map[string]interface{}{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": description,
		"schema":      schema,
	}
}

func queryParam(name, description string, schema map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"in":          "query",
		"required":    false,
		"description": description,
		"schema":      schema,
	}
}

func operationID(parts ...string) string {
	raw := strings.Join(parts, "_")
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c >= 'a' && c <= 'z':
			b.WriteByte(c)
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + ('a' - 'A'))
		case c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == '_' || c == '-':
			b.WriteByte('_')
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func copyValue(v interface{}) interface{} {
	switch tv := v.(type) {
	case map[string]interface{}:
		copied := make(map[string]interface{}, len(tv))
		for key, val := range tv {
			copied[key] = copyValue(val)
		}
		return copied
	case map[interface{}]interface{}:
		copied := make(map[string]interface{}, len(tv))
		for key, val := range tv {
			copied[fmt.Sprintf("%v", key)] = copyValue(val)
		}
		return copied
	case []interface{}:
		copied := make([]interface{}, len(tv))
		for i, val := range tv {
			copied[i] = copyValue(val)
		}
		return copied
	default:
		return tv
	}
}

func copyBytes(in []byte) []byte {
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
