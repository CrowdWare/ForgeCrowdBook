package mailer

import (
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
)

func TestIsMilestone(t *testing.T) {
	if !IsMilestone(10) {
		t.Fatal("expected 10 to be a milestone")
	}
	if IsMilestone(7) {
		t.Fatal("expected 7 to not be a milestone")
	}
}

func TestSendMagicLinkUsesI18nSubject(t *testing.T) {
	bundle, err := i18n.Load(filepath.Join("..", "..", "i18n"))
	if err != nil {
		t.Fatalf("i18n.Load failed: %v", err)
	}

	m := New(config.SMTPConfig{})
	var gotSubject string
	var gotBody string
	m.send = func(to, subject, body string) error {
		gotSubject = subject
		gotBody = body
		return nil
	}

	if err := m.SendMagicLink("user@example.com", "https://example.org/auth?token=abc", "en", bundle); err != nil {
		t.Fatalf("SendMagicLink failed: %v", err)
	}

	if gotSubject != "Your magic login link" {
		t.Fatalf("unexpected subject: %q", gotSubject)
	}
	if !strings.Contains(gotBody, "https://example.org/auth?token=abc") {
		t.Fatalf("expected magic link in body, got: %q", gotBody)
	}
}
