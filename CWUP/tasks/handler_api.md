# handler_api

## Goal
JSON API endpoint for the like system.

## Tasks

### 1 — `POST /api/like/{chapter-id}`
- No authentication required.
- Compute fingerprint: `SHA256(remoteIP + userAgent)`.
- Call `model.AddLike`.
- If new like: check milestone, send email to author if milestone reached.
- Return JSON: `{"count": N, "liked": true|false}`.

### 2 — Milestone email trigger
After a successful new like, check if `newCount` is in `[1, 5, 10, 25, 50, 100]`.
If yes: load chapter author, send milestone email via mailer.

## Acceptance Criteria
- Returns `{"liked": false}` without error on duplicate (no 4xx).
- Returns 404 for unknown chapter ID.
- Milestone email is sent exactly once per threshold.
- Response is always `application/json`.
