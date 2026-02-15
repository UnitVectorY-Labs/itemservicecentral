package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/schema"
	"gopkg.in/yaml.v3"
)

var (
	nameRegexp     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	keyFieldRegexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Tables []TableConfig `yaml:"tables"`
}

type ServerConfig struct {
	Port int       `yaml:"port"`
	JWT  JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
	Enabled  bool   `yaml:"enabled"`
	JWKSUrl  string `yaml:"jwksUrl"`
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
}

type TableConfig struct {
	Name           string        `yaml:"name"`
	PrimaryKey     KeyConfig     `yaml:"primaryKey"`
	RangeKey       *KeyConfig    `yaml:"rangeKey"`
	AllowTableScan bool          `yaml:"allowTableScan"`
	Schema         interface{}   `yaml:"schema"`
	Indexes        []IndexConfig `yaml:"indexes"`
}

type KeyConfig struct {
	Field   string `yaml:"field"`
	Pattern string `yaml:"pattern"`
}

type IndexProjection struct {
	Type             string   `yaml:"type"`
	NonKeyAttributes []string `yaml:"nonKeyAttributes"`
}

type IndexConfig struct {
	Name           string           `yaml:"name"`
	PrimaryKey     KeyConfig        `yaml:"primaryKey"`
	RangeKey       *KeyConfig       `yaml:"rangeKey"`
	Projection     *IndexProjection `yaml:"projection"`
	AllowIndexScan bool             `yaml:"allowIndexScan"`
}

// Load reads and parses a YAML configuration file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for correctness.
func Validate(cfg *Config) error {
	// Default port to 8080 if not set
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	if len(cfg.Tables) == 0 {
		return fmt.Errorf("at least one table must be defined")
	}

	tableNames := make(map[string]bool)
	for i, t := range cfg.Tables {
		// Table name validation
		if t.Name == "" {
			return fmt.Errorf("table[%d]: name is required", i)
		}
		if !nameRegexp.MatchString(t.Name) {
			return fmt.Errorf("table[%d]: name %q must match %s", i, t.Name, nameRegexp.String())
		}
		if tableNames[t.Name] {
			return fmt.Errorf("table[%d]: duplicate table name %q", i, t.Name)
		}
		tableNames[t.Name] = true

		// PrimaryKey validation
		if t.PrimaryKey.Field == "" {
			return fmt.Errorf("table %q: primaryKey field is required", t.Name)
		}
		if !keyFieldRegexp.MatchString(t.PrimaryKey.Field) {
			return fmt.Errorf("table %q: primaryKey field %q must match %s", t.Name, t.PrimaryKey.Field, keyFieldRegexp.String())
		}
		if t.PrimaryKey.Pattern == "" {
			return fmt.Errorf("table %q: primaryKey pattern is required", t.Name)
		}

		// RangeKey validation
		if t.RangeKey != nil {
			if t.RangeKey.Field == "" {
				return fmt.Errorf("table %q: rangeKey field is required when rangeKey is set", t.Name)
			}
			if !keyFieldRegexp.MatchString(t.RangeKey.Field) {
				return fmt.Errorf("table %q: rangeKey field %q must match %s", t.Name, t.RangeKey.Field, keyFieldRegexp.String())
			}
			if t.RangeKey.Pattern == "" {
				return fmt.Errorf("table %q: rangeKey pattern is required when rangeKey is set", t.Name)
			}
			if t.PrimaryKey.Field == t.RangeKey.Field {
				return fmt.Errorf("table %q: primaryKey field and rangeKey field must be different", t.Name)
			}
		}

		// Schema validation
		if t.Schema == nil {
			return fmt.Errorf("table %q: schema is required", t.Name)
		}
		if err := schema.ValidateDefinition(t.Schema); err != nil {
			return fmt.Errorf("table %q: invalid schema: %w", t.Name, err)
		}

		if err := validateSchemaKeys(t); err != nil {
			return err
		}

		// Index validation
		indexNames := make(map[string]bool)
		for j, idx := range t.Indexes {
			if idx.Name == "" {
				return fmt.Errorf("table %q: index[%d]: name is required", t.Name, j)
			}
			if !nameRegexp.MatchString(idx.Name) {
				return fmt.Errorf("table %q: index[%d]: name %q must match %s", t.Name, j, idx.Name, nameRegexp.String())
			}
			if indexNames[idx.Name] {
				return fmt.Errorf("table %q: duplicate index name %q", t.Name, idx.Name)
			}
			indexNames[idx.Name] = true

			// Index PrimaryKey validation
			if idx.PrimaryKey.Field == "" {
				return fmt.Errorf("table %q: index %q: primaryKey field is required", t.Name, idx.Name)
			}
			if !keyFieldRegexp.MatchString(idx.PrimaryKey.Field) {
				return fmt.Errorf("table %q: index %q: primaryKey field %q must match %s", t.Name, idx.Name, idx.PrimaryKey.Field, keyFieldRegexp.String())
			}
			if idx.PrimaryKey.Field == t.PrimaryKey.Field {
				return fmt.Errorf("table %q: index %q: primaryKey field must be different from base primaryKey field", t.Name, idx.Name)
			}
			if t.RangeKey != nil && idx.PrimaryKey.Field == t.RangeKey.Field {
				return fmt.Errorf("table %q: index %q: primaryKey field must be different from base rangeKey field", t.Name, idx.Name)
			}

			// Index RangeKey validation
			if idx.RangeKey != nil {
				if idx.RangeKey.Field == "" {
					return fmt.Errorf("table %q: index %q: rangeKey field is required when rangeKey is set", t.Name, idx.Name)
				}
				if !keyFieldRegexp.MatchString(idx.RangeKey.Field) {
					return fmt.Errorf("table %q: index %q: rangeKey field %q must match %s", t.Name, idx.Name, idx.RangeKey.Field, keyFieldRegexp.String())
				}
				if idx.RangeKey.Field == t.PrimaryKey.Field {
					return fmt.Errorf("table %q: index %q: rangeKey field must be different from base primaryKey field", t.Name, idx.Name)
				}
				if t.RangeKey != nil && idx.RangeKey.Field == t.RangeKey.Field {
					return fmt.Errorf("table %q: index %q: rangeKey field must be different from base rangeKey field", t.Name, idx.Name)
				}
				if idx.RangeKey.Field == idx.PrimaryKey.Field {
					return fmt.Errorf("table %q: index %q: rangeKey field must be different from index primaryKey field", t.Name, idx.Name)
				}
			}

			// Index Projection validation
			if idx.Projection != nil {
				switch idx.Projection.Type {
				case "ALL", "KEYS_ONLY":
					if len(idx.Projection.NonKeyAttributes) > 0 {
						return fmt.Errorf("table %q: index %q: projection nonKeyAttributes must be empty when type is %q", t.Name, idx.Name, idx.Projection.Type)
					}
				case "INCLUDE":
					if len(idx.Projection.NonKeyAttributes) == 0 {
						return fmt.Errorf("table %q: index %q: projection nonKeyAttributes must not be empty when type is INCLUDE", t.Name, idx.Name)
					}
				default:
					return fmt.Errorf("table %q: index %q: projection type must be one of ALL, KEYS_ONLY, or INCLUDE", t.Name, idx.Name)
				}
			}
		}
	}

	return nil
}

