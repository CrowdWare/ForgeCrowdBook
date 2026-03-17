# ForgeCrowdBook — Technical Specification

This document is the authoritative specification for **ForgeCrowdBook**, a standalone Go web application for a collaborative, branching-narrative book platform. It is intended as a complete handoff document for a developer or AI code generator (e.g. Codex). All design decisions, data models, routes, workflows, and constraints are described here. Nothing should be inferred from outside this document.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Technology Stack](#2-technology-stack)
3. [Database Schema](#3-database-schema)
4. [Internationalisation (i18n)](#4-internationalisation-i18n)
5. [Authentication — Magic Link](#5-authentication--magic-link)
6. [Book-First Workflow](#6-book-first-workflow)
7. [Chapter Registration Workflow](#7-chapter-registration-workflow)
8. [HTTP Routes](#8-http-routes)
9. [Navigation Menu](#9-navigation-menu)
10. [Email Notifications](#10-email-notifications)
11. [Configuration — app.sml](#11-configuration--appsml)
12. [Package Structure](#12-package-structure)
13. [Markdown Integrity](#13-markdown-integrity)
14. [Like System](#14-like-system)
15. [Social Sharing](#15-social-sharing)
16. [Future Scope (v2+)](#16-future-scope-v2)

---

## 1. Project Overview

ForgeCrowdBook is a collaborative book platform where anonymous authors write chapters in Markdown. Chapters form a branching narrative — inspired by *The Egg* by Andy Weir. Each chapter can branch into multiple continuations, creating a tree of parallel story paths.

This application replaces a WordPress plugin. It is a **separate repository** from ForgeCMS but depends on it for content fetching and Markdown compilation (`codeberg.org/crowdware/forgecms/internal/content` and `internal/compiler`).

### Core Philosophy

| Principle | Implementation |
|---|---|
| No privacy problem | Everything runs locally on the server — no analytics, no tracking |
| No third parties | No external services, no CDNs |
| Anonymous-friendly | Pseudonyms (display names) are welcome |
| Passwordless | Magic Link authentication — no passwords stored |
| Simple | Authors only need to write Markdown |

---

## 2. Technology Stack

| Concern | Choice | Notes |
|---|---|---|
| Language | Go | stdlib-first approach |
| Database | SQLite | Single file, no external DB server |
| SQLite driver | `modernc.org/sqlite` | Pure Go, no CGO |
| HTTP server | `net/http` | Go stdlib |
| HTML templates | `html/template` | Go stdlib |
| Email | `net/smtp` | Go stdlib |
| SML parsing | `codeberg.org/crowdware/sml-go` | For i18n strings files and config |
| Markdown rendering | goldmark | Server-side, reuses ForgeCMS compiler |
| Content fetching | ForgeCMS `internal/content` | ETag caching, IPFS normalization |

**Content source rule (non-negotiable):** Markdown is never stored in the database. Only the `source_url` is stored. Raw bytes fetched from the source URL are passed directly to goldmark. See [Section 13](#13-markdown-integrity).

---

## 3. Database Schema

The SQLite database is a single file at the path specified in `app.sml`. All migrations run at startup.

### 3.1 `users`

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    bio TEXT DEFAULT '',
    lang TEXT NOT NULL DEFAULT 'en',
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'banned')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2 `magic_tokens`

```sql
CREATE TABLE magic_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used INTEGER NOT NULL DEFAULT 0
);
```

### 3.3 `books`

```sql
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.4 `chapters`

```sql
CREATE TABLE chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL REFERENCES books(id),
    author_id INTEGER NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    path_label TEXT DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'pending_review', 'published', 'rejected')),
    like_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME
);
```

`path_label` is a free-text field authors use to describe which story branch this chapter continues (e.g. "The red door path").

`source_url` is the raw URL to the author's Markdown file. ForgeCrowdBook never stores the Markdown content itself — only this URL.

### 3.5 `likes`

```sql
CREATE TABLE likes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id),
    fingerprint TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chapter_id, fingerprint)
);
```

`fingerprint` is a SHA256 hash of `IP + User-Agent`. See [Section 14](#14-like-system).

---

## 3.5 Content Sources

Chapter content is not stored in ForgeCrowdBook's database. Authors host their own
Markdown files and register the URL. ForgeCrowdBook fetches and caches the content.

### Supported URL types

| Type | Example |
|------|---------|
| GitHub raw | `https://raw.githubusercontent.com/user/repo/main/chapter.md` |
| Codeberg raw | `https://codeberg.org/user/repo/raw/branch/main/chapter.md` |
| IPFS protocol | `ipfs://QmXxxxxxxxxxxxx` |
| IPFS gateway | `https://ipfs.io/ipfs/QmXxxxxxxxxxxxx` |
| Any public URL | Any URL returning `text/plain` or `text/markdown` |

### IPFS handling

`ipfs://` scheme URLs are normalized to the configured gateway URL:
```
ipfs://QmABC... → https://ipfs.io/ipfs/QmABC...
```
The gateway is configurable via `ipfs_gateway` in `app.sml`.

IPFS content is content-addressed: the CID is the cryptographic hash of the content.
This means:
- If the CID hasn't changed → cached content is valid forever (no ETag roundtrip needed)
- Author updates content → gets a new CID → updates `source_url` in their chapter
- Cache key = CID (for IPFS) or URL hash (for HTTP sources)

### Caching

Content fetching and caching reuses the `internal/content` package from ForgeCMS
(`codeberg.org/crowdware/forgecms/internal/content`). This provides:
- ETag / If-None-Match conditional fetching for HTTP sources
- SHA256 content-addressed caching
- TTL-based stale fallback when origin is unreachable
- Manifest-based cache persistence across restarts

### Author sovereignty

Authors retain full ownership of their content:
- Content lives in their own repository or IPFS node
- They can update, delete, or migrate at any time
- ForgeCrowdBook is a publishing layer, not a content host
- If an author deletes their content, the chapter shows a "content unavailable" message

---

## 4. Internationalisation (i18n)

Translation is handled entirely via SML string files — one file per language — parsed at application startup using `sml-go`. No external i18n library is used.

### 4.1 Supported Languages

`de`, `en`, `eo`, `pt`, `fr`, `es`

### 4.2 File Location and Naming

```
i18n/strings-{lang}.sml
```

Example: `i18n/strings-de.sml`

### 4.3 File Format

```
Strings {
    nav_home: "Startseite"
    nav_books: "Bücher"
    nav_dashboard: "Dashboard"
    nav_login: "Anmelden"
    nav_register: "Registrieren"
    nav_logout: "Abmelden"
    btn_edit: "Bearbeiten"
    btn_save: "Speichern"
    btn_submit: "Einreichen"
    btn_preview: "Vorschau"
    label_title: "Titel"
    label_path: "Pfad-Label"
    label_content: "Inhalt"
    msg_saved: "Gespeichert"
    msg_submitted: "Eingereicht"
    lang_de: "Deutsch"
    lang_en: "English"
    lang_eo: "Esperanto"
    lang_pt: "Português"
    lang_fr: "Français"
    lang_es: "Español"
}
```

All other language files follow the same structure with translated values.

### 4.4 In-Memory Structure

```go
// map[lang]map[key]value
type Strings map[string]map[string]string
```

Loaded once at startup. Lookup: `strings[lang][key]`, fallback to `strings["en"][key]`.

### 4.5 Language Detection Order

1. `lang` cookie (set by `POST /lang`)
2. User profile `lang` field (if the user is logged in)
3. Default: `en`

### 4.6 Language Switch Endpoint

`POST /lang` — form field `lang` — sets a `lang` cookie, redirects back to the referring page.

---

## 5. Authentication — Magic Link

There are no passwords. The user's email address is their identity.

### 5.1 Registration Flow

1. User submits email + display name at `GET /register` -> `POST /register`.
2. If the email is unknown: create a new user record (`status = 'active'`).
3. If the email is already known: proceed to step 4 (effectively a login).
4. Generate a cryptographically secure token: `crypto/rand` — 32 bytes encoded as hex (64 characters).
5. Insert into `magic_tokens`: `expires_at = now + 15 minutes`, `used = 0`.
6. Send email with link: `https://{base_url}/auth?token={token}`.
7. Show "Check your email" page.

### 5.2 Login Flow

1. User submits email at `GET /login` -> `POST /login`.
2. Generate and send a Magic Link (same as steps 4–7 above).

### 5.3 Token Validation (`GET /auth?token=...`)

1. Look up the token in `magic_tokens`.
2. Validate: must exist, `used = 0`, `expires_at > now`.
3. Mark token `used = 1`.
4. Create a signed session cookie containing the user's email (HMAC-SHA256, secret from config).
5. Redirect to `/dashboard`.
6. On failure (invalid or expired token): render an error page.

### 5.4 Session Cookie

| Property | Value |
|---|---|
| Name | `session` |
| Value | HMAC-SHA256 signed, contains user email |
| HttpOnly | Yes |
| SameSite | Lax |
| Expiry | 30 days |
| Secret | `session_secret` from `app.sml` |

All protected routes are wrapped in a session middleware that validates this cookie.

### 5.5 Admin Sessions

The admin flag is derived from comparing the session user's email against `admin_email` in `app.sml`. No separate admin table is needed.

### 5.6 Rate Limiting

Maximum 3 Magic Link requests per email per hour. Enforced by counting rows in `magic_tokens` where `email = ?` and `created_at > now - 1 hour`.

---

## 6. Book-First Workflow

This workflow is critical. It prevents chapters from being accidentally assigned to the wrong book.

1. User visits `/dashboard` — sees a list of all books.
2. User clicks a book -> the application stores that book's `id` as `active_book_id` in the session via `POST /dashboard/book/{id}/select`.
3. `/dashboard/chapters` displays only chapters belonging to the active book.
4. `/dashboard/chapters/new` (the chapter editor) has **no book dropdown**. The active book is taken silently from the session.
5. To work on a different book, the user must return to `/dashboard` and select another book.

This single-active-book constraint is intentional and must not be circumvented by any UI affordance.

---

## 7. Chapter Registration Workflow

1. Author creates their Markdown file in their own GitHub/Codeberg repo or pins it to IPFS.
2. Author goes to `/dashboard/chapters/new`.
3. Author fills in:
   - Title
   - Path Label (e.g. "Ubuntu Way", "Rainbow Path")
   - Source URL (GitHub raw, Codeberg raw, or IPFS URL)
4. Author clicks **"Load Preview"** → ForgeCrowdBook fetches the URL and renders the Markdown.
5. If preview looks correct: author clicks **"Submit for Review"**.
6. Admin sees the preview, publishes or rejects.
7. When author updates content at the source URL: the next fetch automatically picks up changes
   (for IPFS: author updates the source_url to the new CID).

### Registration State Summary

| State | URL | Editable |
|---|---|---|
| New chapter | `/dashboard/chapters/new` | Yes (registration form) |
| Edit chapter | `/dashboard/chapters/{id}/edit` | Yes (registration form, explicit navigation) |
| Preview chapter | `/dashboard/chapters/{id}` | No (fetched + rendered, read-only) |

### Preview widget

The preview widget fetches the source URL server-side (via `POST /dashboard/preview`)
and returns rendered HTML. This is a POST to avoid CORS issues and to reuse the server-side
cache. The author sees exactly how their chapter will appear to readers.

---

## 8. HTTP Routes

### 8.1 Public Routes

| Method | Path | Description |
|---|---|---|
| GET | `/` | Home page |
| GET | `/books` | List all published books |
| GET | `/books/{slug}` | Book index — published chapters |
| GET | `/books/{slug}/{chapter-slug}` | Read a single chapter |
| GET | `/login` | Login form |
| POST | `/login` | Send magic link |
| GET | `/register` | Registration form |
| POST | `/register` | Create user and send magic link |
| GET | `/auth` | Validate magic link token, create session |
| POST | `/lang` | Set language cookie |

### 8.2 Protected Routes (require valid session)

| Method | Path | Description |
|---|---|---|
| GET | `/dashboard` | Book selection and user overview |
| POST | `/dashboard/book/{id}/select` | Set active book in session |
| GET | `/dashboard/chapters` | List chapters for the active book |
| GET | `/dashboard/chapters/new` | New chapter registration form |
| POST | `/dashboard/preview` | Fetch + render source URL (returns HTML fragment) |
| POST | `/dashboard/chapters` | Save chapter registration (title, path, source_url) |
| GET | `/dashboard/chapters/{id}` | Preview chapter (fetched + rendered) |
| GET | `/dashboard/chapters/{id}/edit` | Edit chapter registration (explicit action) |
| POST | `/dashboard/chapters/{id}` | Update chapter registration |
| POST | `/dashboard/chapters/{id}/submit` | Submit chapter for review |
| POST | `/logout` | Clear session cookie |

### 8.3 Admin Routes (require session email == `admin_email`)

| Method | Path | Description |
|---|---|---|
| GET | `/admin/chapters` | All chapters with status filters |
| POST | `/admin/chapters/{id}/publish` | Publish a chapter |
| POST | `/admin/chapters/{id}/reject` | Reject a chapter |
| POST | `/admin/chapters/{id}/delete` | Delete a chapter |
| GET | `/admin/users` | User list |
| POST | `/admin/users/{id}/ban` | Ban a user |

### 8.4 API Routes

| Method | Path | Description |
|---|---|---|
| POST | `/api/like/{chapter-id}` | Toggle like (no auth required, fingerprint-based) |

The like endpoint returns JSON: `{"count": N, "liked": true}`.

---

## 9. Navigation Menu

Templates receive a `Nav` struct. The server controls menu state — templates do not contain any conditional logic for auth state.

### 9.1 Unauthenticated

```
Home | Books | Login | Register | [Language Dropdown]
```

### 9.2 Authenticated

```
Home | Books | Dashboard | Logout | [Language Dropdown]
```

### 9.3 Language Dropdown

| Flag | Language |
|---|---|
| DE | Deutsch |
| EN | English |
| EO | Esperanto |
| PT | Portugues |
| FR | Francais |
| ES | Espanol |

The dropdown posts to `POST /lang` with a `lang` form field.

### 9.4 Nav Struct (Go)

```go
type Nav struct {
    IsLoggedIn bool
    ActiveLang string
    Langs      []LangOption
}

type LangOption struct {
    Code  string
    Label string
    Flag  string
}
```

---

## 10. Email Notifications

All email is sent via `net/smtp` using the credentials in `app.sml`. No third-party email service is used.

### 10.1 Notification Events

| Event | Recipient | Subject i18n key |
|---|---|---|
| Magic link requested | Requesting user | `email_magic_link_subject` |
| Chapter published | Chapter author | `email_chapter_published_subject` |
| Chapter rejected | Chapter author | `email_chapter_rejected_subject` |
| 10 likes reached | Chapter author | `email_likes_10_subject` |
| 50 likes reached | Chapter author | `email_likes_50_subject` |

### 10.2 Like Milestone Thresholds

Emails are sent when `like_count` crosses the following values: **1, 5, 10, 25, 50, 100**.

The check is performed after each successful like toggle. If the new count equals a milestone value, the notification email is dispatched.

---

## 11. Configuration — app.sml

The application is configured via a single `app.sml` file in the project root. It is parsed at startup using `sml-go`.

```
App {
    name: "ForgeCrowdBook"
    base_url: "https://crowdware.info"
    db: "./data/crowdbook.db"
    port: "8090"
    session_secret: "change-me-in-production"
    admin_email: "admin@crowdware.info"
    ipfs_gateway: "https://ipfs.io/ipfs/"

    SMTP {
        host: "smtp.example.com"
        port: "587"
        user: "noreply@crowdware.info"
        pass: "secret"
        from: "CrowdBook <noreply@crowdware.info>"
    }
}
```

### 11.1 Config Struct (Go)

```go
type Config struct {
    Name          string
    BaseURL       string
    DB            string
    Port          string
    SessionSecret string
    AdminEmail    string
    IPFSGateway   string
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

---

## 12. Package Structure

```
ForgeCrowdBook/
├── main.go                        # Entry point, wires everything
├── app.sml                        # Configuration
├── go.mod
├── go.sum
├── internal/
│   ├── config/
│   │   └── config.go              # Load app.sml via sml-go
│   ├── db/
│   │   └── db.go                  # SQLite init, migrations
│   ├── auth/
│   │   └── auth.go                # Magic link, session, middleware
│   ├── i18n/
│   │   └── i18n.go                # Load strings-{lang}.sml, lookup
│   ├── fetcher/
│   │   └── fetcher.go             # Wraps ForgeCMS content cache; IPFS normalization
│   ├── handler/
│   │   ├── home.go
│   │   ├── books.go
│   │   ├── auth.go
│   │   ├── dashboard.go
│   │   ├── chapters.go
│   │   ├── admin.go
│   │   ├── api.go
│   │   └── middleware.go
│   ├── model/
│   │   ├── user.go
│   │   ├── book.go
│   │   ├── chapter.go
│   │   └── token.go
│   └── mailer/
│       └── mailer.go              # net/smtp email sender
├── i18n/
│   ├── strings-de.sml
│   ├── strings-en.sml
│   ├── strings-eo.sml
│   ├── strings-pt.sml
│   ├── strings-fr.sml
│   └── strings-es.sml
├── templates/
│   ├── base.html                  # Outer chrome, nav, language dropdown
│   ├── home.html
│   ├── books.html
│   ├── book.html
│   ├── chapter.html
│   ├── login.html
│   ├── register.html
│   ├── dashboard.html
│   ├── chapters.html
│   ├── chapter-register.html      # Source URL registration form + preview widget
│   ├── chapter-preview.html       # Read-only preview (fetched + rendered) + Edit button
│   ├── admin-chapters.html
│   └── admin-users.html
├── static/
│   ├── style.css
│   └── preview.js                 # Minimal vanilla JS for the preview widget (~30 lines)
└── data/
    └── .gitkeep                   # SQLite DB file is created here at runtime
```

### 12.1 main.go Responsibilities

- Parse `app.sml` via `internal/config`
- Open and migrate the SQLite database via `internal/db`
- Load all i18n string files via `internal/i18n`
- Register all HTTP routes and middleware
- Start `net/http` server on the configured port

---

## 13. Markdown Integrity

This is a hard requirement. The WordPress predecessor had a bug where Markdown content was silently transformed, breaking `---` horizontal rules, blockquotes, and HTML entities.

Since Markdown is never stored in the database, the integrity concern shifts to the fetch-and-render pipeline.

### 13.1 Rules

1. Raw bytes are fetched from the author's source URL — no transformation before passing to goldmark.
2. Goldmark compiles the raw bytes to HTML.
3. The resulting HTML string is cast to `template.HTML` and injected into the template.
4. The DB only stores the `source_url` string — never the Markdown content itself.
5. Go's `html/template` must **never receive raw Markdown** as a template variable — only the compiled `template.HTML` value.

### 13.2 Specific Characters That Must Not Be Transformed

| Character / Sequence | Must render as |
|---|---|
| `---` | `<hr>` |
| `> quote` | `<blockquote>` |
| `` `code` `` | `<code>` |

### 13.3 Correct Pattern for Injecting Rendered HTML into a Template

```go
html, err := fetcher.FetchHTML(chapter.SourceURL)
if err != nil {
    // Show "Content unavailable" message
}
// Pass template.HTML(html) to the template — signals it is pre-rendered and safe
```

```html
<!-- In the template -->
<div class="chapter-content">{{ .RenderedHTML }}</div>
```

Where `.RenderedHTML` is of type `template.HTML` — `html/template` will not escape it.

---

## 14. Like System

Likes require no user account. They use an anonymous fingerprint to prevent duplicates without tracking individuals.

### 14.1 Fingerprint

```
fingerprint = hex(SHA256(IP + User-Agent))
```

This value is stored in the `likes` table. It is not reversible to a specific person and is not shared with any external service.

### 14.2 Endpoint Behaviour

`POST /api/like/{chapter-id}`

1. Compute fingerprint from request.
2. Attempt `INSERT INTO likes (chapter_id, fingerprint)`.
3. If insert succeeds (new like): increment `chapters.like_count`.
4. If insert fails with UNIQUE constraint (already liked): delete the row (toggle off), decrement `chapters.like_count`.
5. Check if new `like_count` equals a milestone (1, 5, 10, 25, 50, 100) — if so, send notification email to the chapter author.
6. Return: `{"count": N, "liked": true|false}`.

### 14.3 Like Milestone Emails

Milestones: **1, 5, 10, 25, 50, 100**

Only the crossing of a milestone triggers an email. Toggling a like off and back on does not re-trigger the same milestone email (because `like_count` would need to cross the threshold again from below).

---

## 15. Social Sharing

Each chapter page (`GET /books/{slug}/{chapter-slug}`) includes Open Graph meta tags and share buttons.

### 15.1 Open Graph Meta Tags

```html
<meta property="og:title" content="{chapter title} - {book title}">
<meta property="og:description" content="{first 150 characters of rendered Markdown, plain text}">
<meta property="og:url" content="{base_url}/books/{book-slug}/{chapter-slug}">
<meta property="og:type" content="article">
<meta name="twitter:card" content="summary_large_image">
```

The 150-character description is generated server-side by stripping Markdown syntax to plain text. This is the only place where Markdown is processed on the server — strictly for the meta description, not for page display.

### 15.2 Share Buttons

Plain HTML links — no JavaScript SDK, no third-party scripts.

| Platform | URL Pattern |
|---|---|
| Telegram | `https://t.me/share/url?url={url}&text={title}` |
| WhatsApp | `https://api.whatsapp.com/send?text={title}%20{url}` |
| Facebook | `https://www.facebook.com/sharer/sharer.php?u={url}` |
| X (Twitter) | `https://twitter.com/intent/tweet?url={url}&text={title}` |
| Copy Link | Clipboard API (`navigator.clipboard.writeText`) — one line of inline JS |

---

## 16. Future Scope (v2+)

The following items are out of scope for v1 and are listed here only for architectural awareness. No code should be written for these in v1.

| Feature | Notes |
|---|---|
| Android app with Whisper.cpp | Mobile authors dictate chapters by voice |
| ForgeSTA integration | Markdown submitted directly from Mac desktop app |
| Multiple simultaneous active books | Currently one active book per session |
| Chapter tree visualisation | Interactive graph of branching narrative paths |
| ForgeCMS integration | Expose chapters via `/api/pages` endpoint for ForgeCMS consumption |

---

*End of specification.*
