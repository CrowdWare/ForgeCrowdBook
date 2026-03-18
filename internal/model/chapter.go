package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var slugNonAlphaNumeric = regexp.MustCompile(`[^a-z0-9-]+`)
var slugMultiDash = regexp.MustCompile(`-+`)

type Chapter struct {
	ID          int
	BookID      int
	AuthorID    int
	AuthorName  string
	Title       string
	Slug        string
	PathLabel   string
	SourceURL   string
	Status      string
	LikeCount   int
	CreatedAt   time.Time
	PublishedAt *time.Time
}

func GetChapterByID(db *sql.DB, id int) (*Chapter, error) {
	row := db.QueryRow(chapterSelect+`
		WHERE c.id = ?;
	`, id)
	return scanChapter(row)
}

func GetChapterBySlug(db *sql.DB, slug string) (*Chapter, error) {
	row := db.QueryRow(chapterSelect+`
		WHERE c.slug = ?;
	`, slug)
	return scanChapter(row)
}

func ListChaptersByBook(db *sql.DB, bookID int) ([]Chapter, error) {
	rows, err := db.Query(chapterSelect+`
		WHERE c.book_id = ?
		ORDER BY c.created_at ASC;
	`, bookID)
	if err != nil {
		return nil, fmt.Errorf("list chapters by book: %w", err)
	}
	defer rows.Close()

	return scanChapterRows(rows)
}

func ListChaptersByAuthor(db *sql.DB, authorID int) ([]Chapter, error) {
	rows, err := db.Query(chapterSelect+`
		WHERE c.author_id = ?
		ORDER BY c.created_at DESC;
	`, authorID)
	if err != nil {
		return nil, fmt.Errorf("list chapters by author: %w", err)
	}
	defer rows.Close()

	return scanChapterRows(rows)
}

func ListPublishedChaptersByBook(db *sql.DB, bookID int) ([]Chapter, error) {
	rows, err := db.Query(chapterSelect+`
		WHERE c.book_id = ? AND c.status = 'published'
		ORDER BY c.published_at ASC, c.created_at ASC;
	`, bookID)
	if err != nil {
		return nil, fmt.Errorf("list published chapters by book: %w", err)
	}
	defer rows.Close()

	return scanChapterRows(rows)
}

func CreateChapter(db *sql.DB, bookID, authorID int, title, slug, pathLabel, sourceURL string) (*Chapter, error) {
	result, err := db.Exec(`
		INSERT INTO chapters (book_id, author_id, title, slug, path_label, source_url)
		VALUES (?, ?, ?, ?, ?, ?);
	`, bookID, authorID, title, slug, pathLabel, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("insert chapter: %w", err)
	}

	id64, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read chapter id: %w", err)
	}

	return GetChapterByID(db, int(id64))
}

func UpdateChapter(db *sql.DB, id int, title, pathLabel, sourceURL string) error {
	_, err := db.Exec(`
		UPDATE chapters
		SET title = ?, path_label = ?, source_url = ?
		WHERE id = ?;
	`, title, pathLabel, sourceURL, id)
	if err != nil {
		return fmt.Errorf("update chapter: %w", err)
	}
	return nil
}

func SubmitChapter(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE chapters
		SET status = 'pending_review'
		WHERE id = ?;
	`, id)
	if err != nil {
		return fmt.Errorf("submit chapter: %w", err)
	}
	return nil
}

func PublishChapter(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE chapters
		SET status = 'published', published_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`, id)
	if err != nil {
		return fmt.Errorf("publish chapter: %w", err)
	}
	return nil
}

func RejectChapter(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE chapters
		SET status = 'rejected'
		WHERE id = ?;
	`, id)
	if err != nil {
		return fmt.Errorf("reject chapter: %w", err)
	}
	return nil
}

func DeleteChapter(db *sql.DB, id int) error {
	_, err := db.Exec(`
		DELETE FROM chapters
		WHERE id = ?;
	`, id)
	if err != nil {
		return fmt.Errorf("delete chapter: %w", err)
	}
	return nil
}

func IncrementLikes(db *sql.DB, id int) (int, error) {
	_, err := db.Exec(`
		UPDATE chapters
		SET like_count = like_count + 1
		WHERE id = ?;
	`, id)
	if err != nil {
		return 0, fmt.Errorf("increment chapter likes: %w", err)
	}

	var count int
	if err := db.QueryRow(`
		SELECT like_count
		FROM chapters
		WHERE id = ?;
	`, id).Scan(&count); err != nil {
		return 0, fmt.Errorf("read chapter like count: %w", err)
	}
	return count, nil
}

func GenerateSlug(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	base = strings.ReplaceAll(base, " ", "-")
	base = slugNonAlphaNumeric.ReplaceAllString(base, "")
	base = slugMultiDash.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "chapter"
	}
	return base + "-" + randomSuffix(3)
}

func randomSuffix(bytesLen int) string {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())[:bytesLen*2]
	}
	return hex.EncodeToString(buf)
}

func scanChapterRows(rows *sql.Rows) ([]Chapter, error) {
	chapters := []Chapter{}
	for rows.Next() {
		chapter, err := scanChapterFromScanner(rows)
		if err != nil {
			return nil, err
		}
		chapters = append(chapters, *chapter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chapters: %w", err)
	}
	return chapters, nil
}

func scanChapter(row *sql.Row) (*Chapter, error) {
	chapter, err := scanChapterFromScanner(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return chapter, nil
}

type chapterScanner interface {
	Scan(dest ...any) error
}

func scanChapterFromScanner(scanner chapterScanner) (*Chapter, error) {
	var (
		chapter     Chapter
		publishedAt sql.NullTime
	)

	if err := scanner.Scan(
		&chapter.ID,
		&chapter.BookID,
		&chapter.AuthorID,
		&chapter.AuthorName,
		&chapter.Title,
		&chapter.Slug,
		&chapter.PathLabel,
		&chapter.SourceURL,
		&chapter.Status,
		&chapter.LikeCount,
		&chapter.CreatedAt,
		&publishedAt,
	); err != nil {
		return nil, fmt.Errorf("scan chapter: %w", err)
	}

	if publishedAt.Valid {
		chapter.PublishedAt = &publishedAt.Time
	}

	return &chapter, nil
}

const chapterSelect = `
	SELECT
		c.id,
		c.book_id,
		c.author_id,
		COALESCE(u.display_name, '') AS author_name,
		c.title,
		c.slug,
		c.path_label,
		c.source_url,
		c.status,
		c.like_count,
		c.created_at,
		c.published_at
	FROM chapters c
	LEFT JOIN users u ON u.id = c.author_id
`
