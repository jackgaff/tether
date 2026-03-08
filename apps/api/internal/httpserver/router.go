package httpserver

import (
	"net/http"
	"os"
	"path/filepath"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver/middleware"
	"nova-echoes/api/internal/httpserver/respond"
	"nova-echoes/api/internal/modules/checkins"
	"nova-echoes/api/internal/modules/health"
)

func New(cfg config.Config, store checkins.Store) http.Handler {
	mux := http.NewServeMux()

	checkInHandler := checkins.NewHandler(store)
	apiMiddleware := middleware.Chain(
		middleware.APIKeyAuth(cfg),
	)
	openAPIPath := resolveOpenAPIPath()

	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{
			"name":        cfg.AppName,
			"environment": cfg.AppEnv,
			"message":     "Nova Echoes API is ready for hackathon development.",
			"docsPath":    "/openapi.yaml",
		}, nil)
	}))

	mux.Handle("GET /openapi.yaml", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, openAPIPath)
	}))
	mux.Handle("GET /health", health.NewHandler(cfg))
	mux.Handle("GET /api/v1/check-ins", middleware.Apply(http.HandlerFunc(checkInHandler.List), apiMiddleware...))
	mux.Handle("POST /api/v1/check-ins", middleware.Apply(http.HandlerFunc(checkInHandler.Create), apiMiddleware...))

	return middleware.Apply(
		mux,
		middleware.RequestLogger(),
		middleware.Recoverer(),
		middleware.CORS(cfg),
	)
}

func resolveOpenAPIPath() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return filepath.Join("docs", "openapi.yaml")
	}

	for {
		for _, candidate := range []string{
			filepath.Join(currentDir, "docs", "openapi.yaml"),
			filepath.Join(currentDir, "apps", "api", "docs", "openapi.yaml"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	return filepath.Join("docs", "openapi.yaml")
}
