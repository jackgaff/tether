import type { FamilyMember, Patient, PatientInput } from "../api/contracts";

export interface PatientProfileFormValues {
  displayName: string;
  preferredName: string;
  phoneE164: string;
  timezone: string;
  notes: string;
  profilePhotoDataUrl: string;
  routineAnchors: string;
  favoriteTopics: string;
  calmingCues: string;
  topicsToAvoid: string;
  likes: string;
  significantPlaces: string;
  lifeChapters: string;
  favoriteMusic: string;
  favoriteShowsFilms: string;
  topicsToRevisit: string;
  reminiscenceNotes: string;
  familyMembers: FamilyMember[];
  preferredGreetingStyle: string;
  calmingTopics: string;
  upsettingTopics: string;
  hearingOrPacingNotes: string;
  bestTimeOfDay: string;
  doNotMention: string;
}

export const COMMON_TIMEZONES = [
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Anchorage",
  "Pacific/Honolulu",
  "Europe/London",
  "Europe/Paris",
  "Australia/Sydney"
];

export function createEmptyPatientProfile(defaultTimezone: string): PatientProfileFormValues {
  return {
    displayName: "",
    preferredName: "",
    phoneE164: "",
    timezone: defaultTimezone,
    notes: "",
    profilePhotoDataUrl: "",
    routineAnchors: "",
    favoriteTopics: "",
    calmingCues: "",
    topicsToAvoid: "",
    likes: "",
    significantPlaces: "",
    lifeChapters: "",
    favoriteMusic: "",
    favoriteShowsFilms: "",
    topicsToRevisit: "",
    reminiscenceNotes: "",
    familyMembers: [],
    preferredGreetingStyle: "",
    calmingTopics: "",
    upsettingTopics: "",
    hearingOrPacingNotes: "",
    bestTimeOfDay: "",
    doNotMention: ""
  };
}

export function patientToProfileForm(patient: Patient): PatientProfileFormValues {
  return {
    displayName: patient.displayName,
    preferredName: patient.preferredName,
    phoneE164: patient.phoneE164 ?? "",
    timezone: patient.timezone,
    notes: patient.notes ?? "",
    profilePhotoDataUrl: patient.profilePhotoDataUrl ?? "",
    routineAnchors: stringifyList(patient.routineAnchors),
    favoriteTopics: stringifyList(patient.favoriteTopics),
    calmingCues: stringifyList(patient.calmingCues),
    topicsToAvoid: stringifyList(patient.topicsToAvoid),
    likes: stringifyList(patient.memoryProfile.likes),
    significantPlaces: stringifyList(patient.memoryProfile.significantPlaces),
    lifeChapters: stringifyList(patient.memoryProfile.lifeChapters),
    favoriteMusic: stringifyList(patient.memoryProfile.favoriteMusic),
    favoriteShowsFilms: stringifyList(patient.memoryProfile.favoriteShowsFilms),
    topicsToRevisit: stringifyList(patient.memoryProfile.topicsToRevisit),
    reminiscenceNotes: patient.memoryProfile.reminiscenceNotes ?? "",
    familyMembers: patient.memoryProfile.familyMembers,
    preferredGreetingStyle: patient.conversationGuidance.preferredGreetingStyle ?? "",
    calmingTopics: stringifyList(patient.conversationGuidance.calmingTopics),
    upsettingTopics: stringifyList(patient.conversationGuidance.upsettingTopics),
    hearingOrPacingNotes: patient.conversationGuidance.hearingOrPacingNotes ?? "",
    bestTimeOfDay: patient.conversationGuidance.bestTimeOfDay ?? "",
    doNotMention: stringifyList(patient.conversationGuidance.doNotMention)
  };
}

export function profileFormToInput(
  caregiverId: string,
  values: PatientProfileFormValues
): PatientInput {
  return {
    primaryCaregiverId: caregiverId.trim(),
    displayName: values.displayName.trim(),
    preferredName: (values.preferredName.trim() || values.displayName.trim()),
    phoneE164: values.phoneE164.trim(),
    timezone: values.timezone.trim(),
    notes: values.notes.trim(),
    profilePhotoDataUrl: values.profilePhotoDataUrl.trim(),
    routineAnchors: parseList(values.routineAnchors),
    favoriteTopics: parseList(values.favoriteTopics),
    calmingCues: parseList(values.calmingCues),
    topicsToAvoid: parseList(values.topicsToAvoid),
    memoryProfile: {
      likes: parseList(values.likes),
      familyMembers: values.familyMembers
        .map((member) => ({
          name: member.name.trim(),
          relation: member.relation.trim(),
          notes: member.notes?.trim()
        }))
        .filter((member) => member.name && member.relation),
      lifeEvents: [],
      reminiscenceNotes: values.reminiscenceNotes.trim(),
      significantPlaces: parseList(values.significantPlaces),
      lifeChapters: parseList(values.lifeChapters),
      favoriteMusic: parseList(values.favoriteMusic),
      favoriteShowsFilms: parseList(values.favoriteShowsFilms),
      topicsToRevisit: parseList(values.topicsToRevisit)
    },
    conversationGuidance: {
      preferredGreetingStyle: values.preferredGreetingStyle.trim(),
      calmingTopics: parseList(values.calmingTopics),
      upsettingTopics: parseList(values.upsettingTopics),
      hearingOrPacingNotes: values.hearingOrPacingNotes.trim(),
      bestTimeOfDay: values.bestTimeOfDay.trim(),
      doNotMention: parseList(values.doNotMention)
    }
  };
}

export function parseList(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function stringifyList(values: string[]): string {
  return values.join(", ");
}
