package swagger

import (
	"fmt"
	"strings"
	"sync"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	keyValuePathPattern = `^[A-Za-z_][A-Za-z0-9._-]*$`
	itemTypeName        = "item"
	itemsTypeName       = "items"
	errorTypeName       = "error"
)

type orderedEntry struct {
	Key   string
	Value interface{}
}

type orderedMap []orderedEntry

func (m orderedMap) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	for _, entry := range m {
		key := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: entry.Key,
		}
		value := &yaml.Node{}
		if err := value.Encode(entry.Value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, key, value)
	}
	return node, nil
}

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

func buildDocument(table config.TableConfig, jwtEnabled bool) orderedMap {
	paths := buildPaths(table, jwtEnabled)
	components := orderedMap{
		{Key: "schemas", Value: buildSchemas(table)},
	}

	doc := orderedMap{
		{Key: "openapi", Value: "3.0.3"},
		{Key: "info", Value: orderedMap{
			{Key: "title", Value: fmt.Sprintf("itemservicecentral - %s", table.Name)},
			{Key: "version", Value: "1.0.0"},
			{Key: "description", Value: fmt.Sprintf("Dynamic OpenAPI document for table %q.", table.Name)},
		}},
		{Key: "paths", Value: paths},
		{Key: "components", Value: components},
	}

	if jwtEnabled {
		components = append(components, orderedEntry{
			Key: "securitySchemes",
			Value: orderedMap{
				{
					Key: "bearerAuth",
					Value: orderedMap{
						{Key: "type", Value: "http"},
						{Key: "scheme", Value: "bearer"},
						{Key: "bearerFormat", Value: "JWT"},
					},
				},
			},
		})
		doc[3].Value = components
		doc = append(doc, orderedEntry{
			Key: "security",
			Value: []interface{}{
				orderedMap{
					{Key: "bearerAuth", Value: []interface{}{}},
				},
			},
		})
	}

	return doc
}

func buildSchemas(table config.TableConfig) orderedMap {
	return orderedMap{
		{Key: "Item", Value: copyValue(table.Schema)},
		{Key: "ItemResponse", Value: buildItemResponseSchema(table)},
		{Key: "PatchItem", Value: buildPatchSchema(table)},
		{Key: "ListMeta", Value: orderedMap{
			{Key: "type", Value: "object"},
			{Key: "additionalProperties", Value: false},
			{Key: "properties", Value: orderedMap{
				{Key: "nextPageToken", Value: orderedMap{{Key: "type", Value: "string"}}},
				{Key: "previousPageToken", Value: orderedMap{{Key: "type", Value: "string"}}},
			}},
		}},
		{Key: "ListResponse", Value: orderedMap{
			{Key: "type", Value: "object"},
			{Key: "additionalProperties", Value: false},
			{Key: "properties", Value: orderedMap{
				{Key: "_type", Value: orderedMap{
					{Key: "type", Value: "string"},
					{Key: "enum", Value: []interface{}{itemsTypeName}},
				}},
				{Key: "items", Value: orderedMap{
					{Key: "type", Value: "array"},
					{Key: "items", Value: orderedMap{
						{Key: "$ref", Value: "#/components/schemas/ItemResponse"},
					}},
				}},
				{Key: "_meta", Value: orderedMap{
					{Key: "$ref", Value: "#/components/schemas/ListMeta"},
				}},
			}},
			{Key: "required", Value: []interface{}{"_type", "items", "_meta"}},
		}},
		{Key: "ErrorResponse", Value: orderedMap{
			{Key: "type", Value: "object"},
			{Key: "additionalProperties", Value: false},
			{Key: "properties", Value: orderedMap{
				{Key: "_type", Value: orderedMap{
					{Key: "type", Value: "string"},
					{Key: "enum", Value: []interface{}{errorTypeName}},
				}},
				{Key: "_error", Value: orderedMap{{Key: "type", Value: "string"}}},
			}},
			{Key: "required", Value: []interface{}{"_type", "error"}},
		}},
	}
}

func buildItemResponseSchema(table config.TableConfig) map[string]interface{} {
	root, ok := copyValue(table.Schema).(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"_type": map[string]interface{}{
					"type": "string",
					"enum": []interface{}{itemTypeName},
				},
			},
			"required": []interface{}{"_type"},
		}
	}

	if _, ok := root["type"]; !ok {
		root["type"] = "object"
	}
	props, ok := root["properties"].(map[string]interface{})
	if !ok {
		props = map[string]interface{}{}
		root["properties"] = props
	}
	props["_type"] = map[string]interface{}{
		"type": "string",
		"enum": []interface{}{itemTypeName},
	}
	root["required"] = appendRequiredField(root["required"], "_type")
	return root
}

