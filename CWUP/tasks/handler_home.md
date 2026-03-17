# handler_home

## Goal
Public home page handler.

## Tasks

### 1 — `GET /` → `templates/home.html`
Shows: app name, short description, call-to-action ("Read the book" → `/books`, "Write your chapter" → `/register`).

### 2 — `POST /logout`
Clears session cookie and active book cookie, redirects to `/`.

## Acceptance Criteria
- Home page renders without a session.
- Logout redirects to `/` and session is gone.
