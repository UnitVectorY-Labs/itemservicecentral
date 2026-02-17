package config

import (
	"reflect"
	"testing"
)

func TestBuildMinimalTableStructure_ExtractsOnlyStructuralFields(t *testing.T) {
	tables := []TableConfig{
		{
			Name: "items",
			PrimaryKey: KeyConfig{
				Field:   "itemId",
				Pattern: "ignored",
			},
			RangeKey: &KeyConfig{
				Field:   "tenantId",
				Pattern: "ignored",
			},
			AllowTableScan: true,
			Schema: map[string]interface{}{
				"type": "object",
			},
			Indexes: []IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: KeyConfig{
						Field:   "status",
						Pattern: "ignored",
					},
					AllowIndexScan: true,
				},
			},
		},
	}

	got := BuildMinimalTableStructure(tables)
	want := MinimalTableStructure{
		Tables: []MinimalTable{
			{
				Name: "items",
				PrimaryKey: MinimalKey{
					Field: "itemId",
				},
				RangeKey: &MinimalKey{
					Field: "tenantId",
				},
				Indexes: []MinimalIndex{
					{
						Name: "by_status",
						PrimaryKey: MinimalKey{
							Field: "status",
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected minimal structure\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestBuildMinimalTableStructure_CanonicalOrdering(t *testing.T) {
	tables := []TableConfig{
		{
			Name: "zebra",
			PrimaryKey: KeyConfig{
				Field: "zId",
			},
			Indexes: []IndexConfig{
				{
					Name: "zz",
					PrimaryKey: KeyConfig{
						Field: "zzField",
					},
				},
				{
					Name: "aa",
					PrimaryKey: KeyConfig{
						Field: "aaField",
					},
				},
			},
		},
		{
			Name: "alpha",
			PrimaryKey: KeyConfig{
				Field: "aId",
			},
		},
	}

	got := BuildMinimalTableStructure(tables)
	if len(got.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(got.Tables))
	}
	if got.Tables[0].Name != "alpha" || got.Tables[1].Name != "zebra" {
		t.Fatalf("tables not sorted by name: %#v", got.Tables)
	}
	if len(got.Tables[1].Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(got.Tables[1].Indexes))
	}
	if got.Tables[1].Indexes[0].Name != "aa" || got.Tables[1].Indexes[1].Name != "zz" {
		t.Fatalf("indexes not sorted by name: %#v", got.Tables[1].Indexes)
	}
}
