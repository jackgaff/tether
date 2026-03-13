package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"nova-echoes/api/internal/modules/voice"
)

type VoiceSessionCreator interface {
	CreateSession(ctx context.Context, input voice.CreateSessionRequest) (voice.SessionDescriptor, error)
}

type AnalysisRunner interface {
	Analyze(ctx context.Context, promptContext AnalysisPromptContext) (AnalysisPayload, error)
}

type Service struct {
	store           Store
	voiceCreator    VoiceSessionCreator
	analyzer        AnalysisRunner
	analysisModelID string
	now             func() time.Time
}

func NewService(store Store, voiceCreator VoiceSessionCreator, analyzer AnalysisRunner, analysisModelID string) *Service {
	return &Service{
		store:           store,
		voiceCreator:    voiceCreator,
		analyzer:        analyzer,
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

	triggerType := strings.TrimSpace(input.TriggerType)
	if triggerType == "" {
		triggerType = CallTriggerManual
	}

	var (
		template     CallTemplate
		activePlanID string
	)

	switch triggerType {
	case CallTriggerManual:
		if strings.TrimSpace(input.CallTemplateID) != "" && strings.TrimSpace(input.CallType) != "" {
			return CreateCallResponse{}, newValidationError("callTemplateId and callType cannot both be provided")
		}

		if strings.TrimSpace(input.CallTemplateID) != "" {
			var templateFound bool
			template, templateFound, err = s.store.GetCallTemplateByID(ctx, input.CallTemplateID)
			if err != nil {
				return CreateCallResponse{}, err
			}
			if !templateFound || !template.IsActive {
				return CreateCallResponse{}, ErrCallTemplateNotFound
			}
		} else if strings.TrimSpace(input.CallType) != "" {
			template, err = s.store.ResolveActiveCallTemplateByType(ctx, input.CallType)
			if err != nil {
				return CreateCallResponse{}, err
			}
		} else {
			return CreateCallResponse{}, newValidationError("callTemplateId or callType is required")
		}
	case CallTriggerApprovedNextCall:
		plan, planFound, planErr := s.store.GetActiveNextCallPlan(ctx, patient.ID)
		if planErr != nil {
			return CreateCallResponse{}, planErr
		}
		if !planFound || plan.ApprovalStatus != NextCallStatusApproved {
			return CreateCallResponse{}, ErrApprovedNextCallPlanRequired
		}

		activePlanID = plan.ID
		var templateFound bool
		template, templateFound, err = s.store.GetCallTemplateByID(ctx, plan.CallTemplateID)
		if err != nil {
			return CreateCallResponse{}, err
		}
		if !templateFound || !template.IsActive {
			return CreateCallResponse{}, ErrCallTemplateNotFound
		}
	default:
		return CreateCallResponse{}, newValidationError("triggerType must be one of manual or approved_next_call")
	}

	callRun, err := s.store.CreateCallRun(ctx, CreateCallRunParams{
		PatientID:      patient.ID,
		CaregiverID:    patient.PrimaryCaregiverID,
		CallTemplate:   template,
		Channel:        channel,
		TriggerType:    triggerType,
		RequestedAt:    s.now(),
		NextCallPlanID: activePlanID,
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
			return CreateCallResponse{}, sessionErr
		}
		response.VoiceSession = session
	}

	return response, nil
}

func (s *Service) AnalyzeCall(ctx context.Context, callRunID string, force bool) (AnalysisRecord, error) {
	if !force {
		record, ok, err := s.store.GetAnalysisRecord(ctx, callRunID)
		if err != nil {
			return AnalysisRecord{}, err
		}
		if ok {
			return record, nil
		}
	}

	promptContext, err := s.store.GetAnalysisPromptContext(ctx, callRunID)
	if err != nil {
		return AnalysisRecord{}, err
	}

	if s.analyzer == nil {
		return AnalysisRecord{}, errors.New("analysis runner is not configured")
	}

	payload, err := s.analyzer.Analyze(ctx, promptContext)
	if err != nil {
		return AnalysisRecord{}, err
	}

	if err := validateAnalysisPayload(payload); err != nil {
		return AnalysisRecord{}, err
	}

	record, err := s.store.SaveAnalysisResult(ctx, SaveAnalysisResultInput{
		CallRunID:     promptContext.CallRun.ID,
		PatientID:     promptContext.Patient.ID,
		ModelID:       chooseString(s.analysisModelID, "analysis-runner"),
		SchemaVersion: AnalysisSchemaVersion,
		Result:        payload,
		RiskFlags:     deriveRiskFlags(payload),
		CreatedAt:     s.now(),
	})
	if err != nil {
		return AnalysisRecord{}, err
	}

	return record, nil
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

func validateAnalysisPayload(payload AnalysisPayload) error {
	if !contains(validCallTypes(), payload.CallTypeCompleted) {
		return newValidationError("analysis result call_type_completed must be one of orientation, reminder, wellbeing, or reminiscence")
	}
	if !contains(validAnalysisOrientations(), payload.PatientState.Orientation) {
		return newValidationError("analysis result orientation is invalid")
	}
	if !contains(validAnalysisMoods(), payload.PatientState.Mood) {
		return newValidationError("analysis result mood is invalid")
	}
	if !contains(validAnalysisEngagement(), payload.PatientState.Engagement) {
		return newValidationError("analysis result engagement is invalid")
	}
	if payload.PatientState.Confidence < 0 || payload.PatientState.Confidence > 1 {
		return newValidationError("analysis result confidence must be between 0 and 1")
	}
	if !contains(validEscalationLevels(), payload.EscalationLevel) {
		return newValidationError("analysis result escalation_level is invalid")
	}
	if !contains(validCallTypes(), payload.RecommendedNextCall.Type) {
		return newValidationError("analysis result recommended_next_call.type is invalid")
	}
	if payload.RecommendedNextCall.DurationMinutes <= 0 {
		return newValidationError("analysis result recommended_next_call.duration_minutes must be greater than 0")
	}
	return nil
}

func deriveRiskFlags(payload AnalysisPayload) []RiskFlagSeed {
	evidenceQuote := ""
	whyItMatters := ""
	if len(payload.Evidence) > 0 {
		evidenceQuote = strings.TrimSpace(payload.Evidence[0].Quote)
		whyItMatters = strings.TrimSpace(payload.Evidence[0].WhyItMatters)
	}

	add := func(flags []RiskFlagSeed, condition bool, flagType string, severity string, confidence float64) []RiskFlagSeed {
		if !condition {
			return flags
		}
		return append(flags, RiskFlagSeed{
			FlagType:      flagType,
			Severity:      severity,
			EvidenceQuote: evidenceQuote,
			WhyItMatters:  whyItMatters,
			Confidence:    confidence,
		})
	}

	flags := make([]RiskFlagSeed, 0, 8)
	flags = add(flags, payload.Signals.Repetition > 0, "repetition", RiskSeverityInfo, payload.PatientState.Confidence)
	flags = add(flags, payload.Signals.RoutineAdherenceIssue, "routine_adherence_issue", RiskSeverityWatch, payload.PatientState.Confidence)
	flags = add(flags, payload.Signals.SleepConcern, "sleep_concern", RiskSeverityWatch, payload.PatientState.Confidence)
	flags = add(flags, payload.Signals.NutritionOrHydrationConcern, "nutrition_or_hydration_concern", RiskSeverityWatch, payload.PatientState.Confidence)
	flags = add(flags, payload.Signals.PossibleSafetyConcern, "possible_safety_concern", RiskSeverityUrgent, payload.PatientState.Confidence)
	flags = add(flags, payload.Signals.SocialConnectionNeed, "social_connection_need", RiskSeverityInfo, payload.PatientState.Confidence)
	for _, signal := range payload.Signals.PossibleBPSDSignals {
		flags = append(flags, RiskFlagSeed{
			FlagType:      "bpsd_" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(signal), " ", "_")),
			Severity:      RiskSeverityWatch,
			EvidenceQuote: evidenceQuote,
			WhyItMatters:  whyItMatters,
			Confidence:    payload.PatientState.Confidence,
		})
	}
	if payload.EscalationLevel == EscalationCaregiverNow || payload.EscalationLevel == EscalationClinicalReview {
		flags = append(flags, RiskFlagSeed{
			FlagType:      "escalation",
			Severity:      RiskSeverityUrgent,
			EvidenceQuote: evidenceQuote,
			WhyItMatters:  whyItMatters,
			Confidence:    payload.PatientState.Confidence,
		})
	}
	return flags
}

func validCallTypes() []string {
	return []string{CallTypeOrientation, CallTypeReminder, CallTypeWellbeing, CallTypeReminiscence}
}

func validAnalysisOrientations() []string {
	return []string{AnalysisOrientationGood, AnalysisOrientationMixed, AnalysisOrientationPoor, AnalysisOrientationUnclear}
}

func validAnalysisMoods() []string {
	return []string{AnalysisMoodPositive, AnalysisMoodNeutral, AnalysisMoodAnxious, AnalysisMoodSad, AnalysisMoodDistressed, AnalysisMoodUnclear}
}

func validAnalysisEngagement() []string {
	return []string{AnalysisEngagementHigh, AnalysisEngagementMedium, AnalysisEngagementLow}
}

func validEscalationLevels() []string {
	return []string{EscalationNone, EscalationCaregiverSoon, EscalationCaregiverNow, EscalationClinicalReview}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
