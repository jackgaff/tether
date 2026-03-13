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
  callType: "orientation" | "reminder" | "wellbeing" | "reminiscence";
  description: string;
  durationMinutes: number;
  promptVersion: string;
  systemPromptTemplate: string;
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
  callType: "orientation" | "reminder" | "wellbeing" | "reminiscence";
  channel: "browser" | "connect";
  triggerType: "manual" | "approved_next_call";
  status: "requested" | "in_progress" | "completed" | "failed" | "cancelled";
  sourceVoiceSessionId?: string;
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
  modality: "audio" | "text";
  text: string;
  occurredAt: string;
  stopReason?: string;
}

export interface AnalysisPatientState {
  orientation: "good" | "mixed" | "poor" | "unclear";
  mood: "positive" | "neutral" | "anxious" | "sad" | "distressed" | "unclear";
  engagement: "high" | "medium" | "low";
  confidence: number;
}

export interface AnalysisSignals {
  repetition: number;
  routine_adherence_issue: boolean;
  sleep_concern: boolean;
  nutrition_or_hydration_concern: boolean;
  possible_safety_concern: boolean;
  possible_bpsd_signals: string[];
  social_connection_need: boolean;
}

export interface AnalysisEvidence {
  quote: string;
  why_it_matters: string;
}

export interface RecommendedNextCall {
  type: "orientation" | "reminder" | "wellbeing" | "reminiscence";
  timing: string;
  duration_minutes: number;
  goal: string;
}

export interface AnalysisPayload {
  call_type_completed: "orientation" | "reminder" | "wellbeing" | "reminiscence";
  patient_state: AnalysisPatientState;
  signals: AnalysisSignals;
  evidence: AnalysisEvidence[];
  dashboard_summary: string;
  caregiver_summary: string;
  recommended_next_call: RecommendedNextCall;
  escalation_level: "none" | "caregiver_soon" | "caregiver_now" | "clinical_review";
  uncertainties: string[];
}

export interface RiskFlag {
  id: string;
  analysisResultId: string;
  flagType: string;
  severity: "info" | "watch" | "urgent";
  evidenceQuote?: string;
  whyItMatters?: string;
  confidence: number;
  createdAt: string;
}

export interface AnalysisRecord {
  id: string;
  callRunId: string;
  modelId: string;
  schemaVersion: string;
  result: AnalysisPayload;
  riskFlags: RiskFlag[];
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
  callType: "orientation" | "reminder" | "wellbeing" | "reminiscence";
  suggestedTimeNote?: string;
  plannedFor?: string;
  durationMinutes: number;
  goal: string;
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

export interface DashboardSnapshot {
  patient: Patient;
  caregiver: Caregiver;
  consent: ConsentState;
  latestCall?: CallRun;
  recentCalls: CallRun[];
  latestAnalysis?: AnalysisRecord;
  activeNextCallPlan?: NextCallPlan;
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
  callType?: CallRun["callType"];
  channel: "browser";
  triggerType?: CallRun["triggerType"];
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
