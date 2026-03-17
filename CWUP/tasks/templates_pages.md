# templates_pages

## Goal
All page templates beyond base.html.

## Tasks

Each template extends `{{ template "base" . }}` and defines `{{ block "content" . }}`.

### Templates to implement
| File | Route | Notes |
|------|-------|-------|
| `home.html` | `GET /` | CTA buttons, short intro |
| `books.html` | `GET /books` | Book cards grid |
| `book.html` | `GET /books/{slug}` | Chapter cards with excerpt |
| `chapter.html` | `GET /books/{slug}/{chapter-slug}` | Rendered HTML, likes, share, OG tags |
| `login.html` | `GET /login` | Email form |
| `register.html` | `GET /register` | Display name + email form |
| `confirm.html` | after login/register | "Check your email" message |
| `auth-error.html` | invalid token | Friendly error + retry link |
| `dashboard.html` | `GET /dashboard` | Book selection cards |
| `chapters.html` | `GET /dashboard/chapters` | Author's chapter list for active book |
| `chapter-editor.html` | new / edit | EasyMDE, title, path label inputs |
| `chapter-preview.html` | after save / preview | Read-only rendered view + Edit button |
| `admin-chapters.html` | `GET /admin/chapters` | Moderation table |
| `admin-users.html` | `GET /admin/users` | User management table |

### EasyMDE integration in `chapter-editor.html`
```html
{{ block "scripts" . }}
<link rel="stylesheet" href="/static/easymde.min.css">
<script src="/static/easymde.min.js"></script>
<script>
var easyMDE = new EasyMDE({
    element: document.getElementById("content"),
    initialValue: {{ .MarkdownJSON }},
    spellChecker: false,
    toolbar: ["bold","italic","heading","|","quote","unordered-list","ordered-list","|","link","image","|","preview","side-by-side","fullscreen"],
    previewRender: function(text) { return easyMDE.markdown(text); }
});
</script>
{{ end }}
```
`MarkdownJSON` = `json.Marshal(chapter.MarkdownContent)` — safe for `<script>` context.

## Acceptance Criteria
- All templates render without error when given valid data.
- `chapter-preview.html` has no form or textarea — strictly read-only.
- "Edit" button on preview is the only path back to the editor.
