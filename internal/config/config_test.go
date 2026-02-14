package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to write a temp YAML file and return its path
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

// minimal valid config YAML
const validYAML = `
server:
  port: 9090
  jwt:
    enabled: true
    jwksUrl: https://example.com/.well-known/jwks.json
    issuer: https://example.com
    audience: my-api
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z0-9]+$"
    schema:
      type: object
    defaultFields:
      - userId
      - name
`

func TestLoad_ValidFile(t *testing.T) {
	path := writeTempConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if !cfg.Server.JWT.Enabled {
		t.Error("expected JWT to be enabled")
	}
	if len(cfg.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(cfg.Tables))
	}
	if cfg.Tables[0].Name != "users" {
		t.Errorf("expected table name 'users', got %q", cfg.Tables[0].Name)
	}
	if cfg.Tables[0].PK.Field != "userId" {
		t.Errorf("expected pk field 'userId', got %q", cfg.Tables[0].PK.Field)
	}
	if cfg.Tables[0].RK != nil {
		t.Error("expected rk to be nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempConfig(t, "{{invalid yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	path := writeTempConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_DefaultPort(t *testing.T) {
	yaml := `
tables:
  - name: items
    pk:
      field: itemId
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestValidate_NoTables(t *testing.T) {
	yaml := `
server:
  port: 8080
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for no tables")
	}
	if !strings.Contains(err.Error(), "at least one table") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_DuplicateTableName(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for duplicate table name")
	}
	if !strings.Contains(err.Error(), "duplicate table name") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidTableName(t *testing.T) {
	yaml := `
tables:
  - name: Users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid table name")
	}
	if !strings.Contains(err.Error(), "must match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingTableName(t *testing.T) {
	yaml := `
tables:
  - pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing table name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingPKField(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing pk field")
	}
	if !strings.Contains(err.Error(), "pk field is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingPKPattern(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing pk pattern")
	}
	if !strings.Contains(err.Error(), "pk pattern is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_RKFieldAndPattern(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
      pattern: "^[0-9]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_RKMissingField(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      pattern: "^[0-9]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing rk field")
	}
	if !strings.Contains(err.Error(), "rk field is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_RKMissingPattern(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing rk pattern")
	}
	if !strings.Contains(err.Error(), "rk pattern is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_PKAndRKSameField(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: orderId
      pattern: "^[0-9]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for same pk and rk field")
	}
	if !strings.Contains(err.Error(), "pk field and rk field must be different") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingSchema(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing schema")
	}
	if !strings.Contains(err.Error(), "schema is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexValid(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: by_email
        pk:
          field: email
          pattern: "^.+$"
        projection:
          - userId
          - email
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_DuplicateIndexName(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: by_email
        pk:
          field: email
          pattern: "^.+$"
      - name: by_email
        pk:
          field: name
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for duplicate index name")
	}
	if !strings.Contains(err.Error(), "duplicate index name") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidIndexName(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: ByEmail
        pk:
          field: email
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid index name")
	}
	if !strings.Contains(err.Error(), "must match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexPKSameAsBasePK(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: by_user
        pk:
          field: userId
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for index pk same as base pk")
	}
	if !strings.Contains(err.Error(), "different from base pk") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexPKSameAsBaseRK(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
      pattern: "^[0-9]+$"
    schema:
      type: object
    indexes:
      - name: by_item
        pk:
          field: itemId
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for index pk same as base rk")
	}
	if !strings.Contains(err.Error(), "different from base rk") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexRKSameAsBasePK(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
      pattern: "^[0-9]+$"
    schema:
      type: object
    indexes:
      - name: by_status
        pk:
          field: status
          pattern: "^.+$"
        rk:
          field: orderId
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for index rk same as base pk")
	}
	if !strings.Contains(err.Error(), "different from base pk") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexRKSameAsBaseRK(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
      pattern: "^[0-9]+$"
    schema:
      type: object
    indexes:
      - name: by_status
        pk:
          field: status
          pattern: "^.+$"
        rk:
          field: itemId
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for index rk same as base rk")
	}
	if !strings.Contains(err.Error(), "different from base rk") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexRKSameAsIndexPK(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: by_status
        pk:
          field: status
          pattern: "^.+$"
        rk:
          field: status
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for index rk same as index pk")
	}
	if !strings.Contains(err.Error(), "different from index pk") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidKeyFieldName(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: "123bad"
      pattern: "^[a-z]+$"
    schema:
      type: object
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid key field name")
	}
	if !strings.Contains(err.Error(), "must match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_IndexWithRKValid(t *testing.T) {
	yaml := `
tables:
  - name: orders
    pk:
      field: orderId
      pattern: "^[a-z]+$"
    rk:
      field: itemId
      pattern: "^[0-9]+$"
    schema:
      type: object
    indexes:
      - name: by_status
        pk:
          field: status
          pattern: "^.+$"
        rk:
          field: createdAt
          pattern: "^.+$"
        projection:
          - orderId
          - status
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidate_MissingIndexPKField(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - name: by_email
        pk:
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing index pk field")
	}
	if !strings.Contains(err.Error(), "pk field is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingIndexName(t *testing.T) {
	yaml := `
tables:
  - name: users
    pk:
      field: userId
      pattern: "^[a-z]+$"
    schema:
      type: object
    indexes:
      - pk:
          field: email
          pattern: "^.+$"
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing index name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
