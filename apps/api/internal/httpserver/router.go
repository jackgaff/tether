package httpserver

import (
	"net/http"
	"os"
	"path/filepath"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver/middleware"
	"nova-echoes/api/internal/httpserver/respond"
	"nova-echoes/api/internal/modules/admin"
	"nova-echoes/api/internal/modules/checkins"
	"nova-echoes/api/internal/modules/health"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voice"
)

type Dependencies struct {
	CheckIns    checkins.Handler
	Preferences preferences.Handler
	Voice       voice.Handler
	Admin       *admin.Handler
	AdminAuth   middleware.Middleware
}

func New(cfg config.Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()

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
	mux.Handle("GET /api/v1/voice/voices", http.HandlerFunc(deps.Voice.ListVoices))
	mux.Handle("GET /api/v1/voice/lab/conversations", http.HandlerFunc(deps.Voice.ListLabConversations))
	mux.Handle("POST /api/v1/voice/sessions", http.HandlerFunc(deps.Voice.CreateSession))
	mux.Handle("GET /api/v1/voice/sessions/{id}/stream", http.HandlerFunc(deps.Voice.Stream))
	mux.Handle("GET /api/v1/patients/{id}/preferences", http.HandlerFunc(deps.Preferences.Get))
	mux.Handle("PUT /api/v1/patients/{id}/preferences", http.HandlerFunc(deps.Preferences.Put))
	mux.Handle("GET /api/v1/check-ins", middleware.Apply(http.HandlerFunc(deps.CheckIns.List), apiMiddleware...))
	mux.Handle("POST /api/v1/check-ins", middleware.Apply(http.HandlerFunc(deps.CheckIns.Create), apiMiddleware...))
	if deps.Admin != nil {
		adminReadMiddleware := middleware.Chain()
		if deps.AdminAuth != nil {
			adminReadMiddleware = append(adminReadMiddleware, deps.AdminAuth)
		}
		adminWriteMiddleware := middleware.Chain(middleware.RequireTrustedOrigin(cfg))
		if deps.AdminAuth != nil {
			adminWriteMiddleware = append(adminWriteMiddleware, deps.AdminAuth)
		}

		mux.Handle("POST /api/v1/admin/session/login", middleware.Apply(http.HandlerFunc(deps.Admin.Login), middleware.RequireTrustedOrigin(cfg)))
		mux.Handle("GET /api/v1/admin/session", middleware.Apply(http.HandlerFunc(deps.Admin.CurrentSession), adminReadMiddleware...))
		mux.Handle("POST /api/v1/admin/session/logout", middleware.Apply(http.HandlerFunc(deps.Admin.Logout), adminWriteMiddleware...))
		mux.Handle("POST /api/v1/admin/caregivers", middleware.Apply(http.HandlerFunc(deps.Admin.CreateCaregiver), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/caregivers/{id}", middleware.Apply(http.HandlerFunc(deps.Admin.GetCaregiver), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/caregivers/{id}", middleware.Apply(http.HandlerFunc(deps.Admin.UpdateCaregiver), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/patients", middleware.Apply(http.HandlerFunc(deps.Admin.ListPatients), adminReadMiddleware...))
		mux.Handle("POST /api/v1/admin/patients", middleware.Apply(http.HandlerFunc(deps.Admin.CreatePatient), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}", middleware.Apply(http.HandlerFunc(deps.Admin.GetPatient), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/patients/{id}", middleware.Apply(http.HandlerFunc(deps.Admin.UpdatePatient), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/people", middleware.Apply(http.HandlerFunc(deps.Admin.ListPatientPeople), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/patients/{id}/people/{personId}", middleware.Apply(http.HandlerFunc(deps.Admin.UpdatePatientPerson), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/memory-bank", middleware.Apply(http.HandlerFunc(deps.Admin.ListMemoryBankEntries), adminReadMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/reminders", middleware.Apply(http.HandlerFunc(deps.Admin.ListPatientReminders), adminReadMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/screening-schedule", middleware.Apply(http.HandlerFunc(deps.Admin.GetScreeningSchedule), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/patients/{id}/screening-schedule", middleware.Apply(http.HandlerFunc(deps.Admin.PutScreeningSchedule), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/consent", middleware.Apply(http.HandlerFunc(deps.Admin.GetConsentState), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/patients/{id}/consent", middleware.Apply(http.HandlerFunc(deps.Admin.PutConsentState), adminWriteMiddleware...))
		mux.Handle("POST /api/v1/admin/patients/{id}/pause", middleware.Apply(http.HandlerFunc(deps.Admin.PausePatient), adminWriteMiddleware...))
		mux.Handle("DELETE /api/v1/admin/patients/{id}/pause", middleware.Apply(http.HandlerFunc(deps.Admin.UnpausePatient), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/call-templates", middleware.Apply(http.HandlerFunc(deps.Admin.ListCallTemplates), adminReadMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/dashboard", middleware.Apply(http.HandlerFunc(deps.Admin.GetDashboard), adminReadMiddleware...))
		mux.Handle("POST /api/v1/admin/patients/{id}/calls", middleware.Apply(http.HandlerFunc(deps.Admin.CreateCall), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/calls/{id}", middleware.Apply(http.HandlerFunc(deps.Admin.GetCall), adminReadMiddleware...))
		mux.Handle("POST /api/v1/admin/calls/{id}/analyze", middleware.Apply(http.HandlerFunc(deps.Admin.AnalyzeCall), adminWriteMiddleware...))
		mux.Handle("GET /api/v1/admin/calls/{id}/analysis-job", middleware.Apply(http.HandlerFunc(deps.Admin.GetAnalysisJob), adminReadMiddleware...))
		mux.Handle("GET /api/v1/admin/calls/{id}/analysis", middleware.Apply(http.HandlerFunc(deps.Admin.GetCallAnalysis), adminReadMiddleware...))
		mux.Handle("GET /api/v1/admin/patients/{id}/next-call", middleware.Apply(http.HandlerFunc(deps.Admin.GetNextCallPlan), adminReadMiddleware...))
		mux.Handle("PUT /api/v1/admin/patients/{id}/next-call", middleware.Apply(http.HandlerFunc(deps.Admin.PutNextCallPlan), adminWriteMiddleware...))
	}

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
