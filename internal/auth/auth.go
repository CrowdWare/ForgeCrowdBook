package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrRateLimited  = errors.New("magic link rate limit exceeded")
	ErrInvalidToken = errors.New("invalid or expired token")
)

func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func CreateMagicToken(db *sql.DB, email string) (string, error) {
	var recentCount int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM magic_tokens
		WHERE email = ?
		  AND created_at > datetime('now', '-1 hour');
	`, email).Scan(&recentCount)
	if err != nil {
		return "", fmt.Errorf("count recent magic tokens: %w", err)
	}
	if recentCount >= 3 {
		return "", ErrRateLimited
	}

	token, err := GenerateToken()
	if err != nil {
		return "", err
	}

	_, err = db.Exec(`
		INSERT INTO magic_tokens (email, token, expires_at, used, created_at)
		VALUES (?, ?, datetime('now', '+15 minutes'), 0, datetime('now'));
	`, email, token)
	if err != nil {
		return "", fmt.Errorf("insert magic token: %w", err)
	}

	return token, nil
}

func ValidateToken(db *sql.DB, token string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var email string
	err = tx.QueryRow(`
		SELECT email
		FROM magic_tokens
		WHERE token = ?
		  AND used = 0
		  AND expires_at > datetime('now')
		LIMIT 1;
	`, token).Scan(&email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidToken
		}
		return "", fmt.Errorf("query magic token: %w", err)
	}

	result, err := tx.Exec(`
		UPDATE magic_tokens
		SET used = 1
		WHERE token = ?
		  AND used = 0;
	`, token)
	if err != nil {
		return "", fmt.Errorf("mark magic token as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("read rows affected: %w", err)
	}
	if rowsAffected != 1 {
		return "", ErrInvalidToken
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return email, nil
}
