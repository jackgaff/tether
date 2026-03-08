package preferences_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voicecatalog"
)

func TestGetPreferencesReturnsDefaultWhenMissing(t *testing.T) {
	t.Parallel()

	catalog, err := voicecatalog.New("matthew", []string{"matthew", "tiffany"})
	if err != nil {
		t.Fatalf("voicecatalog.New: %v", err)
	}

	handler := preferences.NewHandler(preferences.NewMemoryStore(), catalog)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patients/patient-001/preferences", nil)
	req.SetPathValue("id", "patient-001")
	recorder := httptest.NewRecorder()

	handler.Get(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Data preferences.Preference `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if payload.Data.DefaultVoiceID != "matthew" {
		t.Fatalf("expected default voice matthew, got %q", payload.Data.DefaultVoiceID)
	}

	if payload.Data.IsConfigured {
		t.Fatal("expected missing preference to be unconfigured")
	}
}

func TestPutPreferencesValidatesVoiceChoice(t *testing.T) {
	t.Parallel()

	catalog, err := voicecatalog.New("matthew", []string{"matthew", "tiffany"})
	if err != nil {
		t.Fatalf("voicecatalog.New: %v", err)
	}

	handler := preferences.NewHandler(preferences.NewMemoryStore(), catalog)
	body, err := json.Marshal(map[string]any{"defaultVoiceId": "unknown"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/patients/patient-001/preferences", bytes.NewReader(body))
	req.SetPathValue("id", "patient-001")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.Put(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestPutPreferencesPersistsChoice(t *testing.T) {
	t.Parallel()

	catalog, err := voicecatalog.New("matthew", []string{"matthew", "tiffany"})
	if err != nil {
		t.Fatalf("voicecatalog.New: %v", err)
	}

	store := preferences.NewMemoryStore()
	handler := preferences.NewHandler(store, catalog)
	body, err := json.Marshal(map[string]any{"defaultVoiceId": "tiffany"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/patients/patient-001/preferences", bytes.NewReader(body))
	req.SetPathValue("id", "patient-001")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.Put(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	preference, ok, err := store.Get(req.Context(), "patient-001")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}

	if !ok {
		t.Fatal("expected preference to be stored")
	}

	if preference.DefaultVoiceID != "tiffany" {
		t.Fatalf("expected stored voice tiffany, got %q", preference.DefaultVoiceID)
	}
}
