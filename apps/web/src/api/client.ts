import type {
  ApiEnvelope,
  CheckIn,
  CreateCheckInInput,
  HealthSnapshot
} from "./contracts";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export const apiBaseUrl = API_BASE_URL;

export async function fetchHealth(): Promise<HealthSnapshot> {
  const payload = await request<ApiEnvelope<HealthSnapshot>>("/health");

  return (
    payload.data ?? {
      status: "unknown",
      service: "nova-echoes-api",
      env: "development",
      authMode: "off",
      databaseURLConfigured: false,
      time: new Date().toISOString()
    }
  );
}

export async function fetchCheckIns(): Promise<CheckIn[]> {
  const payload = await request<ApiEnvelope<CheckIn[]>>("/api/v1/check-ins");
  return payload.data ?? [];
}

export async function createDemoCheckIn(): Promise<CheckIn> {
  return createCheckIn({
    patientId: "patient-001",
    summary:
      "Scheduled voice check-in completed. The caller remembered breakfast, tomorrow's ride, and where the medication card is stored.",
    status: "completed",
    agent: "analysis-agent",
    reminder: "Place tomorrow's appointment card beside the front door."
  });
}

export async function createCheckIn(input: CreateCheckInInput): Promise<CheckIn> {
  const payload = await request<ApiEnvelope<CheckIn>>("/api/v1/check-ins", {
    method: "POST",
    body: JSON.stringify(input)
  });

  if (!payload.data) {
    throw new Error("The API returned an empty response.");
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
