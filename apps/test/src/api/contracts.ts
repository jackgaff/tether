export interface ApiEnvelope<T> {
  data?: T;
  meta?: {
    count?: number;
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

export interface PatientPreference {
  patientId: string;
  defaultVoiceId: string;
  isConfigured: boolean;
  updatedAt?: string;
}

export interface UpdatePatientPreferenceInput {
  defaultVoiceId: string;
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
