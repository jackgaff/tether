package testsupport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"tether/api/internal/config"
	"tether/api/internal/httpserver"
	"tether/api/internal/modules/checkins"
	"tether/api/internal/modules/patients/preferences"
	"tether/api/internal/modules/voice"
	"tether/api/internal/modules/voicecatalog"
)

func NewHandler(cfg config.Config) http.Handler {
	catalog, err := voicecatalog.New(defaultVoiceID(cfg), allowedVoiceIDs(cfg))
	if err != nil {
		panic(err)
	}

	preferenceStore := preferences.NewMemoryStore()
	allowedOrigins := cfg.AllowedFrontendOrigins
	if len(allowedOrigins) == 0 && cfg.FrontendOrigin != "" {
		allowedOrigins = []string{cfg.FrontendOrigin}
	}

	voiceService := voice.NewService(
		cfg,
		catalog,
		newVoiceRepo(),
		preferenceStore,
		noopLiveSessionStarter{},
		nil,
		voice.NewSessionManager(),
	)

	return httpserver.New(cfg, httpserver.Dependencies{
		CheckIns:    checkins.NewHandler(checkins.NewMemoryStore()),
		Preferences: preferences.NewHandler(preferenceStore, catalog),
		Voice:       voice.NewHandler(voiceService, allowedOrigins),
	})
}

func defaultVoiceID(cfg config.Config) string {
	if cfg.NovaDefaultVoiceID != "" {
		return cfg.NovaDefaultVoiceID
	}

	return "matthew"
}

func allowedVoiceIDs(cfg config.Config) []string {
	if len(cfg.NovaAllowedVoiceIDs) > 0 {
		return cfg.NovaAllowedVoiceIDs
	}

	return []string{"matthew", "tiffany"}
}

type voiceRepo struct {
	mu       sync.Mutex
	sessions map[string]voice.SessionRecord
}

func newVoiceRepo() *voiceRepo {
	return &voiceRepo{
		sessions: make(map[string]voice.SessionRecord),
	}
}

func (r *voiceRepo) CreateSession(_ context.Context, session voice.SessionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *voiceRepo) LinkCallRun(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

func (r *voiceRepo) ConsumeAttachToken(_ context.Context, sessionID string, tokenHash []byte, now time.Time) (voice.SessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return voice.SessionRecord{}, voice.ErrSessionNotFound
	}

	if session.StreamTokenExpiresAt.Before(now) {
		return voice.SessionRecord{}, voice.ErrTokenExpired
	}

	if session.StreamTokenConsumedAt != nil {
		return voice.SessionRecord{}, voice.ErrStreamConsumed
	}

	session.StreamTokenConsumedAt = &now
	r.sessions[sessionID] = session
	return session, nil
}

func (r *voiceRepo) MarkSessionStreaming(_ context.Context, sessionID, promptName string, sessionExpiresAt, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	session.PromptName = promptName
	session.SessionExpiresAt = &sessionExpiresAt
	session.Status = voice.StatusStreaming
	session.LastActivityAt = now
	r.sessions[sessionID] = session
	return nil
}

func (r *voiceRepo) MarkCallRunInProgress(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (r *voiceRepo) UpdateSessionMetadata(_ context.Context, sessionID, bedrockSessionID, promptName string, sessionExpiresAt *time.Time, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	if bedrockSessionID != "" {
		session.BedrockSessionID = bedrockSessionID
	}
	if promptName != "" {
		session.PromptName = promptName
	}
	if sessionExpiresAt != nil {
		session.SessionExpiresAt = sessionExpiresAt
	}
	session.LastActivityAt = now
	r.sessions[sessionID] = session
	return nil
}

func (r *voiceRepo) MarkDisconnectGrace(_ context.Context, sessionID string, disconnectedAt, graceExpiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	session.Status = voice.StatusDisconnectGrace
	session.ClientDisconnectedAt = &disconnectedAt
	session.DisconnectGraceExpiresAt = &graceExpiresAt
	r.sessions[sessionID] = session
	return nil
}

func (r *voiceRepo) MarkSessionEnded(_ context.Context, sessionID, status, stopReason, failureCode, failureMessage string, endedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	session.Status = status
	session.StopReason = stopReason
	session.FailureCode = failureCode
	session.FailureMessage = failureMessage
	session.EndedAt = &endedAt
	r.sessions[sessionID] = session
	return nil
}

func (r *voiceRepo) MarkCallRunEnded(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

func (r *voiceRepo) TouchSession(_ context.Context, sessionID string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	session.LastActivityAt = now
	r.sessions[sessionID] = session
	return nil
}

func (r *voiceRepo) SaveTranscriptTurn(_ context.Context, _ voice.TranscriptTurn) error {
	return nil
}

func (r *voiceRepo) SaveUsageEvent(_ context.Context, _ voice.UsageEvent) error {
	return nil
}

func (r *voiceRepo) ListTranscriptTurns(_ context.Context, _ string) ([]voice.TranscriptTurn, error) {
	return nil, nil
}

func (r *voiceRepo) ListUsageEvents(_ context.Context, _ string) ([]voice.UsageEvent, error) {
	return nil, nil
}

type noopLiveSessionStarter struct{}

func (noopLiveSessionStarter) StartSession(_ context.Context, _ voice.StartLiveSessionInput) (voice.LiveSession, error) {
	return &noopLiveSession{
		events: make(chan voice.LiveSessionEvent),
	}, nil
}

type noopLiveSession struct {
	events chan voice.LiveSessionEvent
	closed bool
	mu     sync.Mutex
}

func (s *noopLiveSession) SendAudio(_ context.Context, _ []byte) error {
	return nil
}

func (s *noopLiveSession) SendText(_ context.Context, _ string) error {
	return nil
}

func (s *noopLiveSession) EndConversation(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	close(s.events)
	s.closed = true
	return nil
}

func (s *noopLiveSession) Events() <-chan voice.LiveSessionEvent {
	return s.events
}

func (s *noopLiveSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	close(s.events)
	s.closed = true
	return nil
}

func DecodeResponseBody[T any](body []byte, target *T) error {
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}

	if target == nil {
		return errors.New("target is required")
	}

	*target = envelope.Data
	return nil
}
