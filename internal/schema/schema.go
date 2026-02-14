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

// Compile takes a raw JSON Schema (as map[string]interface{} from YAML) and compiles it.
func Compile(rawSchema interface{}) (*Validator, error) {
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
