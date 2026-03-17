# ForgeCrowdBook

A collaborative book platform built in Go. Authors host their own Markdown content on GitHub, Codeberg, or IPFS — ForgeCrowdBook handles discovery, moderation, and publishing.

## Philosophy

- No passwords — Magic Link authentication only
- No content lock-in — authors own their files, we only store a URL
- No third parties — runs entirely on your own server
- No database server — SQLite, single file

## Status

Planning phase. See [`SPEC.md`](SPEC.md) for the full specification and [`CWUP/BACKLOG.md`](CWUP/BACKLOG.md) for the task backlog.

## Related

- [ForgeCMS](https://codeberg.org/crowdware/forgecms) — the content rendering engine that powers the public reading view
