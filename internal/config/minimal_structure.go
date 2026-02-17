package config

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// MinimalTableStructure is the schema-impacting subset of configuration
// used for migration compatibility hashing.
type MinimalTableStructure struct {
	Tables []MinimalTable `yaml:"tables"`
}

type MinimalTable struct {
	Name       string         `yaml:"name"`
	PrimaryKey MinimalKey     `yaml:"primaryKey"`
	RangeKey   *MinimalKey    `yaml:"rangeKey,omitempty"`
	Indexes    []MinimalIndex `yaml:"indexes,omitempty"`
}

type MinimalIndex struct {
	Name       string      `yaml:"name"`
	PrimaryKey MinimalKey  `yaml:"primaryKey"`
	RangeKey   *MinimalKey `yaml:"rangeKey,omitempty"`
}

type MinimalKey struct {
	Field string `yaml:"field"`
}

// BuildMinimalTableStructure extracts and canonicalizes only DB-structure
// affecting fields from full table configuration.
func BuildMinimalTableStructure(tables []TableConfig) MinimalTableStructure {
	out := MinimalTableStructure{
		Tables: make([]MinimalTable, 0, len(tables)),
	}

	for _, t := range tables {
		mt := MinimalTable{
			Name: t.Name,
			PrimaryKey: MinimalKey{
				Field: t.PrimaryKey.Field,
			},
			Indexes: make([]MinimalIndex, 0, len(t.Indexes)),
		}

		if t.RangeKey != nil {
			mt.RangeKey = &MinimalKey{Field: t.RangeKey.Field}
		}

		for _, idx := range t.Indexes {
			mi := MinimalIndex{
				Name: idx.Name,
				PrimaryKey: MinimalKey{
					Field: idx.PrimaryKey.Field,
				},
			}
			if idx.RangeKey != nil {
				mi.RangeKey = &MinimalKey{Field: idx.RangeKey.Field}
			}
			mt.Indexes = append(mt.Indexes, mi)
		}

		sort.Slice(mt.Indexes, func(i, j int) bool {
			return mt.Indexes[i].Name < mt.Indexes[j].Name
		})
		out.Tables = append(out.Tables, mt)
	}

	sort.Slice(out.Tables, func(i, j int) bool {
		return out.Tables[i].Name < out.Tables[j].Name
	})

	return out
}

// MarshalMinimalTableStructureYAML serializes the canonical minimal structure to YAML.
func MarshalMinimalTableStructureYAML(tables []TableConfig) ([]byte, error) {
	minimal := BuildMinimalTableStructure(tables)
	payload, err := yaml.Marshal(minimal)
	if err != nil {
		return nil, fmt.Errorf("marshal minimal structure yaml: %w", err)
	}
	return payload, nil
}
