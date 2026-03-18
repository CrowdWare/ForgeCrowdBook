package model

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Book struct {
	ID          int
	Slug        string
	Title       string
	Description string
	CreatedAt   time.Time
}

func ListBooks(db *sql.DB) ([]Book, error) {
	rows, err := db.Query(`
		SELECT id, slug, title, description, created_at
		FROM books
		ORDER BY title ASC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	defer rows.Close()

	books := []Book{}
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.ID, &book.Slug, &book.Title, &book.Description, &book.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan book: %w", err)
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate books: %w", err)
	}

	return books, nil
}

func GetBookByID(db *sql.DB, id int) (*Book, error) {
	row := db.QueryRow(`
		SELECT id, slug, title, description, created_at
		FROM books
		WHERE id = ?;
	`, id)
	return scanBook(row)
}

func GetBookBySlug(db *sql.DB, slug string) (*Book, error) {
	row := db.QueryRow(`
		SELECT id, slug, title, description, created_at
		FROM books
		WHERE slug = ?;
	`, slug)
	return scanBook(row)
}

func scanBook(row *sql.Row) (*Book, error) {
	var book Book
	if err := row.Scan(&book.ID, &book.Slug, &book.Title, &book.Description, &book.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan book: %w", err)
	}
	return &book, nil
}
