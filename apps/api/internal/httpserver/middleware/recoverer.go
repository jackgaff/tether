package middleware

import (
	"net/http"

	"nova-echoes/api/internal/httpserver/respond"
)

func Recoverer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					respond.Error(w, http.StatusInternalServerError, "internal_error", "The request failed unexpectedly.")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
