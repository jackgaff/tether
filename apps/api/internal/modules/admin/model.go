package admin

import (
	"encoding/json"
	"time"
)

const (
	CallingStateActive = "active"
	CallingStatePaused = "paused"
)

const (
	ConsentStatusPending = "pending"
	ConsentStatusGranted = "granted"
	ConsentStatusRevoked = "revoked"
)

const (
	CallTypeScreening    = "screening"
	CallTypeCheckIn      = "check_in"
	CallTypeReminiscence = "reminiscence"
)

const (
	PersonStatusConfirmedLiving = "confirmed_living"
	PersonStatusUnknown         = "unknown"
	PersonStatusDeceased        = "deceased"
)

const (
	RelationshipQualityCloseActive = "close_active"
	RelationshipQualityUnclear     = "unclear"
	RelationshipQualityEstranged   = "estranged"
	RelationshipQualityUnknown     = "unknown"
)

const (
	ReminderKindCallPerson  = "call_person"
	ReminderKindMusic       = "music"
	ReminderKindShowFilm    = "show_film"
	ReminderKindJournal     = "journal"
	ReminderKindAppointment = "appointment"
	ReminderKindGeneral     = "general"
)

const (
	ReminderStatusPending   = "pending"
	ReminderStatusCompleted = "completed"
	ReminderStatusDeclined  = "declined"
	ReminderStatusCancelled = "cancelled"
)

const (
	ReminderCreatedByAnalysisWorker = "analysis_worker"
	ReminderCreatedByAdmin          = "admin"
)

const (
	OrientationStatusOriented       = "oriented"
	OrientationStatusMildlyConfused = "mildly_confused"
	OrientationStatusDisoriented    = "disoriented"
)

const (
	CheckInCaptureReported    = "reported"
	CheckInCaptureUncertain   = "uncertain"
	CheckInCaptureNotRecalled = "not_recalled"
)

const (
	SocialContactYes = "yes"
	SocialContactNo  = "no"
)

const (
	CheckInMoodCalm       = "calm"
	CheckInMoodWithdrawn  = "withdrawn"
	CheckInMoodDistressed = "distressed"
	CheckInMoodElevated   = "elevated"
)

const (
	SleepStatusGood     = "good"
	SleepStatusPoor     = "poor"
	SleepStatusReversed = "reversed"
)

const (
	AnchorTypeCall     = "call"
	AnchorTypeMusic    = "music"
	AnchorTypeShowFilm = "show_film"
	AnchorTypeJournal  = "journal"
	AnchorTypeNone     = "none"
)

const (
	CallChannelBrowser = "browser"
	CallChannelConnect = "connect"
)

const (
	CallTriggerCaregiverRequested     = "caregiver_requested"
	CallTriggerScheduled              = "scheduled"
	CallTriggerFollowUpRecommendation = "follow_up_recommendation"
	CallTriggerLegacyManual           = "manual"
	CallTriggerLegacyApprovedNextCall = "approved_next_call"
)

const (
	CallRunStatusScheduled  = "scheduled"
	CallRunStatusRequested  = "requested"
	CallRunStatusInProgress = "in_progress"
	CallRunStatusCompleted  = "completed"
	CallRunStatusFailed     = "failed"
	CallRunStatusCancelled  = "cancelled"
)

const (
	AnalysisJobStatusPending   = "pending"
	AnalysisJobStatusRunning   = "running"
	AnalysisJobStatusSucceeded = "succeeded"
	AnalysisJobStatusFailed    = "failed"
)

const (
	EscalationNone           = "none"
	EscalationCaregiverSoon  = "caregiver_soon"
	EscalationCaregiverNow   = "caregiver_now"
	EscalationClinicalReview = "clinical_review"
)

const (
	RiskSeverityInfo   = "info"
	RiskSeverityWatch  = "watch"
	RiskSeverityUrgent = "urgent"
)

