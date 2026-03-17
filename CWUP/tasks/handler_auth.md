# handler_auth

## Goal
Login, register, magic link validation, and logout handlers.

## Tasks

### 1 — `GET /login` → `templates/login.html`
Form: email input + "Send login link" button.

### 2 — `POST /login`
- Validate email format.
- Call `auth.CreateMagicToken`. If `ErrRateLimited` → show error.
- Send magic link email via mailer.
- Redirect to confirmation page (generic — no user enumeration).

### 3 — `GET /register` → `templates/register.html`
Form: display name + email input.

### 4 — `POST /register`
- Validate inputs.
- `GetUserByEmail`: if not found → `CreateUser`.
- Call `auth.CreateMagicToken`.
- Send magic link email.
- Redirect to confirmation page.

### 5 — `GET /auth?token=...`
- Call `auth.ValidateToken`.
- If `ErrInvalidToken` → render error page.
- Create session cookie.
- Redirect to `/dashboard`.

### 6 — `POST /logout`
(implemented in handler_home — just reference here)

## Acceptance Criteria
- Invalid token shows friendly error, not 500.
- Rate-limited email shows friendly error.
- Both login and register show identical confirmation message (no user enumeration).
