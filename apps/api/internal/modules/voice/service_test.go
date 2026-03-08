package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voicecatalog"
)

func TestCreateSessionUsesPatientPreferenceWhenVoiceNotProvided(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	if _, err := service.preferencesStore.Put(context.Background(), "patient-001", "tiffany"); err != nil {
		t.Fatalf("preferencesStore.Put: %v", err)
	}

	session, err := service.CreateSession(context.Background(), CreateSessionRequest{
		PatientID: "patient-001",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if session.VoiceID != "tiffany" {
		t.Fatalf("expected voice tiffany, got %q", session.VoiceID)
	}
}

func TestCreateSessionExplicitVoiceOverridesPreference(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	if _, err := service.preferencesStore.Put(context.Background(), "patient-001", "tiffany"); err != nil {
		t.Fatalf("preferencesStore.Put: %v", err)
	}

	session, err := service.CreateSession(context.Background(), CreateSessionRequest{
		PatientID: "patient-001",
		VoiceID:   "matthew",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if session.VoiceID != "matthew" {
		t.Fatalf("expected voice matthew, got %q", session.VoiceID)
	}
}

func TestCreateSessionRejectsOversizedSystemPrompt(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	oversizedPrompt := strings.Repeat("a", maxSystemPromptBytes+1)

	_, err := service.CreateSession(context.Background(), CreateSessionRequest{
		PatientID:    "patient-001",
		SystemPrompt: oversizedPrompt,
	})
	if !errors.Is(err, ErrSystemPromptTooLarge) {
		t.Fatalf("expected ErrSystemPromptTooLarge, got %v", err)
	}
}

func TestVoiceHandlerListsVoices(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	handler := NewHandler(service, []string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/voices", nil)
	recorder := httptest.NewRecorder()
	handler.ListVoices(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Data []voicecatalog.Voice `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if len(payload.Data) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(payload.Data))
	}
}

func TestVoiceHandlerListsLabConversations(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	exporter := NewFileArtifactExporter(service.cfg.VoiceLabExportDir)
	endedAt := time.Now().UTC()

	_, err := exporter.Export(context.Background(), SessionArtifact{
		Session: SessionRecord{
			ID:                     "session-history-001",
			PatientID:              "prompt-lab",
			VoiceID:                "matthew",
			SystemPrompt:           "Greet the caller and ask about their morning.",
			ModelID:                "amazon.nova-2-sonic-v1:0",
			InputSampleRateHz:      16000,
			OutputSampleRateHz:     24000,
			EndpointingSensitivity: "LOW",
			CreatedAt:              endedAt.Add(-time.Minute),
		},
		Status:     StatusCompleted,
		StopReason: "END_TURN",
		EndedAt:    endedAt,
		Transcripts: []TranscriptTurn{
			{
				VoiceSessionID: "session-history-001",
				SequenceNo:     1,
				Direction:      "assistant",
				Modality:       "audio",
				TranscriptText: "Good morning. How are you feeling today?",
				OccurredAt:     endedAt.Add(-30 * time.Second),
			},
		},
	})
	if err != nil {
		t.Fatalf("exporter.Export: %v", err)
	}

	handler := NewHandler(service, []string{"http://localhost:5173"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/lab/conversations", nil)
	recorder := httptest.NewRecorder()
	handler.ListLabConversations(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Data []LabConversation `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(payload.Data))
	}

	if payload.Data[0].SystemPrompt != "Greet the caller and ask about their morning." {
		t.Fatalf("unexpected system prompt: %q", payload.Data[0].SystemPrompt)
	}
}

func TestVoiceHandlerCreateSession(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	handler := NewHandler(service, []string{"http://localhost:5173"})
	body := bytes.NewBufferString(`{"patientId":"patient-001"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.CreateSession(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var payload struct {
		Data SessionDescriptor `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if payload.Data.StreamToken == "" {
		t.Fatal("expected stream token")
	}
}

func TestVoiceHandlerCreateSessionRejectsOversizedSystemPrompt(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	handler := NewHandler(service, []string{"http://localhost:5173"})
	body := bytes.NewBufferString(`{"patientId":"patient-001","systemPrompt":"` + strings.Repeat("a", maxSystemPromptBytes+1) + `"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.CreateSession(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestVoiceWebSocketSessionReadyAndClose(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	handler := NewHandler(service, []string{"http://localhost:5173"})

	session, err := service.CreateSession(context.Background(), CreateSessionRequest{PatientID: "patient-001"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/voice/sessions/{id}/stream", http.HandlerFunc(handler.Stream))

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("cannot open local listener in this environment: %v", err)
		}
		t.Fatalf("net.Listen: %v", err)
	}

	server := httptest.NewUnstartedServer(mux)
	server.Listener = listener
	server.Start()
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = session.WebSocketPath
	q := u.Query()
	q.Set("token", session.StreamToken)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{"Origin": []string{"http://localhost:5173"}})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	var ready map[string]any
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("ReadJSON(session_ready): %v", err)
	}

	if ready["type"] != wsMessageSessionReady {
		t.Fatalf("expected session_ready, got %v", ready["type"])
	}

	if err := conn.WriteJSON(map[string]any{"type": wsMessageClientClose}); err != nil {
		t.Fatalf("WriteJSON(client_close): %v", err)
	}

	var ended map[string]any
	if err := conn.ReadJSON(&ended); err != nil {
		t.Fatalf("ReadJSON(session_ended): %v", err)
	}

	if ended["type"] != wsMessageSessionEnded {
		t.Fatalf("expected session_ended, got %v", ended["type"])
	}

	artifacts, ok := ended["artifacts"].(map[string]any)
	if !ok {
		t.Fatalf("expected artifacts payload, got %#v", ended["artifacts"])
	}

	jsonPath, _ := artifacts["jsonPath"].(string)
	markdownPath, _ := artifacts["markdownPath"].(string)
	if jsonPath == "" || markdownPath == "" {
		t.Fatalf("expected artifact paths, got %#v", artifacts)
	}

	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected json artifact to exist: %v", err)
	}

	if _, err := os.Stat(markdownPath); err != nil {
		t.Fatalf("expected markdown artifact to exist: %v", err)
	}
}

func TestRuntimeSessionFinalizeMarksUnexpectedDisconnectAsFailed(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	now := time.Now().UTC()
	repo.sessions["session-001"] = SessionRecord{
		ID:        "session-001",
		PatientID: "patient-001",
		VoiceID:   "matthew",
		CreatedAt: now,
		UpdatedAt: now,
	}

	runtime := &runtimeSession{
		repo:             repo,
		sessionID:        "session-001",
		record:           repo.sessions["session-001"],
		sessionExpiresAt: now.Add(5 * time.Minute),
		now:              func() time.Time { return now },
	}

	if err := runtime.finalize(context.Background(), errClientDisconnected); err == nil {
		t.Fatal("expected finalize to return an error for abrupt disconnect")
	}

	stored := repo.sessions["session-001"]
	if stored.Status != StatusFailed {
		t.Fatalf("expected failed status, got %q", stored.Status)
	}

	if stored.FailureCode != "client_disconnected" {
		t.Fatalf("expected client_disconnected failure code, got %q", stored.FailureCode)
	}
}

func TestRuntimeSessionFinalizeKeepsIntentionalCloseCompleted(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	now := time.Now().UTC()
	repo.sessions["session-002"] = SessionRecord{
		ID:        "session-002",
		PatientID: "patient-001",
		VoiceID:   "matthew",
		CreatedAt: now,
		UpdatedAt: now,
	}

	runtime := &runtimeSession{
		repo:             repo,
		sessionID:        "session-002",
		record:           repo.sessions["session-002"],
		sessionExpiresAt: now.Add(5 * time.Minute),
		now:              func() time.Time { return now },
	}

	if err := runtime.finalize(context.Background(), errClientRequestedClose); err != nil {
		t.Fatalf("expected intentional close to succeed, got %v", err)
	}

	stored := repo.sessions["session-002"]
	if stored.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", stored.Status)
	}
}

func TestRuntimeSessionStartCallOnlyOnce(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	now := time.Now().UTC()
	live := &fakeLiveSession{
		events: make(chan LiveSessionEvent),
	}

	runtime := &runtimeSession{
		repo:      repo,
		live:      live,
		sessionID: "session-004",
		now:       func() time.Time { return now },
	}

	if err := runtime.startCall(context.Background()); err != nil {
		t.Fatalf("startCall: %v", err)
	}

	if err := runtime.startCall(context.Background()); err != nil {
		t.Fatalf("second startCall: %v", err)
	}

	if len(live.sentTexts) != 1 {
		t.Fatalf("expected exactly one start text, got %d", len(live.sentTexts))
	}

	if live.sentTexts[0] != defaultStartCallPrompt {
		t.Fatalf("unexpected start text: %q", live.sentTexts[0])
	}
}

func TestRuntimeSessionTouchSessionThrottlesWrites(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	now := time.Now().UTC()
	repo.sessions["session-003"] = SessionRecord{
		ID:             "session-003",
		PatientID:      "patient-001",
		VoiceID:        "matthew",
		LastActivityAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	runtime := &runtimeSession{
		repo:        repo,
		sessionID:   "session-003",
		record:      repo.sessions["session-003"],
		lastTouchAt: now,
		now:         func() time.Time { return now },
	}

	runtime.touchSession(context.Background(), now.Add(200*time.Millisecond))
	runtime.touchSession(context.Background(), now.Add(500*time.Millisecond))
	runtime.touchSession(context.Background(), now.Add(1200*time.Millisecond))

	if repo.touchCount != 1 {
		t.Fatalf("expected 1 persisted touch, got %d", repo.touchCount)
	}
}

func TestNormalizeTranscriptText(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"  Hello there.  ":           "Hello there.",
		"{ \"interrupted\": true }":  "",
		"{\"interrupted\":false}":    "{\"interrupted\":false}",
		"   ":                        "",
	}

	for input, expected := range cases {
		if actual := normalizeTranscriptText(input); actual != expected {
			t.Fatalf("normalizeTranscriptText(%q): expected %q, got %q", input, expected, actual)
		}
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	cfg := config.Config{
		AppName:                    "Nova Echoes",
		AppEnv:                     "test",
		FrontendOrigin:             "http://localhost:5173",
		AllowedFrontendOrigins:     []string{"http://localhost:5173"},
		VoiceLabExportDir:          t.TempDir(),
		NovaVoiceModelID:           "amazon.nova-2-sonic-v1:0",
		NovaDefaultVoiceID:         "matthew",
		NovaAllowedVoiceIDs:        []string{"matthew", "tiffany"},
		NovaInputSampleRate:        16000,
		NovaOutputSampleRate:       24000,
		NovaEndpointingSensitivity: "LOW",
		AWSRegion:                  "us-east-1",
		BedrockRegion:              "us-east-1",
	}

	catalog, err := voicecatalog.New(cfg.NovaDefaultVoiceID, cfg.NovaAllowedVoiceIDs)
	if err != nil {
		t.Fatalf("voicecatalog.New: %v", err)
	}

	return NewService(
		cfg,
		catalog,
		newMemoryRepository(),
		preferences.NewMemoryStore(),
		newFakeLiveSessionStarter(),
		NewFileArtifactExporter(cfg.VoiceLabExportDir),
		NewSessionManager(),
	)
}

type memoryRepository struct {
	mu          sync.Mutex
	sessions    map[string]SessionRecord
	transcripts []TranscriptTurn
	usageEvents []UsageEvent
	touchCount  int
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		sessions: make(map[string]SessionRecord),
	}
}

func (r *memoryRepository) CreateSession(_ context.Context, session SessionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *memoryRepository) ConsumeAttachToken(_ context.Context, sessionID string, tokenHash []byte, now time.Time) (SessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return SessionRecord{}, ErrSessionNotFound
	}

	if !bytes.Equal(session.StreamTokenHash, tokenHash) {
		return SessionRecord{}, ErrInvalidStreamToken
	}

	if session.StreamTokenExpiresAt.Before(now) {
		return SessionRecord{}, ErrTokenExpired
	}

	if session.StreamTokenConsumedAt != nil {
		return SessionRecord{}, ErrStreamConsumed
	}

	session.StreamTokenConsumedAt = &now
	r.sessions[sessionID] = session
	return session, nil
}

func (r *memoryRepository) MarkSessionStreaming(_ context.Context, sessionID, promptName string, sessionExpiresAt, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := r.sessions[sessionID]
	session.Status = StatusStreaming
	session.PromptName = promptName
	session.SessionExpiresAt = &sessionExpiresAt
	session.LastActivityAt = now
	r.sessions[sessionID] = session
	return nil
}

func (r *memoryRepository) UpdateSessionMetadata(_ context.Context, sessionID, bedrockSessionID, promptName string, sessionExpiresAt *time.Time, now time.Time) error {
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

func (r *memoryRepository) MarkDisconnectGrace(_ context.Context, sessionID string, disconnectedAt, graceExpiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	session.Status = StatusDisconnectGrace
	session.ClientDisconnectedAt = &disconnectedAt
	session.DisconnectGraceExpiresAt = &graceExpiresAt
	r.sessions[sessionID] = session
	return nil
}

func (r *memoryRepository) MarkSessionEnded(_ context.Context, sessionID, status, stopReason, failureCode, failureMessage string, endedAt time.Time) error {
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

func (r *memoryRepository) TouchSession(_ context.Context, sessionID string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	session.LastActivityAt = now
	r.sessions[sessionID] = session
	r.touchCount++
	return nil
}

func (r *memoryRepository) SaveTranscriptTurn(_ context.Context, turn TranscriptTurn) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transcripts = append(r.transcripts, turn)
	return nil
}

func (r *memoryRepository) SaveUsageEvent(_ context.Context, event UsageEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usageEvents = append(r.usageEvents, event)
	return nil
}

func (r *memoryRepository) ListTranscriptTurns(_ context.Context, sessionID string) ([]TranscriptTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	turns := make([]TranscriptTurn, 0, len(r.transcripts))
	for _, turn := range r.transcripts {
		if turn.VoiceSessionID == sessionID {
			turns = append(turns, turn)
		}
	}

	return turns, nil
}

func (r *memoryRepository) ListUsageEvents(_ context.Context, sessionID string) ([]UsageEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	events := make([]UsageEvent, 0, len(r.usageEvents))
	for _, event := range r.usageEvents {
		if event.VoiceSessionID == sessionID {
			events = append(events, event)
		}
	}

	return events, nil
}

type fakeLiveSessionStarter struct{}

func newFakeLiveSessionStarter() fakeLiveSessionStarter {
	return fakeLiveSessionStarter{}
}

func (fakeLiveSessionStarter) StartSession(_ context.Context, _ StartLiveSessionInput) (LiveSession, error) {
	return &fakeLiveSession{
		events: make(chan LiveSessionEvent),
	}, nil
}

type fakeLiveSession struct {
	mu        sync.Mutex
	events    chan LiveSessionEvent
	sentTexts []string
	once      sync.Once
}

func (s *fakeLiveSession) SendAudio(_ context.Context, _ []byte) error {
	return nil
}

func (s *fakeLiveSession) SendText(_ context.Context, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentTexts = append(s.sentTexts, text)
	return nil
}

func (s *fakeLiveSession) EndConversation(_ context.Context) error {
	s.once.Do(func() { close(s.events) })
	return nil
}

func (s *fakeLiveSession) Events() <-chan LiveSessionEvent {
	return s.events
}

func (s *fakeLiveSession) Close() error {
	s.once.Do(func() { close(s.events) })
	return nil
}
