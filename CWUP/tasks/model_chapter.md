# model_chapter

## Goal
DB query functions for the `chapters` table.

## Context
Chapters no longer store Markdown content in the database. Each chapter has a `source_url`
pointing to the author's own hosted Markdown file. The model layer stores and retrieves the
URL as-is — no normalization, no fetching.

## Tasks

### 1 — `internal/model/chapter.go`
```go
type Chapter struct {
    ID          int
    BookID      int
    AuthorID    int
    AuthorName  string    // joined from users
    Title       string
    Slug        string
    PathLabel   string
    SourceURL   string    // raw URL to the author's Markdown file
    Status      string
    LikeCount   int
    CreatedAt   time.Time
    PublishedAt *time.Time
}

func GetChapterByID(db *sql.DB, id int) (*Chapter, error)
func GetChapterBySlug(db *sql.DB, slug string) (*Chapter, error)
func ListChaptersByBook(db *sql.DB, bookID int) ([]Chapter, error)
func ListChaptersByAuthor(db *sql.DB, authorID int) ([]Chapter, error)
func ListPublishedChaptersByBook(db *sql.DB, bookID int) ([]Chapter, error)
func CreateChapter(db *sql.DB, bookID, authorID int, title, slug, pathLabel, sourceURL string) (*Chapter, error)
func UpdateChapter(db *sql.DB, id int, title, pathLabel, sourceURL string) error
func SubmitChapter(db *sql.DB, id int) error   // status → 'pending_review'
func PublishChapter(db *sql.DB, id int) error  // status → 'published', set published_at
func RejectChapter(db *sql.DB, id int) error   // status → 'rejected'
func DeleteChapter(db *sql.DB, id int) error
func IncrementLikes(db *sql.DB, id int) (newCount int, err error)
```

### 2 — Slug generation helper
```go
func GenerateSlug(title string) string
```
- Lowercase, replace spaces with `-`, strip non-alphanumeric except `-`.
- Append short random suffix to avoid collisions.

## Acceptance Criteria
- `SourceURL` is stored as-is — no normalization in the model layer.
- IPFS URL normalization (`ipfs://` → gateway URL) happens in the fetcher layer, not here.
- `UpdateChapter` only touches `title`, `path_label`, `source_url`.
- All queries use parameterized statements.
