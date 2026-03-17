# handler_middleware

## Goal
Shared middleware and template data helpers used by all handlers.

## Tasks

### 1 — `internal/handler/middleware.go`

**`NavData` struct** — passed to every template:
```go
type NavData struct {
    LoggedIn   bool
    IsAdmin    bool
    Email      string
    Lang       string
    Strings    map[string]string
    ActiveBook *model.Book  // nil if none selected
}
```

**`baseData(r, cfg, i18n)` helper** — builds `NavData` from request context.

**`RequireAuth` middleware** — wraps protected routes.

**`RequireAdmin` middleware** — wraps admin routes.

**`SetLang` handler** — `POST /lang`: sets `lang` cookie, redirects to `Referer`.

### 2 — Active book in session
Active book is stored as `active_book_id` in a plain cookie (not signed — not sensitive).
```go
func GetActiveBook(r *http.Request, db *sql.DB) (*model.Book, error)
func SetActiveBook(w http.ResponseWriter, bookID int)
func ClearActiveBook(w http.ResponseWriter)
```

## Acceptance Criteria
- All templates receive consistent `NavData`.
- Language cookie is set with `SameSite=Lax`, no expiry (session cookie).
- Active book cookie is cleared on logout.
