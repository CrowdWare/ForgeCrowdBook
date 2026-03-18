package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "session"

type contextKey string

const (
	contextKeyEmail   contextKey = "email"
	contextKeyIsAdmin contextKey = "is_admin"
)

func CreateSession(w http.ResponseWriter, email string, secret string) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", email, expiresAt)
	signature := sign(payload, secret)
	rawValue := payload + "|" + signature
	encoded := base64.RawURLEncoding.EncodeToString([]byte(rawValue))

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	})
}

func GetSession(r *http.Request, secret string) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", false
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 {
		return "", false
	}

	email := parts[0]
	expiresRaw := parts[1]
	signature := parts[2]

	payload := email + "|" + expiresRaw
	if !hmac.Equal([]byte(signature), []byte(sign(payload, secret))) {
		return "", false
	}

	expiresUnix, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil {
		return "", false
	}
	if time.Now().Unix() > expiresUnix {
		return "", false
	}

	return email, true
}

func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func IsAdmin(email, adminEmail string) bool {
	return email != "" && email == adminEmail
}

func RequireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := GetSession(r, secret)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyEmail, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAdmin(secret, adminEmail string, next http.Handler) http.Handler {
	authWrapped := RequireAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := EmailFromContext(r.Context())
		if !IsAdmin(email, adminEmail) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyIsAdmin, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
	return authWrapped
}

func EmailFromContext(ctx context.Context) string {
	email, _ := ctx.Value(contextKeyEmail).(string)
	return email
}

func IsAdminFromContext(ctx context.Context) bool {
	value, _ := ctx.Value(contextKeyIsAdmin).(bool)
	return value
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
