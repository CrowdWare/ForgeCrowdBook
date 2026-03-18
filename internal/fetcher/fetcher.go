package fetcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
)

type Fetcher struct {
	cacheDir     string
	manifestPath string
	ttl          time.Duration
	ipfsGateway  string
	client       *http.Client

	mu       sync.Mutex
	manifest map[string]cacheEntry
}

type cacheEntry struct {
	FileName  string    `json:"file_name"`
	ETag      string    `json:"etag,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}

func New(cacheDir, manifestPath string, ttl int, ipfsGateway string) (*Fetcher, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("cache directory is empty")
	}
	if manifestPath == "" {
		return nil, fmt.Errorf("manifest path is empty")
	}
	if ttl <= 0 {
		ttl = 300
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return nil, fmt.Errorf("create manifest directory: %w", err)
	}

	f := &Fetcher{
		cacheDir:     cacheDir,
		manifestPath: manifestPath,
		ttl:          time.Duration(ttl) * time.Second,
		ipfsGateway:  normalizeGateway(ipfsGateway),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		manifest: map[string]cacheEntry{},
	}

	if err := f.loadManifest(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *Fetcher) NormalizeURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "ipfs://") {
		cid := strings.TrimPrefix(rawURL, "ipfs://")
		return f.ipfsGateway + strings.TrimPrefix(cid, "/")
	}
	return rawURL
}

func (f *Fetcher) FetchHTML(url string) (string, error) {
	normalizedURL := f.NormalizeURL(url)
	if normalizedURL == "" {
		return "", fmt.Errorf("URL is empty")
	}

	content, err := f.fetchContent(normalizedURL)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := goldmark.Convert(content, &out); err != nil {
		return "", fmt.Errorf("compile markdown: %w", err)
	}

	return out.String(), nil
}

func (f *Fetcher) FetchMarkdown(url string) (string, error) {
	normalizedURL := f.NormalizeURL(url)
	if normalizedURL == "" {
		return "", fmt.Errorf("URL is empty")
	}

	content, err := f.fetchContent(normalizedURL)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (f *Fetcher) Invalidate(url string) error {
	normalizedURL := f.NormalizeURL(url)
	if normalizedURL == "" {
		return fmt.Errorf("URL is empty")
	}

	f.mu.Lock()
	entry, ok := f.manifest[normalizedURL]
	if ok {
		delete(f.manifest, normalizedURL)
	}
	f.mu.Unlock()

	if ok {
		_ = os.Remove(filepath.Join(f.cacheDir, entry.FileName))
	}

	if err := f.saveManifest(); err != nil {
		return err
	}
	return nil
}

func (f *Fetcher) fetchContent(url string) ([]byte, error) {
	f.mu.Lock()
	entry, hasEntry := f.manifest[url]
	f.mu.Unlock()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if hasEntry && entry.ETag != "" {
		req.Header.Set("If-None-Match", entry.ETag)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		if hasEntry {
			return f.readCacheIfFresh(entry)
		}
		return nil, fmt.Errorf("fetch content: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
		updatedEntry, err := f.writeCache(url, body, resp.Header.Get("ETag"))
		if err != nil {
			return nil, err
		}
		f.mu.Lock()
		f.manifest[url] = updatedEntry
		f.mu.Unlock()
		if err := f.saveManifest(); err != nil {
			return nil, err
		}
		return body, nil
	case http.StatusNotModified:
		if hasEntry {
			return f.readCache(entry)
		}
		return nil, fmt.Errorf("received 304 without local cache for %s", url)
	default:
		return nil, fmt.Errorf("fetch content: unexpected HTTP status %d", resp.StatusCode)
	}
}

func (f *Fetcher) readCacheIfFresh(entry cacheEntry) ([]byte, error) {
	if time.Since(entry.FetchedAt) > f.ttl {
		return nil, fmt.Errorf("fetch content: origin unavailable and cache is stale")
	}
	return f.readCache(entry)
}

func (f *Fetcher) readCache(entry cacheEntry) ([]byte, error) {
	path := filepath.Join(f.cacheDir, entry.FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cached content: %w", err)
	}
	return data, nil
}

func (f *Fetcher) writeCache(url string, body []byte, etag string) (cacheEntry, error) {
	hash := sha256.Sum256([]byte(url))
	fileName := hex.EncodeToString(hash[:]) + ".md"
	path := filepath.Join(f.cacheDir, fileName)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return cacheEntry{}, fmt.Errorf("write cached content: %w", err)
	}
	return cacheEntry{
		FileName:  fileName,
		ETag:      etag,
		FetchedAt: time.Now().UTC(),
	}, nil
}

func (f *Fetcher) loadManifest() error {
	raw, err := os.ReadFile(f.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read manifest: %w", err)
	}

	if len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, &f.manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	return nil
}

func (f *Fetcher) saveManifest() error {
	f.mu.Lock()
	raw, err := json.MarshalIndent(f.manifest, "", "  ")
	f.mu.Unlock()
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(f.manifestPath, raw, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func normalizeGateway(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "https://ipfs.io/ipfs/"
	}
	if !strings.HasSuffix(trimmed, "/") {
		trimmed += "/"
	}
	return trimmed
}
