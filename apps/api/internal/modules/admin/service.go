package admin

import (
	"context"
	"strings"
	"time"

	"nova-echoes/api/internal/modules/voice"
)

type VoiceSessionCreator interface {
	CreateSession(ctx context.Context, input voice.CreateSessionRequest) (voice.SessionDescriptor, error)
}

type serviceStore interface {
	GetPatient(ctx context.Context, patientID string) (Patient, bool, error)
	GetConsentState(ctx context.Context, patientID string) (ConsentState, bool, error)
	GetCallTemplateByID(ctx context.Context, templateID string) (CallTemplate, bool, error)
	ResolveActiveCallTemplateByType(ctx context.Context, callType string) (CallTemplate, error)
	CreateCallRun(ctx context.Context, input CreateCallRunParams) (CallRun, error)
	MarkCallRunFailed(ctx context.Context, callRunID, stopReason string, endedAt time.Time) error
	GetActiveNextCallPlan(ctx context.Context, patientID string) (NextCallPlan, bool, error)
	GetCallRun(ctx context.Context, callRunID string) (CallRun, bool, error)
	GetAnalysisJob(ctx context.Context, callRunID string) (AnalysisJob, bool, error)
	UpsertAnalysisJob(ctx context.Context, input UpsertAnalysisJobParams) (AnalysisJob, error)
	UpdateNextCallPlan(ctx context.Context, patientID string, input UpdateNextCallPlanStoreInput) (NextCallPlan, error)
}

type Service struct {
	store           serviceStore
	voiceCreator    VoiceSessionCreator
	analysisModelID string
	now             func() time.Time
}

func NewService(store serviceStore, voiceCreator VoiceSessionCreator, analysisModelID string) *Service {
	return &Service{
		store:           store,
		voiceCreator:    voiceCreator,
		analysisModelID: strings.TrimSpace(analysisModelID),
		now:             func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) CreateCall(ctx context.Context, patientID string, input CreateCallRequest) (CreateCallResponse, error) {
	patient, ok, err := s.store.GetPatient(ctx, patientID)
	if err != nil {
		return CreateCallResponse{}, err
	}
	if !ok {
		return CreateCallResponse{}, ErrPatientNotFound
	}

	consent, ok, err := s.store.GetConsentState(ctx, patient.ID)
	if err != nil {
		return CreateCallResponse{}, err
	}
	if !ok {
		return CreateCallResponse{}, ErrConsentStateNotFound
	}
	if consent.OutboundCallStatus != ConsentStatusGranted || consent.TranscriptStorageStatus != ConsentStatusGranted {
		return CreateCallResponse{}, ErrPatientConsentRequired
	}
	if patient.CallingState == CallingStatePaused {
		return CreateCallResponse{}, ErrPatientPaused
	}

	channel := strings.TrimSpace(input.Channel)
	if channel == "" {
		channel = CallChannelBrowser
	}
	if !contains([]string{CallChannelBrowser, CallChannelConnect}, channel) {
		return CreateCallResponse{}, newValidationError("channel must be browser or connect")
	}

	triggerType := normalizeRequestedCallTrigger(input.TriggerType)
	if !contains(validCallTriggersForRequests(), triggerType) {
		return CreateCallResponse{}, newValidationError("triggerType must be caregiver_requested or follow_up_recommendation")
	}

	template, err := s.resolveCallTemplate(ctx, patient, input, triggerType)
	if err != nil {
		return CreateCallResponse{}, err
	}

	callRun, err := s.store.CreateCallRun(ctx, CreateCallRunParams{
		PatientID:    patient.ID,
		CaregiverID:  patient.PrimaryCaregiverID,
		CallTemplate: template,
		CallType:     template.CallType,
		Channel:      channel,
		TriggerType:  triggerType,
		Status:       CallRunStatusRequested,
		RequestedAt:  s.now(),
	})
	if err != nil {
		return CreateCallResponse{}, err
	}

	response := CreateCallResponse{CallRun: callRun}
	if channel == CallChannelBrowser {
		session, sessionErr := s.voiceCreator.CreateSession(ctx, voice.CreateSessionRequest{
			PatientID:    patient.ID,
			SystemPrompt: template.SystemPromptTemplate,
			CallRunID:    callRun.ID,
		})
		if sessionErr != nil {
			_ = s.store.MarkCallRunFailed(ctx, callRun.ID, "voice_session_bootstrap_failed", s.now())
			return CreateCallResponse{}, sessionErr
		}
		response.VoiceSession = session
	}

	return response, nil
}

func (s *Service) EnqueueAnalysis(ctx context.Context, callRunID string, force bool) (AnalysisJob, error) {
	callRun, ok, err := s.store.GetCallRun(ctx, callRunID)
	if err != nil {
		return AnalysisJob{}, err
	}
	if !ok {
		return AnalysisJob{}, ErrCallRunNotFound
	}
	if callRun.Status != CallRunStatusCompleted {
		return AnalysisJob{}, ErrCallRunNotCompleted
	}
	if strings.TrimSpace(callRun.SourceVoiceSessionID) == "" {
		return AnalysisJob{}, ErrCallRunVoiceSessionMissing
	}

	template, ok, err := s.store.GetCallTemplateByID(ctx, callRun.CallTemplateID)
	if err != nil {
		return AnalysisJob{}, err
	}
	if !ok {
		return AnalysisJob{}, ErrCallTemplateNotFound
	}

	return s.store.UpsertAnalysisJob(ctx, UpsertAnalysisJobParams{
		CallRunID:             callRun.ID,
		Force:                 force,
		AnalysisPromptVersion: template.AnalysisPromptVersion,
		AnalysisSchemaVersion: AnalysisSchemaVersion,
		ModelProvider:         AnalysisModelProvider,
		ModelName:             chooseString(s.analysisModelID, "analysis-runner"),
		Now:                   s.now(),
	})
}

func (s *Service) UpdateNextCallPlan(ctx context.Context, patientID string, input UpdateNextCallPlanRequest, adminUsername string) (NextCallPlan, error) {
	patient, ok, err := s.store.GetPatient(ctx, patientID)
	if err != nil {
		return NextCallPlan{}, err
	}
	if !ok {
		return NextCallPlan{}, ErrPatientNotFound
	}

	var plannedFor *time.Time
	if strings.TrimSpace(input.PlannedFor) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(input.PlannedFor))
		if parseErr != nil {
			return NextCallPlan{}, newValidationError("plannedFor must be a valid RFC3339 timestamp")
		}
		plannedFor = &parsed
	}

	var callTemplate *CallTemplate
	if strings.TrimSpace(input.CallTemplateID) != "" {
		template, ok, err := s.store.GetCallTemplateByID(ctx, input.CallTemplateID)
		if err != nil {
			return NextCallPlan{}, err
		}
		if !ok || !template.IsActive {
			return NextCallPlan{}, ErrCallTemplateNotFound
		}
		callTemplate = &template
	}

	return s.store.UpdateNextCallPlan(ctx, patient.ID, UpdateNextCallPlanStoreInput{
		Action:              input.Action,
		CallTemplate:        callTemplate,
		SuggestedTimeNote:   input.SuggestedTimeNote,
		PlannedFor:          plannedFor,
		DurationMinutes:     input.DurationMinutes,
		Goal:                input.Goal,
		Reason:              input.Reason,
		AdminUsername:       adminUsername,
		ApprovedCaregiverID: patient.PrimaryCaregiverID,
		Now:                 s.now(),
	})
}

