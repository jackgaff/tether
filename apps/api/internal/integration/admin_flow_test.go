package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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

func TestAdminCaregiverFlowWithVoiceLifecycleAndAnalysis(t *testing.T) {
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := db.Open(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	if err := db.ResetForTest(ctx, database); err != nil {
		t.Fatalf("db.ResetForTest: %v", err)
	}

	cfg := config.Config{
		AppName:                    "Nova Echoes",
		AppEnv:                     "test",
		FrontendOrigin:             "http://localhost:5173",
		AllowedFrontendOrigins:     []string{"http://localhost:5173"},
		DatabaseURL:                testDatabaseURL,
		VoiceLabExportDir:          t.TempDir(),
		AuthMode:                   "off",
		AWSRegion:                  "us-east-1",
		BedrockRegion:              "us-east-1",
		NovaVoiceModelID:           "amazon.nova-2-sonic-v1:0",
		NovaAnalysisModelID:        "amazon.nova-2-lite-v1:0",
		NovaDefaultVoiceID:         "matthew",
		NovaAllowedVoiceIDs:        []string{"matthew", "tiffany"},
		NovaInputSampleRate:        16000,
		NovaOutputSampleRate:       24000,
		NovaEndpointingSensitivity: "LOW",
		AdminUsername:              "demo-admin",
		AdminPassword:              "demo-admin-password",
		AdminSessionSecret:         "demo-admin-session-secret-change-me",
	}

	handler := newAdminIntegrationHandler(t, database, cfg)
	server := httptest.NewServer(handler)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{Jar: jar}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/call-templates", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do unauthorized request: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized templates request, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	loginPayload := map[string]any{
		"username": cfg.AdminUsername,
		"password": cfg.AdminPassword,
	}
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/session/login", loginPayload, http.StatusOK, nil)

	var caregiver admin.Caregiver
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/caregivers", map[string]any{
		"displayName": "Ava Carter",
		"email":       "ava@example.com",
		"phoneE164":   "+15551234567",
		"timezone":    "America/Detroit",
	}, http.StatusCreated, &caregiver)

	var patient admin.Patient
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients", map[string]any{
		"primaryCaregiverId": caregiver.ID,
		"displayName":        "Eleanor Carter",
		"preferredName":      "Ellie",
		"phoneE164":          "+15557654321",
		"timezone":           "America/Detroit",
		"routineAnchors":     []string{"breakfast", "medication card"},
		"favoriteTopics":     []string{"gardening"},
		"calmingCues":        []string{"slow breathing"},
		"topicsToAvoid":      []string{"driving"},
	}, http.StatusCreated, &patient)

	assertStatus(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callType": "orientation",
		"channel":  "browser",
	}, http.StatusBadRequest)

	doJSON(t, client, http.MethodPut, server.URL+"/api/v1/admin/patients/"+patient.ID+"/consent", map[string]any{
		"outboundCallStatus":      "granted",
		"transcriptStorageStatus": "granted",
		"notes":                   "Caregiver approved AI check-ins and transcript storage.",
	}, http.StatusOK, nil)

	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/pause", map[string]any{
		"reason": "Quiet during lunch",
	}, http.StatusOK, nil)

	assertStatus(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callType": "orientation",
		"channel":  "browser",
	}, http.StatusBadRequest)

	doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/patients/"+patient.ID+"/pause", nil, http.StatusOK, nil)

	assertStatus(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callTemplateId": "tmpl-orientation-v1",
		"callType":       "orientation",
		"channel":        "browser",
	}, http.StatusBadRequest)

	var created admin.CreateCallResponse
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callType": "orientation",
		"channel":  "browser",
	}, http.StatusCreated, &created)

	voiceSessionBytes, err := json.Marshal(created.VoiceSession)
	if err != nil {
		t.Fatalf("json.Marshal(voiceSession): %v", err)
	}
	var descriptor voice.SessionDescriptor
	if err := json.Unmarshal(voiceSessionBytes, &descriptor); err != nil {
		t.Fatalf("json.Unmarshal(voiceSession): %v", err)
	}

	assertStatus(t, client, http.MethodPost, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID+"/analyze", nil, http.StatusBadRequest)

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	wsURL.Scheme = "ws"
	wsURL.Path = descriptor.WebSocketPath
	query := wsURL.Query()
	query.Set("token", descriptor.StreamToken)
	wsURL.RawQuery = query.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), http.Header{"Origin": []string{"http://localhost:5173"}})
	if err != nil {
		t.Fatalf("websocket.Dial: %v", err)
	}
	defer conn.Close()

	var ready map[string]any
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("ReadJSON(session_ready): %v", err)
	}
	if ready["type"] != "session_ready" {
		t.Fatalf("expected session_ready, got %#v", ready)
	}

	if err := conn.WriteJSON(map[string]any{"type": "client_close"}); err != nil {
		t.Fatalf("WriteJSON(client_close): %v", err)
	}

	var ended map[string]any
	if err := conn.ReadJSON(&ended); err != nil {
		t.Fatalf("ReadJSON(session_ended): %v", err)
	}
	if ended["type"] != "session_ended" {
		t.Fatalf("expected session_ended, got %#v", ended)
	}

	var callDetail admin.CallRunDetail
	doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID, nil, http.StatusOK, &callDetail)
	if callDetail.CallRun.Status != admin.CallRunStatusCompleted {
		t.Fatalf("expected completed call run, got %q", callDetail.CallRun.Status)
	}
	if callDetail.CallRun.SourceVoiceSessionID == "" {
		t.Fatal("expected linked sourceVoiceSessionId")
	}
	if callDetail.CallRun.StartedAt == nil || callDetail.CallRun.EndedAt == nil {
		t.Fatalf("expected startedAt and endedAt to be populated, got %+v", callDetail.CallRun)
	}

	var analysisRecord admin.AnalysisRecord
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID+"/analyze", nil, http.StatusOK, &analysisRecord)
	firstAnalysisID := analysisRecord.ID
	if firstAnalysisID == "" {
		t.Fatal("expected analysis record id")
	}
	if len(analysisRecord.RiskFlags) == 0 {
		t.Fatal("expected derived risk flags")
	}

	var repeated admin.AnalysisRecord
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID+"/analyze", nil, http.StatusOK, &repeated)
	if repeated.ID != firstAnalysisID {
		t.Fatalf("expected repeated analyze call to return existing analysis id %q, got %q", firstAnalysisID, repeated.ID)
	}

	var nextCallPlan admin.NextCallPlan
	doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/patients/"+patient.ID+"/next-call", nil, http.StatusOK, &nextCallPlan)
	if nextCallPlan.ApprovalStatus != admin.NextCallStatusPendingApproval {
		t.Fatalf("expected pending_approval plan, got %q", nextCallPlan.ApprovalStatus)
	}

	assertStatus(t, client, http.MethodPut, server.URL+"/api/v1/admin/patients/"+patient.ID+"/next-call", map[string]any{
		"action":     "edit",
		"plannedFor": "not-a-timestamp",
	}, http.StatusBadRequest)

	doJSON(t, client, http.MethodPut, server.URL+"/api/v1/admin/patients/"+patient.ID+"/next-call", map[string]any{
		"action": "approve",
	}, http.StatusOK, &nextCallPlan)
	if nextCallPlan.ApprovalStatus != admin.NextCallStatusApproved {
		t.Fatalf("expected approved plan, got %q", nextCallPlan.ApprovalStatus)
	}

	var dashboard admin.DashboardSnapshot
	doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/patients/"+patient.ID+"/dashboard", nil, http.StatusOK, &dashboard)
	if dashboard.LatestAnalysis == nil {
		t.Fatal("expected latest analysis in dashboard")
	}
	if dashboard.ActiveNextCallPlan == nil || dashboard.ActiveNextCallPlan.ApprovalStatus != admin.NextCallStatusApproved {
		t.Fatalf("expected approved active next-call plan, got %+v", dashboard.ActiveNextCallPlan)
	}
}

