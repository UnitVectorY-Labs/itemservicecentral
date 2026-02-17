package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/lib/pq"
)

const metaConfigHashRowName = "_meta"

type storedConfigHash struct {
	ConfigHash string `json:"configHash"`
}

// TablesConfigHash returns a deterministic SHA-256 hash for the canonical
// minimal table-structure YAML (table names, key fields, and index key fields).
func TablesConfigHash(tables []config.TableConfig) (string, error) {
	payload, err := config.MarshalMinimalTableStructureYAML(tables)
	if err != nil {
		return "", fmt.Errorf("marshal minimal structure payload: %w", err)
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// ValidateTablesConfigHash checks whether the DB-stored config hash matches the current config.
func ValidateTablesConfigHash(db *sql.DB, tables []config.TableConfig) error {
	expectedHash, err := TablesConfigHash(tables)
	if err != nil {
		return fmt.Errorf("compute tables config hash: %w", err)
	}

	var configJSON []byte
	err = db.QueryRow(`SELECT config FROM _meta WHERE table_name = $1`, metaConfigHashRowName).Scan(&configJSON)
	switch {
	case err == sql.ErrNoRows:
		return fmt.Errorf("database config hash is missing in _meta; run migrate or use --skip-config-validation")
	case err != nil:
		if isUndefinedTableError(err) {
			return fmt.Errorf("_meta table does not exist; run migrate or use --skip-config-validation")
		}
		return fmt.Errorf("querying _meta config hash: %w", err)
	}

	var stored storedConfigHash
	if err := json.Unmarshal(configJSON, &stored); err != nil {
		return fmt.Errorf("parse stored _meta config hash: %w", err)
	}
	if strings.TrimSpace(stored.ConfigHash) == "" {
		return fmt.Errorf("stored _meta config hash is empty; run migrate or use --skip-config-validation")
	}
	if stored.ConfigHash != expectedHash {
		return fmt.Errorf("configuration hash mismatch: database=%s config=%s", stored.ConfigHash, expectedHash)
	}

	return nil
}

func upsertTablesConfigHash(tx *sql.Tx, tables []config.TableConfig, dryRun bool) error {
	hash, err := TablesConfigHash(tables)
	if err != nil {
		return fmt.Errorf("compute tables config hash: %w", err)
	}

	configJSON, err := json.Marshal(storedConfigHash{ConfigHash: hash})
	if err != nil {
		return fmt.Errorf("marshal _meta config hash: %w", err)
	}

	if dryRun {
		log.Printf("[dry-run] would upsert _meta config hash %q", hash)
		return nil
	}

	if _, err := tx.Exec(
		`INSERT INTO _meta (table_name, config)
		 VALUES ($1, $2)
		 ON CONFLICT (table_name)
		 DO UPDATE SET config = EXCLUDED.config, updated_at = now()`,
		metaConfigHashRowName,
		configJSON,
	); err != nil {
		return fmt.Errorf("upsert _meta config hash: %w", err)
	}

	return nil
}

func isUndefinedTableError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "42P01"
}
