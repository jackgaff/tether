package middleware

import (
	"net/http"

	"tether/api/internal/config"
	"tether/api/internal/httpserver/respond"
)

func APIKeyAuth(cfg config.Config) Middleware {
	return func(next http.Handler) http.Handler {
		if cfg.AuthMode != "api-key" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != cfg.InternalAPIKey {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "A valid X-API-Key header is required.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
