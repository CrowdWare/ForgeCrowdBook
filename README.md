# ForgeCrowdBook

A collaborative book platform built in Go. Authors host their own Markdown content on GitHub, Codeberg, or IPFS — ForgeCrowdBook handles discovery, moderation, and publishing.

## Features

- No passwords — Magic Link authentication only
- No content lock-in — authors own their files, we only store a URL
- No third parties — runs entirely on your own server
- No database server — SQLite, single file

## Requirements

- Go `1.23+`
- SQLite is embedded via `modernc.org/sqlite` (no external DB server needed)

## Quick Start

1. Clone and enter the repo.
2. Copy the config template.
3. Start the server.

```bash
git clone https://codeberg.org/crowdware/forgecrowdbook.git
cd forgecrowdbook
cp app-demo.sml app.sml
go run .
```

The app starts on `http://localhost:8090` by default.

## Configuration

Create `app.sml` from the template:

```bash
cp app-demo.sml app.sml
```

Non-sensitive settings (host, port, DB path, SMTP host) go directly into `app.sml`. **Secrets are never stored in the file** — they are passed via environment variables instead.

### Environment Variables

| Variable | Description | Required |
|---|---|---|
| `FCB_SESSION_SECRET` | HMAC secret for session cookies (min. 32 chars) | yes |
| `FCB_ADMIN_EMAIL` | E-mail address of the admin account | yes |
| `FCB_SMTP_USER` | SMTP login username | yes (for mail) |
| `FCB_SMTP_PASS` | SMTP login password | yes (for mail) |

`app.sml` is listed in `.gitignore` and will never be committed. `app-demo.sml` is the safe-to-commit template with all secrets left blank.

### Local Setup

**1. Generate a session secret and set secrets in your shell (or a `.envrc` file):**

```bash
# Generate a secure random secret (pick one):
openssl rand -hex 32          # macOS / Linux with OpenSSL
python3 -c "import secrets; print(secrets.token_hex(32))"  # Python 3

export FCB_SESSION_SECRET="<paste-generated-secret-here>"
export FCB_ADMIN_EMAIL="you@example.com"
export FCB_SMTP_USER="your-smtp-user"
export FCB_SMTP_PASS="your-smtp-password"
```

**2. Edit `app.sml` — non-secret values only:**

```sml
App {
    name: "ForgeCrowdBook"
    base_url: "http://localhost:8090"
    db: "./data/crowdbook.db"
    port: "8090"
    session_secret: ""
    admin_email: ""
    SMTP {
        host: "smtp.example.com"
        port: "587"
        user: ""
        pass: ""
        from: "noreply@example.com"
    }
}
```

**3. Start:**

```bash
./run.sh start
```

## Running Tests

```bash
go test ./...
```

## Helper Script

```bash
./run.sh build
./run.sh test
./run.sh start
./run.sh mirror github
```

## Project Docs

- Full specification: [`spec.md`](spec.md)
- Work backlog: [`CWUP/BACKLOG.md`](CWUP/BACKLOG.md)
- Release checklist: [`RELEASE.md`](RELEASE.md)

## Related

- [ForgeCMS](https://codeberg.org/crowdware/forgecms) — the content rendering engine that powers the public reading view
