# model_book

## Goal
DB query functions for the `books` table.

## Tasks

### 1 — `internal/model/book.go`
```go
type Book struct {
    ID          int
    Slug        string
    Title       string
    Description string
    CreatedAt   time.Time
}

func ListBooks(db *sql.DB) ([]Book, error)
func GetBookByID(db *sql.DB, id int) (*Book, error)
func GetBookBySlug(db *sql.DB, slug string) (*Book, error)
```

## Acceptance Criteria
- `GetBookBySlug` returns `nil, nil` when not found.
- `ListBooks` returns all books ordered by title.