func (s *Service) resolveCallTemplate(ctx context.Context, patient Patient, input CreateCallRequest, triggerType string) (CallTemplate, error) {
	switch triggerType {
	case CallTriggerCaregiverRequested:
		if strings.TrimSpace(input.CallTemplateID) != "" && strings.TrimSpace(input.CallType) != "" {
			return CallTemplate{}, newValidationError("callTemplateId and callType cannot both be provided")
		}

		if strings.TrimSpace(input.CallTemplateID) != "" {
			template, ok, err := s.store.GetCallTemplateByID(ctx, input.CallTemplateID)
			if err != nil {
				return CallTemplate{}, err
			}
			if !ok || !template.IsActive {
				return CallTemplate{}, ErrCallTemplateNotFound
			}
			return template, nil
		}

		if strings.TrimSpace(input.CallType) == "" {
			return CallTemplate{}, newValidationError("callTemplateId or callType is required")
		}

		return s.store.ResolveActiveCallTemplateByType(ctx, input.CallType)
	case CallTriggerFollowUpRecommendation:
		plan, ok, err := s.store.GetActiveNextCallPlan(ctx, patient.ID)
		if err != nil {
			return CallTemplate{}, err
		}
		if !ok || plan.ApprovalStatus != NextCallStatusApproved {
			return CallTemplate{}, ErrApprovedNextCallPlanRequired
		}

		template, ok, err := s.store.GetCallTemplateByID(ctx, plan.CallTemplateID)
		if err != nil {
			return CallTemplate{}, err
		}
		if !ok || !template.IsActive {
			return CallTemplate{}, ErrCallTemplateNotFound
		}

		return template, nil
	default:
		return CallTemplate{}, newValidationError("unsupported triggerType")
	}
}

func validCallTypes() []string {
	return []string{CallTypeScreening, CallTypeCheckIn, CallTypeReminiscence}
}

func validCallTriggers() []string {
	return []string{CallTriggerCaregiverRequested, CallTriggerScheduled, CallTriggerFollowUpRecommendation}
}

func validCallTriggersForRequests() []string {
	return []string{CallTriggerCaregiverRequested, CallTriggerFollowUpRecommendation}
}

func validEscalationLevels() []string {
	return []string{EscalationNone, EscalationCaregiverSoon, EscalationCaregiverNow, EscalationClinicalReview}
}

func validCadences() []string {
	return []string{CadenceWeekly, CadenceBiweekly}
}

func validTimeframeBuckets() []string {
	return []string{TimeframeSameDay, TimeframeTomorrow, TimeframeFewDays, TimeframeNextWeek, TimeframeTwoWeeks, TimeframeUnspecified}
}

func validRiskSeverities() []string {
	return []string{RiskSeverityInfo, RiskSeverityWatch, RiskSeverityUrgent}
}

func validScreeningCompletionStatuses() []string {
	return []string{ScreeningCompletionComplete, ScreeningCompletionPartial, ScreeningCompletionAborted}
}

func validScreeningInterpretations() []string {
	return []string{
		ScreeningInterpretationRoutineFollowUp,
		ScreeningInterpretationCaregiverReview,
		ScreeningInterpretationClinicalReview,
		ScreeningInterpretationIncomplete,
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
