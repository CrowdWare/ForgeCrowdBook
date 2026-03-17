# Risks — ForgeCrowdBook

## R-01: SMTP configuration on server
**Risk:** `net/smtp` with STARTTLS requires correct server config. Self-hosted mail may have TLS cert issues.
**Mitigation:** Test SMTP in isolation early. Provide clear config docs. Consider a `--dry-run-mail` flag that logs emails without sending.

## R-02: SQLite concurrency under load
**Risk:** SQLite WAL mode handles concurrent reads well, but write contention under high load is possible.
**Mitigation:** Acceptable for the expected traffic. If needed, add a write queue. Not a concern for v1.

## R-03: Magic link email delivery
**Risk:** Magic link emails may land in spam, especially from a self-hosted server without SPF/DKIM.
**Mitigation:** Document SPF/DKIM setup. Add plain-text fallback. Consider showing token on screen as backup (debug mode only).

## R-04: EasyMDE and Markdown encoding
**Risk:** EasyMDE submits Markdown via HTML form POST. Browser may encode `+` as `%2B` etc. Server must decode correctly.
**Mitigation:** Use `r.FormValue("content")` which Go's `net/http` URL-decodes automatically. Test with `---`, `>`, `<`, `+` explicitly (TC-05).

## R-05: Session secret rotation
**Risk:** Changing `session_secret` in config invalidates all existing sessions (all users logged out).
**Mitigation:** Document this behavior. Provide guidance to only change secret intentionally.
