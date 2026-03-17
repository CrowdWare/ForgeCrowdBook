# auth_session

## Goal
Implement signed cookie sessions and authentication middleware.

## Context
Session is a signed cookie (HMAC-SHA256). No server-side session store needed.
Cookie contains: `email|expires_unix` signed with `session_secret` from config.

## Tasks

### 1 — `internal/auth/session.go`
```go
func CreateSession(w http.ResponseWriter, email string, secret string)
func GetSession(r *http.Request, secret string) (email string, ok bool)
func ClearSession(w http.ResponseWriter)
```
- `CreateSession`: encode `email + "|" + unixTimestamp`, HMAC-SHA256 sign, base64url encode, set cookie (HttpOnly, SameSite=Lax, 30 days).
- `GetSession`: decode cookie, verify HMAC, check expiry, return email.
- `ClearSession`: set expired cookie.

### 2 — Admin detection
Admin = email matches `config.AdminEmail`.
```go
func IsAdmin(email, adminEmail string) bool
```

### 3 — Middleware: `RequireAuth`
```go
func RequireAuth(secret string, next http.Handler) http.Handler
```
- Calls `GetSession`. If no valid session → redirect to `/login`.
- Injects email into request context.

### 4 — Middleware: `RequireAdmin`
Wraps `RequireAuth` + checks `IsAdmin`. If not admin → 403.

### 5 — Context helpers
```go
func EmailFromContext(ctx context.Context) string
func IsAdminFromContext(ctx context.Context) bool
```

## Acceptance Criteria
- Tampered cookie signature is rejected.
- Expired session redirects to `/login`.
- `POST /logout` clears the cookie and redirects to `/`.
