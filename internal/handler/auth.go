package handler

import (
	"database/sql"
	"errors"
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
	renderPage(w, r, h.DB, h.Config, h.I18N, "login", map[string]any{"Title": "Login"})
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	renderPage(w, r, h.DB, h.Config, h.I18N, "register", map[string]any{"Title": "Register"})
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

	h.renderConfirm(w, r, lang)
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

	h.renderConfirm(w, r, lang)
}

func (h *AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		h.renderAuthError(w, r, http.StatusBadRequest, "Missing token.")
		return
	}

	email, err := auth.ValidateToken(h.DB, token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			h.renderAuthError(w, r, http.StatusUnauthorized, "This login link is invalid or has expired.")
			return
		}
		http.Error(w, "failed to validate login link", http.StatusInternalServerError)
		return
	}

	user, err := model.GetUserByEmail(h.DB, email)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if user == nil || user.Status == "banned" {
		h.renderAuthError(w, r, http.StatusForbidden, "This account is not allowed to sign in.")
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

func (h *AuthHandler) renderConfirm(w http.ResponseWriter, r *http.Request, lang string) {
	renderPage(w, r, h.DB, h.Config, h.I18N, "confirm", map[string]any{
		"Title":   "Check your email",
		"Message": h.I18N.T(lang, "msg_check_email"),
	})
}

func (h *AuthHandler) renderAuthError(w http.ResponseWriter, r *http.Request, code int, message string) {
	w.WriteHeader(code)
	renderPage(w, r, h.DB, h.Config, h.I18N, "auth-error", map[string]any{
		"Title":   "Login link error",
		"Message": message,
	})
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
