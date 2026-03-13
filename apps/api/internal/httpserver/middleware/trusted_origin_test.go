package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nova-echoes/api/internal/config"
	"nova-echoes/api/internal/httpserver/middleware"
)

func TestRequireTrustedOriginAllowsSafeMethods(t *testing.T) {
	t.Parallel()

	handler := middleware.RequireTrustedOrigin(config.Config{
		AllowedFrontendOrigins: []string{"http://localhost:5173"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/patients", nil)
	req.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected safe GET request to pass, got %d", recorder.Code)
	}
}

func TestRequireTrustedOriginAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	handler := middleware.RequireTrustedOrigin(config.Config{
		AllowedFrontendOrigins: []string{"http://localhost:5173"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/patients", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected allowed POST request to pass, got %d", recorder.Code)
	}
}

func TestRequireTrustedOriginAllowsOriginlessClients(t *testing.T) {
	t.Parallel()

	handler := middleware.RequireTrustedOrigin(config.Config{
		AllowedFrontendOrigins: []string{"http://localhost:5173"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/patients", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected originless POST request to pass, got %d", recorder.Code)
	}
}

func TestRequireTrustedOriginRejectsUnexpectedOrigin(t *testing.T) {
	t.Parallel()

	handler := middleware.RequireTrustedOrigin(config.Config{
		AllowedFrontendOrigins: []string{"http://localhost:5173"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/patients/demo/next-call", nil)
	req.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden request, got %d", recorder.Code)
	}
}

func TestRequireTrustedOriginFallsBackToReferer(t *testing.T) {
	t.Parallel()

	handler := middleware.RequireTrustedOrigin(config.Config{
		AllowedFrontendOrigins: []string{"http://localhost:5173"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/patients/demo/consent", nil)
	req.Header.Set("Referer", "http://localhost:5173/dashboard")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected referer-derived origin to pass, got %d", recorder.Code)
	}
}
