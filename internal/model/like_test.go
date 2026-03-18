package model

import (
	"net/http/httptest"
	"testing"
)

func TestFingerprintStableForSameInput(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "203.0.113.10:1234"
	req1.Header.Set("User-Agent", "TestAgent/1.0")

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "203.0.113.10:9999"
	req2.Header.Set("User-Agent", "TestAgent/1.0")

	if Fingerprint(req1) != Fingerprint(req2) {
		t.Fatal("expected identical fingerprints")
	}
}

func TestAddLikeDeduplicatesByFingerprint(t *testing.T) {
	database := openModelTestDB(t)
	authorID := mustCreateAuthor(t, database, "author-like@example.com")
	bookID := mustGetDefaultBookID(t, database)

	chapter, err := CreateChapter(database, bookID, authorID, "Likes", "likes", "Path", "https://example.com/likes.md")
	if err != nil {
		t.Fatalf("CreateChapter failed: %v", err)
	}

	count, alreadyLiked, err := AddLike(database, chapter.ID, "abc")
	if err != nil {
		t.Fatalf("AddLike first failed: %v", err)
	}
	if alreadyLiked {
		t.Fatal("expected first like to be new")
	}
	if count != 1 {
		t.Fatalf("expected like count 1, got %d", count)
	}

	count, alreadyLiked, err = AddLike(database, chapter.ID, "abc")
	if err != nil {
		t.Fatalf("AddLike duplicate failed: %v", err)
	}
	if !alreadyLiked {
		t.Fatal("expected duplicate like")
	}
	if count != 1 {
		t.Fatalf("expected like count to stay 1, got %d", count)
	}
}