func newAdminIntegrationHandler(t *testing.T, database *sql.DB, cfg config.Config) http.Handler {
	t.Helper()

	catalog, err := voicecatalog.New(cfg.NovaDefaultVoiceID, cfg.NovaAllowedVoiceIDs)
	if err != nil {
		t.Fatalf("voicecatalog.New: %v", err)
	}

	preferenceStore := preferences.NewPostgresStore(database)
	voiceService := voice.NewService(
		cfg,
		catalog,
		voice.NewPostgresRepository(database),
		preferenceStore,
		testLiveSessionStarter{},
		voice.NewNoopArtifactExporter(),
		voice.NewSessionManager(),
	)
	adminStore := admin.NewPostgresStore(database)
	adminService := admin.NewService(adminStore, voiceService, staticAnalysisRunner{}, cfg.NovaAnalysisModelID)
	adminSessions := adminsession.New(cfg)

	return httpserver.New(cfg, httpserver.Dependencies{
		CheckIns:    checkins.NewHandler(checkins.NewPostgresStore(database)),
		Preferences: preferences.NewHandler(preferenceStore, catalog),
		Voice:       voice.NewHandler(voiceService, cfg.AllowedFrontendOrigins),
		Admin:       admin.NewHandler(adminStore, adminService, adminSessions),
		AdminAuth:   adminSessions.Middleware(),
	})
}

