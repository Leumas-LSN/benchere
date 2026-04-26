package db

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct{ *sql.DB }

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if err := runMigrations(conn); err != nil {
		return nil, err
	}
	return &DB{conn}, nil
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE name = ?", e.Name()).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", e.Name(), err)
		}
		if count > 0 {
			continue // already applied
		}

		data, err := migrations.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", e.Name(), err)
		}

		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", e.Name(), err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations(name) VALUES(?)", e.Name()); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", e.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", e.Name(), err)
		}
	}
	return nil
}
