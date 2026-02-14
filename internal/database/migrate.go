package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
)

// Migrate creates or updates tables and indexes based on the provided configuration.
// All operations run inside a single transaction.
func Migrate(db *sql.DB, tables []config.TableConfig) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	for _, t := range tables {
		if err := createTable(tx, t); err != nil {
			return fmt.Errorf("table %q: %w", t.Name, err)
		}
		if err := createIndexes(tx, t); err != nil {
			return fmt.Errorf("table %q indexes: %w", t.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	return nil
}

func createTable(tx *sql.Tx, t config.TableConfig) error {
	var stmt string
	if t.RK != nil {
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

func createIndexes(tx *sql.Tx, t config.TableConfig) error {
	for _, idx := range t.Indexes {
		if idx.RK != nil {
			// PK+RK GSI: composite index on two JSONB fields
			stmt := fmt.Sprintf(
				`CREATE INDEX IF NOT EXISTS %q ON %q ((data->>%s), (data->>%s)) WHERE data->>%s IS NOT NULL AND data->>%s IS NOT NULL`,
				fmt.Sprintf("idx_%s_%s", t.Name, idx.Name),
				t.Name,
				quoteStringLiteral(idx.PK.Field),
				quoteStringLiteral(idx.RK.Field),
				quoteStringLiteral(idx.PK.Field),
				quoteStringLiteral(idx.RK.Field),
			)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("index %q: %w", idx.Name, err)
			}
		} else {
			// PK-only GSI: index on single JSONB field
			stmt := fmt.Sprintf(
				`CREATE INDEX IF NOT EXISTS %q ON %q ((data->>%s)) WHERE data->>%s IS NOT NULL`,
				fmt.Sprintf("idx_%s_%s", t.Name, idx.Name),
				t.Name,
				quoteStringLiteral(idx.PK.Field),
				quoteStringLiteral(idx.PK.Field),
			)
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("index %q: %w", idx.Name, err)
			}
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
