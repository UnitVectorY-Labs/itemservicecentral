package validate

import (
	"fmt"
	"regexp"
)

var (
	keyValueRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]*$`)
	jsonKeyRegexp  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

const maxKeyValueLength = 512

// ValidateKeyValue validates that a value used as a PK, RK, or index key
// matches the URL id rule: starts with a letter or underscore, followed by
// alphanumeric, dot, dash, or underscore characters.
func ValidateKeyValue(value string) error {
	if value == "" {
		return fmt.Errorf("key value must not be empty")
	}
	if len(value) > maxKeyValueLength {
		return fmt.Errorf("key value must not exceed %d characters", maxKeyValueLength)
	}
	if !keyValueRegexp.MatchString(value) {
		return fmt.Errorf("key value %q must match %s", value, keyValueRegexp.String())
	}
	return nil
}

// ValidateJSONKeys recursively validates that all JSON object keys at any
// depth match the allowed pattern. The input should be the result of
// json.Unmarshal into interface{}.
func ValidateJSONKeys(data interface{}) error {
	return validateJSONKeys(data, "")
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func validateJSONKeys(data interface{}, path string) error {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			childPath := joinPath(path, key)
			if !jsonKeyRegexp.MatchString(key) {
				return fmt.Errorf("invalid JSON key %q at path %q", key, childPath)
			}
			if err := validateJSONKeys(val, childPath); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, item := range v {
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := validateJSONKeys(item, elemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateKeyPattern validates a key value against a regex pattern from config.
func ValidateKeyPattern(value string, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	if !re.MatchString(value) {
		return fmt.Errorf("key value %q does not match pattern %q", value, pattern)
	}
	return nil
}
