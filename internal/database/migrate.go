package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

// MigrateOptions controls the behavior of the Migrate function.
type MigrateOptions struct {
	Cleanup bool // if true, delete tables and indexes not in config
	DryRun  bool // if true, only print what would change
}

// metaConfig is the JSON structure stored in the _meta table.
type metaConfig struct {
	PrimaryKeyField string `json:"primaryKeyField"`
	RangeKeyField   string `json:"rangeKeyField"`
}

// Migrate creates or updates tables and indexes based on the provided configuration.
// All operations run inside a single transaction.
func Migrate(db *sql.DB, tables []config.TableConfig, opts MigrateOptions) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	if err := createMetaTable(tx, opts.DryRun); err != nil {
		return fmt.Errorf("_meta table: %w", err)
	}

	configuredTables := make(map[string]bool)
	for _, t := range tables {
		configuredTables[t.Name] = true

		if err := reconcileTable(tx, t, opts); err != nil {
			return fmt.Errorf("table %q: %w", t.Name, err)
		}
	}

	if opts.Cleanup {
		if err := cleanupTables(tx, configuredTables, opts.DryRun); err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}
	}

	if err := upsertTablesConfigHash(tx, tables, opts.DryRun); err != nil {
		return fmt.Errorf("store config hash: %w", err)
	}

	if opts.DryRun {
		return tx.Rollback()
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	return nil
}

func createMetaTable(tx *sql.Tx, dryRun bool) error {
	const stmt = `CREATE TABLE IF NOT EXISTS _meta (
		table_name TEXT NOT NULL,
		config JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (table_name)
	)`

	if dryRun {
		log.Printf("[dry-run] ensure _meta table exists")
	}

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to create _meta table: %w", err)
	}
	return nil
}

// reconcileTable handles creating or verifying a single table and its indexes.
func reconcileTable(tx *sql.Tx, t config.TableConfig, opts MigrateOptions) error {
	mc := metaConfig{PrimaryKeyField: t.PrimaryKey.Field}
	if t.RangeKey != nil {
		mc.RangeKeyField = t.RangeKey.Field
	}

	// Check if table exists in _meta
	var existingJSON []byte
	err := tx.QueryRow(`SELECT config FROM _meta WHERE table_name = $1`, t.Name).Scan(&existingJSON)

	switch {
	case err == sql.ErrNoRows:
		// New table
		if opts.DryRun {
			log.Printf("[dry-run] would create table %q", t.Name)
		} else {
			if err := createTable(tx, t); err != nil {
				return err
			}
		}

		configJSON, err := json.Marshal(mc)
		if err != nil {
			return fmt.Errorf("failed to marshal meta config: %w", err)
		}
		if opts.DryRun {
			log.Printf("[dry-run] would insert _meta entry for table %q", t.Name)
		} else {
			if _, err := tx.Exec(
				`INSERT INTO _meta (table_name, config) VALUES ($1, $2)`,
				t.Name, configJSON,
			); err != nil {
				return fmt.Errorf("failed to insert _meta entry: %w", err)
			}
		}
	case err != nil:
		return fmt.Errorf("failed to query _meta: %w", err)
	default:
		// Table exists in _meta — verify key fields haven't changed
		var existing metaConfig
		if err := json.Unmarshal(existingJSON, &existing); err != nil {
			return fmt.Errorf("failed to parse _meta config: %w", err)
		}
		if existing.PrimaryKeyField != mc.PrimaryKeyField {
			return fmt.Errorf("primaryKey field changed from %q to %q; this is not allowed", existing.PrimaryKeyField, mc.PrimaryKeyField)
		}
		if existing.RangeKeyField != mc.RangeKeyField {
			return fmt.Errorf("rangeKey field changed from %q to %q; this is not allowed", existing.RangeKeyField, mc.RangeKeyField)
		}
	}

	// Reconcile indexes
	if err := reconcileIndexes(tx, t, opts); err != nil {
		return fmt.Errorf("indexes: %w", err)
	}

	return nil
}

