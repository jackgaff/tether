export interface ApiEnvelope<T> {
  data?: T;
  meta?: {
    count?: number;
    limit?: number;
  };
  error?: {
    code: string;
    message: string;
  };
}

export type CallType = "screening" | "check_in" | "reminiscence";
export type LegacyCallType = CallType | "orientation" | "reminder" | "wellbeing";
export type CallTriggerType =
  | "caregiver_requested"
  | "scheduled"
  | "follow_up_recommendation";
export type LegacyCallTriggerType = CallTriggerType | "manual" | "approved_next_call";
export type CallRunStatus =
  | "scheduled"
  | "requested"
  | "in_progress"
  | "completed"
  | "failed"
  | "cancelled";
export type AnalysisJobStatus = "pending" | "running" | "succeeded" | "failed";
export type TimeframeBucket =
  | "same_day"
  | "tomorrow"
  | "few_days"
  | "next_week"
  | "two_weeks"
  | "unspecified";

export interface HealthSnapshot {
  status: string;
  service: string;
  env: string;
  authMode: string;
  databaseURLConfigured?: boolean;
  time: string;
}

export interface AdminSession {
  username: string;
  expiresAt: string;
}

export interface Caregiver {
  id: string;
  displayName: string;
  email: string;
  phoneE164?: string;
  timezone: string;
  createdAt: string;
  updatedAt: string;
}

export interface FamilyMember {
  name: string;
  relation: string;
  notes?: string;
}

export interface LifeEvent {
  label: string;
  approximateDate?: string;
  notes?: string;
}

export interface MemoryProfile {
  likes: string[];
  familyMembers: FamilyMember[];
  lifeEvents: LifeEvent[];
  reminiscenceNotes?: string;
  significantPlaces: string[];
  lifeChapters: string[];
  favoriteMusic: string[];
  favoriteShowsFilms: string[];
  topicsToRevisit: string[];
}

export interface ConversationGuidance {
  preferredGreetingStyle?: string;
  calmingTopics: string[];
  upsettingTopics: string[];
  hearingOrPacingNotes?: string;
  bestTimeOfDay?: string;
  doNotMention: string[];
}

export interface Patient {
  id: string;
  primaryCaregiverId: string;
  displayName: string;
  preferredName: string;
  phoneE164?: string;
  timezone: string;
  notes?: string;
  callingState: "active" | "paused";
  pauseReason?: string;
  pausedAt?: string;
  routineAnchors: string[];
  favoriteTopics: string[];
  calmingCues: string[];
  topicsToAvoid: string[];
  memoryProfile: MemoryProfile;
  conversationGuidance: ConversationGuidance;
  createdAt: string;
  updatedAt: string;
}

