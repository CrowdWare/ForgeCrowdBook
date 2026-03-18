package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const (
	CookieName = "_csrf"
	FieldName  = "_csrf"
	HeaderName = "X-CSRF-Token"
)

func generate() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("csrf: crypto/rand unavailable")
	}
	return hex.EncodeToString(b)
}

// Token returns the CSRF token from the request cookie, or "" if absent.
func Token(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// EnsureToken sets a CSRF token cookie if one is not already present and returns the token.
func EnsureToken(w http.ResponseWriter, r *http.Request) string {
	if tok := Token(r); tok != "" {
		return tok
	}
	tok := generate()
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    tok,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	return tok
}

// Validate checks that the submitted token matches the cookie token.
// It checks the form field "_csrf" first, then the "X-CSRF-Token" header.
func Validate(r *http.Request) bool {
	cookie := Token(r)
	if cookie == "" {
		return false
	}
	submitted := r.FormValue(FieldName)
	if submitted == "" {
		submitted = r.Header.Get(HeaderName)
	}
	if submitted == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie), []byte(submitted)) == 1
}

// Middleware enforces CSRF protection on all POST/PUT/DELETE/PATCH requests.
// It also ensures a CSRF token cookie is set on every response.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		EnsureToken(w, r)
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodDelete || r.Method == http.MethodPatch {
			if !Validate(r) {
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
