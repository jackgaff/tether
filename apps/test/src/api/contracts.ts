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

export interface VoiceOption {
  id: string;
  displayName: string;
  locale: string;
  polyglot: boolean;
  isDefault: boolean;
  browserSupported: boolean;
  connectNativeSupported: boolean;
}

export interface AudioConfig {
  encoding: string;
  sampleRateHz: 8000 | 16000 | 24000;
  channels: number;
}

export interface CreateVoiceSessionInput {
  patientId: string;
  voiceId?: string;
  systemPrompt?: string;
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

export interface ArtifactPaths {
  jsonPath?: string;
  markdownPath?: string;
}

export interface LabConversationTurn {
  sequenceNo: number;
  direction: "user" | "assistant";
  modality: "audio" | "text";
  text: string;
  occurredAt: string;
  stopReason?: string;
}

export interface LabConversation {
  id: string;
  voiceId: string;
  status: string;
  systemPrompt?: string;
  stopReason?: string;
  createdAt: string;
  endedAt: string;
  jsonPath?: string;
  markdownPath?: string;
  turns: LabConversationTurn[];
}
