package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"nova-echoes/api/db"
	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver"
	"nova-echoes/api/internal/modules/checkins"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voice"
	"nova-echoes/api/internal/modules/voicecatalog"
)

type App struct {
	Config   config.Config
	Handler  http.Handler
	db       *sql.DB
	sessions *voice.SessionManager
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	migrateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.Migrate(migrateCtx, database); err != nil {
		_ = database.Close()
		return nil, err
	}

	awsCfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(cfg.BedrockRegion))
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	catalog, err := voicecatalog.New(cfg.NovaDefaultVoiceID, cfg.NovaAllowedVoiceIDs)
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	checkInHandler := checkins.NewHandler(checkins.NewPostgresStore(database))
	preferenceStore := preferences.NewPostgresStore(database)
	preferenceHandler := preferences.NewHandler(preferenceStore, catalog)
	sessionManager := voice.NewSessionManager()
	voiceService := voice.NewService(
		cfg,
		catalog,
		voice.NewPostgresRepository(database),
		preferenceStore,
		voice.NewBedrockAdapter(bedrockruntime.NewFromConfig(awsCfg)),
		voice.NewFileArtifactExporter(cfg.VoiceLabExportDir),
		sessionManager,
	)
	voiceHandler := voice.NewHandler(voiceService, cfg.AllowedFrontendOrigins)

	return &App{
		Config: cfg,
		Handler: httpserver.New(cfg, httpserver.Dependencies{
			CheckIns:    checkInHandler,
			Preferences: preferenceHandler,
			Voice:       voiceHandler,
		}),
		db:       database,
		sessions: sessionManager,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.sessions != nil {
		_ = a.sessions.CloseAll()
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
