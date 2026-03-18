package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

import _ "modernc.org/sqlite"

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if _, err := database.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := database.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := applySchema(database); err != nil {
		_ = database.Close()
		return nil, err
	}

	return database, nil
}

func applySchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			bio TEXT DEFAULT '',
			lang TEXT NOT NULL DEFAULT 'en',
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'banned')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS magic_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS chapters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER NOT NULL REFERENCES books(id),
			author_id INTEGER NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			path_label TEXT DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'pending_review', 'published', 'rejected')),
			like_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			published_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chapter_id INTEGER NOT NULL REFERENCES chapters(id),
			fingerprint TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chapter_id, fingerprint)
		);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	if err := ensureMagicTokensCreatedAtColumn(db); err != nil {
		return err
	}

	if err := seedDefaultBook(db); err != nil {
		return err
	}

	return nil
}

func ensureMagicTokensCreatedAtColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(magic_tokens);`)
	if err != nil {
		return fmt.Errorf("inspect magic_tokens columns: %w", err)
	}
	defer rows.Close()

	hasCreatedAt := false
	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultV   sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultV, &primaryKey); err != nil {
			return fmt.Errorf("scan magic_tokens column metadata: %w", err)
		}
		if name == "created_at" {
			hasCreatedAt = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate magic_tokens columns: %w", err)
	}

	if hasCreatedAt {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE magic_tokens ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;`); err != nil {
		return fmt.Errorf("add magic_tokens.created_at column: %w", err)
	}

	return nil
}

func seedDefaultBook(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM books;`).Scan(&count); err != nil {
		return fmt.Errorf("count books: %w", err)
	}

	if count > 0 {
		return nil
	}

	_, err := db.Exec(`
		INSERT INTO books (slug, title, description)
		VALUES ('choose-your-incarnation', 'Choose Your Incarnation', 'A collaborative branching narrative.');
	`)
	if err != nil {
		return fmt.Errorf("seed default book: %w", err)
	}

	return nil
}
