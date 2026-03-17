# templates_base

## Goal
Base HTML template with context-sensitive navigation and language dropdown.

## Tasks

### 1 — `templates/base.html`
Defines `{{ define "base" }}` block used by all pages via `{{ template "base" . }}`.

Structure:
```html
<!DOCTYPE html>
<html lang="{{ .Nav.Lang }}">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Title }} – ForgeCrowdBook</title>
    {{ block "head" . }}{{ end }}
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <nav>
        <!-- Logo / app name -->
        <!-- Home, Books links (always) -->
        <!-- Dashboard (if logged in) -->
        <!-- Login / Register (if not logged in) -->
        <!-- Logout (if logged in) -->
        <!-- Active book badge (if book selected) -->
        <!-- Language dropdown with flag emoji -->
    </nav>
    <main>
        {{ block "content" . }}{{ end }}
    </main>
    <footer>...</footer>
    {{ block "scripts" . }}{{ end }}
</body>
</html>
```

### 2 — Language dropdown
```html
<form method="POST" action="/lang">
    <select name="lang" onchange="this.form.submit()">
        <option value="de" {{ if eq .Nav.Lang "de" }}selected{{ end }}>🇩🇪 Deutsch</option>
        <option value="en" {{ if eq .Nav.Lang "en" }}selected{{ end }}>🇬🇧 English</option>
        <option value="eo" {{ if eq .Nav.Lang "eo" }}selected{{ end }}>🟢 Esperanto</option>
        <option value="pt" {{ if eq .Nav.Lang "pt" }}selected{{ end }}>🇧🇷 Português</option>
        <option value="fr" {{ if eq .Nav.Lang "fr" }}selected{{ end }}>🇫🇷 Français</option>
        <option value="es" {{ if eq .Nav.Lang "es" }}selected{{ end }}>🇪🇸 Español</option>
    </select>
</form>
```

### 3 — Active book badge in nav
If `Nav.ActiveBook != nil`: show book title as a small badge/pill next to Dashboard link.

## Acceptance Criteria
- Nav shows correct items based on `Nav.LoggedIn`.
- Language dropdown submits and reloads in new language.
- Active book badge only shown when a book is selected.
