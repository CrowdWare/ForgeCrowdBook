# config_parser

## Goal
Parse `app.sml` into a typed `Config` struct using `sml-go`.

## Context
`app.sml` is the application configuration file written in SML. It declares the app name,
base URL, database path, port, session secret, admin email, and SMTP settings.
The `sml-go` package (`codeberg.org/crowdware/sml-go`) provides `ParseDocument(source string)`.

## Tasks

### 1 — Define structs in `internal/config/config.go`
```go
type Config struct {
    Name          string
    BaseURL       string
    DBPath        string
    Port          string
    SessionSecret string
    AdminEmail    string
    SMTP          SMTPConfig
}

type SMTPConfig struct {
    Host string
    Port string
    User string
    Pass string
    From string
}
```

### 2 — Implement `LoadConfig(path string) (*Config, error)`
- Read file from disk.
- Call `sml.ParseDocument(source)`.
- Walk the `App{}` root node for scalar properties.
- Walk `SMTP{}` child node for SMTP settings.
- Apply sensible defaults: port `"8090"`, db `"./data/crowdbook.db"`.
- Return populated `Config` or descriptive error.

## Acceptance Criteria
- `LoadConfig("app.sml")` returns a fully populated `Config`.
- Missing optional fields use defaults.
- Malformed SML returns an error, not a panic.
