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
	CallTypeOrientation  = "orientation"
	CallTypeReminder     = "reminder"
	CallTypeWellbeing    = "wellbeing"
	CallTypeReminiscence = "reminiscence"
)

const (
	CallChannelBrowser = "browser"
	CallChannelConnect = "connect"
)

const (
	CallTriggerManual           = "manual"
	CallTriggerApprovedNextCall = "approved_next_call"
)

const (
	CallRunStatusRequested  = "requested"
	CallRunStatusInProgress = "in_progress"
	CallRunStatusCompleted  = "completed"
	CallRunStatusFailed     = "failed"
	CallRunStatusCancelled  = "cancelled"
)

const (
	AnalysisOrientationGood    = "good"
	AnalysisOrientationMixed   = "mixed"
	AnalysisOrientationPoor    = "poor"
	AnalysisOrientationUnclear = "unclear"
)

const (
	AnalysisMoodPositive   = "positive"
	AnalysisMoodNeutral    = "neutral"
	AnalysisMoodAnxious    = "anxious"
	AnalysisMoodSad        = "sad"
	AnalysisMoodDistressed = "distressed"
	AnalysisMoodUnclear    = "unclear"
)

const (
	AnalysisEngagementHigh   = "high"
	AnalysisEngagementMedium = "medium"
	AnalysisEngagementLow    = "low"
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

const AnalysisSchemaVersion = "v1"

type Caregiver struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
	PhoneE164   string    `json:"phoneE164,omitempty"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Patient struct {
	ID                 string     `json:"id"`
	PrimaryCaregiverID string     `json:"primaryCaregiverId"`
	DisplayName        string     `json:"displayName"`
	PreferredName      string     `json:"preferredName"`
	PhoneE164          string     `json:"phoneE164,omitempty"`
	Timezone           string     `json:"timezone"`
	Notes              string     `json:"notes,omitempty"`
	CallingState       string     `json:"callingState"`
	PauseReason        string     `json:"pauseReason,omitempty"`
	PausedAt           *time.Time `json:"pausedAt,omitempty"`
	RoutineAnchors     []string   `json:"routineAnchors"`
	FavoriteTopics     []string   `json:"favoriteTopics"`
	CalmingCues        []string   `json:"calmingCues"`
	TopicsToAvoid      []string   `json:"topicsToAvoid"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
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
	ID                   string          `json:"id"`
	Slug                 string          `json:"slug"`
	DisplayName          string          `json:"displayName"`
	CallType             string          `json:"callType"`
	Description          string          `json:"description"`
	DurationMinutes      int             `json:"durationMinutes"`
	PromptVersion        string          `json:"promptVersion"`
	SystemPromptTemplate string          `json:"systemPromptTemplate"`
	Checklist            json.RawMessage `json:"checklist"`
	IsActive             bool            `json:"isActive"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
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
	RequestedAt          time.Time  `json:"requestedAt"`
	StartedAt            *time.Time `json:"startedAt,omitempty"`
	EndedAt              *time.Time `json:"endedAt,omitempty"`
	StopReason           string     `json:"stopReason,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type CallTranscriptTurn struct {
	SequenceNo int       `json:"sequenceNo"`
	Direction  string    `json:"direction"`
	Modality   string    `json:"modality"`
	Text       string    `json:"text"`
	OccurredAt time.Time `json:"occurredAt"`
	StopReason string    `json:"stopReason,omitempty"`
}

type AnalysisPatientState struct {
	Orientation string  `json:"orientation"`
	Mood        string  `json:"mood"`
	Engagement  string  `json:"engagement"`
	Confidence  float64 `json:"confidence"`
}

type AnalysisSignals struct {
	Repetition                  int      `json:"repetition"`
	RoutineAdherenceIssue       bool     `json:"routine_adherence_issue"`
	SleepConcern                bool     `json:"sleep_concern"`
	NutritionOrHydrationConcern bool     `json:"nutrition_or_hydration_concern"`
	PossibleSafetyConcern       bool     `json:"possible_safety_concern"`
	PossibleBPSDSignals         []string `json:"possible_bpsd_signals"`
	SocialConnectionNeed        bool     `json:"social_connection_need"`
}

type AnalysisEvidence struct {
	Quote        string `json:"quote"`
	WhyItMatters string `json:"why_it_matters"`
}

type RecommendedNextCall struct {
	Type            string `json:"type"`
	Timing          string `json:"timing"`
	DurationMinutes int    `json:"duration_minutes"`
	Goal            string `json:"goal"`
}

