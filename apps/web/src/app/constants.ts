export const STORAGE_KEYS = {
  caregiverId: "nova-echoes.minimal-admin.caregiver-id",
  patientId: "nova-echoes.minimal-admin.patient-id",
  callId: "nova-echoes.minimal-admin.call-id"
} as const;

export const CALL_TYPES = [
  "orientation",
  "reminder",
  "wellbeing",
  "reminiscence"
] as const;

export const CONSENT_STATUSES = ["pending", "granted", "revoked"] as const;

export const NEXT_CALL_ACTIONS = ["approve", "edit", "reject", "cancel"] as const;