const (
	NextCallStatusPendingApproval = "pending_approval"
	NextCallStatusApproved        = "approved"
	NextCallStatusRejected        = "rejected"
	NextCallStatusExecuted        = "executed"
	NextCallStatusSuperseded      = "superseded"
	NextCallStatusCancelled       = "cancelled"
)

const (
	NextCallActionApprove = "approve"
	NextCallActionEdit    = "edit"
	NextCallActionReject  = "reject"
	NextCallActionCancel  = "cancel"
)

const (
	CadenceWeekly   = "weekly"
	CadenceBiweekly = "biweekly"
)

const (
	TimeframeSameDay     = "same_day"
	TimeframeTomorrow    = "tomorrow"
	TimeframeFewDays     = "few_days"
	TimeframeNextWeek    = "next_week"
	TimeframeTwoWeeks    = "two_weeks"
	TimeframeUnspecified = "unspecified"
)

const (
	ScreeningCompletionComplete = "complete"
	ScreeningCompletionPartial  = "partial"
	ScreeningCompletionAborted  = "aborted"
)

const (
	ScreeningInterpretationRoutineFollowUp = "routine_follow_up"
	ScreeningInterpretationCaregiverReview = "caregiver_review_suggested"
	ScreeningInterpretationClinicalReview  = "clinical_review_suggested"
	ScreeningInterpretationIncomplete      = "incomplete"
)

const AnalysisSchemaVersion = "v2"
const AnalysisModelProvider = "amazon"

type Caregiver struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
	PhoneE164   string    `json:"phoneE164,omitempty"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type FamilyMember struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Notes    string `json:"notes,omitempty"`
}

type LifeEvent struct {
	Label           string `json:"label"`
	ApproximateDate string `json:"approximateDate,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

type MemoryProfile struct {
	Likes              []string       `json:"likes"`
	FamilyMembers      []FamilyMember `json:"familyMembers"`
	LifeEvents         []LifeEvent    `json:"lifeEvents"`
	ReminiscenceNotes  string         `json:"reminiscenceNotes,omitempty"`
	SignificantPlaces  []string       `json:"significantPlaces"`
	LifeChapters       []string       `json:"lifeChapters"`
	FavoriteMusic      []string       `json:"favoriteMusic"`
	FavoriteShowsFilms []string       `json:"favoriteShowsFilms"`
	TopicsToRevisit    []string       `json:"topicsToRevisit"`
}

type ConversationGuidance struct {
	PreferredGreetingStyle string   `json:"preferredGreetingStyle,omitempty"`
	CalmingTopics          []string `json:"calmingTopics"`
	UpsettingTopics        []string `json:"upsettingTopics"`
	HearingOrPacingNotes   string   `json:"hearingOrPacingNotes,omitempty"`
	BestTimeOfDay          string   `json:"bestTimeOfDay,omitempty"`
	DoNotMention           []string `json:"doNotMention"`
}

type Patient struct {
	ID                   string               `json:"id"`
	PrimaryCaregiverID   string               `json:"primaryCaregiverId"`
	DisplayName          string               `json:"displayName"`
	PreferredName        string               `json:"preferredName"`
	PhoneE164            string               `json:"phoneE164,omitempty"`
	Timezone             string               `json:"timezone"`
	Notes                string               `json:"notes,omitempty"`
	CallingState         string               `json:"callingState"`
	PauseReason          string               `json:"pauseReason,omitempty"`
	PausedAt             *time.Time           `json:"pausedAt,omitempty"`
	RoutineAnchors       []string             `json:"routineAnchors"`
	FavoriteTopics       []string             `json:"favoriteTopics"`
	CalmingCues          []string             `json:"calmingCues"`
	TopicsToAvoid        []string             `json:"topicsToAvoid"`
	MemoryProfile        MemoryProfile        `json:"memoryProfile"`
	ConversationGuidance ConversationGuidance `json:"conversationGuidance"`
	CreatedAt            time.Time            `json:"createdAt"`
	UpdatedAt            time.Time            `json:"updatedAt"`
}

