package checkins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nova-echoes/api/internal/httpserver/respond"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) Handler {
	return Handler{store: store}
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.URL.Query().Get("patientId"))

	items, err := h.store.List(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load check-ins.")
		return
	}

	respond.JSON(w, http.StatusOK, items, map[string]any{
		"count": len(items),
	})
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateCheckInRequest

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

	if err := validateCreateCheckInRequest(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	item, err := h.store.Create(r.Context(), input)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not save check-in.")
		return
	}

	respond.JSON(w, http.StatusCreated, item, nil)
}

func validateCreateCheckInRequest(input CreateCheckInRequest) error {
	if strings.TrimSpace(input.PatientID) == "" {
		return fmt.Errorf("patientId is required")
	}

	if strings.TrimSpace(input.Summary) == "" {
		return fmt.Errorf("summary is required")
	}

	if strings.TrimSpace(input.Agent) == "" {
		return fmt.Errorf("agent is required")
	}

	switch input.Status {
	case StatusScheduled, StatusCompleted, StatusNeedsFollowUp:
	default:
		return fmt.Errorf("status must be one of scheduled, completed, or needs_follow_up")
	}

	return nil
}
