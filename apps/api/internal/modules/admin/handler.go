package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tether/api/internal/adminsession"
	"tether/api/internal/httpserver/respond"
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

func (h *Handler) ListCaregivers(w http.ResponseWriter, r *http.Request) {
	caregivers, err := h.store.ListCaregivers(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not list caregivers.")
		return
	}
	respond.JSON(w, http.StatusOK, caregivers, nil)
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

func (h *Handler) ListPatients(w http.ResponseWriter, r *http.Request) {
	patients, err := h.store.ListPatients(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not list patients.")
		return
	}
	if patients == nil {
		patients = []Patient{}
	}
	respond.JSON(w, http.StatusOK, patients, nil)
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

func (h *Handler) ListPatientPeople(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load patient people.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	people, err := h.store.ListPatientPeople(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load patient people.")
		return
	}
	if people == nil {
		people = []PatientPerson{}
	}
	respond.JSON(w, http.StatusOK, people, map[string]any{"count": len(people)})
}

func (h *Handler) CreatePatientPerson(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create patient person.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	var input CreatePatientPersonRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	normalizePatientPersonInput(&input)
	if err := validatePatientPersonInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	person, err := h.store.CreatePatientPerson(r.Context(), patientID, input, time.Now().UTC())
	if err != nil {
		if isValidationError(err) {
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create patient person.")
		return
	}

	respond.JSON(w, http.StatusCreated, person, nil)
}

func (h *Handler) UpdatePatientPerson(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	personID := strings.TrimSpace(r.PathValue("personId"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update patient person.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	var input UpdatePatientPersonRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	normalizePatientPersonInput(&input)
	if err := validatePatientPersonInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	person, err := h.store.UpdatePatientPerson(r.Context(), patientID, personID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPatientPersonNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Patient person not found.")
		case isValidationError(err):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update patient person.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, person, nil)
}

func (h *Handler) ListMemoryBankEntries(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load memory bank entries.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	entries, err := h.store.ListMemoryBankEntries(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load memory bank entries.")
		return
	}
	if entries == nil {
		entries = []MemoryBankEntry{}
	}
	respond.JSON(w, http.StatusOK, entries, map[string]any{"count": len(entries)})
}

func (h *Handler) CreateMemoryBankEntry(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create memory bank entry.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	var input CreateMemoryBankEntryRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	normalizeMemoryBankInput(&input)
	if err := validateMemoryBankInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	entry, err := h.store.CreateMemoryBankEntry(r.Context(), patientID, input, time.Now().UTC())
	if err != nil {
		if isValidationError(err) {
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not create memory bank entry.")
		return
	}

	respond.JSON(w, http.StatusCreated, entry, nil)
}

func (h *Handler) UpdateMemoryBankEntry(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	entryID := strings.TrimSpace(r.PathValue("entryId"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update memory bank entry.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	var input UpdateMemoryBankEntryRequest
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	normalizeMemoryBankInput(&input)
	if err := validateMemoryBankInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	entry, err := h.store.UpdateMemoryBankEntry(r.Context(), patientID, entryID, input, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, ErrMemoryBankEntryNotFound):
			respond.Error(w, http.StatusNotFound, "not_found", "Memory bank entry not found.")
		case isValidationError(err):
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			respond.Error(w, http.StatusInternalServerError, "store_error", "Could not update memory bank entry.")
		}
		return
	}

	respond.JSON(w, http.StatusOK, entry, nil)
}

func (h *Handler) ListPatientReminders(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	if _, ok, err := h.store.GetPatient(r.Context(), patientID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load reminders.")
		return
	} else if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
		return
	}

	reminders, err := h.store.ListPatientReminders(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load reminders.")
		return
	}
	if reminders == nil {
		reminders = []Reminder{}
	}
	respond.JSON(w, http.StatusOK, reminders, map[string]any{"count": len(reminders)})
}

func (h *Handler) GetScreeningSchedule(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	schedule, ok, err := h.store.GetScreeningSchedule(r.Context(), patientID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the screening schedule.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Screening schedule not found.")
		return
	}

	respond.JSON(w, http.StatusOK, schedule, nil)
}

func (h *Handler) PutScreeningSchedule(w http.ResponseWriter, r *http.Request) {
	patientID := strings.TrimSpace(r.PathValue("id"))
	var input ScreeningScheduleInput
	if err := decodeJSONBody(r, &input); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateScreeningScheduleInput(input); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	schedule, err := h.store.PutScreeningSchedule(r.Context(), patientID, input, time.Now().UTC())
	if err != nil {
		if errors.Is(err, ErrPatientNotFound) {
			respond.Error(w, http.StatusNotFound, "not_found", "Patient not found.")
			return
		}
		if isValidationError(err) {
			respond.Error(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not save the screening schedule.")
		return
	}

	respond.JSON(w, http.StatusOK, schedule, nil)
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
	input.TriggerType = normalizeRequestedCallTrigger(input.TriggerType)
	if trigger := strings.TrimSpace(input.TriggerType); trigger != "" && !contains(validCallTriggersForRequests(), trigger) {
		respond.Error(w, http.StatusBadRequest, "validation_error", "triggerType must be caregiver_requested or follow_up_recommendation.")
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

	var analysisJob *AnalysisJob
	job, ok, err := h.store.GetAnalysisJob(r.Context(), callRun.ID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the analysis job.")
		return
	}
	if ok {
		analysisJob = &job
	}

	respond.JSON(w, http.StatusOK, CallRunDetail{
		CallRun:         callRun,
		TranscriptTurns: transcriptTurns,
		Analysis:        analysis,
		AnalysisJob:     analysisJob,
	}, nil)
}

func (h *Handler) AnalyzeCall(w http.ResponseWriter, r *http.Request) {
	callRunID := strings.TrimSpace(r.PathValue("id"))
	force, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("force")))

	job, err := h.service.EnqueueAnalysis(r.Context(), callRunID, force)
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

	respond.JSON(w, http.StatusOK, job, nil)
}

func (h *Handler) GetAnalysisJob(w http.ResponseWriter, r *http.Request) {
	callRunID := strings.TrimSpace(r.PathValue("id"))
	job, ok, err := h.store.GetAnalysisJob(r.Context(), callRunID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "store_error", "Could not load the analysis job.")
		return
	}
	if !ok {
		respond.Error(w, http.StatusNotFound, "not_found", "Analysis job not found.")
		return
	}

	respond.JSON(w, http.StatusOK, job, nil)
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
	if err := validateProfilePhotoDataURL(input.ProfilePhotoDataURL); err != nil {
		return err
	}
	return nil
}

func validatePatientPersonInput(input UpdatePatientPersonRequest) error {
	if strings.TrimSpace(input.Name) == "" {
		return newValidationError("name is required")
	}
	if !contains([]string{PersonStatusConfirmedLiving, PersonStatusUnknown, PersonStatusDeceased}, strings.TrimSpace(input.Status)) {
		return newValidationError("status must be confirmed_living, unknown, or deceased")
	}
	if !contains([]string{RelationshipQualityCloseActive, RelationshipQualityUnclear, RelationshipQualityEstranged, RelationshipQualityUnknown}, strings.TrimSpace(input.RelationshipQuality)) {
		return newValidationError("relationshipQuality must be close_active, unclear, estranged, or unknown")
	}
	return nil
}

func normalizePatientPersonInput(input *UpdatePatientPersonRequest) {
	if input == nil {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Relationship = strings.TrimSpace(input.Relationship)
	input.Context = strings.TrimSpace(input.Context)
	input.Notes = strings.TrimSpace(input.Notes)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = PersonStatusUnknown
	}
	input.RelationshipQuality = strings.TrimSpace(input.RelationshipQuality)
	if input.RelationshipQuality == "" {
		input.RelationshipQuality = RelationshipQualityUnknown
	}
}

func normalizeMemoryBankInput(input *CreateMemoryBankEntryRequest) {
	if input == nil {
		return
	}
	input.Topic = strings.TrimSpace(input.Topic)
	input.Summary = strings.TrimSpace(input.Summary)
	input.EmotionalTone = strings.TrimSpace(input.EmotionalTone)
	input.AnchorType = strings.TrimSpace(input.AnchorType)
	if input.AnchorType == "" {
		input.AnchorType = AnchorTypeNone
	}
	input.AnchorDetail = strings.TrimSpace(input.AnchorDetail)
	input.SuggestedFollowUp = strings.TrimSpace(input.SuggestedFollowUp)
}

func validateMemoryBankInput(input CreateMemoryBankEntryRequest) error {
	if strings.TrimSpace(input.Topic) == "" {
		return newValidationError("topic is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return newValidationError("summary is required")
	}
	if !contains(validAnchorTypes(), input.AnchorType) {
		return newValidationError("anchorType must be call, music, show_film, journal, or none")
	}
	if input.AnchorAccepted && !input.AnchorOffered {
		return newValidationError("anchorAccepted cannot be true when anchorOffered is false")
	}
	if input.AnchorAccepted && input.AnchorType == AnchorTypeNone {
		return newValidationError("anchorType must not be none when anchorAccepted is true")
	}
	return nil
}

func validateProfilePhotoDataURL(profilePhotoDataURL string) error {
	normalized := strings.TrimSpace(profilePhotoDataURL)
	if normalized == "" {
		return nil
	}
	if len(normalized) > 2_000_000 {
		return errors.New("profilePhotoDataUrl is too large")
	}
	if !strings.HasPrefix(normalized, "data:image/") || !strings.Contains(normalized, ";base64,") {
		return errors.New("profilePhotoDataUrl must be a base64-encoded image data URL")
	}
	return nil
}

func validateScreeningScheduleInput(input ScreeningScheduleInput) error {
	if !contains(validCadences(), input.Cadence) {
		return newValidationError("cadence must be weekly or biweekly")
	}
	if strings.TrimSpace(input.Timezone) == "" {
		return newValidationError("timezone is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(input.Timezone)); err != nil {
		return newValidationError("timezone must be a valid IANA timezone")
	}
	if input.PreferredWeekday < 0 || input.PreferredWeekday > 6 {
		return newValidationError("preferredWeekday must be between 0 and 6")
	}
	if _, _, err := parseLocalClock(strings.TrimSpace(input.PreferredLocalTime)); err != nil {
		return err
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
