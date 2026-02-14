package config

import (
	"fmt"
	"os"
	"regexp"

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
	Name           string      `yaml:"name"`
	PK             KeyConfig   `yaml:"pk"`
	RK             *KeyConfig  `yaml:"rk"`
	AllowTableScan bool        `yaml:"allowTableScan"`
	Schema         interface{} `yaml:"schema"`
	DefaultFields  []string    `yaml:"defaultFields"`
	Indexes        []IndexConfig `yaml:"indexes"`
}

type KeyConfig struct {
	Field   string `yaml:"field"`
	Pattern string `yaml:"pattern"`
}

type IndexConfig struct {
	Name           string     `yaml:"name"`
	PK             KeyConfig  `yaml:"pk"`
	RK             *KeyConfig `yaml:"rk"`
	Projection     []string   `yaml:"projection"`
	AllowIndexScan bool       `yaml:"allowIndexScan"`
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

		// PK validation
		if t.PK.Field == "" {
			return fmt.Errorf("table %q: pk field is required", t.Name)
		}
		if !keyFieldRegexp.MatchString(t.PK.Field) {
			return fmt.Errorf("table %q: pk field %q must match %s", t.Name, t.PK.Field, keyFieldRegexp.String())
		}
		if t.PK.Pattern == "" {
			return fmt.Errorf("table %q: pk pattern is required", t.Name)
		}

		// RK validation
		if t.RK != nil {
			if t.RK.Field == "" {
				return fmt.Errorf("table %q: rk field is required when rk is set", t.Name)
			}
			if !keyFieldRegexp.MatchString(t.RK.Field) {
				return fmt.Errorf("table %q: rk field %q must match %s", t.Name, t.RK.Field, keyFieldRegexp.String())
			}
			if t.RK.Pattern == "" {
				return fmt.Errorf("table %q: rk pattern is required when rk is set", t.Name)
			}
			if t.PK.Field == t.RK.Field {
				return fmt.Errorf("table %q: pk field and rk field must be different", t.Name)
			}
		}

		// Schema validation
		if t.Schema == nil {
			return fmt.Errorf("table %q: schema is required", t.Name)
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

			// Index PK validation
			if idx.PK.Field == "" {
				return fmt.Errorf("table %q: index %q: pk field is required", t.Name, idx.Name)
			}
			if !keyFieldRegexp.MatchString(idx.PK.Field) {
				return fmt.Errorf("table %q: index %q: pk field %q must match %s", t.Name, idx.Name, idx.PK.Field, keyFieldRegexp.String())
			}
			if idx.PK.Field == t.PK.Field {
				return fmt.Errorf("table %q: index %q: pk field must be different from base pk field", t.Name, idx.Name)
			}
			if t.RK != nil && idx.PK.Field == t.RK.Field {
				return fmt.Errorf("table %q: index %q: pk field must be different from base rk field", t.Name, idx.Name)
			}

			// Index RK validation
			if idx.RK != nil {
				if idx.RK.Field == "" {
					return fmt.Errorf("table %q: index %q: rk field is required when rk is set", t.Name, idx.Name)
				}
				if !keyFieldRegexp.MatchString(idx.RK.Field) {
					return fmt.Errorf("table %q: index %q: rk field %q must match %s", t.Name, idx.Name, idx.RK.Field, keyFieldRegexp.String())
				}
				if idx.RK.Field == t.PK.Field {
					return fmt.Errorf("table %q: index %q: rk field must be different from base pk field", t.Name, idx.Name)
				}
				if t.RK != nil && idx.RK.Field == t.RK.Field {
					return fmt.Errorf("table %q: index %q: rk field must be different from base rk field", t.Name, idx.Name)
				}
				if idx.RK.Field == idx.PK.Field {
					return fmt.Errorf("table %q: index %q: rk field must be different from index pk field", t.Name, idx.Name)
				}
			}
		}
	}

	return nil
}
