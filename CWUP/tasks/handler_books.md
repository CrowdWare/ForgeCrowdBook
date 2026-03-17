# handler_books

## Goal
Public book listing and chapter reading handlers.

## Tasks

### 1 — `GET /books` → `templates/books.html`
Lists all books with title and description.

### 2 — `GET /books/{slug}` → `templates/book.html`
Lists all published chapters for the book.
Per chapter card: title, author (display name), path label, first 150 chars of markdown, like count, "Read" button.

### 3 — `GET /books/{slug}/{chapter-slug}` → `templates/chapter.html`
Renders a published chapter:
- Markdown compiled server-side via `goldmark` for public display
- Open Graph meta tags in `<head>`
- Like button (`POST /api/like/{id}` via fetch)
- Share buttons (Telegram, WhatsApp, Facebook, X, Copy Link)
- Back link to book index

## Context — Markdown rendering for public display
For public reading pages, Markdown is compiled server-side using `goldmark` (same as ForgeCMS).
Raw Markdown must not be escaped before reaching goldmark.
The compiled HTML is passed to the template as `template.HTML`.

## Acceptance Criteria
- Unknown book slug → 404.
- Unknown chapter slug → 404.
- Draft/rejected chapters are not accessible on public routes.
- Open Graph tags contain chapter title and first 150 chars of content.
