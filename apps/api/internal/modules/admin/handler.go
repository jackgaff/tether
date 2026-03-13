package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nova-echoes/api/internal/adminsession"
	"nova-echoes/api/internal/httpserver/respond"
)

type Handler struct {
	store    Store
	service  *Service
	sessions *adminsession.Manager
}

func NewHandler(store Store, service *Service, sessions *adminsession.Manager) *Handler {
	return &Handler{
		store:    store,
		service:  service,
		sessions: sessions,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if err := h.sessions.ValidateCredentials(input.Username, input.Password); err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized", "A valid admin username and password are required.")
		return
	}

	h.sessions.SetSession(w, strings.TrimSpace(input.Username))
	claims, err := h.sessions.SessionClaims(withCookieRequest(r, w))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "session_error", "Could not establish the admin session.")
		return
	}

	respond.JSON(w, http.StatusOK, SessionResponse{
		Username:  claims.Username,
		ExpiresAt: claims.ExpiresAt,
	}, nil)
}

func (h *Handler) CurrentSession(w http.ResponseWriter, r *http.Request) {
	claims, err := h.sessions.SessionClaims(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized", "A valid admin session is required.")
		return
	}

	respond.JSON(w, http.StatusOK, SessionResponse{
		Username:  claims.Username,
		ExpiresAt: claims.ExpiresAt,
	}, nil)
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	h.sessions.ClearSession(w)
	respond.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"}, nil)
}

func (h *Handler) CreateCaregiver(w http.ResponseWriter, r *http.Request) {
	var input CreateCaregiverRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateCaregiverInput(input.DisplayName, input.Email, input.Timezone); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	caregiver, err := h.store.CreateCaregiver(r.Context(), input)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create caregiver.")
		return
	}

	respond.JSON(w, http.StatusCreated, caregiver, nil)
}

func (h *Handler) GetCaregiver(w http.ResponseWriter, r *http.Request) {
	caregiverID := strings.TrimSpace(r.PathValue("id"))
	caregiver, ok, err := h.store.GetCaregiver(r.Context(), caregiverID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load caregiver.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Caregiver not found.")
		return
	}

	respond.JSON(w, http.StatusOK, caregiver, nil)
}

func (h *Handler) UpdateCaregiver(w http.ResponseWriter, r *http.Request) {
	caregiverID := strings.TrimSpace(r.PathValue("id"))
	var input UpdateCaregiverRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateCaregiverInput(input.DisplayName, input.Email, input.Timezone); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	caregiver, err := h.store.UpdateCaregiver(r.Context(), caregiverID, input)
	if err != nil {
		if errors.Is(err, ErrCaregiverNotFound) {
			respond.Error(w, http.StatusNotFound, "not_found", "Caregiver not found.")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update caregiver.")
		return
	}

	respond.JSON(w, http.StatusOK, caregiver, nil)
}

func (h *Handler) CreatePatient(w http.ResponseWriter, r *http.Request) {
	var input CreatePatientRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validatePatientInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	patient, err := h.store.CreatePatient(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrCaregiverNotFound):
			respond.Error(w, http.StatusBadRequest, "validation_error", "primaryCaregiverId must reference an existing caregiver.")
		case errors.Is(err, ErrPatientAlreadyAssigned):
			respond.Error(w, http.StatusConflict, "conflict", "This caregiver already has a patient assigned in the MVP.")
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create patient.")
		}
		return
	}

	respond.JSON(w, http.StatusCreated, patient, nil)
}

func (h *Handler) GetPatient(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	patient, ok, err := h.store.GetPatient(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load patient.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	respond.JSON(w, http.StatusOK, patient, nil)
}

func (h *Handler) UpdatePatient(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input UpdatePatientRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validatePatientInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	patient, err := h.store.UpdatePatient(r.Context(), patientID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		case errors.Is(err, ErrCaregiverNotFound):
			respond.Error(w, http.StatusBadRequest, "validation_error", "primaryCaregiverId must reference an existing caregiver.")
		case errors.Is(err, ErrPatientAlreadyAssigned):
			respond.Error(w, http.StatusConflict, "conflict", "This caregiver already has a patient assigned in the MVP.")
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update patient.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, patient, nil)
}

func (h *Handler) GetConsentState(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	state, ok, err := h.store.GetConsentState(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load consent state.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient consent state not found.")
		return
	}

	respond.JSON(w, http.StatusOK, state, nil)
}

func (h *Handler) PutConsentState(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input UpdateConsentRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !contains([]string{ConsentStatusPending, ConsentStatusGranted, ConsentStatusRevoked}, input.OutboundCallStatus) ||
		!contains([]string{ConsentStatusPending, ConsentStatusGranted, ConsentStatusRevoked}, input.TranscriptStorageStatus) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "Consent statuses must be pending, granted, or revoked.")
		return
	}

	state, err := h.store.PutConsentState(r.Context(), patientID, input, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound), errors.Is(err, ErrConsentStateNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Patient consent state not found.")
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update consent state.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, state, nil)
}

