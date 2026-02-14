package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Store provides CRUD operations against PostgreSQL tables.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ListOptions controls pagination and range-key filtering for list/scan/query operations.
type ListOptions struct {
	Limit        int
	Cursor       string // base64url-encoded JSON cursor
	RKBeginsWith string
	RKGt         string
	RKGte        string
	RKLt         string
	RKLte        string
}

// ListResult holds a page of items and an optional cursor for the next page.
type ListResult struct {
	Items      []map[string]interface{}
	NextCursor string // empty if no more pages
}

// IndexQueryConfig describes the key fields for a GSI query.
type IndexQueryConfig struct {
	PKField string
	RKField string // empty if pk-only GSI
}

// cursor is the internal representation of a pagination cursor.
type cursor struct {
	PK string `json:"pk"`
	RK string `json:"rk,omitempty"`
}

func encodeCursor(pk, rk string) string {
	c := cursor{PK: pk, RK: rk}
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return cursor{}, fmt.Errorf("invalid cursor format: %w", err)
	}
	return c, nil
}

// GetItem retrieves a single item by PK (and optionally RK).
func (s *Store) GetItem(ctx context.Context, table string, pk string, rk *string) (map[string]interface{}, error) {
	var row *sql.Row
	if rk != nil {
		row = s.db.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT data FROM %q WHERE pk = $1 AND rk = $2`, table),
			pk, *rk,
		)
	} else {
		row = s.db.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT data FROM %q WHERE pk = $1`, table),
			pk,
		)
	}

	var dataBytes []byte
	if err := row.Scan(&dataBytes); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return data, nil
}

// PutItem creates or replaces an item (full upsert).
// The data column stores the payload WITHOUT pk/rk fields.
func (s *Store) PutItem(ctx context.Context, table string, pk string, rk *string, data map[string]interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if rk != nil {
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(
				`INSERT INTO %q (pk, rk, data, created_at, updated_at)
				 VALUES ($1, $2, $3, now(), now())
				 ON CONFLICT (pk, rk) DO UPDATE SET data = $3, updated_at = now()`,
				table,
			),
			pk, *rk, dataBytes,
		)
	} else {
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(
				`INSERT INTO %q (pk, data, created_at, updated_at)
				 VALUES ($1, $2, now(), now())
				 ON CONFLICT (pk) DO UPDATE SET data = $2, updated_at = now()`,
				table,
			),
			pk, dataBytes,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}
	return nil
}