func validateSchemaKeys(t TableConfig) error {
	schemaMap, ok := t.Schema.(map[string]interface{})
	if !ok {
		return nil
	}
	propsRaw, ok := schemaMap["properties"]
	if !ok {
		return nil
	}
	props, ok := propsRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	if err := validateSchemaKeyField(t.Name, props, t.PrimaryKey.Field, "primaryKey"); err != nil {
		return err
	}
	if t.RangeKey != nil {
		if err := validateSchemaKeyField(t.Name, props, t.RangeKey.Field, "rangeKey"); err != nil {
			return err
		}
	}
	return nil
}

func validateSchemaKeyField(tableName string, props map[string]interface{}, field, keyLabel string) error {
	propRaw, ok := props[field]
	if !ok {
		return fmt.Errorf("table %q: schema must define property %q for %s field", tableName, field, keyLabel)
	}
	prop, ok := propRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("table %q: schema must define property %q for %s field", tableName, field, keyLabel)
	}
	typeVal, _ := prop["type"].(string)
	if typeVal != "string" {
		return fmt.Errorf("table %q: schema property %q for %s must have type \"string\"", tableName, field, keyLabel)
	}
	patternVal, hasPattern := prop["pattern"]
	if !hasPattern {
		return fmt.Errorf("table %q: schema property %q for %s must have a \"pattern\" constraint", tableName, field, keyLabel)
	}
	patternStr, ok := patternVal.(string)
	if !ok || patternStr == "" {
		return fmt.Errorf("table %q: schema property %q for %s must have a \"pattern\" constraint", tableName, field, keyLabel)
	}
	if _, err := regexp.Compile(patternStr); err != nil {
		return fmt.Errorf("table %q: schema property %q for %s has invalid pattern: %w", tableName, field, keyLabel, err)
	}
	return nil
}
