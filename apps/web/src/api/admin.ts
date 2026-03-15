import { request } from "./client";
import type {
  AdminSession,
  AnalysisRecord,
  CallRunDetail,
  CallTemplate,
  Caregiver,
  CaregiverInput,
  ConsentInput,
  ConsentState,
  CreateCallResponse,
  CreatePatientCallInput,
  DashboardSnapshot,
  LoginAdminInput,
  NextCallPlan,
  PausePatientInput,
  Patient,
  PatientInput,
  UpdateNextCallInput
} from "./contracts";

export function loginAdmin(input: LoginAdminInput): Promise<AdminSession> {
  return request<AdminSession>("/api/v1/admin/session/login", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function getAdminSession(): Promise<AdminSession> {
  return request<AdminSession>("/api/v1/admin/session");
}

export function logoutAdmin(): Promise<{ status: "logged_out" }> {
  return request<{ status: "logged_out" }>("/api/v1/admin/session/logout", {
    method: "POST"
  });
}

export function createCaregiver(input: CaregiverInput): Promise<Caregiver> {
  return request<Caregiver>("/api/v1/admin/caregivers", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function getCaregiver(id: string): Promise<Caregiver> {
  return request<Caregiver>(`/api/v1/admin/caregivers/${encodeURIComponent(id)}`);
}

export function updateCaregiver(id: string, input: CaregiverInput): Promise<Caregiver> {
  return request<Caregiver>(`/api/v1/admin/caregivers/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export function listPatients(): Promise<Patient[]> {
  return request<Patient[]>("/api/v1/admin/patients");
}

export function createPatient(input: PatientInput): Promise<Patient> {
  return request<Patient>("/api/v1/admin/patients", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function getPatient(id: string): Promise<Patient> {
  return request<Patient>(`/api/v1/admin/patients/${encodeURIComponent(id)}`);
}

export function updatePatient(id: string, input: PatientInput): Promise<Patient> {
  return request<Patient>(`/api/v1/admin/patients/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export function getConsent(patientId: string): Promise<ConsentState> {
  return request<ConsentState>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/consent`
  );
}

export function updateConsent(patientId: string, input: ConsentInput): Promise<ConsentState> {
  return request<ConsentState>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/consent`,
    {
      method: "PUT",
      body: JSON.stringify(input)
    }
  );
}

export function pausePatient(patientId: string, input: PausePatientInput): Promise<Patient> {
  return request<Patient>(`/api/v1/admin/patients/${encodeURIComponent(patientId)}/pause`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function unpausePatient(patientId: string): Promise<Patient> {
  return request<Patient>(`/api/v1/admin/patients/${encodeURIComponent(patientId)}/pause`, {
    method: "DELETE"
  });
}

export function listCallTemplates(): Promise<CallTemplate[]> {
  return request<CallTemplate[]>("/api/v1/admin/call-templates");
}

export function getDashboard(patientId: string): Promise<DashboardSnapshot> {
  return request<DashboardSnapshot>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/dashboard`
  );
}

export function createPatientCall(
  patientId: string,
  input: CreatePatientCallInput
): Promise<CreateCallResponse> {
  return request<CreateCallResponse>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/calls`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function getCall(callId: string): Promise<CallRunDetail> {
  return request<CallRunDetail>(`/api/v1/admin/calls/${encodeURIComponent(callId)}`);
}

export function analyzeCall(callId: string): Promise<AnalysisRecord> {
  return request<AnalysisRecord>(
    `/api/v1/admin/calls/${encodeURIComponent(callId)}/analyze`,
    {
      method: "POST"
    }
  );
}

export function getCallAnalysis(callId: string): Promise<AnalysisRecord> {
  return request<AnalysisRecord>(
    `/api/v1/admin/calls/${encodeURIComponent(callId)}/analysis`
  );
}

export function getNextCall(patientId: string): Promise<NextCallPlan> {
  return request<NextCallPlan>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/next-call`
  );
}

export function updateNextCall(
  patientId: string,
  input: UpdateNextCallInput
): Promise<NextCallPlan> {
  return request<NextCallPlan>(
    `/api/v1/admin/patients/${encodeURIComponent(patientId)}/next-call`,
    {
      method: "PUT",
      body: JSON.stringify(input)
    }
  );
}
