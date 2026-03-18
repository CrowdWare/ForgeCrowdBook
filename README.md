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

Create `app.sml` from `app-demo.sml`.

For quick local testing you can also use the demo file:

```bash
cp app-demo.sml app.sml
```

or rename it:

```bash
mv app-demo.sml app.sml
```

Minimal local setup:

```sml
App {
    name: "ForgeCrowdBook"
    base_url: "http://localhost:8090"
    db: "./data/crowdbook.db"
    port: "8090"
    session_secret: "replace-with-a-long-random-secret"
    admin_email: "admin@example.com"
}
```

With SMTP (recommended for real Magic-Link login mails):

```sml
App {
    name: "ForgeCrowdBook"
    base_url: "http://localhost:8090"
    db: "./data/crowdbook.db"
    port: "8090"
    session_secret: "replace-with-a-long-random-secret"
    admin_email: "admin@example.com"
    SMTP {
        host: "smtp.example.com"
        port: "587"
        user: "smtp-user"
        pass: "smtp-password"
        from: "noreply@example.com"
    }
}
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
