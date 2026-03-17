# mailer

## Goal
Send transactional emails via `net/smtp` using config SMTP settings.

## Tasks

### 1 — `internal/mailer/mailer.go`
```go
type Mailer struct { cfg SMTPConfig }

func New(cfg config.SMTPConfig) *Mailer
func (m *Mailer) Send(to, subject, body string) error
```
- Uses `net/smtp` STARTTLS.
- `body` is plain text (magic links) or HTML (notifications).

### 2 — Typed send methods
```go
func (m *Mailer) SendMagicLink(to, link, lang string, i18n *i18n.Bundle) error
func (m *Mailer) SendChapterPublished(to, title, lang string, i18n *i18n.Bundle) error
func (m *Mailer) SendChapterRejected(to, title, lang string, i18n *i18n.Bundle) error
func (m *Mailer) SendLikeMilestone(to string, count int, title, lang string, i18n *i18n.Bundle) error
```
- Subject and body text come from i18n bundle.

### 3 — Like milestone thresholds
Milestones: 1, 5, 10, 25, 50, 100.
```go
func IsMilestone(count int) bool
```

## Acceptance Criteria
- `Send` returns a wrapped error on SMTP failure — never panics.
- All typed methods use i18n for subject lines.
- `IsMilestone(10)` → true, `IsMilestone(7)` → false.
