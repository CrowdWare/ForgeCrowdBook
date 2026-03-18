package model

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
)

func Fingerprint(r *http.Request) string {
	if r == nil {
		sum := sha256.Sum256([]byte(""))
		return hex.EncodeToString(sum[:])
	}

	ip := requestIP(r)
	input := ip + r.UserAgent()
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func AddLike(db *sql.DB, chapterID int, fingerprint string) (newCount int, alreadyLiked bool, err error) {
	result, err := db.Exec(`
		INSERT OR IGNORE INTO likes (chapter_id, fingerprint)
		VALUES (?, ?);
	`, chapterID, fingerprint)
	if err != nil {
		return 0, false, fmt.Errorf("insert like: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, false, fmt.Errorf("read like insert result: %w", err)
	}

	if rowsAffected == 0 {
		count, err := chapterLikeCount(db, chapterID)
		if err != nil {
			return 0, true, err
		}
		return count, true, nil
	}

	count, err := IncrementLikes(db, chapterID)
	if err != nil {
		return 0, false, err
	}
	return count, false, nil
}

func chapterLikeCount(db *sql.DB, chapterID int) (int, error) {
	var count int
	if err := db.QueryRow(`
		SELECT like_count
		FROM chapters
		WHERE id = ?;
	`, chapterID).Scan(&count); err != nil {
		return 0, fmt.Errorf("read chapter like count: %w", err)
	}
	return count, nil
}

func requestIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
