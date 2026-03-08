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

export interface CheckIn {
  id: string;
  patientId: string;
  summary: string;
  status: "scheduled" | "completed" | "needs_follow_up";
  agent: string;
  reminder?: string;
  recordedAt: string;
}

export interface CreateCheckInInput {
  patientId: string;
  summary: string;
  status: CheckIn["status"];
  agent: string;
  reminder?: string;
}
