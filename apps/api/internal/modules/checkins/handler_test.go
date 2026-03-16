package checkins_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tether/api/internal/config"
	"tether/api/internal/testsupport"
)

func TestCreateCheckInSuccess(t *testing.T) {
	t.Parallel()

	handler := testsupport.NewHandler(config.Config{
		AppName:        "Tether",
		AppEnv:         "test",
		Port:           "8080",
		FrontendOrigin: "http://localhost:5173",
		AuthMode:       "off",
	})

	body := map[string]any{
		"patientId": "patient-001",
		"summary":   "Caller completed a mood and memory check-in.",
		"status":    "completed",
		"agent":     "call-agent",
		"reminder":  "Bring water to the living room before lunch.",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/check-ins", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestCreateCheckInValidationError(t *testing.T) {
	t.Parallel()

	handler := testsupport.NewHandler(config.Config{
		AppName:        "Tether",
		AppEnv:         "test",
		Port:           "8080",
		FrontendOrigin: "http://localhost:5173",
		AuthMode:       "off",
	})

	body := map[string]any{
		"patientId": "",
		"summary":   "",
		"status":    "unknown",
		"agent":     "",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/check-ins", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestCreateCheckInRequiresAPIKeyWhenEnabled(t *testing.T) {
	t.Parallel()

	handler := testsupport.NewHandler(config.Config{
		AppName:        "Tether",
		AppEnv:         "test",
		Port:           "8080",
		FrontendOrigin: "http://localhost:5173",
		AuthMode:       "api-key",
		InternalAPIKey: "secret-key",
	})

	body := map[string]any{
		"patientId": "patient-001",
		"summary":   "Caller completed a mood and memory check-in.",
		"status":    "completed",
		"agent":     "call-agent",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/check-ins", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
