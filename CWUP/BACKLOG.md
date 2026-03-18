# Backlog — ForgeCrowdBook

## Functional Requirements

See `SPEC.md` for full specification.

### Core Platform
- Standalone Go web application — no WordPress, no PHP
- SQLite database (single file, no external server)
- Magic Link authentication (no passwords)
- Book-first workflow: select book on dashboard before editing chapters
- Markdown is never stored in the database — only source_url is stored
- Content fetched on demand from author-hosted URLs (GitHub, Codeberg, IPFS)
- Context-sensitive navigation menu

### i18n
- Translation via SML files (`i18n/strings-{lang}.sml`) parsed by `sml-go`
- Supported languages: de, en, eo, pt, fr, es
- Language selection via cookie, with flag dropdown in nav

### Chapter Registration
- Authors register a source URL pointing to their own Markdown file
- Preview widget: paste URL → see rendered preview before submitting
- After saving: show Preview (fetched + rendered, read-only)
- Return to Edit mode only via explicit "Edit" button
- IPFS URLs (`ipfs://`) normalized to configurable gateway

### Moderation
- Admin dashboard: publish, reject, delete chapters
- Admin user management: ban users
- No automatic spam filter (manual moderation only)

### Likes
- No login required
- Fingerprint-based (SHA256 of IP + User-Agent)
- Email notifications at milestones: 1, 5, 10, 25, 50, 100 likes

### Social Sharing
- Open Graph meta tags per chapter
- Plain HTML share buttons (Telegram, WhatsApp, Facebook, X, Copy Link)

---

## Tasks

### Phase 1 — Foundation
- [x] tasks/go_module_setup.md
- [x] tasks/config_parser.md
- [x] tasks/db_init.md
- [x] tasks/i18n.md
- [x] tasks/content_fetcher.md

### Phase 2 — Auth
- [x] tasks/auth_magic_link.md
- [x] tasks/auth_session.md
- [x] tasks/mailer.md

### Phase 3 — Models
- [x] tasks/model_user.md
- [x] tasks/model_book.md
- [x] tasks/model_chapter.md  ← stores source_url, not markdown_content
- [x] tasks/model_like.md

### Phase 4 — Handlers
- [x] tasks/handler_middleware.md
- [x] tasks/handler_home.md
- [x] tasks/handler_books.md
- [x] tasks/handler_auth.md
- [x] tasks/handler_dashboard.md
- [x] tasks/handler_chapters.md  ← registration form + preview widget (no EasyMDE)
- [x] tasks/handler_admin.md
- [x] tasks/handler_api.md

### Phase 5 — Frontend
- [x] tasks/templates_base.md
- [x] tasks/templates_pages.md
- [x] tasks/static_assets.md  ← style.css + preview.js only (no EasyMDE)

### Phase 6 — Tests
- [x] tasks/unit_tests_config.md
- [x] tasks/unit_tests_auth.md
- [x] tasks/unit_tests_fetcher.md
- [x] tasks/integration_test.md
