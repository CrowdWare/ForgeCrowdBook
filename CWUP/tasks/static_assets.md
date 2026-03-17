# static_assets

## Goal
Bundle static assets — no CDN, no JavaScript framework, minimal footprint.

## Tasks

### 1 — `static/style.css`
Base styles:
- CSS reset / normalize
- Responsive nav with language dropdown
- Book and chapter cards
- Registration form layout
- Preview widget (source URL input + rendered preview area)
- Status badges (draft, pending_review, published, rejected)
- Like button
- Share buttons

### 2 — `static/preview.js`
Minimal vanilla JS for the preview widget:
```javascript
// Calls POST /dashboard/preview with source_url,
// injects returned HTML into #preview-area,
// enables the Submit button on success.
```
No framework. ~30 lines.

### 3 — Serve via `http.FileServer`
```go
mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
```

## Acceptance Criteria
- App runs with no external HTTP requests for assets.
- Preview widget works with GitHub, Codeberg, and IPFS gateway URLs.
- CSS is responsive on mobile.
- Total JS footprint: < 50 lines vanilla JS.
