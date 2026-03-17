# go_module_setup

## Goal
Initialize the Go module and directory structure for ForgeCrowdBook.

## Tasks

### 1 — `go mod init`
```
go mod init codeberg.org/crowdware/forgecrowdbook
```

### 2 — Add dependencies
```
go get modernc.org/sqlite
go get codeberg.org/crowdware/sml-go
```

### 3 — Create directory skeleton
```
internal/config/
internal/db/
internal/auth/
internal/i18n/
internal/handler/
internal/model/
internal/mailer/
i18n/
templates/
static/
data/
CWUP/
```

### 4 — Create `data/.gitkeep`
SQLite DB file goes here, must not be committed.

### 5 — Update `.gitignore`
```
data/*.db
data/*.db-shm
data/*.db-wal
```

### 6 — Stub `main.go`
Minimal entry point that loads config and prints version — no HTTP yet.

## Acceptance Criteria
- `go build ./...` succeeds
- `go mod tidy` produces no warnings
