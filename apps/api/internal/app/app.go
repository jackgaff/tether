package app

import (
	"net/http"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver"
	"nova-echoes/api/internal/modules/checkins"
)

type App struct {
	Config  config.Config
	Handler http.Handler
}

func New(cfg config.Config) *App {
	store := checkins.NewMemoryStore()

	return &App{
		Config:  cfg,
		Handler: httpserver.New(cfg, store),
	}
}
