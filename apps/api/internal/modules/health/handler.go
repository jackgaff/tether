package health

import (
	"net/http"
	"time"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver/respond"
)

func NewHandler(cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{
			"status":                "ok",
			"service":               cfg.AppName,
			"env":                   cfg.AppEnv,
			"authMode":              cfg.AuthMode,
			"databaseURLConfigured": cfg.DatabaseURL != "",
			"time":                  time.Now().UTC().Format(time.RFC3339),
		}, nil)
	})
}
