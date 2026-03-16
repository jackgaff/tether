import {
  type Dispatch,
  type FormEvent,
  type ReactNode,
  type SetStateAction,
  useEffect,
  useState
} from "react";
import {
  BookHeart,
  type LucideIcon,
  Save,
  Settings2,
  Sparkles,
  UserRoundPlus,
  Users
} from "lucide-react";
import {
  createMemoryBankEntry,
  createPatientPerson,
  listMemoryBankEntries,
  listPatientPeople,
  updateMemoryBankEntry,
  updatePatientPerson
} from "../api/admin";
import type {
  Caregiver,
  CaregiverInput,
  MemoryBankEntry,
  MemoryBankEntryInput,
  Patient,
  PatientInput,
  PatientPerson,
  PatientPersonInput
} from "../api/contracts";
import { formatError, getDefaultTimezone } from "../app/forms";
import { PatientProfileForm } from "../components/patient/PatientProfileForm";

type SettingsTab = "profile" | "memory" | "people" | "caregiver";

interface PatientSettingsProps {
  patientId: string;
  patient: Patient | null;
  caregiver: Caregiver | null;
  onSavePatient: (input: PatientInput) => Promise<void>;
  onSaveCaregiver: (input: CaregiverInput) => Promise<void>;
  onRefresh: () => Promise<void> | void;
}

const tabs: { id: SettingsTab; label: string; Icon: LucideIcon }[] = [
  { id: "profile", label: "Profile", Icon: Settings2 },
  { id: "memory", label: "Memory Bank", Icon: BookHeart },
  { id: "people", label: "People", Icon: Users },
  { id: "caregiver", label: "Caregiver", Icon: Sparkles }
];

