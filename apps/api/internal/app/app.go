package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"nova-echoes/api/db"
	"nova-echoes/api/internal/adminsession"
	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver"
	"nova-echoes/api/internal/modules/admin"
	"nova-echoes/api/internal/modules/checkins"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voice"
	"nova-echoes/api/internal/modules/voicecatalog"
)

type App struct {
	Config           config.Config
	Handler          http.Handler
	db               *sql.DB
	sessions         *voice.SessionManager
	backgroundCancel context.CancelFunc
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

	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	checkInHandler := checkins.NewHandler(checkins.NewPostgresStore(database))
	preferenceStore := preferences.NewPostgresStore(database)
	preferenceHandler := preferences.NewHandler(preferenceStore, catalog)
	sessionManager := voice.NewSessionManager()
	voiceService := voice.NewService(
		cfg,
		catalog,
		voice.NewPostgresRepository(database),
		preferenceStore,
		voice.NewBedrockAdapter(bedrockClient),
		voice.NewFileArtifactExporter(cfg.VoiceLabExportDir),
		sessionManager,
	)
	voiceHandler := voice.NewHandler(voiceService, cfg.AllowedFrontendOrigins)
	adminStore := admin.NewPostgresStore(database)
	adminAnalyzer := admin.NewBedrockAnalyzer(bedrockClient, cfg.NovaAnalysisModelID)
	adminService := admin.NewService(adminStore, voiceService, cfg.NovaAnalysisModelID)
	adminSessions := adminsession.New(cfg)
	adminHandler := admin.NewHandler(adminStore, adminService, adminSessions)

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	if cfg.AnalysisWorkerEnabled {
		go admin.NewAnalysisWorker(adminStore, adminAnalyzer).Run(backgroundCtx, cfg.AnalysisWorkerPollInterval)
	}
	if cfg.ScreeningSchedulerEnabled {
		go admin.NewScreeningScheduler(adminStore).Run(backgroundCtx, cfg.ScreeningSchedulerPollInterval)
	}

	return &App{
		Config: cfg,
		Handler: httpserver.New(cfg, httpserver.Dependencies{
			CheckIns:    checkInHandler,
			Preferences: preferenceHandler,
			Voice:       voiceHandler,
			Admin:       adminHandler,
			AdminAuth:   adminSessions.Middleware(),
		}),
		db:               database,
		sessions:         sessionManager,
		backgroundCancel: backgroundCancel,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.backgroundCancel != nil {
		a.backgroundCancel()
	}

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
