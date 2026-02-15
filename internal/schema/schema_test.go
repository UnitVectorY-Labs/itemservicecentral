package schema

import (
	"strings"
	"testing"
)

func validSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
		},
		"required":             []interface{}{"name", "age"},
		"additionalProperties": false,
	}
}

func TestCompileValidSchema(t *testing.T) {
	v, err := Compile(validSchema())
	if err != nil {
		t.Fatalf("expected no error compiling valid schema, got: %v", err)
	}
	if v == nil {
		t.Fatal("expected non-nil Validator")
	}
}

func TestValidateValidDocument(t *testing.T) {
	v, err := Compile(validSchema())
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	doc := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}
	if err := v.Validate(doc); err != nil {
		t.Fatalf("expected valid document, got error: %v", err)
	}
}

func TestValidateMissingRequiredFields(t *testing.T) {
	v, err := Compile(validSchema())
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	doc := map[string]interface{}{
		"name": "Alice",
	}
	if err := v.Validate(doc); err == nil {
		t.Fatal("expected error for missing required field 'age', got nil")
	}
}

func TestValidateWrongTypes(t *testing.T) {
	v, err := Compile(validSchema())
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	doc := map[string]interface{}{
		"name": 123,
		"age":  "not a number",
	}
	if err := v.Validate(doc); err == nil {
		t.Fatal("expected error for wrong types, got nil")
	}
}

func TestValidateAdditionalProperties(t *testing.T) {
	v, err := Compile(validSchema())
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	doc := map[string]interface{}{
		"name":  "Alice",
		"age":   30,
		"extra": "not allowed",
	}
	if err := v.Validate(doc); err == nil {
		t.Fatal("expected error for additional properties, got nil")
	}
}

func TestCompileInvalidSchema(t *testing.T) {
	invalidSchema := map[string]interface{}{
		"type": "invalid_type",
	}
	_, err := Compile(invalidSchema)
	if err == nil {
		t.Fatal("expected error compiling invalid schema, got nil")
	}
}

func TestCompileRejectsMissingAdditionalProperties(t *testing.T) {
	invalidSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	_, err := Compile(invalidSchema)
	if err == nil {
		t.Fatal("expected error for missing additionalProperties, got nil")
	}
}

func TestCompileRejectsNestedObjectWithoutAdditionalProperties(t *testing.T) {
	invalidSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"profile": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		"additionalProperties": false,
	}
	_, err := Compile(invalidSchema)
	if err == nil {
		t.Fatal("expected error for nested object missing additionalProperties, got nil")
	}
}

func TestCompileRejectsRefKeyword(t *testing.T) {
	invalidSchema := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"$ref":                 "#/$defs/foo",
	}
	_, err := Compile(invalidSchema)
	if err == nil {
		t.Fatal("expected error for unsupported $ref keyword, got nil")
	}
}

func TestValidateErrorDoesNotLeakSchemaPath(t *testing.T) {
	v, err := Compile(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"active", "inactive"},
			},
		},
		"required": []interface{}{"status"},
	})
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	err = v.Validate(map[string]interface{}{
		"status": "invalid",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	msg := err.Error()
	if strings.Contains(msg, "file:///") {
		t.Fatalf("validation error leaked file URI: %s", msg)
	}
	if !strings.Contains(msg, "at '/status'") {
		t.Fatalf("expected instance location in error message, got: %s", msg)
	}
}
