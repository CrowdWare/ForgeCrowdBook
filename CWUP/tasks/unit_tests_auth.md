# unit_tests_auth

## Goal
Unit tests for `internal/auth` (tokens and sessions).

## Tasks

### Token tests (`internal/auth/auth_test.go`)
- `GenerateToken` → 64 hex chars, unique on repeated calls
- `CreateMagicToken` → token stored in DB, expires in ~15min
- `ValidateToken` valid → returns email, marks used
- `ValidateToken` expired → `ErrInvalidToken`
- `ValidateToken` already used → `ErrInvalidToken`
- `ValidateToken` unknown token → `ErrInvalidToken`
- Rate limit: 4th call within 1h → `ErrRateLimited`

### Session tests
- `CreateSession` + `GetSession` round-trip → returns email
- Tampered cookie → `ok = false`
- Expired session → `ok = false`

## Acceptance Criteria
- All tests use an in-memory SQLite DB (`file::memory:?cache=shared`)
- No real email sent
