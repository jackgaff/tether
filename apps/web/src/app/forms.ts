import type {
  CaregiverInput,
  ConsentInput,
  NextCallPlan,
  Patient,
  PatientInput,
  UpdateNextCallInput
} from "../api/contracts";

export type PatientFormState = {
  primaryCaregiverId: string;
  displayName: string;
  preferredName: string;
  phoneE164: string;
  timezone: string;
  notes: string;
  routineAnchors: string;
  favoriteTopics: string;
  calmingCues: string;
  topicsToAvoid: string;
};

export type NextCallFormState = {
  action: "approve" | "edit" | "reject" | "cancel";
  callTemplateId: string;
  suggestedTimeNote: string;
  plannedFor: string;
  durationMinutes: string;
  goal: string;
  reason: string;
};

export function createCaregiverForm(defaultTimezone: string): CaregiverInput {
  return {
    displayName: "",
    email: "",
    phoneE164: "",
    timezone: defaultTimezone
  };
}

export function createPatientForm(
  defaultTimezone: string,
  caregiverId = ""
): PatientFormState {
  return {
    primaryCaregiverId: caregiverId,
    displayName: "",
    preferredName: "",
    phoneE164: "",
    timezone: defaultTimezone,
    notes: "",
    routineAnchors: "",
    favoriteTopics: "",
    calmingCues: "",
    topicsToAvoid: ""
  };
}

export function createConsentForm(): ConsentInput {
  return {
    outboundCallStatus: "pending",
    transcriptStorageStatus: "pending",
    notes: ""
  };
}

export function createNextCallForm(nextPlan?: NextCallPlan | null): NextCallFormState {
  if (!nextPlan) {
    return {
      action: "approve",
      callTemplateId: "",
      suggestedTimeNote: "",
      plannedFor: "",
      durationMinutes: "",
      goal: "",
      reason: ""
    };
  }

  return {
    action: "approve",
    callTemplateId: nextPlan.callTemplateId,
    suggestedTimeNote: nextPlan.suggestedTimeNote ?? "",
    plannedFor: nextPlan.plannedFor ?? "",
    durationMinutes: String(nextPlan.durationMinutes),
    goal: nextPlan.goal,
    reason: ""
  };
}

export function patientToForm(patient: Patient): PatientFormState {
  return {
    primaryCaregiverId: patient.primaryCaregiverId,
    displayName: patient.displayName,
    preferredName: patient.preferredName,
    phoneE164: patient.phoneE164 ?? "",
    timezone: patient.timezone,
    notes: patient.notes ?? "",
    routineAnchors: joinList(patient.routineAnchors),
    favoriteTopics: joinList(patient.favoriteTopics),
    calmingCues: joinList(patient.calmingCues),
    topicsToAvoid: joinList(patient.topicsToAvoid)
  };
}

export function patientFormToInput(form: PatientFormState): PatientInput {
  return {
    primaryCaregiverId: form.primaryCaregiverId.trim(),
    displayName: form.displayName.trim(),
    preferredName: form.preferredName.trim(),
    phoneE164: form.phoneE164.trim(),
    timezone: form.timezone.trim(),
    notes: form.notes.trim(),
    routineAnchors: parseList(form.routineAnchors),
    favoriteTopics: parseList(form.favoriteTopics),
    calmingCues: parseList(form.calmingCues),
    topicsToAvoid: parseList(form.topicsToAvoid)
  };
}

export function buildNextCallInput(form: NextCallFormState): UpdateNextCallInput | Error {
  if (form.action === "approve") {
    return { action: "approve" };
  }

  if (form.action === "edit") {
    const input: UpdateNextCallInput = { action: "edit" };

    if (form.callTemplateId.trim()) {
      input.callTemplateId = form.callTemplateId.trim();
    }
    if (form.suggestedTimeNote.trim()) {
      input.suggestedTimeNote = form.suggestedTimeNote.trim();
    }
    if (form.plannedFor.trim()) {
      input.plannedFor = form.plannedFor.trim();
    }
    if (form.durationMinutes.trim()) {
      const parsed = Number.parseInt(form.durationMinutes.trim(), 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        return new Error("Duration minutes must be a positive integer.");
      }
      input.durationMinutes = parsed;
    }
    if (form.goal.trim()) {
      input.goal = form.goal.trim();
    }
    if (form.reason.trim()) {
      input.reason = form.reason.trim();
    }

    return input;
  }

  if (form.reason.trim()) {
    return {
      action: form.action,
      reason: form.reason.trim()
    };
  }

  return {
    action: form.action
  };
}

export function formatError(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong.";
}

export function getDefaultTimezone(): string {
  if (typeof Intl !== "undefined") {
    const resolved = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (resolved) {
      return resolved;
    }
  }

  return "America/Detroit";
}

function parseList(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinList(values: string[]): string {
  return values.join(", ");
}
