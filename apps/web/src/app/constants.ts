export const STORAGE_KEYS = {
  caregiverId: "tether.minimal-admin.caregiver-id",
  patientId: "tether.minimal-admin.patient-id",
  callId: "tether.minimal-admin.call-id"
} as const;

export const CALL_TYPES = [
  "orientation",
  "reminder",
  "wellbeing",
  "reminiscence"
] as const;

export const CONSENT_STATUSES = ["pending", "granted", "revoked"] as const;

export const NEXT_CALL_ACTIONS = ["approve", "edit", "reject", "cancel"] as const;