type staticAnalysisRunner struct{}

func (staticAnalysisRunner) Analyze(_ context.Context, promptContext admin.AnalysisPromptContext) (admin.AnalysisPayload, error) {
	return admin.AnalysisPayload{
		CallTypeCompleted: promptContext.CallTemplate.CallType,
		PatientState: admin.AnalysisPatientState{
			Orientation: admin.AnalysisOrientationMixed,
			Mood:        admin.AnalysisMoodNeutral,
			Engagement:  admin.AnalysisEngagementMedium,
			Confidence:  0.91,
		},
		Signals: admin.AnalysisSignals{
			Repetition:           1,
			SleepConcern:         true,
			SocialConnectionNeed: true,
		},
		Evidence: []admin.AnalysisEvidence{
			{
				Quote:        "The caller paused several times and repeated the morning question.",
				WhyItMatters: "Suggests a gentle follow-up and another brief orientation call.",
			},
		},
		DashboardSummary: "The call completed successfully and suggested a gentle follow-up.",
		CaregiverSummary: "Ellie completed the call. Repetition was mild, and a short reminder-style follow-up is recommended.",
		RecommendedNextCall: admin.RecommendedNextCall{
			Type:            admin.CallTypeReminder,
			Timing:          "Later this evening",
			DurationMinutes: 4,
			Goal:            "Repeat the main routine cue one more time.",
		},
		EscalationLevel: admin.EscalationCaregiverSoon,
		Uncertainties:   []string{"Transcript evidence is limited because the test call ended early."},
	}, nil
}

type testLiveSessionStarter struct{}

func (testLiveSessionStarter) StartSession(_ context.Context, _ voice.StartLiveSessionInput) (voice.LiveSession, error) {
	return &testLiveSession{events: make(chan voice.LiveSessionEvent)}, nil
}

type testLiveSession struct {
	events chan voice.LiveSessionEvent
	closed bool
}

func (s *testLiveSession) SendAudio(_ context.Context, _ []byte) error {
	return nil
}

func (s *testLiveSession) SendText(_ context.Context, _ string) error {
	return nil
}

func (s *testLiveSession) EndConversation(_ context.Context) error {
	if !s.closed {
		close(s.events)
		s.closed = true
	}
	return nil
}

func (s *testLiveSession) Events() <-chan voice.LiveSessionEvent {
	return s.events
}

func (s *testLiveSession) Close() error {
	if !s.closed {
		close(s.events)
		s.closed = true
	}
	return nil
}

type apiEnvelope[T any] struct {
	Data  T `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func doJSON(t *testing.T, client *http.Client, method string, targetURL string, body any, expectedStatus int, out any) {
	t.Helper()

	res := doRequest(t, client, method, targetURL, body)
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		rawBody := new(strings.Builder)
		_, _ = io.Copy(rawBody, res.Body)
		t.Fatalf("expected %d from %s %s, got %d: %s", expectedStatus, method, targetURL, res.StatusCode, rawBody.String())
	}

	if out == nil {
		return
	}

	envelope := struct {
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response envelope: %v", err)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		t.Fatalf("decode response data: %v", err)
	}
}

func assertStatus(t *testing.T, client *http.Client, method string, targetURL string, body any, expectedStatus int) {
	t.Helper()
	res := doRequest(t, client, method, targetURL, body)
	defer res.Body.Close()
	if res.StatusCode != expectedStatus {
		t.Fatalf("expected %d from %s %s, got %d", expectedStatus, method, targetURL, res.StatusCode)
	}
}

func doRequest(t *testing.T, client *http.Client, method string, targetURL string, body any) *http.Response {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal request: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, targetURL, requestBody)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	return res
}
