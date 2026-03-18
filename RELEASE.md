# Release Guide

## Principle

Reproducibility first.

- Never change a running system.
- No "please update 14 plugins" workflow.
- Deploy a tested binary with pinned dependencies (`go.mod`, `go.sum`).

## Pre-Release Checklist

1. Ensure working tree is clean.
2. Run full test suite:
   ```bash
   go test ./...
   ```
3. Verify config file for target environment (`app.sml`).
4. Build release binary:
   ```bash
   go build -o forgecrowdbook .
   ```

## Release Steps

1. Tag the release commit:
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```
2. Ship the binary plus templates/static/i18n files.
3. Keep previous binary for rollback.

## Smoke Test After Deploy

1. Open `/` and `/books`.
2. Trigger login flow (`/login` -> magic link).
3. Open `/dashboard` as authenticated user.
4. Run one like action on a published chapter (`/api/like/{id}`).

## Rollback

1. Stop current process.
2. Restore previous binary and previous `app.sml`.
3. Start service again.
4. Re-run smoke test.
