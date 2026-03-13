package middleware

import (
	"net/http"
	"slices"
	"strings"

	"nova-echoes/api/internal/config"
)

func CORS(cfg config.Config) Middleware {
	allowedOrigins := make([]string, 0, len(cfg.AllowedFrontendOrigins))
	for _, origin := range cfg.AllowedFrontendOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowedOrigins = append(allowedOrigins, origin)
	}
	if len(allowedOrigins) == 0 && strings.TrimSpace(cfg.FrontendOrigin) != "" {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(cfg.FrontendOrigin))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))

			if origin != "" && (slices.Contains(allowedOrigins, "*") || slices.Contains(allowedOrigins, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
