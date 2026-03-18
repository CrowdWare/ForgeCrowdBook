package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

const activeBookCookieName = "active_book_id"

type NavData struct {
	LoggedIn   bool
	IsAdmin    bool
	Email      string
	Lang       string
	Strings    map[string]string
	ActiveBook *model.Book
}

func baseData(r *http.Request, cfg *config.Config, bundle *i18n.Bundle, db *sql.DB) (NavData, error) {
	lang := i18n.DetectLang(r, i18n.SupportedLanguages)
	email, loggedIn := auth.GetSession(r, cfg.SessionSecret)
	isAdmin := loggedIn && auth.IsAdmin(email, cfg.AdminEmail)

	activeBook, err := GetActiveBook(r, db)
	if err != nil {
		return NavData{}, err
	}

	return NavData{
		LoggedIn:   loggedIn,
		IsAdmin:    isAdmin,
		Email:      email,
		Lang:       lang,
		Strings:    bundle.Map(lang),
		ActiveBook: activeBook,
	}, nil
}

func RequireAuth(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.GetSession(r, cfg.SessionSecret); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := auth.GetSession(r, cfg.SessionSecret)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !auth.IsAdmin(email, cfg.AdminEmail) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func SetLang(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	lang := strings.TrimSpace(r.FormValue("lang"))
	if !isSupportedLang(lang) {
		lang = "en"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 60 * 60,
	})

	// Only redirect to a relative path to prevent open redirect via Referer.
	target := "/"
	if ref := r.Referer(); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Path != "" {
			target = u.RequestURI()
		}
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func GetActiveBook(r *http.Request, db *sql.DB) (*model.Book, error) {
	cookie, err := r.Cookie(activeBookCookieName)
	if err != nil {
		return nil, nil
	}

	bookID, err := strconv.Atoi(strings.TrimSpace(cookie.Value))
	if err != nil || bookID <= 0 {
		return nil, nil
	}

	book, err := model.GetBookByID(db, bookID)
	if err != nil {
		return nil, fmt.Errorf("load active book: %w", err)
	}
	return book, nil
}

func SetActiveBook(w http.ResponseWriter, bookID int) {
	http.SetCookie(w, &http.Cookie{
		Name:     activeBookCookieName,
		Value:    strconv.Itoa(bookID),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearActiveBook(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     activeBookCookieName,
		Value:    "",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func isSupportedLang(lang string) bool {
	for _, candidate := range i18n.SupportedLanguages {
		if lang == candidate {
			return true
		}
	}
	return false
}
