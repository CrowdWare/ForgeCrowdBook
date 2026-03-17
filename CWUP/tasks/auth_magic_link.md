# auth_magic_link

## Goal
Implement Magic Link token generation, email dispatch, and token validation.

## Context
No passwords. The email is the identity. Token is 64 hex chars (32 random bytes),
valid for 15 minutes, single-use.

## Tasks

### 1 — `internal/auth/auth.go`: Token generation
```go
func GenerateToken() (string, error)
// crypto/rand 32 bytes → hex string (64 chars)
```

### 2 — Store token in DB
```go
func CreateMagicToken(db *sql.DB, email string) (token string, err error)
```
- Check rate limit: max 3 tokens for this email in the last hour → return `ErrRateLimited`.
- Generate token.
- Insert into `magic_tokens` with `expires_at = now + 15min`, `used = 0`.

### 3 — Validate token
```go
func ValidateToken(db *sql.DB, token string) (email string, err error)
```
- Query: `token = ? AND used = 0 AND expires_at > now`.
- If not found: return `ErrInvalidToken`.
- Mark `used = 1`.
- Return `email`.

### 4 — Registration + Login handlers wire-up
`POST /register`: create user if not exists, call `CreateMagicToken`, send email, redirect to confirmation page.
`POST /login`: call `CreateMagicToken` for existing email, send email, redirect.
`GET /auth?token=...`: call `ValidateToken`, create session, redirect to `/dashboard`.

## Acceptance Criteria
- Token is exactly 64 hex chars.
- Expired token returns `ErrInvalidToken`.
- Used token cannot be reused.
- 4th request within one hour returns `ErrRateLimited`.
- Unknown email on `/login` shows generic "check your email" (no user enumeration).