export function PatientSettings({
  patientId,
  patient,
  caregiver,
  onSavePatient,
  onSaveCaregiver,
  onRefresh
}: PatientSettingsProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("profile");
  const [people, setPeople] = useState<PatientPerson[]>([]);
  const [memoryEntries, setMemoryEntries] = useState<MemoryBankEntry[]>([]);
  const [isRelatedLoading, setIsRelatedLoading] = useState(false);
  const [relatedError, setRelatedError] = useState<string | null>(null);
  const [caregiverForm, setCaregiverForm] = useState<CaregiverInput>(() => createCaregiverForm(caregiver));
  const [caregiverError, setCaregiverError] = useState<string | null>(null);
  const [savingCaregiver, setSavingCaregiver] = useState(false);

  useEffect(() => {
    setCaregiverForm(createCaregiverForm(caregiver));
    setCaregiverError(null);
  }, [caregiver]);

  useEffect(() => {
    let cancelled = false;
    setIsRelatedLoading(true);
    setRelatedError(null);

    Promise.all([listPatientPeople(patientId), listMemoryBankEntries(patientId)])
      .then(([nextPeople, nextEntries]) => {
        if (cancelled) {
          return;
        }
        setPeople(nextPeople);
        setMemoryEntries(nextEntries);
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        setRelatedError(formatError(error));
      })
      .finally(() => {
        if (!cancelled) {
          setIsRelatedLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [patientId]);

  async function refreshRelated() {
    const [nextPeople, nextEntries] = await Promise.all([
      listPatientPeople(patientId),
      listMemoryBankEntries(patientId)
    ]);
    setPeople(nextPeople);
    setMemoryEntries(nextEntries);
    await onRefresh();
  }

  async function handleSaveCaregiver(event: FormEvent) {
    event.preventDefault();
    if (!caregiver) {
      setCaregiverError("Caregiver details are still loading.");
      return;
    }
    setSavingCaregiver(true);
    setCaregiverError(null);
    try {
      await onSaveCaregiver(caregiverForm);
    } catch (error) {
      setCaregiverError(formatError(error));
    } finally {
      setSavingCaregiver(false);
    }
  }

  return (
    <div className="app-page-enter mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
      <section className="app-panel overflow-hidden p-7 md:p-8">
        <div className="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
          <div className="max-w-3xl">
            <p className="eyebrow mb-2">Caregiver Settings</p>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950">
              Keep the memory bank, people graph, and profile polished
            </h1>
            <p className="mt-3 text-sm leading-6 text-slate-500">
              Review what the assistant has learned, correct anything noisy, and enrich the profile the agent uses before every call.
            </p>
          </div>
          <div className="grid gap-3 rounded-[28px] border border-white/70 bg-white/80 p-4 shadow-[0_20px_40px_rgba(15,23,42,0.06)] md:grid-cols-3">
            <SummaryStat label="People" value={String(people.length)} />
            <SummaryStat label="Memories" value={String(memoryEntries.length)} />
            <SummaryStat label="Status" value={patient?.callingState === "active" ? "Active" : "Paused"} />
          </div>
        </div>
      </section>

      <div className="flex flex-wrap gap-2">
        {tabs.map(({ id, label, Icon }) => {
          const active = id === activeTab;
          return (
            <button
              key={id}
              type="button"
              onClick={() => setActiveTab(id)}
              className={[
                "inline-flex items-center gap-2 rounded-full px-4 py-2.5 text-sm font-semibold transition-all",
                active
                  ? "bg-slate-950 text-white shadow-[0_18px_34px_rgba(15,23,42,0.16)]"
                  : "bg-white/80 text-slate-600 ring-1 ring-slate-200 hover:text-slate-950"
              ].join(" ")}
            >
              <Icon size={16} strokeWidth={2} />
              {label}
            </button>
          );
        })}
      </div>

      {relatedError && (
        <div className="rounded-[28px] border border-rose-200 bg-rose-50/90 px-5 py-4 text-sm text-rose-700">
          {relatedError}
        </div>
      )}

      {activeTab === "profile" && (
        <PatientProfileForm
          mode="edit"
          caregiverId={patient?.primaryCaregiverId ?? ""}
          initialPatient={patient}
          title="Refine the patient profile"
          subtitle="These settings feed the dashboard, guide live calls, and shape the non-voice analysis context after every conversation."
          submitLabel="Save profile"
          onSubmit={async (input) => {
            await onSavePatient(input);
            await refreshRelated();
          }}
        />
      )}

      {activeTab === "people" && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.35fr)]">
          <CreatePersonCard
            onCreate={async (input) => {
              await createPatientPerson(patientId, input);
              await refreshRelated();
            }}
          />
          <section className="app-panel flex flex-col gap-4 p-6">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="eyebrow mb-1">People Registry</p>
                <h2 className="text-xl font-semibold text-slate-950">Review learned people</h2>
              </div>
              {isRelatedLoading && <p className="text-sm text-slate-500">Refreshing…</p>}
            </div>
            {people.length === 0 ? (
              <EmptyState
                title="No people captured yet"
                body="As calls accumulate, any durable people mentions will show up here for caregiver review."
              />
            ) : (
              <div className="space-y-4">
                {people.map((person) => (
                  <EditablePersonCard
                    key={person.id}
                    person={person}
                    onSave={async (input) => {
                      await updatePatientPerson(patientId, person.id, input);
                      await refreshRelated();
                    }}
                  />
                ))}
              </div>
            )}
          </section>
        </div>
      )}

      {activeTab === "memory" && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.35fr)]">
          <CreateMemoryEntryCard
            people={people}
            onCreate={async (input) => {
              await createMemoryBankEntry(patientId, input);
              await refreshRelated();
            }}
          />
          <section className="app-panel flex flex-col gap-4 p-6">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="eyebrow mb-1">Memory Bank</p>
                <h2 className="text-xl font-semibold text-slate-950">Tune the durable memories</h2>
              </div>
              {isRelatedLoading && <p className="text-sm text-slate-500">Refreshing…</p>}
            </div>
            {memoryEntries.length === 0 ? (
              <EmptyState
                title="No memories logged yet"
                body="When the assistant captures a meaningful reminiscence or a durable follow-up thread, it will land here for review."
              />
            ) : (
              <div className="space-y-4">
                {memoryEntries.map((entry) => (
                  <EditableMemoryEntryCard
                    key={entry.id}
                    entry={entry}
                    people={people}
                    onSave={async (input) => {
                      await updateMemoryBankEntry(patientId, entry.id, input);
                      await refreshRelated();
                    }}
                  />
                ))}
              </div>
            )}
          </section>
        </div>
      )}

      {activeTab === "caregiver" && (
        <section className="app-panel max-w-3xl p-6 md:p-7">
          <p className="eyebrow mb-2">Caregiver Profile</p>
          <h2 className="text-2xl font-semibold text-slate-950">Keep caregiver details current</h2>
          <p className="mt-2 text-sm leading-6 text-slate-500">
            This powers next-call approvals, timezone-aware summaries, and any caregiver-facing follow-up language.
          </p>
          <form onSubmit={handleSaveCaregiver} className="mt-6 space-y-4">
            <GridField label="Display name" hint="Used in approvals and consent tracking">
              <input
                value={caregiverForm.displayName}
                onChange={(event) =>
                  setCaregiverForm((current) => ({ ...current, displayName: event.target.value }))
                }
                className={fieldClass}
              />
            </GridField>
            <div className="grid gap-4 md:grid-cols-2">
              <GridField label="Email" hint="Primary caregiver email">
                <input
                  value={caregiverForm.email}
                  onChange={(event) =>
                    setCaregiverForm((current) => ({ ...current, email: event.target.value }))
                  }
                  className={fieldClass}
                />
              </GridField>
              <GridField label="Phone" hint="Optional direct line">
                <input
                  value={caregiverForm.phoneE164}
                  onChange={(event) =>
                    setCaregiverForm((current) => ({ ...current, phoneE164: event.target.value }))
                  }
                  className={fieldClass}
                />
              </GridField>
            </div>
            <GridField label="Timezone" hint="Used for caregiver-facing scheduling">
              <input
                value={caregiverForm.timezone}
                onChange={(event) =>
                  setCaregiverForm((current) => ({ ...current, timezone: event.target.value }))
                }
                className={fieldClass}
              />
            </GridField>
            {caregiverError && (
              <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
                {caregiverError}
              </div>
            )}
            <button type="submit" className="app-btn-primary" disabled={savingCaregiver}>
              <Save size={15} strokeWidth={2.1} />
              {savingCaregiver ? "Saving..." : "Save caregiver"}
            </button>
          </form>
        </section>
      )}
    </div>
  );
}

