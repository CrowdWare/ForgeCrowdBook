# handler_dashboard

## Goal
Authenticated dashboard: book selection and user overview.

## Context
This is the entry point for all author actions. The user MUST select a book here
before they can create or edit chapters. This prevents chapters being assigned to the wrong book.

## Tasks

### 1 — `GET /dashboard` → `templates/dashboard.html`
Shows:
- Welcome message with display name
- List of all books (cards with title, description)
- "Select" button per book — highlights currently active book
- Link to "My Chapters" (only visible after a book is selected)

### 2 — `POST /dashboard/book/{id}/select`
- Validate book exists.
- Set `active_book` cookie.
- Redirect to `/dashboard/chapters`.

## Acceptance Criteria
- Unauthenticated request → redirect to `/login`.
- After selecting a book, active book name shown in nav.
- `/dashboard/chapters` without active book → redirect to `/dashboard`.
