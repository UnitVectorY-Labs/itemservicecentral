package model

import (
	"reflect"
	"testing"
)

func TestStripKeys_BothPkAndRk(t *testing.T) {
	data := map[string]interface{}{
		"id":    "123",
		"sort":  "abc",
		"name":  "test",
		"value": 42,
	}
	result := StripKeys(data, "id", "sort")

	expected := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("StripKeys both: got %v, want %v", result, expected)
	}
}

func TestStripKeys_OnlyPk(t *testing.T) {
	data := map[string]interface{}{
		"id":    "123",
		"name":  "test",
		"value": 42,
	}
	result := StripKeys(data, "id", "")

	expected := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("StripKeys pk only: got %v, want %v", result, expected)
	}
}

func TestInjectKeys_BothPkAndRk(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}
	result := InjectKeys(data, "id", "123", "sort", "abc")

	expected := map[string]interface{}{
		"id":    "123",
		"sort":  "abc",
		"name":  "test",
		"value": 42,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("InjectKeys both: got %v, want %v", result, expected)
	}
}

func TestInjectKeys_OnlyPk(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}
	result := InjectKeys(data, "id", "123", "", "")

	expected := map[string]interface{}{
		"id":    "123",
		"name":  "test",
		"value": 42,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("InjectKeys pk only: got %v, want %v", result, expected)
	}
}

func TestMergePatch_Basic(t *testing.T) {
	target := map[string]interface{}{
		"name": "old",
		"age":  float64(30),
	}
	patch := map[string]interface{}{
		"name": "new",
	}
	result := MergePatch(target, patch)

	if result["name"] != "new" {
		t.Errorf("MergePatch basic: name got %v, want 'new'", result["name"])
	}
	if result["age"] != float64(30) {
		t.Errorf("MergePatch basic: age got %v, want 30", result["age"])
	}
}

func TestMergePatch_NullDeletion(t *testing.T) {
	target := map[string]interface{}{
		"name":  "test",
		"email": "test@example.com",
	}
	patch := map[string]interface{}{
		"email": nil,
	}
	result := MergePatch(target, patch)

	if _, ok := result["email"]; ok {
		t.Error("MergePatch null deletion: email should be removed")
	}
	if result["name"] != "test" {
		t.Errorf("MergePatch null deletion: name got %v, want 'test'", result["name"])
	}
}

func TestMergePatch_NestedMerge(t *testing.T) {
	target := map[string]interface{}{
		"address": map[string]interface{}{
			"city":  "old",
			"state": "CA",
		},
	}
	patch := map[string]interface{}{
		"address": map[string]interface{}{
			"city": "new",
		},
	}
	result := MergePatch(target, patch)

	addr, ok := result["address"].(map[string]interface{})
	if !ok {
		t.Fatal("MergePatch nested: address should be a map")
	}
	if addr["city"] != "new" {
		t.Errorf("MergePatch nested: city got %v, want 'new'", addr["city"])
	}
	if addr["state"] != "CA" {
		t.Errorf("MergePatch nested: state got %v, want 'CA'", addr["state"])
	}
}

func TestMergePatch_NestedNullDeletion(t *testing.T) {
	target := map[string]interface{}{
		"address": map[string]interface{}{
			"city":  "test",
			"state": "CA",
		},
	}
	patch := map[string]interface{}{
		"address": map[string]interface{}{
			"state": nil,
		},
	}
	result := MergePatch(target, patch)

	addr, ok := result["address"].(map[string]interface{})
	if !ok {
		t.Fatal("MergePatch nested null: address should be a map")
	}
	if _, ok := addr["state"]; ok {
		t.Error("MergePatch nested null: state should be removed")
	}
	if addr["city"] != "test" {
		t.Errorf("MergePatch nested null: city got %v, want 'test'", addr["city"])
	}
}

func TestProjectFields_Subset(t *testing.T) {
	data := map[string]interface{}{
		"id":    "123",
		"sort":  "abc",
		"name":  "test",
		"value": 42,
		"extra": "drop",
	}
	result := ProjectFields(data, []string{"name", "value"}, "id", "sort")

	expected := map[string]interface{}{
		"id":    "123",
		"sort":  "abc",
		"name":  "test",
		"value": 42,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ProjectFields subset: got %v, want %v", result, expected)
	}
}

func TestProjectFields_AlwaysIncludeKeys(t *testing.T) {
	data := map[string]interface{}{
		"id":    "123",
		"sort":  "abc",
		"name":  "test",
		"value": 42,
	}
	result := ProjectFields(data, []string{"name"}, "id", "sort")

	if _, ok := result["id"]; !ok {
		t.Error("ProjectFields keys: id should always be included")
	}
	if _, ok := result["sort"]; !ok {
		t.Error("ProjectFields keys: sort should always be included")
	}
	if _, ok := result["value"]; ok {
		t.Error("ProjectFields keys: value should not be included")
	}
}

func TestParseFieldsParam_Basic(t *testing.T) {
	result := ParseFieldsParam("name,value,extra")

	expected := []string{"name", "value", "extra"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ParseFieldsParam basic: got %v, want %v", result, expected)
	}
}

func TestParseFieldsParam_Empty(t *testing.T) {
	result := ParseFieldsParam("")

	if result != nil {
		t.Errorf("ParseFieldsParam empty: got %v, want nil", result)
	}
}

func TestParseFieldsParam_Spaces(t *testing.T) {
	result := ParseFieldsParam(" name , value , extra ")

	expected := []string{"name", "value", "extra"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ParseFieldsParam spaces: got %v, want %v", result, expected)
	}
}
