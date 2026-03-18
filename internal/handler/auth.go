package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type MagicLinkSender interface {
	SendMagicLink(to, link, lang string, bundle *i18n.Bundle) error
}

type AuthHandler struct {
	DB     *sql.DB
	Config *config.Config
	I18N   *i18n.Bundle
	Mailer MagicLinkSender
}

func NewAuthHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, mailer MagicLinkSender) *AuthHandler {
	return &AuthHandler{
		DB:     db,
		Config: cfg,
		I18N:   bundle,
		Mailer: mailer,
	}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprint(w, `<html><body><h1>Login</h1><form method="POST" action="/login"><input type="email" name="email" required><button type="submit">Send login link</button></form></body></html>`)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprint(w, `<html><body><h1>Register</h1><form method="POST" action="/register"><input type="text" name="display_name" required><input type="email" name="email" required><button type="submit">Register</button></form></body></html>`)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if !isValidEmail(email) {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	lang := i18n.DetectLang(r, i18n.SupportedLanguages)
	user, err := model.GetUserByEmail(h.DB, email)
	if err != nil {
		http.Error(w, "failed to query user", http.StatusInternalServerError)
		return
	}

	if user != nil {
		token, err := auth.CreateMagicToken(h.DB, email)
		if err != nil {
			if errors.Is(err, auth.ErrRateLimited) {
				http.Error(w, "Too many login links requested. Please try again later.", http.StatusTooManyRequests)
				return
			}
			http.Error(w, "failed to create login link", http.StatusInternalServerError)
			return
		}

		if err := h.Mailer.SendMagicLink(email, h.authLink(token), lang, h.I18N); err != nil {
			http.Error(w, "failed to send login email", http.StatusInternalServerError)
			return
		}
	}

	h.renderConfirm(w, lang)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if displayName == "" || !isValidEmail(email) {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	lang := i18n.DetectLang(r, i18n.SupportedLanguages)
	user, err := model.GetUserByEmail(h.DB, email)
	if err != nil {
		http.Error(w, "failed to query user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		if _, err := model.CreateUser(h.DB, email, displayName); err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}
	}

	token, err := auth.CreateMagicToken(h.DB, email)
	if err != nil {
		if errors.Is(err, auth.ErrRateLimited) {
			http.Error(w, "Too many login links requested. Please try again later.", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "failed to create login link", http.StatusInternalServerError)
		return
	}

	if err := h.Mailer.SendMagicLink(email, h.authLink(token), lang, h.I18N); err != nil {
		http.Error(w, "failed to send login email", http.StatusInternalServerError)
		return
	}

	h.renderConfirm(w, lang)
}

func (h *AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		h.renderAuthError(w, http.StatusBadRequest, "Missing token.")
		return
	}

	email, err := auth.ValidateToken(h.DB, token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			h.renderAuthError(w, http.StatusUnauthorized, "This login link is invalid or has expired.")
			return
		}
		http.Error(w, "failed to validate login link", http.StatusInternalServerError)
		return
	}

	auth.CreateSession(w, email, h.Config.SessionSecret)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) authLink(token string) string {
	base := strings.TrimRight(h.Config.BaseURL, "/")
	if base == "" {
		base = ""
	}
	return base + "/auth?token=" + token
}

func (h *AuthHandler) renderConfirm(w http.ResponseWriter, lang string) {
	message := h.I18N.T(lang, "msg_check_email")
	fmt.Fprintf(w, "<html><body><h1>Check your email</h1><p>%s</p></body></html>", message)
}

func (h *AuthHandler) renderAuthError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	fmt.Fprintf(w, `<html><body><h1>Login link error</h1><p>%s</p><p><a href="/login">Try again</a></p></body></html>`, message)
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