export interface ScreeningSchedule {
  patientId: string;
  enabled: boolean;
  cadence: "weekly" | "biweekly";
  timezone: string;
  preferredWeekday: number;
  preferredLocalTime: string;
  nextDueAt?: string;
  lastScheduledWindowStart?: string;
  lastScheduledWindowEnd?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ConsentState {
  patientId: string;
  outboundCallStatus: "pending" | "granted" | "revoked";
  transcriptStorageStatus: "pending" | "granted" | "revoked";
  grantedByCaregiverId?: string;
  grantedAt?: string;
  revokedAt?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CallTemplate {
  id: string;
  slug: string;
  displayName: string;
  callType: CallType;
  description: string;
  durationMinutes: number;
  promptVersion: string;
  callPromptVersion: string;
  systemPromptTemplate: string;
  analysisPromptVersion: string;
  analysisPromptTemplate: string;
  checklist: unknown;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AudioConfig {
  encoding: string;
  sampleRateHz: 8000 | 16000 | 24000;
  channels: number;
}

export interface VoiceSessionDescriptor {
  id: string;
  voiceId: string;
  websocketPath: string;
  streamToken: string;
  streamTokenExpiresAt: string;
  audioInput: AudioConfig;
  audioOutput: AudioConfig;
  drainSeconds: number;
  maxSessionSeconds: number;
}

export interface CallRun {
  id: string;
  patientId: string;
  caregiverId: string;
  callTemplateId: string;
  callTemplateSlug?: string;
  callTemplateName?: string;
  callType: CallType;
  channel: "browser" | "connect";
  triggerType: CallTriggerType;
  status: CallRunStatus;
  sourceVoiceSessionId?: string;
  scheduleWindowStart?: string;
  scheduleWindowEnd?: string;
  requestedAt: string;
  startedAt?: string;
  endedAt?: string;
  stopReason?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CallTranscriptTurn {
  sequenceNo: number;
  direction: "user" | "assistant";
  speakerRole?: "patient" | "agent" | "caregiver" | "system";
  modality: "audio" | "text";
  text: string;
  occurredAt: string;
  stopReason?: string;
}

export interface SalientEvidence {
  quote: string;
  reason: string;
}

export interface StructuredRiskFlag {
  flagType: string;
  severity: "info" | "watch" | "urgent";
  evidence?: string;
  reason?: string;
  whyItMatters?: string;
  confidence: number;
}

export interface FollowUpIntent {
  requestedByPatient: boolean;
  timeframeBucket: TimeframeBucket;
  evidence?: string;
  confidence: number;
}

export interface NextCallRecommendation {
  callType: CallType;
  windowBucket: TimeframeBucket;
  goal: string;
}

export interface ScreeningAnalysis {
  screeningItemsAdministered: string[];
  screeningCompletionStatus: "complete" | "partial" | "aborted";
  screeningScoreRaw?: string;
  screeningScoreInterpretation?:
    | "routine_follow_up"
    | "caregiver_review_suggested"
    | "clinical_review_suggested"
    | "incomplete";
  screeningFlags: string[];
  suggestedRescreenWindowBucket?: TimeframeBucket;
}

export interface ReminderNote {
  title: string;
  detail?: string;
}

export interface CheckInAnalysis {
  orientationStatus: "oriented" | "mildly_confused" | "disoriented" | "unknown";
  orientationNotes?: string;
  mealsStatus: "reported" | "uncertain" | "not_recalled";
  mealsDetail?: string;
  fluidsStatus: "reported" | "uncertain" | "not_recalled";
  fluidsDetail?: string;
  activityDetail?: string;
  socialContact: "yes" | "no" | "unknown";
  socialContactDetail?: string;
  remindersNoted: ReminderNote[];
  reminderDeclined: boolean;
  reminderDeclinedTopic?: string;
  mood: "calm" | "withdrawn" | "distressed" | "elevated" | "unknown";
  moodNotes?: string;
  sleep: "good" | "poor" | "reversed" | "unknown";
  sleepNotes?: string;
  memoryFlags: string[];
  deliriumWatch: boolean;
  deliriumWatchNotes?: string;
  deliriumPotentialTriggers: string[];
  caregiverSummary?: string;
}

export interface MentionedPerson {
  name: string;
  relationship?: string;
  context?: string;
}

export interface ReminiscenceAnalysis {
  topic?: string;
  mentionedPeople: MentionedPerson[];
  mentionedPlaces: string[];
  mentionedMusic: string[];
  mentionedShowsFilms: string[];
  lifeChapters: string[];
  summary?: string;
  emotionalTone?: string;
  respondedWellTo: string[];
  anchorOffered: boolean;
  anchorType?: "call" | "music" | "show_film" | "journal" | "none";
  anchorAccepted: boolean;
  anchorDetail?: string;
  suggestedFollowUp?: string;
  caregiverSummary?: string;
}

export interface AnalysisPayload {
  summary: string;
  salientEvidence: SalientEvidence[];
  riskFlags: StructuredRiskFlag[];
  escalationLevel: "none" | "caregiver_soon" | "caregiver_now" | "clinical_review";
  caregiverReviewReason?: string;
  followUpIntent: FollowUpIntent;
  nextCallRecommendation?: NextCallRecommendation;
  screening?: ScreeningAnalysis;
  checkIn?: CheckInAnalysis;
  reminiscence?: ReminiscenceAnalysis;
  dashboard_summary?: string;
  caregiver_summary?: string;
  patient_state?: {
    orientation: string;
    mood: string;
    engagement: string;
    confidence: number;
  };
}

export interface RiskFlag {
  id: string;
  analysisResultId: string;
  flagType: string;
  severity: "info" | "watch" | "urgent";
  evidence?: string;
  reason?: string;
  whyItMatters?: string;
  confidence: number;
  createdAt: string;
}

export interface AnalysisRecord {
  id: string;
  callRunId: string;
  callTemplateId?: string;
  modelId: string;
  modelProvider: string;
  modelName: string;
  callPromptVersion: string;
  analysisPromptVersion: string;
  schemaVersion: string;
  generatedAt: string;
  result: AnalysisPayload;
  riskFlags: RiskFlag[];
  createdAt: string;
  updatedAt: string;
}

export interface AnalysisJob {
  id: string;
  callRunId: string;
  status: AnalysisJobStatus;
  attemptCount: number;
  lastError?: string;
  lockedAt?: string;
  startedAt?: string;
  finishedAt?: string;
  analysisPromptVersion: string;
  schemaVersion: string;
  modelProvider: string;
  modelName: string;
  createdAt: string;
  updatedAt: string;
}

export interface NextCallPlan {
  id: string;
  patientId: string;
  sourceAnalysisResultId: string;
  callTemplateId: string;
  callTemplateSlug?: string;
  callTemplateName?: string;
  callType: CallType;
  suggestedTimeNote?: string;
  suggestedWindowStartAt?: string;
  suggestedWindowEndAt?: string;
  plannedFor?: string;
  durationMinutes: number;
  goal: string;
  followUpRequestedByPatient: boolean;
  followUpEvidence?: string;
  caregiverReviewReason?: string;
  approvalStatus:
    | "pending_approval"
    | "approved"
    | "rejected"
    | "executed"
    | "superseded"
    | "cancelled";
  approvedByCaregiverId?: string;
  approvedByAdminUsername?: string;
  approvedAt?: string;
  rejectionReason?: string;
  rejectedAt?: string;
  executedCallRunId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PatientPerson {
  id: string;
  patientId: string;
  name: string;
  relationship?: string;
  status: string;
  relationshipQuality: string;
  safeToSuggestCall: boolean;
  firstMentionedAt: string;
  firstMentionedCallRunId?: string;
  lastMentionedAt: string;
  lastMentionedCallRunId?: string;
  context?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface MemoryBankEntry {
  id: string;
  patientId: string;
  sourceCallRunId: string;
  sourceAnalysisResultId: string;
  topic: string;
  summary: string;
  emotionalTone?: string;
  respondedWellTo: string[];
  anchorOffered: boolean;
  anchorType: string;
  anchorAccepted: boolean;
  anchorDetail?: string;
  suggestedFollowUp?: string;
  occurredAt: string;
  people: PatientPerson[];
  createdAt: string;
  updatedAt: string;
}

export interface DashboardSnapshot {
  patient: Patient;
  caregiver: Caregiver;
  consent: ConsentState;
  screeningSchedule?: ScreeningSchedule;
  latestCall?: CallRun;
  recentCalls: CallRun[];
  latestAnalysis?: AnalysisRecord;
  activeNextCallPlan?: NextCallPlan;
  patientPeople: PatientPerson[];
  recentMemoryBankEntries: MemoryBankEntry[];
  riskFlags: RiskFlag[];
}

export interface CreateCallResponse {
  callRun: CallRun;
  voiceSession?: VoiceSessionDescriptor;
}

export interface CallRunDetail {
  callRun: CallRun;
  transcriptTurns: CallTranscriptTurn[];
  analysis?: AnalysisRecord;
  analysisJob?: AnalysisJob;
}

export interface LoginAdminInput {
  username: string;
  password: string;
}

export interface CaregiverInput {
  displayName: string;
  email: string;
  phoneE164: string;
  timezone: string;
}

export interface PatientInput {
  primaryCaregiverId: string;
  displayName: string;
  preferredName: string;
  phoneE164: string;
  timezone: string;
  notes: string;
  routineAnchors: string[];
  favoriteTopics: string[];
  calmingCues: string[];
  topicsToAvoid: string[];
  memoryProfile?: MemoryProfile;
  conversationGuidance?: ConversationGuidance;
}

export interface ScreeningScheduleInput {
  enabled: boolean;
  cadence: ScreeningSchedule["cadence"];
  timezone: string;
  preferredWeekday: number;
  preferredLocalTime: string;
}

export interface ConsentInput {
  outboundCallStatus: ConsentState["outboundCallStatus"];
  transcriptStorageStatus: ConsentState["transcriptStorageStatus"];
  notes: string;
}

export interface PausePatientInput {
  reason: string;
}

export interface CreatePatientCallInput {
  callTemplateId?: string;
  callType?: LegacyCallType;
  channel: "browser" | "connect";
  triggerType?: LegacyCallTriggerType;
}

export interface UpdateNextCallInput {
  action: "approve" | "edit" | "reject" | "cancel";
  callTemplateId?: string;
  suggestedTimeNote?: string;
  plannedFor?: string;
  durationMinutes?: number;
  goal?: string;
  reason?: string;
}
