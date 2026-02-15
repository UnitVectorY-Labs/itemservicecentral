package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection and verifies it with a ping.
func Connect(host string, port int, database string, username string, password string, sslmode string) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		host, port, database, username, password, sslmode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}
