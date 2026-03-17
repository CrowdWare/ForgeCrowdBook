# model_user

## Goal
DB query functions for the `users` table.

## Tasks

### 1 — `internal/model/user.go`
```go
type User struct {
    ID          int
    Email       string
    DisplayName string
    Bio         string
    Lang        string
    Status      string
    CreatedAt   time.Time
}

func GetUserByEmail(db *sql.DB, email string) (*User, error)
func GetUserByID(db *sql.DB, id int) (*User, error)
func CreateUser(db *sql.DB, email, displayName string) (*User, error)
func UpdateUserLang(db *sql.DB, id int, lang string) error
func BanUser(db *sql.DB, id int) error
func ListUsers(db *sql.DB) ([]User, error)
```

## Acceptance Criteria
- `GetUserByEmail` returns `nil, nil` when not found (not an error).
- `CreateUser` returns error on duplicate email.
- All queries use parameterized statements.
