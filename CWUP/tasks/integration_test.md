# integration_test

## Goal
End-to-end integration test covering the full author workflow via HTTP.

## Tasks

### `integration_test.go` (package main)

**TC-01: Register new user**
- `POST /register` with email + display name → 200, confirmation page

**TC-02: Magic link validation**
- Read token from test DB → `GET /auth?token=...` → session cookie set → redirect to `/dashboard`

**TC-03: Select book**
- `POST /dashboard/book/1/select` → active book cookie set → redirect to `/dashboard/chapters`

**TC-04: Create chapter with special chars**
- `POST /dashboard/chapters` with markdown containing `---`, `>`, `<code>`
- Redirect to preview
- `GET /dashboard/chapters/{id}` → preview rendered

**TC-05: Markdown integrity**
- Load chapter from DB directly → `MarkdownContent` contains `---` verbatim

**TC-06: Edit chapter**
- `GET /dashboard/chapters/{id}/edit` → 200, editor with pre-filled content

**TC-07: Submit chapter**
- `POST /dashboard/chapters/{id}/submit` → status = `pending_review`

**TC-08: Admin publish**
- With admin session: `POST /admin/chapters/{id}/publish` → status = `published`

**TC-09: Like chapter**
- `POST /api/like/{id}` → `{"count": 1, "liked": true}`
- Repeat → `{"count": 1, "liked": false}`

**TC-10: Public chapter page**
- `GET /books/choose-your-incarnation/{slug}` → 200, contains chapter title

## Acceptance Criteria
- All TCs pass with `go test -run TestIntegration ./...`
- Uses in-memory SQLite and a stub SMTP (captures sent emails without delivering)
- No real network calls