func (h *Handler) PausePatient(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input PausePatientRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	patient, err := h.store.SetPatientPause(r.Context(), patientID, input.Reason, time.Now().UTC())
	if err != nil {
		if errors.Is(err, ErrPatientNotFound) {
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not pause patient calling.")
		return
	}

	respond.JSON(w, http.StatusOK, patient, nil)
}

func (h *Handler) UnpausePatient(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	patient, err := h.store.ClearPatientPause(r.Context(), patientID)
	if err != nil {
		if errors.Is(err, ErrPatientNotFound) {
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not resume patient calling.")
		return
	}

	respond.JSON(w, http.StatusOK, patient, nil)
}

func (h *Handler) ListCallTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.store.ListCallTemplates(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load call templates.")
		return
	}

	respond.JSON(w, http.StatusOK, templates, map[string]any{"count": len(templates)})
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	dashboard, err := h.store.GetDashboard(r.Context(), patientID)
	if err != nil {
		if errors.Is(err, ErrPatientNotFound) {
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the dashboard snapshot.")
		return
	}

	respond.JSON(w, http.StatusOK, dashboard, nil)
}

func (h *Handler) CreateCall(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input CreateCallRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if channel := strings.TrimSpace(input.Channel); channel != "" && !contains([]string{CallChannelBrowser, CallChannelConnect}, channel) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "channel must be browser or connect.")
		return
	}
	if trigger := strings.TrimSpace(input.TriggerType); trigger != "" && !contains([]string{CallTriggerManual, CallTriggerApprovedNextCall}, trigger) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "triggerType must be manual or approved_next_call.")
		return
	}

	response, err := h.service.CreateCall(r.Context(), patientID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		case errors.Is(err, ErrPatientConsentRequired), errors.Is(err, ErrPatientPaused), errors.Is(err, ErrCallTemplateNotFound), errors.Is(err, ErrApprovedNextCallPlanRequired), isValidationError(err):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrCallTemplateConflict):
			respond.Error(w, http.StatusConflict, "conflict", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "call_create_error", "Could not create the call run.")
		}
		return
	}

	respond.JSON(w, http.StatusCreated, response, nil)
}

func (h *Handler) GetCall(w http.ResponseWriter, r *http.Request) {
	callRunID := strings.TrimSpace(r.PathValue("id"))
	callRun, ok, err := h.store.GetCallRun(r.Context(), callRunID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the call run.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Call run not found.")
		return
	}

	transcriptTurns, err := h.store.ListTranscriptTurnsForCallRun(r.Context(), callRun.ID)
	if err != nil && !errors.Is(err, ErrCallRunNotFound) {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the call transcript.")
		return
	}

	var analysis *AnalysisRecord
	record, ok, err := h.store.GetAnalysisRecord(r.Context(), callRun.ID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the call analysis.")
		return
	}
	if ok {
		analysis = &record
	}

	respond.JSON(w, http.StatusOK, CallRunDetail{
		CallRun:         callRun,
		TranscriptTurns: transcriptTurns,
		Analysis:        analysis,
	}, nil)
}

func (h *Handler) AnalyzeCall(w http.ResponseWriter, r *http.Request) {
	callRunID := strings.TrimSpace(r.PathValue("id"))
	force, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("force")))

	record, err := h.service.AnalyzeCall(r.Context(), callRunID, force)
	if err != nil {
		switch {
		case errors.Is(err, ErrCallRunNotFound), errors.Is(err, ErrPatientNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Call run not found.")
		case errors.Is(err, ErrCallRunNotCompleted), errors.Is(err, ErrCallRunVoiceSessionMissing), isValidationError(err):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, ErrCallTemplateConflict):
			respond.Error(w, http.StatusConflict, "conflict", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "analysis_error", "Could not analyze the call run.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, record, nil)
}

func (h *Handler) GetCallAnalysis(w http.ResponseWriter, r *http.Request) {
	callRunID := strings.TrimSpace(r.PathValue("id"))
	record, ok, err := h.store.GetAnalysisRecord(r.Context(), callRunID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the call analysis.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Analysis result not found.")
		return
	}

	respond.JSON(w, http.StatusOK, record, nil)
}

func (h *Handler) GetNextCallPlan(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	plan, ok, err := h.store.GetActiveNextCallPlan(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the next-call plan.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Active next-call plan not found.")
		return
	}

	respond.JSON(w, http.StatusOK, plan, nil)
}

func (h *Handler) PutNextCallPlan(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input UpdateNextCallPlanRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !contains([]string{NextCallActionApprove, NextCallActionEdit, NextCallActionReject, NextCallActionCancel}, input.Action) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "action must be approve, edit, reject, or cancel.")
		return
	}

	adminUsername, ok := adminsession.UsernameFromContext(r.Context())
	if !ok {
		respond.Error(w, http.StatusUnauthorized, "unauthorized", "A valid admin session is required.")
		return
	}

	plan, err := h.service.UpdateNextCallPlan(r.Context(), patientID, input, adminUsername)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientNotFound), errors.Is(err, ErrNextCallPlanNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", err.Error())
		case errors.Is(err, ErrCallTemplateNotFound), isValidationError(err):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update the next-call plan.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, plan, nil)
}

func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func validateCaregiverInput(displayName, email, timezone string) error {
	if strings.TrimSpace(displayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(timezone) == "" {
		return errors.New("timezone is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(timezone)); err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}
	return nil
}

func validatePatientInput(input CreatePatientRequest) error {
	if strings.TrimSpace(input.PrimaryCaregiverID) == "" {
		return errors.New("primaryCaregiverId is required")
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(input.PreferredName) == "" {
		return errors.New("preferredName is required")
	}
	if strings.TrimSpace(input.Timezone) == "" {
		return errors.New("timezone is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(input.Timezone)); err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}
	return nil
}

func withCookieRequest(r *http.Request, w http.ResponseWriter) *http.Request {
	clone := r.Clone(r.Context())
	for _, cookie := range w.Header().Values("Set-Cookie") {
		if strings.HasPrefix(cookie, adminsession.CookieName+"=") {
			parts := strings.SplitN(cookie, ";", 2)
			nameValue := strings.SplitN(parts[0], "=", 2)
			if len(nameValue) == 2 {
				clone.AddCookie(&http.Cookie{Name: nameValue[0], Value: nameValue[1]})
			}
		}
	}
	return clone
}
