package auth

import (
	"database/sql"
	"errors"
	"path/filepath"
	"regexp"
	"testing"

	"codeberg.org/crowdware/forgecrowdbook/internal/db"
)

func TestGenerateTokenFormat(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected token length 64, got %d", len(token))
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(token) {
		t.Fatalf("token is not lowercase hex: %q", token)
	}
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	database := openTestDB(t)

	_, err := database.Exec(`
		INSERT INTO magic_tokens (email, token, expires_at, used, created_at)
		VALUES ('user@example.com', 'expiredtoken', datetime('now', '-1 minute'), 0, datetime('now'));
	`)
	if err != nil {
		t.Fatalf("seed expired token failed: %v", err)
	}

	_, err = ValidateToken(database, "expiredtoken")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}

func TestValidateTokenSingleUse(t *testing.T) {
	database := openTestDB(t)

	token, err := CreateMagicToken(database, "user@example.com")
	if err != nil {
		t.Fatalf("CreateMagicToken failed: %v", err)
	}

	email, err := ValidateToken(database, token)
	if err != nil {
		t.Fatalf("ValidateToken first use failed: %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected email: %s", email)
	}

	_, err = ValidateToken(database, token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on reuse, got: %v", err)
	}
}

func TestCreateMagicTokenRateLimit(t *testing.T) {
	database := openTestDB(t)

	email := "limited@example.com"
	for i := 0; i < 3; i++ {
		if _, err := CreateMagicToken(database, email); err != nil {
			t.Fatalf("CreateMagicToken attempt %d failed: %v", i+1, err)
		}
	}

	_, err := CreateMagicToken(database, email)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited on 4th request, got: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}