// reconcileIndexes creates new indexes and optionally removes stale ones.
func reconcileIndexes(tx *sql.Tx, t config.TableConfig, opts MigrateOptions) error {
	// Build set of desired index names
	desired := make(map[string]bool)
	for _, idx := range t.Indexes {
		desired[fmt.Sprintf("idx_%s_%s", t.Name, idx.Name)] = true
	}

	// Create indexes that don't exist yet
	for _, idx := range t.Indexes {
		idxName := fmt.Sprintf("idx_%s_%s", t.Name, idx.Name)

		// Check if index already exists
		var exists bool
		err := tx.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE tablename = $1 AND indexname = $2)`,
			t.Name, idxName,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking index %q: %w", idxName, err)
		}
		if exists {
			continue
		}

		if opts.DryRun {
			log.Printf("[dry-run] would create index %q on table %q", idxName, t.Name)
		} else {
			if err := createIndex(tx, t, idx); err != nil {
				return fmt.Errorf("index %q: %w", idx.Name, err)
			}
		}
	}

	// Cleanup stale indexes
	if opts.Cleanup {
		rows, err := tx.Query(
			`SELECT indexname FROM pg_indexes WHERE tablename = $1 AND indexname LIKE 'idx_%'`,
			t.Name,
		)
		if err != nil {
			return fmt.Errorf("querying existing indexes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return fmt.Errorf("scanning index name: %w", err)
			}
			if !desired[name] {
				if opts.DryRun {
					log.Printf("[dry-run] would drop index %q from table %q", name, t.Name)
				} else {
					if _, err := tx.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %q`, name)); err != nil {
						return fmt.Errorf("dropping index %q: %w", name, err)
					}
				}
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterating indexes: %w", err)
		}
	}

	return nil
}

// cleanupTables drops tables in _meta that are not in the current config.
func cleanupTables(tx *sql.Tx, configuredTables map[string]bool, dryRun bool) error {
	rows, err := tx.Query(`SELECT table_name FROM _meta`)
	if err != nil {
		return fmt.Errorf("querying _meta: %w", err)
	}
	defer rows.Close()

	var toDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scanning table name: %w", err)
		}
		if name == metaConfigHashRowName {
			continue
		}
		if !configuredTables[name] {
			toDrop = append(toDrop, name)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating _meta: %w", err)
	}

	for _, name := range toDrop {
		if dryRun {
			log.Printf("[dry-run] would drop table %q and remove _meta entry", name)
		} else {
			if _, err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %q`, name)); err != nil {
				return fmt.Errorf("dropping table %q: %w", name, err)
			}
			if _, err := tx.Exec(`DELETE FROM _meta WHERE table_name = $1`, name); err != nil {
				return fmt.Errorf("removing _meta entry for %q: %w", name, err)
			}
		}
	}

	return nil
}

func createTable(tx *sql.Tx, t config.TableConfig) error {
	var stmt string
	if t.RangeKey != nil {
		// PK+RK table: rk is NOT NULL with composite primary key
		stmt = fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q (
				pk TEXT NOT NULL,
				rk TEXT NOT NULL,
				data JSONB NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
				PRIMARY KEY (pk, rk)
			)`,
			t.Name,
		)
	} else {
		// PK-only table: rk is nullable
		stmt = fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q (
				pk TEXT NOT NULL,
				rk TEXT,
				data JSONB NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
				PRIMARY KEY (pk)
			)`,
			t.Name,
		)
	}

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func createIndex(tx *sql.Tx, t config.TableConfig, idx config.IndexConfig) error {
	if idx.RangeKey != nil {
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %q ON %q ((data->>%s), (data->>%s)) WHERE data->>%s IS NOT NULL AND data->>%s IS NOT NULL`,
			fmt.Sprintf("idx_%s_%s", t.Name, idx.Name),
			t.Name,
			quoteStringLiteral(idx.PrimaryKey.Field),
			quoteStringLiteral(idx.RangeKey.Field),
			quoteStringLiteral(idx.PrimaryKey.Field),
			quoteStringLiteral(idx.RangeKey.Field),
		)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	} else {
		stmt := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %q ON %q ((data->>%s)) WHERE data->>%s IS NOT NULL`,
			fmt.Sprintf("idx_%s_%s", t.Name, idx.Name),
			t.Name,
			quoteStringLiteral(idx.PrimaryKey.Field),
			quoteStringLiteral(idx.PrimaryKey.Field),
		)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	return nil
}

// quoteStringLiteral wraps a value in single quotes for use as a SQL string literal.
// It escapes any embedded single quotes by doubling them.
func quoteStringLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('\'')
	for _, c := range s {
		if c == '\'' {
			b.WriteString("''")
		} else {
			b.WriteRune(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
