package preferences

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"nova-echoes/api/internal/httpserver/respond"
	"nova-echoes/api/internal/modules/voicecatalog"
)

type Handler struct {
	store        Store
	voiceCatalog voicecatalog.Catalog
}

func NewHandler(store Store, voiceCatalog voicecatalog.Catalog) Handler {
	return Handler{store: store, voiceCatalog: voiceCatalog}
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if patientID == "" {
		respond.Error(w, http.StatusBadRequest, "validation_error", "patient id is required")
		return
	}

	preference, ok, err := h.store.Get(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load patient preferences.")
		return
	}

	if !ok {
		respond.JSON(w, http.StatusOK, Preference{
			PatientID:      patientID,
			DefaultVoiceID: h.voiceCatalog.DefaultVoiceID(),
			IsConfigured:   false,
		}, nil)
		return
	}

	respond.JSON(w, http.StatusOK, preference, nil)
}

func (h Handler) Put(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if patientID == "" {
		respond.Error(w, http.StatusBadRequest, "validation_error", "patient id is required")
		return
	}

	var input UpdatePreferenceRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		respond.Error(w, http.StatusBadRequest, "invalid_json", "Request body must contain a single JSON object.")
		return
	}

	input.DefaultVoiceID = strings.TrimSpace(input.DefaultVoiceID)
	if input.DefaultVoiceID == "" {
		respond.Error(w, http.StatusBadRequest, "validation_error", "defaultVoiceId is required")
		return
	}

	if !h.voiceCatalog.IsAllowed(input.DefaultVoiceID) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "defaultVoiceId must be one of the allowed voices")
		return
	}

	preference, err := h.store.Put(r.Context(), patientID, input.DefaultVoiceID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not save patient preferences.")
		return
	}

	respond.JSON(w, http.StatusOK, preference, nil)
}
