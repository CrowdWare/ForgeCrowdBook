# unit_tests_config

## Goal
Unit tests for `internal/config` and `internal/i18n`.

## Tasks

### config tests (`internal/config/config_test.go`)
- `LoadConfig` with valid `app.sml` → all fields populated
- `LoadConfig` with missing optional fields → defaults applied
- `LoadConfig` with malformed SML → returns error, no panic

### i18n tests (`internal/i18n/i18n_test.go`)
- `Load` with valid files → bundle populated
- `T("en", "nav_home")` → `"Home"`
- `T("de", "nav_home")` → `"Startseite"`
- `T("en", "nonexistent_key")` → returns `"nonexistent_key"` (key as fallback)
- `T("xx", "nav_home")` → falls back to `"en"` value

## Acceptance Criteria
- All tests pass with `go test ./internal/config/... ./internal/i18n/...`
- No network calls, no filesystem side effects outside `t.TempDir()`
