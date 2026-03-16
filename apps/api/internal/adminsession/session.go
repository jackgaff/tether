package adminsession

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tether/api/internal/config"
	"tether/api/internal/httpserver/middleware"
	"tether/api/internal/httpserver/respond"
)

const (
	CookieName = "tether_admin_session"
	sessionTTL = 12 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid admin credentials")
	ErrMissingSession     = errors.New("admin session is required")
	ErrInvalidSession     = errors.New("admin session is invalid")
)

type Claims struct {
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type contextKey string

const usernameContextKey contextKey = "admin_username"

type Manager struct {
	username string
	password string
	secret   []byte
	secure   bool
	now      func() time.Time
}

func New(cfg config.Config) *Manager {
	return &Manager{
		username: cfg.AdminUsername,
		password: cfg.AdminPassword,
		secret:   []byte(cfg.AdminSessionSecret),
		secure:   strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "production"),
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (m *Manager) ValidateCredentials(username, password string) error {
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte(m.username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(m.password)) != 1 {
		return ErrInvalidCredentials
	}

	return nil
}

func (m *Manager) SessionClaims(r *http.Request) (Claims, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return Claims{}, ErrMissingSession
		}
		return Claims{}, ErrInvalidSession
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidSession
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidSession
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidSession
	}

	if subtle.ConstantTimeCompare(signature, m.sign(payload)) != 1 {
		return Claims{}, ErrInvalidSession
	}

	fields := strings.Split(string(payload), "\n")
	if len(fields) != 2 {
		return Claims{}, ErrInvalidSession
	}

	expiresAt, err := time.Parse(time.RFC3339, fields[1])
	if err != nil {
		return Claims{}, ErrInvalidSession
	}

	if m.now().After(expiresAt) {
		return Claims{}, ErrInvalidSession
	}

	return Claims{
		Username:  fields[0],
		ExpiresAt: expiresAt,
	}, nil
}

func (m *Manager) SetSession(w http.ResponseWriter, username string) {
	expiresAt := m.now().Add(sessionTTL)
	payload := []byte(username + "\n" + expiresAt.Format(time.RFC3339))
	value := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(m.sign(payload))

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (m *Manager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func (m *Manager) Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := m.SessionClaims(r)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "A valid admin session is required.")
				return
			}

			ctx := context.WithValue(r.Context(), usernameContextKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameContextKey).(string)
	return username, ok && strings.TrimSpace(username) != ""
}

func (m *Manager) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func (m *Manager) String() string {
	return fmt.Sprintf("Manager{cookie=%q}", CookieName)
}
