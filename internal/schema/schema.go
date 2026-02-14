package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator wraps a compiled JSON Schema for document validation.
type Validator struct {
	compiled *jsonschema.Schema
}

var (
	allowedKeywords = map[string]struct{}{
		"$schema":              {},
		"type":                 {},
		"properties":           {},
		"required":             {},
		"additionalProperties": {},
		"items":                {},
		"pattern":              {},
		"enum":                 {},
		"const":                {},
		"minLength":            {},
		"maxLength":            {},
		"minimum":              {},
		"maximum":              {},
		"exclusiveMinimum":     {},
		"exclusiveMaximum":     {},
		"multipleOf":           {},
		"minItems":             {},
		"maxItems":             {},
		"uniqueItems":          {},
		"minProperties":        {},
		"maxProperties":        {},
		"format":               {},
		"title":                {},
		"description":          {},
		"default":              {},
		"examples":             {},
	}
	forbiddenKeywords = map[string]string{
		"$ref":                  "references are not supported",
		"$defs":                 "schema definitions are not supported",
		"definitions":           "schema definitions are not supported",
		"allOf":                 "composition is not supported",
		"anyOf":                 "composition is not supported",
		"oneOf":                 "composition is not supported",
		"not":                   "negation schemas are not supported",
		"if":                    "conditional schemas are not supported",
		"then":                  "conditional schemas are not supported",
		"else":                  "conditional schemas are not supported",
		"dependentSchemas":      "dependent schemas are not supported",
		"dependencies":          "dependencies are not supported",
		"patternProperties":     "pattern properties are not supported",
		"propertyNames":         "property name schemas are not supported",
		"unevaluatedProperties": "unevaluated properties are not supported",
		"unevaluatedItems":      "unevaluated items are not supported",
		"contains":              "contains is not supported",
		"prefixItems":           "tuple-style array schemas are not supported",
	}
)

// ValidateDefinition validates that a schema uses the supported subset and
// enforces closed object schemas at every level.
func ValidateDefinition(rawSchema interface{}) error {
	root, ok := rawSchema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must be a JSON object")
	}
	typeVal, ok := root["type"].(string)
	if !ok || typeVal != "object" {
		return fmt.Errorf("schema at $ must set type to \"object\"")
	}
	return validateSchemaNode(root, "$")
}

func validateSchemaNode(node map[string]interface{}, path string) error {
	for key := range node {
		if detail, forbidden := forbiddenKeywords[key]; forbidden {
			return fmt.Errorf("schema at %s uses unsupported keyword %q: %s", path, key, detail)
		}
		if _, allowed := allowedKeywords[key]; !allowed {
			return fmt.Errorf("schema at %s uses unsupported keyword %q", path, key)
		}
	}

	if typeRaw, hasType := node["type"]; hasType {
		if _, ok := typeRaw.(string); !ok {
			return fmt.Errorf("schema at %s: type must be a string", path)
		}
	}

	if requiresClosedObject(node) {
		apRaw, ok := node["additionalProperties"]
		if !ok {
			return fmt.Errorf("schema at %s must set additionalProperties to false", path)
		}
		ap, ok := apRaw.(bool)
		if !ok || ap {
			return fmt.Errorf("schema at %s must set additionalProperties to false", path)
		}
	}

	if propsRaw, hasProps := node["properties"]; hasProps {
		props, ok := propsRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("schema at %s.properties must be an object", path)
		}
		for propName, propRaw := range props {
			propSchema, ok := propRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("schema at %s.properties.%s must be an object", path, propName)
			}
			if err := validateSchemaNode(propSchema, fmt.Sprintf("%s.properties.%s", path, propName)); err != nil {
				return err
			}
		}
	}

	if itemsRaw, hasItems := node["items"]; hasItems {
		switch items := itemsRaw.(type) {
		case map[string]interface{}:
			if err := validateSchemaNode(items, path+".items"); err != nil {
				return err
			}
		case []interface{}:
			return fmt.Errorf("schema at %s.items uses unsupported tuple schema form", path)
		default:
			return fmt.Errorf("schema at %s.items must be an object", path)
		}
	}

	return nil
}

func requiresClosedObject(node map[string]interface{}) bool {
	if t, ok := node["type"].(string); ok && t == "object" {
		return true
	}
	if _, ok := node["properties"]; ok {
		return true
	}
	if _, ok := node["required"]; ok {
		return true
	}
	if _, ok := node["additionalProperties"]; ok {
		return true
	}
	return false
}

// Compile takes a raw JSON Schema (as map[string]interface{} from YAML) and compiles it.
func Compile(rawSchema interface{}) (*Validator, error) {
	if err := ValidateDefinition(rawSchema); err != nil {
		return nil, fmt.Errorf("unsupported schema definition: %w", err)
	}

	schemaBytes, err := json.Marshal(rawSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(schemaBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema JSON: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiled, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return &Validator{compiled: compiled}, nil
}

// Validate validates a JSON document against the compiled schema.
func (v *Validator) Validate(doc map[string]interface{}) error {
	docBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document to JSON: %w", err)
	}

	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(docBytes)))
	if err != nil {
		return fmt.Errorf("failed to unmarshal document JSON: %w", err)
	}

	if err := v.compiled.Validate(inst); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