type ScreeningSchedule struct {
	PatientID                string     `json:"patientId"`
	Enabled                  bool       `json:"enabled"`
	Cadence                  string     `json:"cadence"`
	Timezone                 string     `json:"timezone"`
	PreferredWeekday         int        `json:"preferredWeekday"`
	PreferredLocalTime       string     `json:"preferredLocalTime"`
	NextDueAt                *time.Time `json:"nextDueAt,omitempty"`
	LastScheduledWindowStart *time.Time `json:"lastScheduledWindowStart,omitempty"`
	LastScheduledWindowEnd   *time.Time `json:"lastScheduledWindowEnd,omitempty"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}

type ScreeningScheduleInput struct {
	Enabled            bool   `json:"enabled"`
	Cadence            string `json:"cadence"`
	Timezone           string `json:"timezone"`
	PreferredWeekday   int    `json:"preferredWeekday"`
	PreferredLocalTime string `json:"preferredLocalTime"`
}

type ConsentState struct {
	PatientID               string     `json:"patientId"`
	OutboundCallStatus      string     `json:"outboundCallStatus"`
	TranscriptStorageStatus string     `json:"transcriptStorageStatus"`
	GrantedByCaregiverID    string     `json:"grantedByCaregiverId,omitempty"`
	GrantedAt               *time.Time `json:"grantedAt,omitempty"`
	RevokedAt               *time.Time `json:"revokedAt,omitempty"`
	Notes                   string     `json:"notes,omitempty"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

type CallTemplate struct {
	ID                     string          `json:"id"`
	Slug                   string          `json:"slug"`
	DisplayName            string          `json:"displayName"`
	CallType               string          `json:"callType"`
	Description            string          `json:"description"`
	DurationMinutes        int             `json:"durationMinutes"`
	PromptVersion          string          `json:"promptVersion"`
	CallPromptVersion      string          `json:"callPromptVersion"`
	SystemPromptTemplate   string          `json:"systemPromptTemplate"`
	AnalysisPromptVersion  string          `json:"analysisPromptVersion"`
	AnalysisPromptTemplate string          `json:"analysisPromptTemplate"`
	Checklist              json.RawMessage `json:"checklist"`
	IsActive               bool            `json:"isActive"`
	CreatedAt              time.Time       `json:"createdAt"`
	UpdatedAt              time.Time       `json:"updatedAt"`
}

type CallRun struct {
	ID                   string     `json:"id"`
	PatientID            string     `json:"patientId"`
	CaregiverID          string     `json:"caregiverId"`
	CallTemplateID       string     `json:"callTemplateId"`
	CallTemplateSlug     string     `json:"callTemplateSlug,omitempty"`
	CallTemplateName     string     `json:"callTemplateName,omitempty"`
	CallType             string     `json:"callType"`
	Channel              string     `json:"channel"`
	TriggerType          string     `json:"triggerType"`
	Status               string     `json:"status"`
	SourceVoiceSessionID string     `json:"sourceVoiceSessionId,omitempty"`
	ScheduleWindowStart  *time.Time `json:"scheduleWindowStart,omitempty"`
	ScheduleWindowEnd    *time.Time `json:"scheduleWindowEnd,omitempty"`
	RequestedAt          time.Time  `json:"requestedAt"`
	StartedAt            *time.Time `json:"startedAt,omitempty"`
	EndedAt              *time.Time `json:"endedAt,omitempty"`
	StopReason           string     `json:"stopReason,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type CallTranscriptTurn struct {
	SequenceNo  int       `json:"sequenceNo"`
	Direction   string    `json:"direction"`
	SpeakerRole string    `json:"speakerRole,omitempty"`
	Modality    string    `json:"modality"`
	Text        string    `json:"text"`
	OccurredAt  time.Time `json:"occurredAt"`
	StopReason  string    `json:"stopReason,omitempty"`
}

type SalientEvidence struct {
	Quote  string `json:"quote"`
	Reason string `json:"reason"`
}

type AnalysisRiskFlag struct {
	FlagType     string  `json:"flagType"`
	Severity     string  `json:"severity"`
	Evidence     string  `json:"evidence,omitempty"`
	Reason       string  `json:"reason,omitempty"`
	WhyItMatters string  `json:"whyItMatters,omitempty"`
	Confidence   float64 `json:"confidence"`
}

type FollowUpIntent struct {
	RequestedByPatient bool    `json:"requestedByPatient"`
	TimeframeBucket    string  `json:"timeframeBucket"`
	Evidence           string  `json:"evidence,omitempty"`
	Confidence         float64 `json:"confidence"`
}

type NextCallRecommendation struct {
	CallType     string `json:"callType"`
	WindowBucket string `json:"windowBucket"`
	Goal         string `json:"goal"`
}

type ScreeningAnalysis struct {
	ScreeningItemsAdministered    []string `json:"screeningItemsAdministered"`
	ScreeningCompletionStatus     string   `json:"screeningCompletionStatus"`
	ScreeningScoreRaw             string   `json:"screeningScoreRaw,omitempty"`
	ScreeningScoreInterpretation  string   `json:"screeningScoreInterpretation,omitempty"`
	ScreeningFlags                []string `json:"screeningFlags"`
	SuggestedRescreenWindowBucket string   `json:"suggestedRescreenWindowBucket,omitempty"`
}

type ReminderNote struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type CheckInAnalysis struct {
	OrientationStatus         string         `json:"orientationStatus"`
	OrientationNotes          string         `json:"orientationNotes,omitempty"`
	MealsStatus               string         `json:"mealsStatus"`
	MealsDetail               string         `json:"mealsDetail,omitempty"`
	FluidsStatus              string         `json:"fluidsStatus"`
	FluidsDetail              string         `json:"fluidsDetail,omitempty"`
	ActivityDetail            string         `json:"activityDetail,omitempty"`
	SocialContact             string         `json:"socialContact"`
	SocialContactDetail       string         `json:"socialContactDetail,omitempty"`
	RemindersNoted            []ReminderNote `json:"remindersNoted"`
	ReminderDeclined          bool           `json:"reminderDeclined"`
	ReminderDeclinedTopic     string         `json:"reminderDeclinedTopic,omitempty"`
	Mood                      string         `json:"mood"`
	MoodNotes                 string         `json:"moodNotes,omitempty"`
	Sleep                     string         `json:"sleep"`
	SleepNotes                string         `json:"sleepNotes,omitempty"`
	MemoryFlags               []string       `json:"memoryFlags"`
	DeliriumWatch             bool           `json:"deliriumWatch"`
	DeliriumWatchNotes        string         `json:"deliriumWatchNotes,omitempty"`
	DeliriumPotentialTriggers []string       `json:"deliriumPotentialTriggers"`
	CaregiverSummary          string         `json:"caregiverSummary,omitempty"`
}

type MentionedPerson struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship,omitempty"`
	Context      string `json:"context,omitempty"`
}

type ReminiscenceAnalysis struct {
	Topic               string            `json:"topic,omitempty"`
	MentionedPeople     []MentionedPerson `json:"mentionedPeople"`
	MentionedPlaces     []string          `json:"mentionedPlaces"`
	MentionedMusic      []string          `json:"mentionedMusic"`
	MentionedShowsFilms []string          `json:"mentionedShowsFilms"`
	LifeChapters        []string          `json:"lifeChapters"`
	Summary             string            `json:"summary,omitempty"`
	EmotionalTone       string            `json:"emotionalTone,omitempty"`
	RespondedWellTo     []string          `json:"respondedWellTo"`
	AnchorOffered       bool              `json:"anchorOffered"`
	AnchorType          string            `json:"anchorType,omitempty"`
	AnchorAccepted      bool              `json:"anchorAccepted"`
	AnchorDetail        string            `json:"anchorDetail,omitempty"`
	SuggestedFollowUp   string            `json:"suggestedFollowUp,omitempty"`
	CaregiverSummary    string            `json:"caregiverSummary,omitempty"`
}

type LegacyPatientState struct {
	Orientation string  `json:"orientation"`
	Mood        string  `json:"mood"`
	Engagement  string  `json:"engagement"`
	Confidence  float64 `json:"confidence"`
}

type AnalysisPayload struct {
	Summary                string                  `json:"summary"`
	SalientEvidence        []SalientEvidence       `json:"salientEvidence"`
	RiskFlags              []AnalysisRiskFlag      `json:"riskFlags"`
	EscalationLevel        string                  `json:"escalationLevel"`
	CaregiverReviewReason  string                  `json:"caregiverReviewReason,omitempty"`
	FollowUpIntent         FollowUpIntent          `json:"followUpIntent"`
	NextCallRecommendation *NextCallRecommendation `json:"nextCallRecommendation,omitempty"`
	Screening              *ScreeningAnalysis      `json:"screening,omitempty"`
	CheckIn                *CheckInAnalysis        `json:"checkIn,omitempty"`
	Reminiscence           *ReminiscenceAnalysis   `json:"reminiscence,omitempty"`
	DashboardSummary       string                  `json:"dashboard_summary,omitempty"`
	CaregiverSummary       string                  `json:"caregiver_summary,omitempty"`
	PatientState           *LegacyPatientState     `json:"patient_state,omitempty"`
}

type RiskFlag struct {
	ID               string    `json:"id"`
	AnalysisResultID string    `json:"analysisResultId"`
	FlagType         string    `json:"flagType"`
	Severity         string    `json:"severity"`
	Evidence         string    `json:"evidence,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	WhyItMatters     string    `json:"whyItMatters,omitempty"`
	Confidence       float64   `json:"confidence"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AnalysisRecord struct {
	ID                    string          `json:"id"`
	CallRunID             string          `json:"callRunId"`
	CallTemplateID        string          `json:"callTemplateId,omitempty"`
	ModelID               string          `json:"modelId"`
	ModelProvider         string          `json:"modelProvider"`
	ModelName             string          `json:"modelName"`
	CallPromptVersion     string          `json:"callPromptVersion"`
	AnalysisPromptVersion string          `json:"analysisPromptVersion"`
	SchemaVersion         string          `json:"schemaVersion"`
	GeneratedAt           time.Time       `json:"generatedAt"`
	Result                AnalysisPayload `json:"result"`
	RiskFlags             []RiskFlag      `json:"riskFlags"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

type AnalysisJob struct {
	ID                    string     `json:"id"`
	CallRunID             string     `json:"callRunId"`
	Status                string     `json:"status"`
	AttemptCount          int        `json:"attemptCount"`
	LastError             string     `json:"lastError,omitempty"`
	LockedAt              *time.Time `json:"lockedAt,omitempty"`
	StartedAt             *time.Time `json:"startedAt,omitempty"`
	FinishedAt            *time.Time `json:"finishedAt,omitempty"`
	AnalysisPromptVersion string     `json:"analysisPromptVersion"`
	SchemaVersion         string     `json:"schemaVersion"`
	ModelProvider         string     `json:"modelProvider"`
	ModelName             string     `json:"modelName"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type NextCallPlan struct {
	ID                         string     `json:"id"`
	PatientID                  string     `json:"patientId"`
	SourceAnalysisResultID     string     `json:"sourceAnalysisResultId"`
	CallTemplateID             string     `json:"callTemplateId"`
	CallTemplateSlug           string     `json:"callTemplateSlug,omitempty"`
	CallTemplateName           string     `json:"callTemplateName,omitempty"`
	CallType                   string     `json:"callType"`
	SuggestedTimeNote          string     `json:"suggestedTimeNote,omitempty"`
	SuggestedWindowStartAt     *time.Time `json:"suggestedWindowStartAt,omitempty"`
	SuggestedWindowEndAt       *time.Time `json:"suggestedWindowEndAt,omitempty"`
	PlannedFor                 *time.Time `json:"plannedFor,omitempty"`
	DurationMinutes            int        `json:"durationMinutes"`
	Goal                       string     `json:"goal"`
	FollowUpRequestedByPatient bool       `json:"followUpRequestedByPatient"`
	FollowUpEvidence           string     `json:"followUpEvidence,omitempty"`
	CaregiverReviewReason      string     `json:"caregiverReviewReason,omitempty"`
	ApprovalStatus             string     `json:"approvalStatus"`
	ApprovedByCaregiverID      string     `json:"approvedByCaregiverId,omitempty"`
	ApprovedByAdminUsername    string     `json:"approvedByAdminUsername,omitempty"`
	ApprovedAt                 *time.Time `json:"approvedAt,omitempty"`
	RejectionReason            string     `json:"rejectionReason,omitempty"`
	RejectedAt                 *time.Time `json:"rejectedAt,omitempty"`
	ExecutedCallRunID          string     `json:"executedCallRunId,omitempty"`
	CreatedAt                  time.Time  `json:"createdAt"`
	UpdatedAt                  time.Time  `json:"updatedAt"`
}

type DashboardSnapshot struct {
	Patient            Patient            `json:"patient"`
	Caregiver          Caregiver          `json:"caregiver"`
	Consent            ConsentState       `json:"consent"`
	ScreeningSchedule  *ScreeningSchedule `json:"screeningSchedule,omitempty"`
	LatestCall         *CallRun           `json:"latestCall,omitempty"`
	RecentCalls        []CallRun          `json:"recentCalls"`
	LatestAnalysis     *AnalysisRecord    `json:"latestAnalysis,omitempty"`
	ActiveNextCallPlan *NextCallPlan      `json:"activeNextCallPlan,omitempty"`
	RiskFlags          []RiskFlag         `json:"riskFlags"`
}

type PatientPerson struct {
	ID                      string    `json:"id"`
	PatientID               string    `json:"patientId"`
	Name                    string    `json:"name"`
	Relationship            string    `json:"relationship,omitempty"`
	Status                  string    `json:"status"`
	RelationshipQuality     string    `json:"relationshipQuality"`
	SafeToSuggestCall       bool      `json:"safeToSuggestCall"`
	FirstMentionedAt        time.Time `json:"firstMentionedAt"`
	FirstMentionedCallRunID string    `json:"firstMentionedCallRunId,omitempty"`
	LastMentionedAt         time.Time `json:"lastMentionedAt"`
	LastMentionedCallRunID  string    `json:"lastMentionedCallRunId,omitempty"`
	Context                 string    `json:"context,omitempty"`
	Notes                   string    `json:"notes,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type UpdatePatientPersonRequest struct {
	Name                string `json:"name"`
	Relationship        string `json:"relationship"`
	Status              string `json:"status"`
	RelationshipQuality string `json:"relationshipQuality"`
	Notes               string `json:"notes"`
}

type MemoryBankEntry struct {
	ID                     string          `json:"id"`
	PatientID              string          `json:"patientId"`
	SourceCallRunID        string          `json:"sourceCallRunId"`
	SourceAnalysisResultID string          `json:"sourceAnalysisResultId"`
	Topic                  string          `json:"topic"`
	Summary                string          `json:"summary"`
	EmotionalTone          string          `json:"emotionalTone,omitempty"`
	RespondedWellTo        []string        `json:"respondedWellTo"`
	AnchorOffered          bool            `json:"anchorOffered"`
	AnchorType             string          `json:"anchorType"`
	AnchorAccepted         bool            `json:"anchorAccepted"`
	AnchorDetail           string          `json:"anchorDetail,omitempty"`
	SuggestedFollowUp      string          `json:"suggestedFollowUp,omitempty"`
	OccurredAt             time.Time       `json:"occurredAt"`
	People                 []PatientPerson `json:"people"`
	CreatedAt              time.Time       `json:"createdAt"`
	UpdatedAt              time.Time       `json:"updatedAt"`
}

type Reminder struct {
	ID                           string         `json:"id"`
	PatientID                    string         `json:"patientId"`
	SourceCallRunID              string         `json:"sourceCallRunId,omitempty"`
	SourceAnalysisResultID       string         `json:"sourceAnalysisResultId,omitempty"`
	Kind                         string         `json:"kind"`
	Status                       string         `json:"status"`
	Title                        string         `json:"title"`
	Detail                       string         `json:"detail,omitempty"`
	PersonID                     string         `json:"personId,omitempty"`
	Person                       *PatientPerson `json:"person,omitempty"`
	CaregiverFollowUpRecommended bool           `json:"caregiverFollowUpRecommended"`
	SuggestedFor                 *time.Time     `json:"suggestedFor,omitempty"`
	CreatedBy                    string         `json:"createdBy"`
	CreatedAt                    time.Time      `json:"createdAt"`
	UpdatedAt                    time.Time      `json:"updatedAt"`
}

type CreateCaregiverRequest struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	PhoneE164   string `json:"phoneE164"`
	Timezone    string `json:"timezone"`
}

type UpdateCaregiverRequest = CreateCaregiverRequest

type CreatePatientRequest struct {
	PrimaryCaregiverID   string               `json:"primaryCaregiverId"`
	DisplayName          string               `json:"displayName"`
	PreferredName        string               `json:"preferredName"`
	PhoneE164            string               `json:"phoneE164"`
	Timezone             string               `json:"timezone"`
	Notes                string               `json:"notes"`
	RoutineAnchors       []string             `json:"routineAnchors"`
	FavoriteTopics       []string             `json:"favoriteTopics"`
	CalmingCues          []string             `json:"calmingCues"`
	TopicsToAvoid        []string             `json:"topicsToAvoid"`
	MemoryProfile        MemoryProfile        `json:"memoryProfile"`
	ConversationGuidance ConversationGuidance `json:"conversationGuidance"`
}

type UpdatePatientRequest = CreatePatientRequest

type UpdateConsentRequest struct {
	OutboundCallStatus      string `json:"outboundCallStatus"`
	TranscriptStorageStatus string `json:"transcriptStorageStatus"`
	Notes                   string `json:"notes"`
}

type PausePatientRequest struct {
	Reason string `json:"reason"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SessionResponse struct {
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type CreateCallRequest struct {
	CallTemplateID string `json:"callTemplateId"`
	CallType       string `json:"callType"`
	Channel        string `json:"channel"`
	TriggerType    string `json:"triggerType"`
}

type CreateCallResponse struct {
	CallRun      CallRun `json:"callRun"`
	VoiceSession any     `json:"voiceSession,omitempty"`
}

type CallRunDetail struct {
	CallRun         CallRun              `json:"callRun"`
	TranscriptTurns []CallTranscriptTurn `json:"transcriptTurns"`
	Analysis        *AnalysisRecord      `json:"analysis,omitempty"`
	AnalysisJob     *AnalysisJob         `json:"analysisJob,omitempty"`
}

type UpdateNextCallPlanRequest struct {
	Action            string `json:"action"`
	CallTemplateID    string `json:"callTemplateId"`
	SuggestedTimeNote string `json:"suggestedTimeNote"`
	PlannedFor        string `json:"plannedFor"`
	DurationMinutes   *int   `json:"durationMinutes"`
	Goal              string `json:"goal"`
	Reason            string `json:"reason"`
}

type AnalysisPromptContext struct {
	CallRun           CallRun              `json:"callRun"`
	Patient           Patient              `json:"patient"`
	Caregiver         Caregiver            `json:"caregiver"`
	CallTemplate      CallTemplate         `json:"callTemplate"`
	ScreeningSchedule *ScreeningSchedule   `json:"screeningSchedule,omitempty"`
	TranscriptTurns   []CallTranscriptTurn `json:"transcriptTurns"`
	RecentAnalyses    []AnalysisPayload    `json:"recentAnalyses"`
}

type CallPromptContext struct {
	Patient                 Patient           `json:"patient"`
	SafePeopleForCallAnchor []PatientPerson   `json:"safePeopleForCallAnchor"`
	PeopleToAvoidNaming     []PatientPerson   `json:"peopleToAvoidNaming"`
	RecentMemoryBankEntries []MemoryBankEntry `json:"recentMemoryBankEntries"`
}
