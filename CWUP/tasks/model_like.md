# model_like

## Goal
DB query functions for the `likes` table with fingerprint-based deduplication.

## Tasks

### 1 — `internal/model/like.go`
```go
func Fingerprint(r *http.Request) string
// SHA256(IP + User-Agent) → hex string

func AddLike(db *sql.DB, chapterID int, fingerprint string) (newCount int, alreadyLiked bool, err error)
// INSERT OR IGNORE into likes, then return current like_count from chapters
// Also calls IncrementLikes if not already liked
```

## Acceptance Criteria
- Same IP + User-Agent cannot like the same chapter twice.
- Returns `alreadyLiked = true` without error on duplicate attempt.
- `newCount` reflects the updated `like_count` from `chapters`.
