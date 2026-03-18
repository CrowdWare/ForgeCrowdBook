package i18n

import (
	"path/filepath"
	"testing"
)

func TestLoadBundleAndFallbacks(t *testing.T) {
	bundle, err := Load(filepath.Join("..", "..", "i18n"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if got := bundle.T("en", "nav_home"); got != "Home" {
		t.Fatalf("expected Home, got %q", got)
	}
	if got := bundle.T("de", "nav_home"); got != "Startseite" {
		t.Fatalf("expected Startseite, got %q", got)
	}
	if got := bundle.T("en", "nonexistent_key"); got != "nonexistent_key" {
		t.Fatalf("expected key fallback, got %q", got)
	}
	if got := bundle.T("xx", "nav_home"); got != "Home" {
		t.Fatalf("expected en fallback Home, got %q", got)
	}
}
