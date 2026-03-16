package middleware

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"tether/api/internal/config"
	"tether/api/internal/httpserver/respond"
)

func RequireTrustedOrigin(cfg config.Config) Middleware {
	allowedOrigins := compactOrigins(cfg.AllowedFrontendOrigins)
	if len(allowedOrigins) == 0 && strings.TrimSpace(cfg.FrontendOrigin) != "" {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(cfg.FrontendOrigin))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			origin := requestOrigin(r)
			if origin == "" || slices.Contains(allowedOrigins, "*") || slices.Contains(allowedOrigins, origin) {
				next.ServeHTTP(w, r)
				return
			}

			respond.Error(w, http.StatusForbidden, "forbidden_origin", "The request origin is not allowed.")
		})
	}
}

func compactOrigins(origins []string) []string {
	items := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		items = append(items, origin)
	}
	return items
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func requestOrigin(r *http.Request) string {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return origin
	}

	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return ""
	}

	parsed, err := url.Parse(referer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}