type AnalysisPayload struct {
	CallTypeCompleted   string               `json:"call_type_completed"`
	PatientState        AnalysisPatientState `json:"patient_state"`
	Signals             AnalysisSignals      `json:"signals"`
	Evidence            []AnalysisEvidence   `json:"evidence"`
	DashboardSummary    string               `json:"dashboard_summary"`
	CaregiverSummary    string               `json:"caregiver_summary"`
	RecommendedNextCall RecommendedNextCall  `json:"recommended_next_call"`
	EscalationLevel     string               `json:"escalation_level"`
	Uncertainties       []string             `json:"uncertainties"`
}

type RiskFlag struct {
	ID               string    `json:"id"`
	AnalysisResultID string    `json:"analysisResultId"`
	FlagType         string    `json:"flagType"`
	Severity         string    `json:"severity"`
	EvidenceQuote    string    `json:"evidenceQuote,omitempty"`
	WhyItMatters     string    `json:"whyItMatters,omitempty"`
	Confidence       float64   `json:"confidence"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AnalysisRecord struct {
	ID            string          `json:"id"`
	CallRunID     string          `json:"callRunId"`
	ModelID       string          `json:"modelId"`
	SchemaVersion string          `json:"schemaVersion"`
	Result        AnalysisPayload `json:"result"`
	RiskFlags     []RiskFlag      `json:"riskFlags"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type NextCallPlan struct {
	ID                      string     `json:"id"`
	PatientID               string     `json:"patientId"`
	SourceAnalysisResultID  string     `json:"sourceAnalysisResultId"`
	CallTemplateID          string     `json:"callTemplateId"`
	CallTemplateSlug        string     `json:"callTemplateSlug,omitempty"`
	CallTemplateName        string     `json:"callTemplateName,omitempty"`
	CallType                string     `json:"callType"`
	SuggestedTimeNote       string     `json:"suggestedTimeNote,omitempty"`
	PlannedFor              *time.Time `json:"plannedFor,omitempty"`
	DurationMinutes         int        `json:"durationMinutes"`
	Goal                    string     `json:"goal"`
	ApprovalStatus          string     `json:"approvalStatus"`
	ApprovedByCaregiverID   string     `json:"approvedByCaregiverId,omitempty"`
	ApprovedByAdminUsername string     `json:"approvedByAdminUsername,omitempty"`
	ApprovedAt              *time.Time `json:"approvedAt,omitempty"`
	RejectionReason         string     `json:"rejectionReason,omitempty"`
	RejectedAt              *time.Time `json:"rejectedAt,omitempty"`
	ExecutedCallRunID       string     `json:"executedCallRunId,omitempty"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

type DashboardSnapshot struct {
	Patient            Patient         `json:"patient"`
	Caregiver          Caregiver       `json:"caregiver"`
	Consent            ConsentState    `json:"consent"`
	LatestCall         *CallRun        `json:"latestCall,omitempty"`
	RecentCalls        []CallRun       `json:"recentCalls"`
	LatestAnalysis     *AnalysisRecord `json:"latestAnalysis,omitempty"`
	ActiveNextCallPlan *NextCallPlan   `json:"activeNextCallPlan,omitempty"`
	RiskFlags          []RiskFlag      `json:"riskFlags"`
}

type CreateCaregiverRequest struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	PhoneE164   string `json:"phoneE164"`
	Timezone    string `json:"timezone"`
}

type UpdateCaregiverRequest = CreateCaregiverRequest

type CreatePatientRequest struct {
	PrimaryCaregiverID string   `json:"primaryCaregiverId"`
	DisplayName        string   `json:"displayName"`
	PreferredName      string   `json:"preferredName"`
	PhoneE164          string   `json:"phoneE164"`
	Timezone           string   `json:"timezone"`
	Notes              string   `json:"notes"`
	RoutineAnchors     []string `json:"routineAnchors"`
	FavoriteTopics     []string `json:"favoriteTopics"`
	CalmingCues        []string `json:"calmingCues"`
	TopicsToAvoid      []string `json:"topicsToAvoid"`
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
	CallRun         CallRun              `json:"callRun"`
	Patient         Patient              `json:"patient"`
	Caregiver       Caregiver            `json:"caregiver"`
	CallTemplate    CallTemplate         `json:"callTemplate"`
	TranscriptTurns []CallTranscriptTurn `json:"transcriptTurns"`
	RecentAnalyses  []AnalysisPayload    `json:"recentAnalyses"`
}
