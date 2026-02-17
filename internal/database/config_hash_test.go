package database

import (
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

func TestTablesConfigHash_IgnoresNonStructuralChanges(t *testing.T) {
	base := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field:   "itemId",
				Pattern: "^[A-Za-z_][A-Za-z0-9._-]*$",
			},
			AllowTableScan: false,
			Schema: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"itemId": map[string]interface{}{
						"type":    "string",
						"pattern": "^[A-Za-z_][A-Za-z0-9._-]*$",
					},
					"status": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"itemId"},
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field:   "status",
						Pattern: "^[a-z]+$",
					},
					AllowIndexScan: false,
				},
			},
		},
	}

	changed := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field:   "itemId",
				Pattern: "^.*$", // pattern changed, no DB structure impact
			},
			AllowTableScan: true, // API behavior changed, no DB structure impact
			Schema: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"itemId": map[string]interface{}{
						"type":    "string",
						"pattern": "^.*$",
					},
					"status": map[string]interface{}{
						"type": "string",
					},
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"itemId", "name"},
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field:   "status",
						Pattern: "^.*$",
					},
					AllowIndexScan: true,
				},
			},
		},
	}

	baseHash, err := TablesConfigHash(base)
	if err != nil {
		t.Fatalf("hash base failed: %v", err)
	}
	changedHash, err := TablesConfigHash(changed)
	if err != nil {
		t.Fatalf("hash changed failed: %v", err)
	}

	if baseHash != changedHash {
		t.Fatalf("expected matching hashes for non-structural changes, got %q and %q", baseHash, changedHash)
	}
}

func TestTablesConfigHash_ChangesForStructuralUpdates(t *testing.T) {
	base := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field: "itemId",
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "status",
					},
				},
			},
		},
	}

	changedTableKey := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field: "id",
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "status",
					},
				},
			},
		},
	}

	baseHash, err := TablesConfigHash(base)
	if err != nil {
		t.Fatalf("hash base failed: %v", err)
	}
	changedTableKeyHash, err := TablesConfigHash(changedTableKey)
	if err != nil {
		t.Fatalf("hash changed table key failed: %v", err)
	}
	if baseHash == changedTableKeyHash {
		t.Fatalf("expected hash change for table key-field update")
	}

	changedIndexKey := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field: "itemId",
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "state",
					},
				},
			},
		},
	}

	changedIndexKeyHash, err := TablesConfigHash(changedIndexKey)
	if err != nil {
		t.Fatalf("hash changed index key failed: %v", err)
	}
	if baseHash == changedIndexKeyHash {
		t.Fatalf("expected hash change for index key-field update")
	}
}

func TestTablesConfigHash_IgnoresOrdering(t *testing.T) {
	tablesA := []config.TableConfig{
		{
			Name: "orders",
			PrimaryKey: config.KeyConfig{
				Field: "orderId",
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_customer",
					PrimaryKey: config.KeyConfig{
						Field: "customerId",
					},
				},
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "status",
					},
				},
			},
		},
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field: "itemId",
			},
		},
	}

	tablesB := []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field: "itemId",
			},
		},
		{
			Name: "orders",
			PrimaryKey: config.KeyConfig{
				Field: "orderId",
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "status",
					},
				},
				{
					Name: "by_customer",
					PrimaryKey: config.KeyConfig{
						Field: "customerId",
					},
				},
			},
		},
	}

	hashA, err := TablesConfigHash(tablesA)
	if err != nil {
		t.Fatalf("hash A failed: %v", err)
	}
	hashB, err := TablesConfigHash(tablesB)
	if err != nil {
		t.Fatalf("hash B failed: %v", err)
	}

	if hashA != hashB {
		t.Fatalf("expected matching hashes for re-ordered config, got %q and %q", hashA, hashB)
	}
}
