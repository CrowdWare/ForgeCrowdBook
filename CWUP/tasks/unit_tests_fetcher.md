# unit_tests_fetcher

## Goal
Unit tests for content fetching, IPFS URL normalization, and Markdown compilation.

## Tasks

### Fetcher tests (`internal/fetcher/fetcher_test.go`)
- `NormalizeURL("ipfs://QmABC")` → `"https://ipfs.io/ipfs/QmABC"` (default gateway)
- `NormalizeURL("https://codeberg.org/...")` → unchanged
- `FetchHTML(mockURL)` with valid Markdown → returns non-empty HTML string
- `FetchHTML(mockURL)` with `---` in content → compiled HTML contains `<hr>`
- `FetchHTML(mockURL)` with `> quote` → compiled HTML contains `<blockquote>`
- `FetchHTML(unreachableURL)` → returns error, not panic
- Second call with same URL + unchanged ETag → 304, served from cache

### Content integrity (these guard against the old WordPress bug)
- Markdown with `---` → rendered `<hr>`, never `-` or `&#x2014;`
- Markdown with `>` → rendered `<blockquote>`, never `&gt;`
- Markdown with `<code>` → preserved in output

## Acceptance Criteria
- All tests use `httptest.NewServer` — no real network calls
- Cache uses `t.TempDir()`
