# handler_admin

## Goal
Admin moderation dashboard for chapters and users.

## Tasks

### 1 — `GET /admin/chapters` → `templates/admin-chapters.html`
Table: title, author, book, path label, status badge, like count, created date.
Filters: by status (all / pending / published / rejected), by date.
Actions per row: Preview | Publish | Reject | Delete.

### 2 — `POST /admin/chapters/{id}/publish`
- Set status to `published`, set `published_at`.
- Send "chapter published" email to author.
- Redirect back with success message.

### 3 — `POST /admin/chapters/{id}/reject`
- Set status to `rejected`.
- Send "chapter rejected" email to author.
- Redirect back.

### 4 — `POST /admin/chapters/{id}/delete`
- Hard delete from DB.
- Redirect back.

### 5 — `GET /admin/users` → `templates/admin-users.html`
Table: display name, email, status, chapter count, registered date.
Actions: Ban | Send Email.

### 6 — `POST /admin/users/{id}/ban`
- Set status to `banned`.
- Redirect back.

## Acceptance Criteria
- Non-admin request → 403.
- Publish/reject sends email to author.
- Banned users cannot log in (magic link flow checks status).
