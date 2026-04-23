import type { ApiEnvelope, HealthSnapshot } from "./contracts";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export const apiBaseUrl = API_BASE_URL;

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function buildVoiceWebSocketUrl(path: string, token: string): string {
  const base = new URL(API_BASE_URL);
  base.protocol = base.protocol === "https:" ? "wss:" : "ws:";
  base.pathname = path;
  base.search = "";
  base.searchParams.set("token", token);
  return base.toString();
}

export async function fetchHealth(): Promise<HealthSnapshot> {
  const payload = await requestEnvelope<HealthSnapshot>("/health");

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

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const payload = await requestEnvelope<T>(path, init);

  if (payload.data === undefined) {
    throw new Error("The API returned an empty response.");
  }

  return payload.data;
}

async function requestEnvelope<T>(path: string, init?: RequestInit): Promise<ApiEnvelope<T>> {
  const headers = new Headers(init?.headers);

  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers
  });

  const contentType = response.headers.get("content-type") ?? "";
  const isJson = contentType.includes("application/json");
  const payload = isJson ? ((await response.json()) as ApiEnvelope<T>) : undefined;

  if (!response.ok) {
    const message =
      payload?.error?.message ??
      (isJson ? JSON.stringify(payload) : await response.text()) ??
      `Request failed with ${response.status}`;

    throw new ApiError(message, response.status, payload?.error?.code);
  }

  return payload ?? {};
}
