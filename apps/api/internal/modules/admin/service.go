package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nova-echoes/api/internal/modules/voice"
	"nova-echoes/api/internal/prompts"
)

type VoiceSessionCreator interface {
	CreateSession(ctx context.Context, input voice.CreateSessionRequest) (voice.SessionDescriptor, error)
}

type serviceStore interface {
	GetPatient(ctx context.Context, patientID string) (Patient, bool, error)
	GetCallPromptContext(ctx context.Context, patientID string) (CallPromptContext, error)
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
		systemPrompt, renderErr := s.renderCallPrompt(ctx, patient, template)
		if renderErr != nil {
			_ = s.store.MarkCallRunFailed(ctx, callRun.ID, "call_prompt_render_failed", s.now())
			return CreateCallResponse{}, renderErr
		}
		session, sessionErr := s.voiceCreator.CreateSession(ctx, voice.CreateSessionRequest{
			PatientID:    patient.ID,
			SystemPrompt: systemPrompt,
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
		if !contains(activeCallTypes(), strings.TrimSpace(input.CallType)) {
			return CallTemplate{}, newValidationError("callType must be check_in or reminiscence")
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

func activeCallTypes() []string {
	return prompts.ActiveCallTypes()
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

func validOrientationStatuses() []string {
	return []string{OrientationStatusOriented, OrientationStatusMildlyConfused, OrientationStatusDisoriented}
}

func validCheckInCaptureStatuses() []string {
	return []string{CheckInCaptureReported, CheckInCaptureUncertain, CheckInCaptureNotRecalled}
}

func validSocialContactStatuses() []string {
	return []string{SocialContactYes, SocialContactNo}
}

func validCheckInMoods() []string {
	return []string{CheckInMoodCalm, CheckInMoodWithdrawn, CheckInMoodDistressed, CheckInMoodElevated}
}

func validSleepStatuses() []string {
	return []string{SleepStatusGood, SleepStatusPoor, SleepStatusReversed}
}

func validAnchorTypes() []string {
	return []string{AnchorTypeCall, AnchorTypeMusic, AnchorTypeShowFilm, AnchorTypeJournal, AnchorTypeNone}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *Service) renderCallPrompt(ctx context.Context, patient Patient, template CallTemplate) (string, error) {
	promptContext, err := s.store.GetCallPromptContext(ctx, patient.ID)
	if err != nil {
		return "", err
	}

	now := s.now().In(loadLocationOrUTC(patient.Timezone))
	rendered, err := prompts.RenderCallPrompt(template.SystemPromptTemplate, prompts.RenderContext{
		PatientFirstName:             chooseString(strings.TrimSpace(patient.PreferredName), strings.TrimSpace(patient.DisplayName)),
		CurrentWeekday:               now.Weekday().String(),
		CurrentDateLong:              now.Format("January 2, 2006"),
		RoutineAnchorsBlock:          renderPromptList(promptContext.Patient.RoutineAnchors),
		FavoriteTopicsBlock:          renderPromptList(promptContext.Patient.FavoriteTopics),
		CalmingCuesBlock:             renderPromptList(promptContext.Patient.CalmingCues),
		TopicsToAvoidBlock:           renderPromptList(promptContext.Patient.TopicsToAvoid),
		KnownInterestsBlock:          renderPromptList(promptContext.Patient.MemoryProfile.Likes),
		SignificantPlacesBlock:       renderPromptList(promptContext.Patient.MemoryProfile.SignificantPlaces),
		LifeChaptersBlock:            renderPromptList(promptContext.Patient.MemoryProfile.LifeChapters),
		FavoriteMusicBlock:           renderPromptList(promptContext.Patient.MemoryProfile.FavoriteMusic),
		FavoriteShowsFilmsBlock:      renderPromptList(promptContext.Patient.MemoryProfile.FavoriteShowsFilms),
		TopicsToRevisitBlock:         renderPromptList(promptContext.Patient.MemoryProfile.TopicsToRevisit),
		SafePeopleForCallAnchorBlock: renderPeoplePromptList(promptContext.SafePeopleForCallAnchor),
		PeopleToAvoidNamingBlock:     renderPeoplePromptList(promptContext.PeopleToAvoidNaming),
		RecentMemoryFollowUpsBlock:   renderMemoryFollowUpList(promptContext.RecentMemoryBankEntries),
	})
	if err != nil {
		return "", fmt.Errorf("render call prompt: %w", err)
	}
	return rendered, nil
}

func loadLocationOrUTC(timezone string) *time.Location {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return time.UTC
	}
	return location
}

func renderPromptList(values []string) string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, "- "+trimmed)
	}
	if len(normalized) == 0 {
		return "- None noted yet."
	}
	return strings.Join(normalized, "\n")
}

func renderPeoplePromptList(people []PatientPerson) string {
	lines := make([]string, 0, len(people))
	for _, person := range people {
		parts := []string{strings.TrimSpace(person.Name)}
		if relationship := strings.TrimSpace(person.Relationship); relationship != "" {
			parts = append(parts, "("+relationship+")")
		}
		if context := strings.TrimSpace(person.Context); context != "" {
			parts = append(parts, "- "+context)
		}
		line := strings.TrimSpace(strings.Join(parts, " "))
		if line == "" {
			continue
		}
		lines = append(lines, "- "+line)
	}
	if len(lines) == 0 {
		return "- None confirmed."
	}
	return strings.Join(lines, "\n")
}

func renderMemoryFollowUpList(entries []MemoryBankEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		followUp := strings.TrimSpace(entry.SuggestedFollowUp)
		if followUp == "" {
			continue
		}
		if topic := strings.TrimSpace(entry.Topic); topic != "" {
			lines = append(lines, "- "+topic+": "+followUp)
			continue
		}
		lines = append(lines, "- "+followUp)
	}
	if len(lines) == 0 {
		return "- No follow-up threads logged yet."
	}
	return strings.Join(lines, "\n")
}
