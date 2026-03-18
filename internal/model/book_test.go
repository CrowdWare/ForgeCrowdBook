package model

import "testing"

func TestGetBookBySlugNotFoundReturnsNil(t *testing.T) {
	database := openModelTestDB(t)

	book, err := GetBookBySlug(database, "missing")
	if err != nil {
		t.Fatalf("GetBookBySlug failed: %v", err)
	}
	if book != nil {
		t.Fatalf("expected nil book, got %+v", book)
	}
}

func TestListBooksOrdersByTitle(t *testing.T) {
	database := openModelTestDB(t)

	if _, err := database.Exec(`
		INSERT INTO books (slug, title, description)
		VALUES
			('zeta', 'Zeta', ''),
			('alpha', 'Alpha', '');
	`); err != nil {
		t.Fatalf("insert books: %v", err)
	}

	books, err := ListBooks(database)
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}
	if len(books) < 3 {
		t.Fatalf("expected at least 3 books, got %d", len(books))
	}
	if books[0].Title != "Alpha" {
		t.Fatalf("expected first title Alpha, got %q", books[0].Title)
	}
}
