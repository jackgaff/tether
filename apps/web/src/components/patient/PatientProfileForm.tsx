import { ArrowLeft, Plus, Save, UserPlus, X } from "lucide-react";
import { type FormEvent, type ReactNode, useEffect, useState } from "react";
import type { FamilyMember, Patient, PatientInput } from "../../api/contracts";
import {
  COMMON_TIMEZONES,
  createEmptyPatientProfile,
  patientToProfileForm,
  profileFormToInput,
  type PatientProfileFormValues
} from "../../app/patientProfile";
import { getDefaultTimezone } from "../../app/forms";
import { ProfilePhotoField } from "./ProfilePhotoField";

interface PatientProfileFormProps {
  mode: "create" | "edit";
  caregiverId: string;
  initialPatient?: Patient | null;
  title: string;
  subtitle: string;
  submitLabel: string;
  onSubmit: (input: PatientInput) => Promise<void>;
  onCancel?: () => void;
}

export function PatientProfileForm({
  mode,
  caregiverId,
  initialPatient,
  title,
  subtitle,
  submitLabel,
  onSubmit,
  onCancel
}: PatientProfileFormProps) {
  const [values, setValues] = useState<PatientProfileFormValues>(() =>
    initialPatient ? patientToProfileForm(initialPatient) : createEmptyPatientProfile(getDefaultTimezone())
  );
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setValues(
      initialPatient ? patientToProfileForm(initialPatient) : createEmptyPatientProfile(getDefaultTimezone())
    );
    setError(null);
  }, [initialPatient]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!values.displayName.trim()) {
      setError("Please add the patient's full name.");
      return;
    }
    if (!caregiverId.trim()) {
      setError("The caregiver profile is still loading. Please try again in a moment.");
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await onSubmit(profileFormToInput(caregiverId, values));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save the patient profile.");
    } finally {
      setIsSubmitting(false);
    }
  }

  function updateField<Key extends keyof PatientProfileFormValues>(
    key: Key,
    nextValue: PatientProfileFormValues[Key]
  ) {
    setValues((current) => ({ ...current, [key]: nextValue }));
  }

  function updateFamilyMember(index: number, patch: Partial<FamilyMember>) {
    setValues((current) => ({
      ...current,
      familyMembers: current.familyMembers.map((member, memberIndex) =>
        memberIndex === index ? { ...member, ...patch } : member
      )
    }));
  }

  return (
    <form onSubmit={handleSubmit} className="mx-auto flex w-full max-w-6xl flex-col gap-6 pb-10">
      {onCancel && (
        <button
          type="button"
          onClick={onCancel}
          className="inline-flex w-fit items-center gap-2 text-sm font-medium text-slate-500 transition-colors hover:text-slate-900"
        >
          <ArrowLeft size={15} strokeWidth={2} />
          Back
        </button>
      )}

      <section className="app-panel overflow-hidden p-7 md:p-8">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl">
            <p className="eyebrow mb-2">{mode === "create" ? "New Patient Setup" : "Profile Studio"}</p>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950">{title}</h1>
            <p className="mt-3 text-sm leading-6 text-slate-500">{subtitle}</p>
          </div>
          <div className="grid gap-3 rounded-[28px] border border-white/70 bg-white/80 p-4 shadow-[0_20px_40px_rgba(15,23,42,0.06)] md:grid-cols-3">
            <Stat label="Profile" value={values.displayName.trim() ? "Ready" : "Missing"} />
            <Stat
              label="Memory cues"
              value={String(values.likes.split(",").filter((item) => item.trim()).length)}
            />
            <Stat
              label="Trusted people"
              value={String(values.familyMembers.filter((member) => member.name.trim()).length)}
            />
          </div>
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.95fr)]">
        <div className="flex flex-col gap-6">
          <FormSection
            title="Core profile"
            description="The essentials caregivers need on every call screen."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Full name" hint="As it appears in care records" required>
                <input
                  value={values.displayName}
                  onChange={(event) => updateField("displayName", event.target.value)}
                  className={fieldClass}
                  autoFocus
                />
              </Field>
              <Field label="Preferred name" hint="What Echo should say out loud">
                <input
                  value={values.preferredName}
                  onChange={(event) => updateField("preferredName", event.target.value)}
                  className={fieldClass}
                />
              </Field>
              <Field label="Phone number" hint="The number Echo will call">
                <input
                  value={values.phoneE164}
                  onChange={(event) => updateField("phoneE164", event.target.value)}
                  className={fieldClass}
                />
              </Field>
              <Field label="Timezone" hint="Used for scheduling and summaries">
                <select
                  value={values.timezone}
                  onChange={(event) => updateField("timezone", event.target.value)}
                  className={fieldClass}
                >
                  {[...new Set([values.timezone, ...COMMON_TIMEZONES])].map((timezone) => (
                    <option key={timezone} value={timezone}>
                      {timezone}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <Field label="Care notes" hint="Context the team should see before starting a call">
              <textarea
                value={values.notes}
                onChange={(event) => updateField("notes", event.target.value)}
                rows={4}
                className={fieldClass}
              />
            </Field>
          </FormSection>

          <ProfilePhotoField
            name={values.preferredName || values.displayName || "Patient"}
            value={values.profilePhotoDataUrl}
            onChange={(next) => updateField("profilePhotoDataUrl", next)}
          />

          <FormSection
            title="Conversation context"
            description="Live call cues that keep the assistant grounded and gentle."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Routine anchors" hint="Regular touchpoints like breakfast, church, or daily walks">
                <textarea
                  value={values.routineAnchors}
                  onChange={(event) => updateField("routineAnchors", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Current favorite topics" hint="Reliable subjects that usually land well">
                <textarea
                  value={values.favoriteTopics}
                  onChange={(event) => updateField("favoriteTopics", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Calming cues" hint="Sensory cues, rituals, or topics that steady them">
                <textarea
                  value={values.calmingCues}
                  onChange={(event) => updateField("calmingCues", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Topics to avoid" hint="Anything distressing, confusing, or unsafe">
                <textarea
                  value={values.topicsToAvoid}
                  onChange={(event) => updateField("topicsToAvoid", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
            </div>
          </FormSection>

          <FormSection
            title="Memory profile"
            description="Longer-lived facts and themes Echo can safely revisit during reminiscence."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Interests and joys" hint="Comma or line separated">
                <textarea
                  value={values.likes}
                  onChange={(event) => updateField("likes", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Significant places" hint="Homes, neighborhoods, schools, favorite trips">
                <textarea
                  value={values.significantPlaces}
                  onChange={(event) => updateField("significantPlaces", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Life chapters" hint="Teacher, parenthood, military service, retirement">
                <textarea
                  value={values.lifeChapters}
                  onChange={(event) => updateField("lifeChapters", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Favorite music" hint="Artists, songs, or genres">
                <textarea
                  value={values.favoriteMusic}
                  onChange={(event) => updateField("favoriteMusic", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Shows and films" hint="Comfort watches and references they enjoy">
                <textarea
                  value={values.favoriteShowsFilms}
                  onChange={(event) => updateField("favoriteShowsFilms", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
              <Field label="Topics to revisit" hint="Open threads Echo should bring back later">
                <textarea
                  value={values.topicsToRevisit}
                  onChange={(event) => updateField("topicsToRevisit", event.target.value)}
                  rows={3}
                  className={fieldClass}
                />
              </Field>
            </div>

            <Field label="Family and close friends" hint="People Echo can mention naturally when appropriate">
              <div className="space-y-3">
                {values.familyMembers.map((member, index) => (
                  <div
                    key={`${member.name}-${index}`}
                    className="rounded-[24px] border border-slate-200 bg-white/90 p-4 shadow-[0_12px_24px_rgba(15,23,42,0.04)]"
                  >
                    <div className="grid gap-3 md:grid-cols-[1.25fr_1fr_auto]">
                      <input
                        value={member.name}
                        onChange={(event) => updateFamilyMember(index, { name: event.target.value })}
                        placeholder="Name"
                        className={fieldClass}
                      />
                      <input
                        value={member.relation}
                        onChange={(event) => updateFamilyMember(index, { relation: event.target.value })}
                        placeholder="Relation"
                        className={fieldClass}
                      />
                      <button
                        type="button"
                        onClick={() =>
                          updateField(
                            "familyMembers",
                            values.familyMembers.filter((_, memberIndex) => memberIndex !== index)
                          )
                        }
                        className="app-btn-secondary justify-center"
                      >
                        <X size={15} strokeWidth={2.1} />
                        Remove
                      </button>
                    </div>
                    <textarea
                      value={member.notes ?? ""}
                      onChange={(event) => updateFamilyMember(index, { notes: event.target.value })}
                      placeholder="Notes or context"
                      rows={2}
                      className={`${fieldClass} mt-3`}
                    />
                  </div>
                ))}
                <button
                  type="button"
                  onClick={() =>
                    updateField("familyMembers", [
                      ...values.familyMembers,
                      { name: "", relation: "", notes: "" }
                    ])
                  }
                  className="app-btn-secondary"
                >
                  <Plus size={15} strokeWidth={2} />
                  Add person
                </button>
              </div>
            </Field>

            <Field label="Reminiscence notes" hint="Stories, emotional cues, and themes worth preserving verbatim">
              <textarea
                value={values.reminiscenceNotes}
                onChange={(event) => updateField("reminiscenceNotes", event.target.value)}
                rows={5}
                className={fieldClass}
              />
            </Field>
          </FormSection>
        </div>

        <div className="flex flex-col gap-6">
          <FormSection
            title="Voice guidance"
            description="Dial in how the non-voice analysis and live agent should approach the patient."
          >
            <Field label="Greeting style" hint="Warm, playful, formal, gently familiar">
              <input
                value={values.preferredGreetingStyle}
                onChange={(event) => updateField("preferredGreetingStyle", event.target.value)}
                className={fieldClass}
              />
            </Field>
            <Field label="Best time of day" hint="Morning, after lunch, late afternoon">
              <input
                value={values.bestTimeOfDay}
                onChange={(event) => updateField("bestTimeOfDay", event.target.value)}
                className={fieldClass}
              />
            </Field>
            <Field label="Calming topics" hint="Subjects that reliably reset or soothe">
              <textarea
                value={values.calmingTopics}
                onChange={(event) => updateField("calmingTopics", event.target.value)}
                rows={3}
                className={fieldClass}
              />
            </Field>
            <Field label="Upsetting topics" hint="Subjects that may trigger confusion or distress">
              <textarea
                value={values.upsettingTopics}
                onChange={(event) => updateField("upsettingTopics", event.target.value)}
                rows={3}
                className={fieldClass}
              />
            </Field>
            <Field label="Do not mention" hint="Explicit no-go names or topics">
              <textarea
                value={values.doNotMention}
                onChange={(event) => updateField("doNotMention", event.target.value)}
                rows={3}
                className={fieldClass}
              />
            </Field>
            <Field label="Hearing and pacing notes" hint="Slow speech, repeat gently, avoid complex phrasing">
              <textarea
                value={values.hearingOrPacingNotes}
                onChange={(event) => updateField("hearingOrPacingNotes", event.target.value)}
                rows={4}
                className={fieldClass}
              />
            </Field>
          </FormSection>

          <section className="app-panel flex flex-col gap-4 p-6">
            <p className="eyebrow">Save State</p>
            <div>
              <h3 className="text-lg font-semibold text-slate-950">
                {mode === "create" ? "Create the patient profile" : "Save profile changes"}
              </h3>
              <p className="mt-2 text-sm leading-6 text-slate-500">
                {mode === "create"
                  ? "This will create the patient, seed their memory profile, and let you continue straight into the dashboard."
                  : "Updates are reflected across the dashboard, call setup, and caregiver review surfaces."}
              </p>
            </div>
            {error && (
              <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
                {error}
              </div>
            )}
            <div className="flex flex-wrap gap-3">
              {onCancel && (
                <button
                  type="button"
                  onClick={onCancel}
                  className="app-btn-secondary"
                  disabled={isSubmitting}
                >
                  Cancel
                </button>
              )}
              <button
                type="submit"
                className="app-btn-primary"
                disabled={isSubmitting}
              >
                {mode === "create" ? (
                  <UserPlus size={16} strokeWidth={2.1} />
                ) : (
                  <Save size={16} strokeWidth={2.1} />
                )}
                {isSubmitting ? "Saving..." : submitLabel}
              </button>
            </div>
          </section>
        </div>
      </div>
    </form>
  );
}

function FormSection({
  title,
  description,
  children
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <section className="app-panel flex flex-col gap-5 p-6 md:p-7">
      <div>
        <p className="eyebrow mb-1">{title}</p>
        <p className="text-sm leading-6 text-slate-500">{description}</p>
      </div>
      {children}
    </section>
  );
}

function Field({
  label,
  hint,
  required = false,
  children
}: {
  label: string;
  hint: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <div className="mb-1 flex items-center gap-1.5 text-sm font-medium text-slate-800">
        <span>{label}</span>
        {required && <span className="text-rose-500">*</span>}
      </div>
      <p className="mb-2 text-xs leading-5 text-slate-500">{hint}</p>
      {children}
    </label>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-slate-200 bg-slate-50/80 px-4 py-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-400">{label}</p>
      <p className="mt-1 text-lg font-semibold text-slate-900">{value}</p>
    </div>
  );
}

const fieldClass =
  "w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 shadow-[0_10px_24px_rgba(15,23,42,0.04)] outline-none transition focus:border-slate-300 focus:ring-4 focus:ring-sky-100";
