# content_fetcher

## Goal
Fetch and cache Markdown content from external URLs (GitHub, Codeberg, IPFS gateways).
Reuses ForgeCMS content infrastructure.

## Context
ForgeCMS already implements URL-based content fetching with ETag caching and stale fallback.
ForgeCrowdBook adds this as a dependency and wraps it with IPFS URL normalization.

## Tasks

### 1 — Add ForgeCMS as dependency
```
go get codeberg.org/crowdware/forgecms
```
Use `codeberg.org/crowdware/forgecms/internal/content` (CacheManager) and
`codeberg.org/crowdware/forgecms/internal/compiler` (Compile).

### 2 — `internal/fetcher/fetcher.go`

```go
type Fetcher struct {
    cache       *content.CacheManager
    ipfsGateway string
}

func New(cacheDir, manifestPath string, ttl int, ipfsGateway string) (*Fetcher, error)

// NormalizeURL converts ipfs:// to gateway URL, passes others through unchanged.
func (f *Fetcher) NormalizeURL(rawURL string) string

// FetchHTML fetches content from URL (normalizing IPFS), compiles Markdown to HTML.
// Returns compiled HTML string.
func (f *Fetcher) FetchHTML(url string) (string, error)
```

### 3 — IPFS normalization
```go
func (f *Fetcher) NormalizeURL(rawURL string) string {
    if strings.HasPrefix(rawURL, "ipfs://") {
        cid := strings.TrimPrefix(rawURL, "ipfs://")
        return f.ipfsGateway + cid
    }
    return rawURL
}
```

### 4 — Wire into Handler
`Handler` receives a `*fetcher.Fetcher`. Used by:
- `POST /dashboard/preview`
- `GET /dashboard/chapters/{id}` (author preview)
- `GET /books/{slug}/{chapter-slug}` (public reader)
- `GET /admin/chapters` (admin preview)

## Acceptance Criteria
- `FetchHTML("ipfs://QmABC")` fetches from configured gateway.
- `FetchHTML("https://codeberg.org/...")` fetches directly.
- Unreachable URL returns error (caller shows "Content unavailable").
- Cache persists across restarts (manifest file).
- `go test ./internal/fetcher/...` passes with mock HTTP server.
