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
	"time"

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
	assertStatusWithOrigin(t, client, http.MethodPost, server.URL+"/api/v1/admin/session/login", loginPayload, "https://evil.example", http.StatusForbidden)
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/session/login", loginPayload, http.StatusOK, nil)

	assertStatusWithOrigin(t, client, http.MethodPost, server.URL+"/api/v1/admin/caregivers", map[string]any{
		"displayName": "Blocked Caregiver",
		"email":       "blocked@example.com",
		"timezone":    "America/Detroit",
	}, "https://evil.example", http.StatusForbidden)

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
		"callType": "check_in",
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
		"callType": "check_in",
		"channel":  "browser",
	}, http.StatusBadRequest)

	doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/patients/"+patient.ID+"/pause", nil, http.StatusOK, nil)

	assertStatus(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callTemplateId": "tmpl-check-in-v1",
		"callType":       "check_in",
		"channel":        "browser",
	}, http.StatusBadRequest)

	var created admin.CreateCallResponse
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/patients/"+patient.ID+"/calls", map[string]any{
		"callType": "check_in",
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

	var analysisJob admin.AnalysisJob
	doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID+"/analyze", nil, http.StatusOK, &analysisJob)
	if analysisJob.ID == "" {
		t.Fatal("expected analysis job id")
	}

	var analysisRecord admin.AnalysisRecord
	waitForAnalysis(t, client, server.URL+"/api/v1/admin/calls/"+created.CallRun.ID+"/analysis", &analysisRecord)
	if len(analysisRecord.RiskFlags) == 0 {
		t.Fatal("expected persisted risk flags")
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
	adminService := admin.NewService(adminStore, voiceService, cfg.NovaAnalysisModelID)
	adminSessions := adminsession.New(cfg)

	go admin.NewAnalysisWorker(adminStore, staticAnalysisRunner{}).Run(context.Background(), 10*time.Millisecond)

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
		Summary: "The call completed successfully and suggested a gentle follow-up.",
		SalientEvidence: []admin.SalientEvidence{
			{
				Quote:  "The caller asked if Echo could check in again in a few days.",
				Reason: "This directly supports a caregiver review for another short check-in.",
			},
		},
		RiskFlags: []admin.AnalysisRiskFlag{
			{
				FlagType:   "follow_up_requested",
				Severity:   admin.RiskSeverityInfo,
				Evidence:   "The caller asked for another check-in in a few days.",
				Reason:     "A caregiver should review and approve the follow-up.",
				Confidence: 0.91,
			},
		},
		EscalationLevel:       admin.EscalationCaregiverSoon,
		CaregiverReviewReason: "The patient asked for another check-in soon.",
		FollowUpIntent: admin.FollowUpIntent{
			RequestedByPatient: true,
			TimeframeBucket:    admin.TimeframeFewDays,
			Evidence:           "Could you check in with me again in a few days?",
			Confidence:         0.91,
		},
		NextCallRecommendation: &admin.NextCallRecommendation{
			CallType:     admin.CallTypeCheckIn,
			WindowBucket: admin.TimeframeFewDays,
			Goal:         "Follow up on the patient's day and comfort.",
		},
		CheckIn: &admin.CheckInAnalysis{
			ReportedDayOverview:     "The patient shared a brief update about the day.",
			FoodAndHydration:        "Breakfast was mentioned.",
			MedicationMentions:      []string{},
			MoodSignals:             []string{"calm"},
			RoutineAdherence:        "No major adherence concern observed.",
			SocialContactMentions:   []string{"Echo"},
			FollowUpRequestDetected: true,
		},
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

func assertStatusWithOrigin(t *testing.T, client *http.Client, method string, targetURL string, body any, origin string, expectedStatus int) {
	t.Helper()
	res := doRequestWithOrigin(t, client, method, targetURL, body, origin)
	defer res.Body.Close()
	if res.StatusCode != expectedStatus {
		rawBody := new(strings.Builder)
		_, _ = io.Copy(rawBody, res.Body)
		t.Fatalf("expected %d from %s %s with origin %s, got %d: %s", expectedStatus, method, targetURL, origin, res.StatusCode, rawBody.String())
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

func doRequestWithOrigin(t *testing.T, client *http.Client, method string, targetURL string, body any, origin string) *http.Response {
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
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	return res
}

func waitForAnalysis(t *testing.T, client *http.Client, targetURL string, out any) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		res := doRequest(t, client, http.MethodGet, targetURL, nil)
		if res.StatusCode == http.StatusOK {
			defer res.Body.Close()
			envelope := struct {
				Data json.RawMessage `json:"data"`
			}{}
			if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
				t.Fatalf("decode analysis envelope: %v", err)
			}
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				t.Fatalf("decode analysis payload: %v", err)
			}
			return
		}
		_ = res.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatal("timed out waiting for analysis result")
}