function CreatePersonCard({
  onCreate
}: {
  onCreate: (input: PatientPersonInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState<PatientPersonInput>({
    name: "",
    relationship: "",
    status: "unknown",
    relationshipQuality: "unknown",
    context: "",
    notes: ""
  });
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  return (
    <section className="app-panel p-6">
      <p className="eyebrow mb-2">Add Person</p>
      <h2 className="text-xl font-semibold text-slate-950">Add someone the caregiver trusts</h2>
      <p className="mt-2 text-sm leading-6 text-slate-500">
        This is useful when the assistant should already know who matters before a name shows up in a transcript.
      </p>
      <form
        className="mt-6 space-y-4"
        onSubmit={async (event) => {
          event.preventDefault();
          setSaving(true);
          setError(null);
          try {
            await onCreate(draft);
            setDraft({
              name: "",
              relationship: "",
              status: "unknown",
              relationshipQuality: "unknown",
              context: "",
              notes: ""
            });
          } catch (nextError) {
            setError(formatError(nextError));
          } finally {
            setSaving(false);
          }
        }}
      >
        <GridField label="Name" hint="Who they are">
          <input
            value={draft.name}
            onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
            className={fieldClass}
          />
        </GridField>
        <GridField label="Relationship" hint="Daughter, neighbor, pastor, old friend">
          <input
            value={draft.relationship}
            onChange={(event) => setDraft((current) => ({ ...current, relationship: event.target.value }))}
            className={fieldClass}
          />
        </GridField>
        <div className="grid gap-4 md:grid-cols-2">
          <GridField label="Status" hint="Whether they are known to be living">
            <select
              value={draft.status}
              onChange={(event) => setDraft((current) => ({ ...current, status: event.target.value }))}
              className={fieldClass}
            >
              <option value="unknown">Unknown</option>
              <option value="confirmed_living">Confirmed living</option>
              <option value="deceased">Deceased</option>
            </select>
          </GridField>
          <GridField label="Relationship quality" hint="Controls safe-to-suggest logic">
            <select
              value={draft.relationshipQuality}
              onChange={(event) =>
                setDraft((current) => ({ ...current, relationshipQuality: event.target.value }))
              }
              className={fieldClass}
            >
              <option value="unknown">Unknown</option>
              <option value="close_active">Close and active</option>
              <option value="unclear">Unclear</option>
              <option value="estranged">Estranged</option>
            </select>
          </GridField>
        </div>
        <GridField label="Context" hint="Why they matter, shared routines, or recent mentions">
          <textarea
            value={draft.context ?? ""}
            onChange={(event) => setDraft((current) => ({ ...current, context: event.target.value }))}
            rows={3}
            className={fieldClass}
          />
        </GridField>
        <GridField label="Notes" hint="Any caregiver review notes">
          <textarea
            value={draft.notes}
            onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))}
            rows={3}
            className={fieldClass}
          />
        </GridField>
        {error && <p className="text-sm text-rose-600">{error}</p>}
        <button type="submit" className="app-btn-primary" disabled={saving}>
          <UserRoundPlus size={15} strokeWidth={2.1} />
          {saving ? "Adding..." : "Add person"}
        </button>
      </form>
    </section>
  );
}

