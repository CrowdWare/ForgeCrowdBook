package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.sml")
	content := `App {
name: "ForgeCrowdBook"
base_url: "http://localhost:8090"
db: "./data/test.db"
port: "8088"
session_secret: "secret"
admin_email: "admin@example.com"
SMTP {
host: "smtp.example.com"
port: "587"
user: "user"
pass: "pass"
from: "noreply@example.com"
}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Name != "ForgeCrowdBook" || cfg.BaseURL != "http://localhost:8090" {
		t.Fatalf("unexpected basic fields: %+v", cfg)
	}
	if cfg.DBPath != "./data/test.db" || cfg.Port != "8088" {
		t.Fatalf("unexpected db/port: %+v", cfg)
	}
	if cfg.SMTP.Host != "smtp.example.com" || cfg.SMTP.From != "noreply@example.com" {
		t.Fatalf("unexpected smtp fields: %+v", cfg.SMTP)
	}
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.sml")
	content := `App {
name: "ForgeCrowdBook"
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
	}
	if cfg.DBPath != defaultDBPath {
		t.Fatalf("expected default DB path %q, got %q", defaultDBPath, cfg.DBPath)
	}
}

func TestLoadConfigMalformedSML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.sml")
	if err := os.WriteFile(path, []byte(`App {`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected error for malformed SML")
	}
}