func buildPatchSchema(table config.TableConfig) map[string]interface{} {
	root, ok := copyValue(table.Schema).(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": true,
		}
	}

	props, ok := root["properties"].(map[string]interface{})
	if !ok {
		props = map[string]interface{}{}
		root["properties"] = props
	}

	rkField := ""
	if table.RangeKey != nil {
		rkField = table.RangeKey.Field
	}

	for fieldName, rawProp := range props {
		if fieldName == table.PrimaryKey.Field || (rkField != "" && fieldName == rkField) {
			continue
		}
		propMap, ok := rawProp.(map[string]interface{})
		if !ok {
			continue
		}
		propCopy, ok := copyValue(propMap).(map[string]interface{})
		if !ok {
			continue
		}
		propCopy["nullable"] = true
		props[fieldName] = propCopy
	}

	required := []interface{}{table.PrimaryKey.Field}
	if rkField != "" {
		required = append(required, rkField)
	}
	root["required"] = required

	return root
}

func buildPaths(table config.TableConfig, jwtEnabled bool) orderedMap {
	paths := orderedMap{}
	hasRK := table.RangeKey != nil

	if hasRK {
		path := fmt.Sprintf("/v1/%s/data/{%s}/{%s}/_item", table.Name, table.PrimaryKey.Field, table.RangeKey.Field)
		paths = append(paths, orderedEntry{
			Key: path,
			Value: orderedMap{
				{Key: "put", Value: putItemOperation(table, true, jwtEnabled)},
				{Key: "get", Value: getItemOperation(table, true, jwtEnabled)},
				{Key: "patch", Value: patchItemOperation(table, true, jwtEnabled)},
				{Key: "delete", Value: deleteItemOperation(table, true, jwtEnabled)},
			},
		})
	} else {
		path := fmt.Sprintf("/v1/%s/data/{%s}/_item", table.Name, table.PrimaryKey.Field)
		paths = append(paths, orderedEntry{
			Key: path,
			Value: orderedMap{
				{Key: "put", Value: putItemOperation(table, false, jwtEnabled)},
				{Key: "get", Value: getItemOperation(table, false, jwtEnabled)},
				{Key: "patch", Value: patchItemOperation(table, false, jwtEnabled)},
				{Key: "delete", Value: deleteItemOperation(table, false, jwtEnabled)},
			},
		})
	}

	partitionPath := fmt.Sprintf("/v1/%s/data/{%s}/_items", table.Name, table.PrimaryKey.Field)
	paths = append(paths, orderedEntry{
		Key: partitionPath,
		Value: orderedMap{
			{Key: "get", Value: listByPKOperation(table, hasRK, jwtEnabled)},
		},
	})

	if table.AllowTableScan {
		tableScanPath := fmt.Sprintf("/v1/%s/_items", table.Name)
		paths = append(paths, orderedEntry{
			Key: tableScanPath,
			Value: orderedMap{
				{Key: "get", Value: scanTableOperation(table, hasRK, jwtEnabled)},
			},
		})
	}

	for _, idx := range table.Indexes {
		indexQueryPath := fmt.Sprintf("/v1/%s/_index/%s/{%s}/_items", table.Name, idx.Name, idx.PrimaryKey.Field)
		paths = append(paths, orderedEntry{
			Key: indexQueryPath,
			Value: orderedMap{
				{Key: "get", Value: queryIndexOperation(table, idx, jwtEnabled)},
			},
		})

		if idx.AllowIndexScan {
			indexScanPath := fmt.Sprintf("/v1/%s/_index/%s/_items", table.Name, idx.Name)
			paths = append(paths, orderedEntry{
				Key: indexScanPath,
				Value: orderedMap{
					{Key: "get", Value: scanIndexOperation(table, idx, jwtEnabled)},
				},
			})
		}

		if idx.RangeKey != nil {
			indexGetItemPath := fmt.Sprintf("/v1/%s/_index/%s/{%s}/{%s}/_item", table.Name, idx.Name, idx.PrimaryKey.Field, idx.RangeKey.Field)
			paths = append(paths, orderedEntry{
				Key: indexGetItemPath,
				Value: orderedMap{
					{Key: "get", Value: getIndexItemOperation(table, idx, jwtEnabled)},
				},
			})
		}
	}

	return paths
}

func getItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		keyPathParam(table, table.PrimaryKey.Field, "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, keyPathParam(table, table.RangeKey.Field, "Range key value.", table.RangeKey.Pattern))
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
		keyPathParam(table, table.PrimaryKey.Field, "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, keyPathParam(table, table.RangeKey.Field, "Range key value.", table.RangeKey.Pattern))
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
		keyPathParam(table, table.PrimaryKey.Field, "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, keyPathParam(table, table.RangeKey.Field, "Range key value.", table.RangeKey.Pattern))
	}

	return map[string]interface{}{
		"operationId": operationID(table.Name, "patch", "item"),
		"summary":     "Patch item",
		"description": "Applies RFC 7396 JSON Merge Patch. primaryKey/rangeKey fields are required in the payload and must match the URL.",
		"parameters":  params,
		"requestBody": map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/merge-patch+json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/PatchItem",
					},
				},
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/PatchItem",
					},
				},
			},
		},
		"responses": patchItemResponses(jwtEnabled),
	}
}

