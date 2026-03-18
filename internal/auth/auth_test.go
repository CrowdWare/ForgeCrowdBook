package auth

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	_ "modernc.org/sqlite"
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

	other, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken second call failed: %v", err)
	}
	if token == other {
		t.Fatal("expected unique token values")
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

func TestCreateMagicTokenStoresAndExpiresInAbout15Minutes(t *testing.T) {
	database := openTestDB(t)

	token, err := CreateMagicToken(database, "expires@example.com")
	if err != nil {
		t.Fatalf("CreateMagicToken failed: %v", err)
	}

	var (
		storedToken string
		expiresAt   time.Time
	)
	if err := database.QueryRow(`
		SELECT token, expires_at
		FROM magic_tokens
		WHERE email = ?
		ORDER BY id DESC
		LIMIT 1;
	`, "expires@example.com").Scan(&storedToken, &expiresAt); err != nil {
		t.Fatalf("query stored token: %v", err)
	}
	if storedToken != token {
		t.Fatalf("stored token mismatch: got %q want %q", storedToken, token)
	}

	remaining := time.Until(expiresAt)
	if remaining < 13*time.Minute || remaining > 16*time.Minute {
		t.Fatalf("expected ~15m ttl, got %s", remaining)
	}
}

func TestValidateTokenRejectsUnknownToken(t *testing.T) {
	database := openTestDB(t)

	_, err := ValidateToken(database, "does-not-exist")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}

func TestValidateTokenRejectsAlreadyUsedToken(t *testing.T) {
	database := openTestDB(t)

	if _, err := database.Exec(`
		INSERT INTO magic_tokens (email, token, expires_at, used, created_at)
		VALUES ('used@example.com', 'used-token', datetime('now', '+15 minutes'), 1, datetime('now'));
	`); err != nil {
		t.Fatalf("seed used token failed: %v", err)
	}

	_, err := ValidateToken(database, "used-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
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

	database, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS magic_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}

	return database
}
