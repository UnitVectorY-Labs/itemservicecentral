package validate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateKeyValue_Valid(t *testing.T) {
	valid := []string{
		"abc",
		"ABC",
		"a1",
		"_private",
		"user.name",
		"my-key",
		"a_b_c",
		"A.B-C_D",
		"_",
		"a",
	}
	for _, v := range valid {
		if err := ValidateKeyValue(v); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", v, err)
		}
	}
}

func TestValidateKeyValue_Invalid(t *testing.T) {
	invalid := []struct {
		value string
		desc  string
	}{
		{"", "empty string"},
		{"1abc", "starts with number"},
		{".abc", "starts with dot"},
		{"-abc", "starts with dash"},
		{"has space", "contains space"},
		{"key!", "contains exclamation"},
		{"key@value", "contains at sign"},
		{"key#val", "contains hash"},
		{string(make([]byte, 513)), "exceeds max length"},
	}
	for _, tc := range invalid {
		if err := ValidateKeyValue(tc.value); err == nil {
			t.Errorf("expected %q (%s) to be invalid, got nil error", tc.value, tc.desc)
		}
	}
}

func TestValidateKeyValue_MaxLength(t *testing.T) {
	// Exactly 512 characters should be valid
	val := "a" + strings.Repeat("b", 511)
	if err := ValidateKeyValue(val); err != nil {
		t.Errorf("expected 512-char value to be valid, got error: %v", err)
	}

	// 513 characters should be invalid
	val = "a" + strings.Repeat("b", 512)
	if err := ValidateKeyValue(val); err == nil {
		t.Error("expected 513-char value to be invalid, got nil error")
	}
}

func TestValidateJSONKeys_ValidNestedObject(t *testing.T) {
	raw := `{
		"name": "test",
		"address": {
			"street": "123 Main St",
			"city": "Springfield",
			"nested": {
				"deep": "value"
			}
		},
		"tags": ["a", "b"],
		"items": [{"id": 1}, {"id": 2}]
	}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if err := ValidateJSONKeys(data); err != nil {
		t.Errorf("expected valid JSON keys, got error: %v", err)
	}
}

func TestValidateJSONKeys_ValidKeysWithDashAndUnderscore(t *testing.T) {
	raw := `{"my-key": 1, "my_key": 2, "Key3": 3}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if err := ValidateJSONKeys(data); err != nil {
		t.Errorf("expected valid JSON keys, got error: %v", err)
	}
}

func TestValidateJSONKeys_InvalidKeyStartingWithUnderscore(t *testing.T) {
	raw := `{"valid": {"_hidden": "value"}}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	err := ValidateJSONKeys(data)
	if err == nil {
		t.Fatal("expected error for key starting with underscore")
	}
	if !strings.Contains(err.Error(), "_hidden") {
		t.Errorf("expected error to mention '_hidden', got: %v", err)
	}
}

func TestValidateJSONKeys_InvalidKeyContainingDot(t *testing.T) {
	raw := `{"level1": {"level2": {"bad.key": "value"}}}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	err := ValidateJSONKeys(data)
	if err == nil {
		t.Fatal("expected error for key containing dot")
	}
	if !strings.Contains(err.Error(), "bad.key") {
		t.Errorf("expected error to mention 'bad.key', got: %v", err)
	}
}

func TestValidateJSONKeys_InvalidKeyInArray(t *testing.T) {
	raw := `{"items": [{"good": 1}, {"$bad": 2}]}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	err := ValidateJSONKeys(data)
	if err == nil {
		t.Fatal("expected error for invalid key in array element")
	}
	if !strings.Contains(err.Error(), "$bad") {
		t.Errorf("expected error to mention '$bad', got: %v", err)
	}
}

func TestValidateJSONKeys_TopLevelInvalidKey(t *testing.T) {
	raw := `{" space": "value"}`
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	err := ValidateJSONKeys(data)
	if err == nil {
		t.Fatal("expected error for key with leading space")
	}
}

func TestValidateJSONKeys_NonObjectInput(t *testing.T) {
	// Non-object inputs should pass without error
	inputs := []string{`"hello"`, `42`, `true`, `null`, `[1, 2, 3]`}
	for _, input := range inputs {
		var data interface{}
		if err := json.Unmarshal([]byte(input), &data); err != nil {
			t.Fatalf("failed to unmarshal %s: %v", input, err)
		}
		if err := ValidateJSONKeys(data); err != nil {
			t.Errorf("expected no error for input %s, got: %v", input, err)
		}
	}
}

func TestValidateKeyPattern_Match(t *testing.T) {
	if err := ValidateKeyPattern("abc123", `^[a-z0-9]+$`); err != nil {
		t.Errorf("expected match, got error: %v", err)
	}
}

func TestValidateKeyPattern_NoMatch(t *testing.T) {
	err := ValidateKeyPattern("ABC", `^[a-z]+$`)
	if err == nil {
		t.Fatal("expected error for non-matching pattern")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateKeyPattern_InvalidPattern(t *testing.T) {
	err := ValidateKeyPattern("test", `[invalid`)
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("unexpected error: %v", err)
	}
}
