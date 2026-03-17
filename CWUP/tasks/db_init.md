# db_init

## Goal
Initialize SQLite database with schema and provide a thin query layer.

## Context
Uses `modernc.org/sqlite` (pure Go, no CGO). DB file path comes from config.
Schema must be applied via `CREATE TABLE IF NOT EXISTS` on every startup — no separate migration tool.

## Tasks

### 1 — `internal/db/db.go`: Open and initialize DB
```go
func Open(path string) (*sql.DB, error)
```
- Opens SQLite file at `path`.
- Sets `PRAGMA journal_mode=WAL` and `PRAGMA foreign_keys=ON`.
- Calls `applySchema(db)`.

### 2 — `applySchema(db *sql.DB) error`
Apply all `CREATE TABLE IF NOT EXISTS` statements:
- `users`
- `magic_tokens`
- `books`
- `chapters`
- `likes`

See `SPEC.md` section "Database Schema" for full SQL.

### 3 — Seed default book on first run
If `books` table is empty, insert:
```sql
INSERT INTO books (slug, title, description)
VALUES ('choose-your-incarnation', 'Choose Your Incarnation', 'A collaborative branching narrative.')
```

## Acceptance Criteria
- DB opens and schema is applied without error on a fresh file.
- Re-running on existing DB causes no errors (`IF NOT EXISTS`).
- Foreign key constraints are enforced.
- WAL mode enabled for concurrent reads.
