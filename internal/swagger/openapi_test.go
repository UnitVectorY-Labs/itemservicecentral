package swagger

import (
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"gopkg.in/yaml.v3"
)

func TestGenerateTableYAML_PKOnly(t *testing.T) {
	table := config.TableConfig{
		Name: "items",
		PrimaryKey: config.KeyConfig{
			Field:   "itemId",
			Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
		},
		AllowTableScan: true,
		Schema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"itemId": map[string]interface{}{"type": "string", "pattern": "^[A-Za-z_][A-Za-z0-9._-]*$"},
				"name":   map[string]interface{}{"type": "string"},
				"status": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"itemId", "name"},
		},
		Indexes: []config.IndexConfig{
			{
				Name: "by_status",
				PrimaryKey: config.KeyConfig{
					Field: "status",
				},
				AllowIndexScan: true,
			},
		},
	}

	doc := parseDoc(t, table, false)
	paths := asMap(t, doc["paths"])

	requirePath(t, paths, "/v1/items/data/{pk}/_item")
	requirePath(t, paths, "/v1/items/data/{pk}/_items")
	requirePath(t, paths, "/v1/items/_items")
	requirePath(t, paths, "/v1/items/_index/by_status/{indexPk}/_items")
	requirePath(t, paths, "/v1/items/_index/by_status/_items")
	forbiddenPath(t, paths, "/v1/items/data/{pk}/{rk}/_item")

	op := getOperation(t, paths, "/v1/items/data/{pk}/_items", "get")
	paramNames := parameterNames(t, op)
	requireParam(t, paramNames, "pk")
	requireParam(t, paramNames, "limit")
	requireParam(t, paramNames, "pageToken")
	requireParam(t, paramNames, "fields")
	forbiddenParam(t, paramNames, "rkBeginsWith")
}

func TestGenerateTableYAML_CompositeAndIndexRangeKey(t *testing.T) {
	table := config.TableConfig{
		Name: "orders",
		PrimaryKey: config.KeyConfig{
			Field:   "orderId",
			Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
		},
		RangeKey: &config.KeyConfig{
			Field:   "lineId",
			Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
		},
		AllowTableScan: false,
		Schema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"orderId": map[string]interface{}{"type": "string", "pattern": "^[A-Za-z_][A-Za-z0-9._-]*$"},
				"lineId":  map[string]interface{}{"type": "string", "pattern": "^[A-Za-z_][A-Za-z0-9._-]*$"},
				"status":  map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"orderId", "lineId"},
		},
		Indexes: []config.IndexConfig{
			{
				Name: "by_status",
				PrimaryKey: config.KeyConfig{
					Field: "status",
				},
				RangeKey: &config.KeyConfig{
					Field: "sortKey",
				},
			},
		},
	}

	doc := parseDoc(t, table, false)
	paths := asMap(t, doc["paths"])

	requirePath(t, paths, "/v1/orders/data/{pk}/{rk}/_item")
	requirePath(t, paths, "/v1/orders/data/{pk}/_items")
	requirePath(t, paths, "/v1/orders/_index/by_status/{indexPk}/_items")
	requirePath(t, paths, "/v1/orders/_index/by_status/{indexPk}/{indexRk}/_item")
	forbiddenPath(t, paths, "/v1/orders/_items")

	partitionOp := getOperation(t, paths, "/v1/orders/data/{pk}/_items", "get")
	partitionParams := parameterNames(t, partitionOp)
	requireParam(t, partitionParams, "rkBeginsWith")
	requireParam(t, partitionParams, "rkGt")
	requireParam(t, partitionParams, "rkGte")
	requireParam(t, partitionParams, "rkLt")
	requireParam(t, partitionParams, "rkLte")

	indexQueryOp := getOperation(t, paths, "/v1/orders/_index/by_status/{indexPk}/_items", "get")
	indexParams := parameterNames(t, indexQueryOp)
	requireParam(t, indexParams, "rkBeginsWith")
}

func TestGenerateTableYAML_WithJWTSecurity(t *testing.T) {
	table := config.TableConfig{
		Name: "items",
		PrimaryKey: config.KeyConfig{
			Field: "itemId",
		},
		Schema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
		},
	}

	doc := parseDoc(t, table, true)
	components := asMap(t, doc["components"])
	securitySchemes := asMap(t, components["securitySchemes"])
	bearer := asMap(t, securitySchemes["bearerAuth"])
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("expected bearer auth security scheme, got: %#v", bearer)
	}

	security, ok := doc["security"].([]interface{})
	if !ok || len(security) == 0 {
		t.Fatalf("expected top-level security requirements, got: %#v", doc["security"])
	}
}

func TestProviderYAML(t *testing.T) {
	table := config.TableConfig{
		Name: "items",
		PrimaryKey: config.KeyConfig{
			Field: "itemId",
		},
		Schema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
		},
	}

	p := NewProvider([]config.TableConfig{table}, false)

	first, err := p.YAML("items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := p.YAML("items")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("expected cached YAML to be stable")
	}

	// Ensure returned slices are copies, not mutable references to cache.
	first[0] = 'X'
	third, err := p.YAML("items")
	if err != nil {
		t.Fatalf("unexpected error on third call: %v", err)
	}
	if string(second) != string(third) {
		t.Fatalf("expected cached YAML to be isolated from caller mutations")
	}

	if _, err := p.YAML("missing"); err == nil {
		t.Fatalf("expected error for unknown table")
	}
}

func parseDoc(t *testing.T, table config.TableConfig, jwtEnabled bool) map[string]interface{} {
	t.Helper()
	out, err := GenerateTableYAML(table, jwtEnabled)
	if err != nil {
		t.Fatalf("generate yaml: %v", err)
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	return doc
}

func asMap(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	result, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", value)
	}
	return result
}

func requirePath(t *testing.T, paths map[string]interface{}, path string) {
	t.Helper()
	if _, ok := paths[path]; !ok {
		t.Fatalf("expected path %q in OpenAPI document", path)
	}
}

func forbiddenPath(t *testing.T, paths map[string]interface{}, path string) {
	t.Helper()
	if _, ok := paths[path]; ok {
		t.Fatalf("did not expect path %q in OpenAPI document", path)
	}
}

func getOperation(t *testing.T, paths map[string]interface{}, path string, method string) map[string]interface{} {
	t.Helper()
	pathItem := asMap(t, paths[path])
	op := asMap(t, pathItem[method])
	return op
}

func parameterNames(t *testing.T, operation map[string]interface{}) map[string]bool {
	t.Helper()
	params := map[string]bool{}
	paramList, ok := operation["parameters"].([]interface{})
	if !ok {
		return params
	}
	for _, paramRaw := range paramList {
		param := asMap(t, paramRaw)
		name, _ := param["name"].(string)
		if name != "" {
			params[name] = true
		}
	}
	return params
}

func requireParam(t *testing.T, params map[string]bool, name string) {
	t.Helper()
	if !params[name] {
		t.Fatalf("expected parameter %q", name)
	}
}

func forbiddenParam(t *testing.T, params map[string]bool, name string) {
	t.Helper()
	if params[name] {
		t.Fatalf("did not expect parameter %q", name)
	}
}
