# handler_chapters

## Goal
Chapter registration, preview, and submission handlers for authenticated authors.

## Context
Authors host their own Markdown on GitHub, Codeberg, or IPFS.
ForgeCrowdBook stores only metadata (title, path label, source URL).
Content is fetched on demand and cached.

## Tasks

### 1 — `GET /dashboard/chapters` → `templates/chapters.html`
Lists the current user's chapters for the active book.
Per row: title, path label, status badge, source URL (truncated), created date, "Preview" / "Edit" actions.
No active book in session → redirect to `/dashboard`.

### 2 — `GET /dashboard/chapters/new` → `templates/chapter-register.html`
Registration form:
- Title input
- Path Label input
- Source URL input (placeholder: "https://codeberg.org/you/mybook/raw/branch/main/chapter.md")
- "Load Preview" button → calls `POST /dashboard/preview` via fetch, shows rendered HTML below
- "Submit for Review" button (only enabled after preview loaded successfully)
- Active book name shown (read-only, no dropdown)

### 3 — `POST /dashboard/preview`
- Accepts `source_url` form field.
- Normalizes `ipfs://` to gateway URL.
- Fetches content via content cache (reuses ForgeCMS fetcher).
- Compiles Markdown to HTML.
- Returns HTML fragment (not full page) — used by the preview widget via fetch.
- Returns 400 if URL is empty, 502 if fetch fails.

### 4 — `POST /dashboard/chapters` (register new)
- Validate title not empty, source_url not empty.
- Normalize source_url (ipfs:// → gateway).
- Generate slug from title.
- Store chapter metadata in DB (status: draft).
- Redirect to `GET /dashboard/chapters/{id}` (preview).

### 5 — `GET /dashboard/chapters/{id}` → `templates/chapter-preview.html`
- Fetch content from source_url via content cache.
- Render Markdown → HTML.
- Show: title, path label, status, rendered content.
- Explicit **"Edit"** button → `/dashboard/chapters/{id}/edit`
- "Submit for Review" button (if status is draft).
- "Refresh Content" button → clears cache entry for this URL, re-fetches.

### 6 — `GET /dashboard/chapters/{id}/edit` → `templates/chapter-register.html`
Same form as new, pre-filled with existing metadata.
This is the ONLY way to re-enter edit mode.

### 7 — `POST /dashboard/chapters/{id}` (update)
- Validate ownership.
- Update title, path_label, source_url.
- Redirect to preview.

### 8 — `POST /dashboard/chapters/{id}/submit`
- Validate ownership.
- Set status to `pending_review`.
- Redirect to preview with status message.

## Acceptance Criteria
- After save: browser shows preview (fetched + rendered), not the form.
- Edit mode only reachable via "Edit" button on preview page.
- User cannot edit another user's chapter (403).
- `ipfs://` URLs are normalized before storage and fetching.
- If source URL is unreachable: show "Content unavailable" message, not 500.
- No active book selected → redirect to `/dashboard`.