func deleteItemOperation(table config.TableConfig, hasRK bool, jwtEnabled bool) map[string]interface{} {
	params := []interface{}{
		keyPathParam(table, table.PrimaryKey.Field, "Primary key value.", table.PrimaryKey.Pattern),
	}
	if hasRK {
		params = append(params, keyPathParam(table, table.RangeKey.Field, "Range key value.", table.RangeKey.Pattern))
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
		keyPathParam(table, table.PrimaryKey.Field, "Primary key value.", table.PrimaryKey.Pattern),
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
		keyPathParam(table, idx.PrimaryKey.Field, fmt.Sprintf("Index %q primary key value.", idx.Name), idx.PrimaryKey.Pattern),
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
		keyPathParam(table, idx.PrimaryKey.Field, fmt.Sprintf("Index %q primary key value.", idx.Name), idx.PrimaryKey.Pattern),
		keyPathParam(table, idx.RangeKey.Field, fmt.Sprintf("Index %q range key value.", idx.Name), idx.RangeKey.Pattern),
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
		"200": jsonResponse("Item found.", map[string]interface{}{"$ref": "#/components/schemas/ItemResponse"}),
		"400": jsonErrorResponse("Invalid request."),
		"404": jsonErrorResponse("Item not found."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func putItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Item stored.", map[string]interface{}{"$ref": "#/components/schemas/ItemResponse"}),
		"400": jsonErrorResponse("Invalid request body or key mismatch."),
		"500": jsonErrorResponse("Internal server error."),
	})
}

func patchItemResponses(jwtEnabled bool) map[string]interface{} {
	return withAuthError(jwtEnabled, map[string]interface{}{
		"200": jsonResponse("Item patched.", map[string]interface{}{"$ref": "#/components/schemas/ItemResponse"}),
		"400": jsonErrorResponse("Invalid patch, key mismatch, or validation failure."),
		"409": jsonErrorResponse("Item was modified by another request."),
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

func keyPathParam(table config.TableConfig, field, description, configuredPattern string) map[string]interface{} {
	return pathParamWithSchema(field, description, keyPathSchema(table, field, configuredPattern))
}

func keyPathSchema(table config.TableConfig, field, configuredPattern string) map[string]interface{} {
	schema := map[string]interface{}{
		"type": "string",
	}

	if enum := schemaStringEnumConstraint(table, field); len(enum) > 0 {
		schema["enum"] = enum
		return schema
	}

	pattern := configuredPattern
	if pattern == "" {
		pattern = schemaStringPatternConstraint(table, field)
	}
	if pattern == "" {
		pattern = keyValuePathPattern
	}
	schema["pattern"] = pattern

	return schema
}

func schemaStringEnumConstraint(table config.TableConfig, field string) []interface{} {
	prop := schemaField(table, field)
	if prop == nil {
		return nil
	}

	rawEnum, ok := prop["enum"].([]interface{})
	if !ok || len(rawEnum) == 0 {
		return nil
	}

	enum := make([]interface{}, 0, len(rawEnum))
	for _, v := range rawEnum {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		enum = append(enum, s)
	}
	return enum
}

func schemaStringPatternConstraint(table config.TableConfig, field string) string {
	prop := schemaField(table, field)
	if prop == nil {
		return ""
	}
	pattern, _ := prop["pattern"].(string)
	return pattern
}

func schemaField(table config.TableConfig, field string) map[string]interface{} {
	root, ok := table.Schema.(map[string]interface{})
	if !ok {
		return nil
	}
	props, ok := root["properties"].(map[string]interface{})
	if !ok {
		return nil
	}
	prop, ok := props[field].(map[string]interface{})
	if !ok {
		return nil
	}
	return prop
}

func pathParamWithSchema(name, description string, schema map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": description,
		"schema":      schema,
	}
}

func appendRequiredField(requiredRaw interface{}, field string) []interface{} {
	var required []interface{}

	switch v := requiredRaw.(type) {
	case []interface{}:
		required = append(required, v...)
	case []string:
		required = make([]interface{}, 0, len(v)+1)
		for _, s := range v {
			required = append(required, s)
		}
	default:
		required = []interface{}{}
	}

	for _, existing := range required {
		if name, ok := existing.(string); ok && name == field {
			return required
		}
	}
	return append(required, field)
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
