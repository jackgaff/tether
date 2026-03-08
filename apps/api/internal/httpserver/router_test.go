package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver"
	"nova-echoes/api/internal/modules/checkins"
)

func TestHealthRouteReportsConfigurationState(t *testing.T) {
	t.Parallel()

	handler := httpserver.New(config.Config{
		AppName:        "Nova Echoes",
		AppEnv:         "test",
		FrontendOrigin: "http://localhost:5173",
		DatabaseURL:    "postgres://example",
		AuthMode:       "off",
	}, checkins.NewMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Data struct {
			Status                string `json:"status"`
			DatabaseURLConfigured bool   `json:"databaseURLConfigured"`
		} `json:"data"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}

	if payload.Data.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Data.Status)
	}

	if !payload.Data.DatabaseURLConfigured {
		t.Fatal("expected databaseURLConfigured to be true")
	}
}

func TestOpenAPIRouteIsServed(t *testing.T) {
	t.Parallel()

	handler := httpserver.New(config.Config{
		AppName:        "Nova Echoes",
		AppEnv:         "test",
		FrontendOrigin: "http://localhost:5173",
		AuthMode:       "off",
	}, checkins.NewMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "openapi: 3.1.0") {
		t.Fatalf("expected OpenAPI document, got %q", recorder.Body.String())
	}
}

func TestCORSPreflightAllowsConfiguredFrontend(t *testing.T) {
	t.Parallel()

	handler := httpserver.New(config.Config{
		AppName:        "Nova Echoes",
		AppEnv:         "test",
		FrontendOrigin: "http://localhost:5173",
		AuthMode:       "off",
	}, checkins.NewMemoryStore())

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/check-ins", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin header, got %q", got)
	}
}
