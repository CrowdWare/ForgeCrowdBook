package model

import (
	"database/sql"
	"regexp"
	"strings"
	"testing"
)

func TestCreateAndUpdateChapterStoresSourceURLAsIs(t *testing.T) {
	database := openModelTestDB(t)
	authorID := mustCreateAuthor(t, database, "author1@example.com")
	bookID := mustGetDefaultBookID(t, database)

	sourceURL := "ipfs://QmExample/Story.md"
	chapter, err := CreateChapter(database, bookID, authorID, "One", "one", "A", sourceURL)
	if err != nil {
		t.Fatalf("CreateChapter failed: %v", err)
	}
	if chapter.SourceURL != sourceURL {
		t.Fatalf("expected source_url %q, got %q", sourceURL, chapter.SourceURL)
	}

	updatedURL := "https://example.com/new.md"
	if err := UpdateChapter(database, chapter.ID, "One+", "B", updatedURL); err != nil {
		t.Fatalf("UpdateChapter failed: %v", err)
	}

	loaded, err := GetChapterByID(database, chapter.ID)
	if err != nil {
		t.Fatalf("GetChapterByID failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected chapter")
	}
	if loaded.SourceURL != updatedURL {
		t.Fatalf("expected updated source_url %q, got %q", updatedURL, loaded.SourceURL)
	}
	if loaded.Title != "One+" || loaded.PathLabel != "B" {
		t.Fatalf("unexpected updated fields: %+v", loaded)
	}
}

func TestChapterStatusFlowAndPublishedList(t *testing.T) {
	database := openModelTestDB(t)
	authorID := mustCreateAuthor(t, database, "author2@example.com")
	bookID := mustGetDefaultBookID(t, database)

	chapter, err := CreateChapter(database, bookID, authorID, "Flow", "flow", "Path", "https://example.com/flow.md")
	if err != nil {
		t.Fatalf("CreateChapter failed: %v", err)
	}

	if err := SubmitChapter(database, chapter.ID); err != nil {
		t.Fatalf("SubmitChapter failed: %v", err)
	}
	if err := PublishChapter(database, chapter.ID); err != nil {
		t.Fatalf("PublishChapter failed: %v", err)
	}

	loaded, err := GetChapterByID(database, chapter.ID)
	if err != nil {
		t.Fatalf("GetChapterByID failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected chapter")
	}
	if loaded.Status != "published" {
		t.Fatalf("expected published, got %q", loaded.Status)
	}
	if loaded.PublishedAt == nil {
		t.Fatal("expected published_at to be set")
	}

	published, err := ListPublishedChaptersByBook(database, bookID)
	if err != nil {
		t.Fatalf("ListPublishedChaptersByBook failed: %v", err)
	}
	if len(published) != 1 {
		t.Fatalf("expected 1 published chapter, got %d", len(published))
	}

	if err := RejectChapter(database, chapter.ID); err != nil {
		t.Fatalf("RejectChapter failed: %v", err)
	}
	rejected, err := GetChapterByID(database, chapter.ID)
	if err != nil {
		t.Fatalf("GetChapterByID after reject failed: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("expected rejected, got %q", rejected.Status)
	}

	if err := DeleteChapter(database, chapter.ID); err != nil {
		t.Fatalf("DeleteChapter failed: %v", err)
	}
	removed, err := GetChapterByID(database, chapter.ID)
	if err != nil {
		t.Fatalf("GetChapterByID after delete failed: %v", err)
	}
	if removed != nil {
		t.Fatalf("expected chapter to be deleted, got %+v", removed)
	}
}

func TestGenerateSlug(t *testing.T) {
	slug := GenerateSlug(" Hello, World! ")
	if !strings.HasPrefix(slug, "hello-world-") {
		t.Fatalf("unexpected slug prefix: %q", slug)
	}
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(slug) {
		t.Fatalf("slug contains invalid chars: %q", slug)
	}
	if len(slug) < len("hello-world-000000") {
		t.Fatalf("slug too short: %q", slug)
	}
}

func mustCreateAuthor(t *testing.T, database *sql.DB, email string) int {
	t.Helper()
	user, err := CreateUser(database, email, "Author")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	return user.ID
}

func mustGetDefaultBookID(t *testing.T, database *sql.DB) int {
	t.Helper()
	book, err := GetBookBySlug(database, "choose-your-incarnation")
	if err != nil {
		t.Fatalf("GetBookBySlug failed: %v", err)
	}
	if book == nil {
		t.Fatal("default book missing")
	}
	return book.ID
}
