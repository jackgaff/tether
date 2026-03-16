import { useState } from "react";
import { Plus, X, UserPlus, ArrowLeft } from "lucide-react";
import { createPatient, updateConsent } from "../api/admin";
import type { Patient } from "../api/contracts";

const TIMEZONES = [
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Anchorage",
  "Pacific/Honolulu",
  "Europe/London",
  "Europe/Paris",
  "Australia/Sydney",
];

interface FamilyMember {
  name: string;
  relation: string;
}

interface Props {
  caregiverId: string;
  onCreated: (patient: Patient) => void;
  onCancel: () => void;
}

export function CreatePatient({ caregiverId, onCreated, onCancel }: Props) {
  // Basic info
  const [displayName, setDisplayName] = useState("");
  const [preferredName, setPreferredName] = useState("");
  const [phone, setPhone] = useState("");
  const [timezone, setTimezone] = useState("America/New_York");
  const [notes, setNotes] = useState("");

  // Memory & reminiscence
  const [interests, setInterests] = useState<string[]>([]);
  const [interestInput, setInterestInput] = useState("");
  const [familyMembers, setFamilyMembers] = useState<FamilyMember[]>([]);
  const [memberName, setMemberName] = useState("");
  const [memberRelation, setMemberRelation] = useState("");
  const [memoriesNotes, setMemoriesNotes] = useState("");
  const [avoidTopics, setAvoidTopics] = useState<string[]>([]);
  const [avoidInput, setAvoidInput] = useState("");

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function addInterest() {
    const val = interestInput.trim();
    if (val && !interests.includes(val)) setInterests([...interests, val]);
    setInterestInput("");
  }

  function addAvoid() {
    const val = avoidInput.trim();
    if (val && !avoidTopics.includes(val)) setAvoidTopics([...avoidTopics, val]);
    setAvoidInput("");
  }

  function addMember() {
    if (!memberName.trim() || !memberRelation.trim()) return;
    setFamilyMembers([...familyMembers, { name: memberName.trim(), relation: memberRelation.trim() }]);
    setMemberName("");
    setMemberRelation("");
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!displayName.trim() || !caregiverId) return;
    setIsSubmitting(true);
    setError(null);

    try {
      const patient = await createPatient({
        primaryCaregiverId: caregiverId,
        displayName: displayName.trim(),
        preferredName: preferredName.trim() || displayName.trim(),
        phoneE164: phone.trim(),
        timezone,
        notes: notes.trim(),
        routineAnchors: [],
        favoriteTopics: interests,
        calmingCues: [],
        topicsToAvoid: avoidTopics,
        memoryProfile: {
          likes: interests,
          familyMembers: familyMembers.map((m) => ({
            name: m.name,
            relation: m.relation,
          })),
          lifeEvents: [],
          reminiscenceNotes: memoriesNotes.trim(),
        },
        conversationGuidance: {
          calmingTopics: interests,
          upsettingTopics: [],
          doNotMention: avoidTopics,
        },
      });

      await updateConsent(patient.id, {
        outboundCallStatus: "granted",
        transcriptStorageStatus: "granted",
        notes: "Consent granted during patient setup.",
      });

      onCreated(patient);
    } catch (err: any) {
      setError(err.message ?? "Failed to create patient");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="p-8 max-w-2xl mx-auto">
      {/* Header */}
      <button
        onClick={onCancel}
        className="flex items-center gap-1.5 text-sm text-gray-400 hover:text-gray-700 transition-colors mb-6"
      >
        <ArrowLeft size={14} strokeWidth={2} />
        Back
      </button>

      <div className="flex items-center gap-3 mb-8">
        <div className="w-9 h-9 rounded-xl bg-gray-900 flex items-center justify-center flex-shrink-0">
          <UserPlus size={16} className="text-white" strokeWidth={1.75} />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Add New Patient</h1>
          <p className="text-sm text-gray-400">
            Fill in the patient's details and memory profile for personalised calls.
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* ── Section 1: Patient Info ── */}
        <section className="bg-white border border-gray-200 rounded-2xl p-6 space-y-5">
          <h2 className="text-sm font-semibold text-gray-900">Patient Information</h2>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelCls}>
                Full name <span className="text-red-400">*</span>
              </label>
              <p className={hintCls}>As it appears on their records</p>
              <input
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                required
                autoFocus
                className={inputCls}
              />
            </div>
            <div>
              <label className={labelCls}>Preferred name</label>
              <p className={hintCls}>What they like to be called</p>
              <input
                value={preferredName}
                onChange={(e) => setPreferredName(e.target.value)}
                className={inputCls}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={labelCls}>Phone number</label>
              <p className={hintCls}>The number Echo will call</p>
              <input
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                className={inputCls}
              />
            </div>
            <div>
              <label className={labelCls}>Timezone</label>
              <p className={hintCls}>Used to schedule calls at the right time</p>
              <select
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className={inputCls}
              >
                {TIMEZONES.map((tz) => (
                  <option key={tz} value={tz}>{tz}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className={labelCls}>Notes</label>
            <p className={hintCls}>Anything the care team should keep in mind</p>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              className={inputCls}
            />
          </div>
        </section>

        {/* ── Section 2: Memory & Reminiscence ── */}
        <section className="bg-white border border-gray-200 rounded-2xl p-6 space-y-5">
          <div>
            <h2 className="text-sm font-semibold text-gray-900">Memory & Reminiscence Profile</h2>
            <p className="text-xs text-gray-400 mt-0.5">
              Echo uses this to make Reminiscence calls feel personal and comforting.
            </p>
          </div>

          {/* Interests */}
          <div>
            <label className={labelCls}>Interests & things they love</label>
            <p className={hintCls}>Hobbies, activities, music, food — anything they enjoy</p>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {interests.map((t) => (
                <span
                  key={t}
                  className="flex items-center gap-1 text-xs bg-gray-100 text-gray-700 rounded-full px-2.5 py-1"
                >
                  {t}
                  <button
                    type="button"
                    onClick={() => setInterests(interests.filter((x) => x !== t))}
                  >
                    <X size={11} strokeWidth={2.5} className="text-gray-400 hover:text-gray-700" />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                value={interestInput}
                onChange={(e) => setInterestInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addInterest(); } }}
                className={inputCls}
              />
              <button
                type="button"
                onClick={addInterest}
                disabled={!interestInput.trim()}
                className={addBtnCls}
              >
                <Plus size={15} strokeWidth={2} />
              </button>
            </div>
          </div>

          {/* Family members */}
          <div>
            <label className={labelCls}>Family members & close friends</label>
            <p className={hintCls}>People Echo can mention naturally during conversation</p>
            <div className="space-y-2 mb-2">
              {familyMembers.map((m, i) => (
                <div key={i} className="flex items-center gap-2 bg-gray-50 rounded-lg px-3 py-2 text-sm">
                  <span className="font-medium text-gray-800">{m.name}</span>
                  <span className="text-gray-400">·</span>
                  <span className="text-gray-500 capitalize">{m.relation}</span>
                  <button
                    type="button"
                    onClick={() => setFamilyMembers(familyMembers.filter((_, j) => j !== i))}
                    className="ml-auto text-gray-400 hover:text-gray-700"
                  >
                    <X size={13} strokeWidth={2.5} />
                  </button>
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                value={memberName}
                onChange={(e) => setMemberName(e.target.value)}
                className={inputCls}
              />
              <input
                value={memberRelation}
                onChange={(e) => setMemberRelation(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addMember(); } }}
                className={inputCls}
              />
              <button
                type="button"
                onClick={addMember}
                disabled={!memberName.trim() || !memberRelation.trim()}
                className={addBtnCls}
              >
                <Plus size={15} strokeWidth={2} />
              </button>
            </div>
          </div>

          {/* Memories & places */}
          <div>
            <label className={labelCls}>Memories, stories & favourite places</label>
            <p className={hintCls}>
              Things from their past Echo can bring up — childhood, work, travel, special moments
            </p>
            <textarea
              value={memoriesNotes}
              onChange={(e) => setMemoriesNotes(e.target.value)}
              rows={4}
              className={inputCls}
            />
          </div>

          {/* Topics to avoid */}
          <div>
            <label className={labelCls}>Topics to avoid</label>
            <p className={hintCls}>Things that upset or confuse them — Echo will never bring these up</p>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {avoidTopics.map((t) => (
                <span
                  key={t}
                  className="flex items-center gap-1 text-xs bg-red-50 text-red-700 rounded-full px-2.5 py-1"
                >
                  {t}
                  <button
                    type="button"
                    onClick={() => setAvoidTopics(avoidTopics.filter((x) => x !== t))}
                  >
                    <X size={11} strokeWidth={2.5} className="text-red-400 hover:text-red-700" />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                value={avoidInput}
                onChange={(e) => setAvoidInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addAvoid(); } }}
                className={inputCls}
              />
              <button
                type="button"
                onClick={addAvoid}
                disabled={!avoidInput.trim()}
                className={addBtnCls}
              >
                <Plus size={15} strokeWidth={2} />
              </button>
            </div>
          </div>
        </section>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <div className="flex justify-end gap-3 pb-8">
          <button
            type="button"
            onClick={onCancel}
            className="px-5 py-2.5 border border-gray-200 text-sm text-gray-600 rounded-lg hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting || !displayName.trim()}
            className="px-6 py-2.5 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-700 disabled:opacity-50 transition-colors"
          >
            {isSubmitting ? "Creating..." : "Add Patient"}
          </button>
        </div>
      </form>
    </div>
  );
}

const inputCls =
  "w-full px-3 py-2 text-sm bg-white border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900";
const labelCls = "block text-xs font-medium text-gray-700 mb-1";
const hintCls = "text-xs text-gray-400 mb-1.5";
const addBtnCls =
  "flex-shrink-0 px-3 py-2 bg-gray-100 text-gray-600 rounded-lg hover:bg-gray-200 disabled:opacity-40 transition-colors";
