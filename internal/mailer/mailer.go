package mailer

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
)

type Mailer struct {
	cfg  config.SMTPConfig
	send func(to, subject, body string) error
}

func New(cfg config.SMTPConfig) *Mailer {
	m := &Mailer{cfg: cfg}
	m.send = m.sendSMTP
	return m
}

func (m *Mailer) Send(to, subject, body string) error {
	if err := m.send(to, subject, body); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

func (m *Mailer) SendMagicLink(to, link, lang string, bundle *i18n.Bundle) error {
	subject := bundle.T(lang, "email_magic_link_subject")
	body := fmt.Sprintf("%s\n\n%s", bundle.T(lang, "email_magic_link_body"), link)
	return m.Send(to, subject, body)
}

func (m *Mailer) SendChapterPublished(to, title, lang string, bundle *i18n.Bundle) error {
	subject := bundle.T(lang, "email_chapter_published_subject")
	body := fmt.Sprintf("%s: %s", bundle.T(lang, "email_chapter_published_body"), title)
	return m.Send(to, subject, body)
}

func (m *Mailer) SendChapterRejected(to, title, lang string, bundle *i18n.Bundle) error {
	subject := bundle.T(lang, "email_chapter_rejected_subject")
	body := fmt.Sprintf("%s: %s", bundle.T(lang, "email_chapter_rejected_body"), title)
	return m.Send(to, subject, body)
}

func (m *Mailer) SendLikeMilestone(to string, count int, title, lang string, bundle *i18n.Bundle) error {
	subject := bundle.T(lang, "email_likes_milestone_subject")
	body := fmt.Sprintf("%s %d: %s", bundle.T(lang, "email_likes_milestone_body"), count, title)
	return m.Send(to, subject, body)
}

func IsMilestone(count int) bool {
	switch count {
	case 1, 5, 10, 25, 50, 100:
		return true
	default:
		return false
	}
}

func (m *Mailer) sendSMTP(to, subject, body string) error {
	host := strings.TrimSpace(m.cfg.Host)
	port := strings.TrimSpace(m.cfg.Port)
	if host == "" || port == "" {
		return fmt.Errorf("SMTP host/port not configured")
	}

	addr := net.JoinHostPort(host, port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         host,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start TLS: %w", err)
		}
	}

	if m.cfg.User != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP user: %w", err)
		}
	}

	from := m.cfg.From
	if from == "" {
		from = m.cfg.User
	}
	if from == "" {
		return fmt.Errorf("SMTP from address not configured")
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set mail sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set mail recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP data stream: %w", err)
	}

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", to, subject, body)
	if _, err := writer.Write([]byte(msg)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close SMTP data stream: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP connection: %w", err)
	}
	return nil
}