function EditablePersonCard({
  person,
  onSave
}: {
  person: PatientPerson;
  onSave: (input: PatientPersonInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState<PatientPersonInput>({
    name: person.name,
    relationship: person.relationship ?? "",
    status: person.status,
    relationshipQuality: person.relationshipQuality,
    context: person.context ?? "",
    notes: person.notes ?? ""
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDraft({
      name: person.name,
      relationship: person.relationship ?? "",
      status: person.status,
      relationshipQuality: person.relationshipQuality,
      context: person.context ?? "",
      notes: person.notes ?? ""
    });
    setError(null);
  }, [person]);

  return (
    <form
      className="rounded-[28px] border border-slate-200 bg-white/90 p-5 shadow-[0_14px_28px_rgba(15,23,42,0.05)]"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        setError(null);
        try {
          await onSave(draft);
        } catch (nextError) {
          setError(formatError(nextError));
        } finally {
          setSaving(false);
        }
      }}
    >
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold text-slate-950">{person.name}</h3>
            <p className="mt-1 text-sm text-slate-500">
              Last mentioned {new Date(person.lastMentionedAt).toLocaleDateString("en-US")}
            </p>
          </div>
          <span
            className={[
              "rounded-full px-3 py-1 text-xs font-semibold",
              person.safeToSuggestCall ? "bg-emerald-50 text-emerald-700" : "bg-amber-50 text-amber-700"
            ].join(" ")}
          >
            {person.safeToSuggestCall ? "Safe to suggest" : "Needs review"}
          </span>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <GridField label="Name" hint="Display name">
            <input
              value={draft.name}
              onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
              className={fieldClass}
            />
          </GridField>
          <GridField label="Relationship" hint="How the patient knows them">
            <input
              value={draft.relationship}
              onChange={(event) => setDraft((current) => ({ ...current, relationship: event.target.value }))}
              className={fieldClass}
            />
          </GridField>
          <GridField label="Status" hint="Used for safe reminder suggestions">
            <select
              value={draft.status}
              onChange={(event) => setDraft((current) => ({ ...current, status: event.target.value }))}
              className={fieldClass}
            >
              <option value="unknown">Unknown</option>
              <option value="confirmed_living">Confirmed living</option>
              <option value="deceased">Deceased</option>
            </select>
          </GridField>
          <GridField label="Relationship quality" hint="Close and active enables safer call anchors">
            <select
              value={draft.relationshipQuality}
              onChange={(event) =>
                setDraft((current) => ({ ...current, relationshipQuality: event.target.value }))
              }
              className={fieldClass}
            >
              <option value="unknown">Unknown</option>
              <option value="close_active">Close and active</option>
              <option value="unclear">Unclear</option>
              <option value="estranged">Estranged</option>
            </select>
          </GridField>
        </div>
        <GridField label="Context" hint="When to mention them and why they matter">
          <textarea
            value={draft.context ?? ""}
            onChange={(event) => setDraft((current) => ({ ...current, context: event.target.value }))}
            rows={3}
            className={fieldClass}
          />
        </GridField>
        <GridField label="Notes" hint="Caregiver review notes">
          <textarea
            value={draft.notes}
            onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))}
            rows={3}
            className={fieldClass}
          />
        </GridField>
        {error && <p className="text-sm text-rose-600">{error}</p>}
        <button type="submit" className="app-btn-secondary w-fit" disabled={saving}>
          <Save size={15} strokeWidth={2.1} />
          {saving ? "Saving..." : "Save person"}
        </button>
      </div>
    </form>
  );
}