// DeleteItem deletes an item by PK (and optionally RK).
func (s *Store) DeleteItem(ctx context.Context, table string, pk string, rk *string) error {
	var err error
	if rk != nil {
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM %q WHERE pk = $1 AND rk = $2`, table),
			pk, *rk,
		)
	} else {
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(`DELETE FROM %q WHERE pk = $1`, table),
			pk,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return nil
}

// ListItems lists items in a partition with pagination and optional RK filtering.
func (s *Store) ListItems(ctx context.Context, table string, pk string, hasRK bool, opts ListOptions) (*ListResult, error) {
	where := []string{"pk = $1"}
	args := []interface{}{pk}
	argIdx := 2

	where, args, argIdx = appendRKFilters(where, args, argIdx, hasRK, opts)
	where, args, argIdx = appendCursorFilter(where, args, argIdx, hasRK, opts.Cursor, "rk")

	return s.queryItems(ctx, table, where, args, argIdx, opts.Limit, hasRK, "pk", "rk")
}

// ScanTable performs a full table scan with pagination.
func (s *Store) ScanTable(ctx context.Context, table string, hasRK bool, opts ListOptions) (*ListResult, error) {
	var where []string
	var args []interface{}
	argIdx := 1

	where, args, argIdx = appendCursorFilter(where, args, argIdx, hasRK, opts.Cursor, "rk")

	return s.queryItems(ctx, table, where, args, argIdx, opts.Limit, hasRK, "pk", "rk")
}

// QueryIndex queries a GSI by its partition key value.
func (s *Store) QueryIndex(ctx context.Context, table string, index IndexQueryConfig, indexPk string, opts ListOptions) (*ListResult, error) {
	pkExpr := fmt.Sprintf("data->>%s", quoteStringLiteral(index.PKField))

	where := []string{pkExpr + " = $1"}
	args := []interface{}{indexPk}
	argIdx := 2

	hasRK := index.RKField != ""
	rkExpr := ""
	if hasRK {
		rkExpr = fmt.Sprintf("data->>%s", quoteStringLiteral(index.RKField))
		where, args, argIdx = appendIndexRKFilters(where, args, argIdx, rkExpr, opts)
	}

	where, args, argIdx = appendCursorFilter(where, args, argIdx, hasRK, opts.Cursor, rkExpr)

	return s.queryItems(ctx, table, where, args, argIdx, opts.Limit, hasRK, pkExpr, rkExpr)
}

// ScanIndex performs a full index scan with pagination.
func (s *Store) ScanIndex(ctx context.Context, table string, index IndexQueryConfig, opts ListOptions) (*ListResult, error) {
	pkExpr := fmt.Sprintf("data->>%s", quoteStringLiteral(index.PKField))

	// Only include rows where the index pk field is present (sparse index)
	where := []string{pkExpr + " IS NOT NULL"}
	var args []interface{}
	argIdx := 1

	hasRK := index.RKField != ""
	rkExpr := ""
	if hasRK {
		rkExpr = fmt.Sprintf("data->>%s", quoteStringLiteral(index.RKField))
		where = append(where, rkExpr+" IS NOT NULL")
	}

	where, args, argIdx = appendCursorFilter(where, args, argIdx, hasRK, opts.Cursor, rkExpr)

	return s.queryItems(ctx, table, where, args, argIdx, opts.Limit, hasRK, pkExpr, rkExpr)
}

// GetItemByIndex retrieves a single item from a GSI by pk+rk.
func (s *Store) GetItemByIndex(ctx context.Context, table string, index IndexQueryConfig, indexPk string, indexRk string) (map[string]interface{}, error) {
	pkExpr := fmt.Sprintf("data->>%s", quoteStringLiteral(index.PKField))
	rkExpr := fmt.Sprintf("data->>%s", quoteStringLiteral(index.RKField))

	query := fmt.Sprintf(
		`SELECT data FROM %q WHERE %s = $1 AND %s = $2 LIMIT 1`,
		table, pkExpr, rkExpr,
	)

	row := s.db.QueryRowContext(ctx, query, indexPk, indexRk)

	var dataBytes []byte
	if err := row.Scan(&dataBytes); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by index: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return data, nil
}

// queryItems builds and executes a paginated SELECT query.
func (s *Store) queryItems(ctx context.Context, table string, where []string, args []interface{}, argIdx int, limit int, hasRK bool, pkExpr string, rkExpr string) (*ListResult, error) {
	if limit <= 0 {
		limit = 50
	}
	fetchLimit := limit + 1

	orderBy := pkExpr
	if hasRK {
		orderBy = pkExpr + ", " + rkExpr
	}

	query := fmt.Sprintf(`SELECT pk, rk, data FROM %q`, table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(` ORDER BY %s LIMIT $%d`, orderBy, argIdx)
	args = append(args, fetchLimit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	type rowData struct {
		pk   string
		rk   string
		data map[string]interface{}
	}
	var collected []rowData

	for rows.Next() {
		var pk string
		var rk sql.NullString
		var dataBytes []byte
		if err := rows.Scan(&pk, &rk, &dataBytes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		rkVal := ""
		if rk.Valid {
			rkVal = rk.String
		}
		collected = append(collected, rowData{pk: pk, rk: rkVal, data: data})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	result := &ListResult{}
	hasMore := len(collected) > limit
	if hasMore {
		collected = collected[:limit]
	}

	result.Items = make([]map[string]interface{}, len(collected))
	for i, r := range collected {
		result.Items[i] = r.data
	}

	if hasMore && len(collected) > 0 {
		last := collected[len(collected)-1]
		result.NextCursor = encodeCursor(last.pk, last.rk)
	}

	return result, nil
}

// appendRKFilters adds range key filter conditions for table queries.
func appendRKFilters(where []string, args []interface{}, argIdx int, hasRK bool, opts ListOptions) ([]string, []interface{}, int) {
	if !hasRK {
		return where, args, argIdx
	}
	if opts.RKBeginsWith != "" {
		where = append(where, fmt.Sprintf("rk LIKE $%d", argIdx))
		args = append(args, opts.RKBeginsWith+"%")
		argIdx++
	}
	if opts.RKGt != "" {
		where = append(where, fmt.Sprintf("rk > $%d", argIdx))
		args = append(args, opts.RKGt)
		argIdx++
	}
	if opts.RKGte != "" {
		where = append(where, fmt.Sprintf("rk >= $%d", argIdx))
		args = append(args, opts.RKGte)
		argIdx++
	}
	if opts.RKLt != "" {
		where = append(where, fmt.Sprintf("rk < $%d", argIdx))
		args = append(args, opts.RKLt)
		argIdx++
	}
	if opts.RKLte != "" {
		where = append(where, fmt.Sprintf("rk <= $%d", argIdx))
		args = append(args, opts.RKLte)
		argIdx++
	}
	return where, args, argIdx
}

// appendIndexRKFilters adds range key filter conditions for GSI queries using JSONB expressions.
func appendIndexRKFilters(where []string, args []interface{}, argIdx int, rkExpr string, opts ListOptions) ([]string, []interface{}, int) {
	if opts.RKBeginsWith != "" {
		where = append(where, fmt.Sprintf("%s LIKE $%d", rkExpr, argIdx))
		args = append(args, opts.RKBeginsWith+"%")
		argIdx++
	}
	if opts.RKGt != "" {
		where = append(where, fmt.Sprintf("%s > $%d", rkExpr, argIdx))
		args = append(args, opts.RKGt)
		argIdx++
	}
	if opts.RKGte != "" {
		where = append(where, fmt.Sprintf("%s >= $%d", rkExpr, argIdx))
		args = append(args, opts.RKGte)
		argIdx++
	}
	if opts.RKLt != "" {
		where = append(where, fmt.Sprintf("%s < $%d", rkExpr, argIdx))
		args = append(args, opts.RKLt)
		argIdx++
	}
	if opts.RKLte != "" {
		where = append(where, fmt.Sprintf("%s <= $%d", rkExpr, argIdx))
		args = append(args, opts.RKLte)
		argIdx++
	}
	return where, args, argIdx
}

// appendCursorFilter adds cursor-based pagination conditions.
func appendCursorFilter(where []string, args []interface{}, argIdx int, hasRK bool, cursorStr string, rkExpr string) ([]string, []interface{}, int) {
	if cursorStr == "" {
		return where, args, argIdx
	}
	c, err := decodeCursor(cursorStr)
	if err != nil {
		// Invalid cursor is silently ignored; start from the beginning
		return where, args, argIdx
	}

	if hasRK && rkExpr != "" {
		// For composite ordering: (pk, rk) > ($a, $b)
		where = append(where, fmt.Sprintf("(pk, %s) > ($%d, $%d)", rkExpr, argIdx, argIdx+1))
		args = append(args, c.PK, c.RK)
		argIdx += 2
	} else {
		where = append(where, fmt.Sprintf("pk > $%d", argIdx))
		args = append(args, c.PK)
		argIdx++
	}

	return where, args, argIdx
}
