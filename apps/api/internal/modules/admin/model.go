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
	CallChannelBrowser = "browser"
	CallChannelConnect = "connect"
)

const (
	CallTriggerCaregiverRequested     = "caregiver_requested"
	CallTriggerScheduled              = "scheduled"
	CallTriggerFollowUpRecommendation = "follow_up_recommendation"
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
	Likes             []string       `json:"likes"`
	FamilyMembers     []FamilyMember `json:"familyMembers"`
	LifeEvents        []LifeEvent    `json:"lifeEvents"`
	ReminiscenceNotes string         `json:"reminiscenceNotes,omitempty"`
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
	FlagType   string  `json:"flagType"`
	Severity   string  `json:"severity"`
	Evidence   string  `json:"evidence,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Confidence float64 `json:"confidence"`
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

type CheckInAnalysis struct {
	ReportedDayOverview     string   `json:"reportedDayOverview,omitempty"`
	FoodAndHydration        string   `json:"foodAndHydration,omitempty"`
	MedicationMentions      []string `json:"medicationMentions"`
	MoodSignals             []string `json:"moodSignals"`
	RoutineAdherence        string   `json:"routineAdherence,omitempty"`
	SocialContactMentions   []string `json:"socialContactMentions"`
	FollowUpRequestDetected bool     `json:"followUpRequestDetected"`
}

type ReminiscenceAnalysis struct {
	TopicsDiscussed              []string `json:"topicsDiscussed"`
	PeopleMentioned              []string `json:"peopleMentioned"`
	PositiveEngagementSignals    []string `json:"positiveEngagementSignals"`
	DistressOrTriggerSignals     []string `json:"distressOrTriggerSignals"`
	FutureReminiscenceCandidates []string `json:"futureReminiscenceCandidates"`
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
}

type RiskFlag struct {
	ID               string    `json:"id"`
	AnalysisResultID string    `json:"analysisResultId"`
	FlagType         string    `json:"flagType"`
	Severity         string    `json:"severity"`
	Evidence         string    `json:"evidence,omitempty"`
	Reason           string    `json:"reason,omitempty"`
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