function CreateMemoryEntryCard({
  people,
  onCreate
}: {
  people: PatientPerson[];
  onCreate: (input: MemoryBankEntryInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState<MemoryBankEntryInput>(() => createMemoryDraft());
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return (
    <section className="app-panel p-6">
      <p className="eyebrow mb-2">New Memory</p>
      <h2 className="text-xl font-semibold text-slate-950">Log a caregiver-authored memory</h2>
      <p className="mt-2 text-sm leading-6 text-slate-500">
        Use this for details the caregiver knows are real and worth carrying forward into later reminiscence calls.
      </p>
      <form
        className="mt-6 space-y-4"
        onSubmit={async (event) => {
          event.preventDefault();
          setSaving(true);
          setError(null);
          try {
            await onCreate(draft);
            setDraft(createMemoryDraft());
          } catch (nextError) {
            setError(formatError(nextError));
          } finally {
            setSaving(false);
          }
        }}
      >
        <MemoryEntryFields draft={draft} setDraft={setDraft} people={people} />
        {error && <p className="text-sm text-rose-600">{error}</p>}
        <button type="submit" className="app-btn-primary" disabled={saving}>
          <Save size={15} strokeWidth={2.1} />
          {saving ? "Adding..." : "Add memory"}
        </button>
      </form>
    </section>
  );
}

function EditableMemoryEntryCard({
  entry,
  people,
  onSave
}: {
  entry: MemoryBankEntry;
  people: PatientPerson[];
  onSave: (input: MemoryBankEntryInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState<MemoryBankEntryInput>(() => memoryEntryToDraft(entry));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDraft(memoryEntryToDraft(entry));
    setError(null);
  }, [entry]);

  return (
    <form
      className="rounded-[28px] border border-slate-200 bg-white/90 p-5 shadow-[0_14px_28px_rgba(15,23,42,0.05)]"
      onSubmit={async (event) => {
        event.preventDefault();
        setSaving(true);
        setError(null);
        try {
          await onSave(draft);
        } catch (nextError) {
          setError(formatError(nextError));
        } finally {
          setSaving(false);
        }
      }}
    >
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold text-slate-950">{entry.topic}</h3>
          <p className="mt-1 text-sm text-slate-500">
            Logged {new Date(entry.occurredAt).toLocaleDateString("en-US")}
          </p>
        </div>
        <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600">
          {entry.createdBy === "admin" ? "Caregiver added" : "Analysis added"}
        </span>
      </div>
      <MemoryEntryFields draft={draft} setDraft={setDraft} people={people} />
      {error && <p className="mt-4 text-sm text-rose-600">{error}</p>}
      <button type="submit" className="app-btn-secondary mt-4 w-fit" disabled={saving}>
        <Save size={15} strokeWidth={2.1} />
        {saving ? "Saving..." : "Save memory"}
      </button>
    </form>
  );
}

function MemoryEntryFields({
  draft,
  setDraft,
  people
}: {
  draft: MemoryBankEntryInput;
  setDraft: Dispatch<SetStateAction<MemoryBankEntryInput>>;
  people: PatientPerson[];
}) {
  return (
    <div className="space-y-4">
      <GridField label="Topic" hint="Short title caregivers will recognize">
        <input
          value={draft.topic}
          onChange={(event) => setDraft((current) => ({ ...current, topic: event.target.value }))}
          className={fieldClass}
        />
      </GridField>
      <GridField label="Summary" hint="The durable memory or note worth preserving">
        <textarea
          value={draft.summary}
          onChange={(event) => setDraft((current) => ({ ...current, summary: event.target.value }))}
          rows={4}
          className={fieldClass}
        />
      </GridField>
      <div className="grid gap-4 md:grid-cols-2">
        <GridField label="Emotional tone" hint="Calm, proud, wistful, joyful">
          <input
            value={draft.emotionalTone ?? ""}
            onChange={(event) =>
              setDraft((current) => ({ ...current, emotionalTone: event.target.value }))
            }
            className={fieldClass}
          />
        </GridField>
        <GridField label="Occurred at" hint="Approximate memory or note date">
          <input
            type="datetime-local"
            value={toDateTimeLocalValue(draft.occurredAt)}
            onChange={(event) =>
              setDraft((current) => ({
                ...current,
                occurredAt: fromDateTimeLocalValue(event.target.value)
              }))
            }
            className={fieldClass}
          />
        </GridField>
      </div>
      <GridField label="Responded well to" hint="Comma-separated cues that worked">
        <input
          value={draft.respondedWellTo.join(", ")}
          onChange={(event) =>
            setDraft((current) => ({
              ...current,
              respondedWellTo: parseCommaList(event.target.value)
            }))
          }
          className={fieldClass}
        />
      </GridField>
      <div className="grid gap-4 md:grid-cols-3">
        <label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm font-medium text-slate-700">
          <input
            type="checkbox"
            checked={draft.anchorOffered}
            onChange={(event) =>
              setDraft((current) => ({ ...current, anchorOffered: event.target.checked }))
            }
          />
          Anchor offered
        </label>
        <label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm font-medium text-slate-700">
          <input
            type="checkbox"
            checked={draft.anchorAccepted}
            onChange={(event) =>
              setDraft((current) => ({ ...current, anchorAccepted: event.target.checked }))
            }
          />
          Anchor accepted
        </label>
        <GridField label="Anchor type" hint="Reminder style">
          <select
            value={draft.anchorType}
            onChange={(event) => setDraft((current) => ({ ...current, anchorType: event.target.value }))}
            className={fieldClass}
          >
            <option value="none">None</option>
            <option value="call">Call</option>
            <option value="music">Music</option>
            <option value="show_film">Show or film</option>
            <option value="journal">Journal</option>
          </select>
        </GridField>
      </div>
      <GridField label="Anchor detail" hint="Exactly what the caregiver wants Echo to carry forward">
        <input
          value={draft.anchorDetail ?? ""}
          onChange={(event) =>
            setDraft((current) => ({ ...current, anchorDetail: event.target.value }))
          }
          className={fieldClass}
        />
      </GridField>
      <GridField label="Suggested follow-up" hint="What to circle back to next time">
        <textarea
          value={draft.suggestedFollowUp ?? ""}
          onChange={(event) =>
            setDraft((current) => ({ ...current, suggestedFollowUp: event.target.value }))
          }
          rows={3}
          className={fieldClass}
        />
      </GridField>
      <div>
        <div className="mb-1 text-sm font-medium text-slate-800">Linked people</div>
        <p className="mb-3 text-xs leading-5 text-slate-500">
          Tie the memory to known people so future calls can use verified relationships.
        </p>
        {people.length === 0 ? (
          <p className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-500">
            Add or review people first, then link them here.
          </p>
        ) : (
          <div className="grid gap-2">
            {people.map((person) => {
              const checked = draft.personIds.includes(person.id);
              return (
                <label
                  key={person.id}
                  className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700"
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={(event) =>
                      setDraft((current) => ({
                        ...current,
                        personIds: event.target.checked
                          ? [...current.personIds, person.id]
                          : current.personIds.filter((id) => id !== person.id)
                      }))
                    }
                  />
                  <span className="font-medium text-slate-800">{person.name}</span>
                  <span className="text-slate-400">
                    {person.relationship || "Relationship pending"}
                  </span>
                </label>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

function SummaryStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-slate-200 bg-slate-50/80 px-4 py-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-400">{label}</p>
      <p className="mt-1 text-lg font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function EmptyState({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-[28px] border border-dashed border-slate-200 bg-slate-50/70 px-5 py-6 text-center">
      <p className="text-base font-semibold text-slate-800">{title}</p>
      <p className="mt-2 text-sm leading-6 text-slate-500">{body}</p>
    </div>
  );
}

function GridField({
  label,
  hint,
  children
}: {
  label: string;
  hint: string;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <div className="mb-1 text-sm font-medium text-slate-800">{label}</div>
      <p className="mb-2 text-xs leading-5 text-slate-500">{hint}</p>
      {children}
    </label>
  );
}

function createCaregiverForm(caregiver: Caregiver | null): CaregiverInput {
  return {
    displayName: caregiver?.displayName ?? "",
    email: caregiver?.email ?? "",
    phoneE164: caregiver?.phoneE164 ?? "",
    timezone: caregiver?.timezone ?? getDefaultTimezone()
  };
}

function createMemoryDraft(): MemoryBankEntryInput {
  return {
    topic: "",
    summary: "",
    emotionalTone: "",
    respondedWellTo: [],
    anchorOffered: false,
    anchorType: "none",
    anchorAccepted: false,
    anchorDetail: "",
    suggestedFollowUp: "",
    occurredAt: new Date().toISOString(),
    personIds: []
  };
}

function memoryEntryToDraft(entry: MemoryBankEntry): MemoryBankEntryInput {
  return {
    topic: entry.topic,
    summary: entry.summary,
    emotionalTone: entry.emotionalTone ?? "",
    respondedWellTo: entry.respondedWellTo,
    anchorOffered: entry.anchorOffered,
    anchorType: entry.anchorType,
    anchorAccepted: entry.anchorAccepted,
    anchorDetail: entry.anchorDetail ?? "",
    suggestedFollowUp: entry.suggestedFollowUp ?? "",
    occurredAt: entry.occurredAt,
    personIds: entry.people.map((person) => person.id)
  };
}

function parseCommaList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function toDateTimeLocalValue(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const timezoneOffset = date.getTimezoneOffset();
  const local = new Date(date.getTime() - timezoneOffset * 60_000);
  return local.toISOString().slice(0, 16);
}

function fromDateTimeLocalValue(value: string) {
  if (!value) {
    return new Date().toISOString();
  }
  return new Date(value).toISOString();
}

const fieldClass =
  "w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 shadow-[0_10px_24px_rgba(15,23,42,0.04)] outline-none transition focus:border-slate-300 focus:ring-4 focus:ring-sky-100";
