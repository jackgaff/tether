import type {
  ApiEnvelope,
  CreateVoiceSessionInput,
  HealthSnapshot,
  LabConversation,
  VoiceOption,
  VoiceSessionDescriptor
} from "./contracts";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export const apiBaseUrl = API_BASE_URL;

export function buildVoiceWebSocketUrl(path: string, token: string): string {
  const base = new URL(API_BASE_URL);
  base.protocol = base.protocol === "https:" ? "wss:" : "ws:";
  base.pathname = path;
  base.search = "";
  base.searchParams.set("token", token);
  return base.toString();
}

export async function fetchHealth(): Promise<HealthSnapshot> {
  const payload = await request<ApiEnvelope<HealthSnapshot>>("/health");
  return (
    payload.data ?? {
      status: "unknown",
      service: "tether-api",
      env: "development",
      authMode: "off",
      databaseURLConfigured: false,
      time: new Date().toISOString()
    }
  );
}

export async function fetchVoices(): Promise<VoiceOption[]> {
  const payload = await request<ApiEnvelope<VoiceOption[]>>("/api/v1/voice/voices");
  return payload.data ?? [];
}

export async function fetchLabConversations(limit = 20): Promise<LabConversation[]> {
  const payload = await request<ApiEnvelope<LabConversation[]>>(
    `/api/v1/voice/lab/conversations?limit=${encodeURIComponent(String(limit))}`
  );
  return payload.data ?? [];
}

export async function createVoiceSession(
  input: CreateVoiceSessionInput
): Promise<VoiceSessionDescriptor> {
  const payload = await request<ApiEnvelope<VoiceSessionDescriptor>>(
    "/api/v1/voice/sessions",
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );

  if (!payload.data) {
    throw new Error("The API returned an empty voice session response.");
  }

  return payload.data;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);

  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers
  });

  const contentType = response.headers.get("content-type") ?? "";
  const isJson = contentType.includes("application/json");
  const payload = isJson ? ((await response.json()) as ApiEnvelope<unknown>) : undefined;

  if (!response.ok) {
    const message =
      payload?.error?.message ??
      (isJson ? JSON.stringify(payload) : await response.text()) ??
      `Request failed with ${response.status}`;

    throw new Error(message);
  }

  return (payload as T) ?? ({} as T);
}
