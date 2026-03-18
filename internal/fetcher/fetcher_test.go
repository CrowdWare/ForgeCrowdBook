package fetcher

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchHTMLValidMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Hello"))
	}))
	defer srv.Close()

	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	html, err := f.FetchHTML(srv.URL)
	if err != nil {
		t.Fatalf("FetchHTML failed: %v", err)
	}
	if html == "" {
		t.Fatal("FetchHTML returned empty HTML")
	}
	if !strings.Contains(html, "<h1>") {
		t.Fatalf("expected heading HTML, got: %s", html)
	}
}

func TestFetchHTMLHorizontalRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("---"))
	}))
	defer srv.Close()

	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	html, err := f.FetchHTML(srv.URL)
	if err != nil {
		t.Fatalf("FetchHTML failed: %v", err)
	}
	if !strings.Contains(html, "<hr") {
		t.Fatalf("expected <hr> in HTML, got: %s", html)
	}
}

func TestFetchHTMLBlockquote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("> quote"))
	}))
	defer srv.Close()

	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	html, err := f.FetchHTML(srv.URL)
	if err != nil {
		t.Fatalf("FetchHTML failed: %v", err)
	}
	if !strings.Contains(html, "<blockquote>") {
		t.Fatalf("expected <blockquote> in HTML, got: %s", html)
	}
}

func TestFetchHTMLUnreachableURL(t *testing.T) {
	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 1, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	_, err = f.FetchHTML("http://127.0.0.1:1/never")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestNormalizeIPFSURL(t *testing.T) {
	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "https://example.gateway/ipfs")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	got := f.NormalizeURL("ipfs://QmABC")
	want := "https://example.gateway/ipfs/QmABC"
	if got != want {
		t.Fatalf("NormalizeURL mismatch: got %q, want %q", got, want)
	}
}

func TestNormalizeURLDefaultGatewayAndHTTPUnchanged(t *testing.T) {
	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if got := f.NormalizeURL("ipfs://QmABC"); got != "https://ipfs.io/ipfs/QmABC" {
		t.Fatalf("default gateway normalization failed: %q", got)
	}

	httpURL := "https://codeberg.org/crowdware/project/raw/main/file.md"
	if got := f.NormalizeURL(httpURL); got != httpURL {
		t.Fatalf("expected unchanged URL, got %q", got)
	}
}

func TestManifestPersistsAcrossRestarts(t *testing.T) {
	cacheDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cached body"))
	}))
	defer srv.Close()

	f1, err := New(cacheDir, manifestPath, 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New (first) failed: %v", err)
	}
	if _, err := f1.FetchHTML(srv.URL); err != nil {
		t.Fatalf("FetchHTML (first) failed: %v", err)
	}

	f2, err := New(cacheDir, manifestPath, 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New (second) failed: %v", err)
	}
	if _, err := f2.FetchHTML(srv.URL); err != nil {
		t.Fatalf("FetchHTML (second) failed: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 HTTP requests, got %d", requestCount)
	}
}

func TestMarkdownIntegritySpecialCharacters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("---\n\n> quote\n\n`<code>`"))
	}))
	defer srv.Close()

	f, err := New(t.TempDir(), filepath.Join(t.TempDir(), "manifest.json"), 300, "https://ipfs.io/ipfs/")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	html, err := f.FetchHTML(srv.URL)
	if err != nil {
		t.Fatalf("FetchHTML failed: %v", err)
	}

	if !strings.Contains(html, "<hr") {
		t.Fatalf("expected <hr>, got %q", html)
	}
	if strings.Contains(html, "&#x2014;") {
		t.Fatalf("unexpected em-dash encoding in html: %q", html)
	}
	if !strings.Contains(html, "<blockquote>") || strings.Contains(html, "&gt; quote") {
		t.Fatalf("blockquote rendering mismatch: %q", html)
	}
	if !strings.Contains(html, "<code>&lt;code&gt;</code>") {
		t.Fatalf("expected code element to be preserved, got %q", html)
	}
}
