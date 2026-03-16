import {
  BookOpen,
  PhoneOutgoing,
  Clock,
  CalendarDays,
  Sparkles,
  ChevronRight,
  Settings2,
  Star,
  Users,
  AlertTriangle,
} from "lucide-react";
import type { Page } from "../App";
import type { DashboardSnapshot } from "../api/contracts";
import { Avatar } from "../components/Avatar";

function formatCallTime(iso?: string) {
  if (!iso) return null;
  const d = new Date(iso);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  const time = d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
  if (d.toDateString() === today.toDateString()) return `Today, ${time}`;
  if (d.toDateString() === yesterday.toDateString()) return `Yesterday, ${time}`;
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" }) + `, ${time}`;
}

function formatDuration(start?: string, end?: string) {
  if (!start || !end) return null;
  const ms = new Date(end).getTime() - new Date(start).getTime();
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}

function formatDateTime(iso?: string, timezone?: string) {
  if (!iso) return null;
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone: timezone
  });
}

function callTypeLabel(type: string) {
  return (
    {
      screening: "Screening",
      check_in: "Check-In",
      reminiscence: "Reminiscence",
      orientation: "Orientation",
      reminder: "Reminder",
      wellbeing: "Wellbeing",
    }[type] ?? type
  );
}

const statusConfig: Record<string, { label: string; bg: string; text: string }> = {
  completed: { label: "Completed", bg: "bg-green-50", text: "text-green-700" },
  in_progress: { label: "In Progress", bg: "bg-blue-50", text: "text-blue-700" },
  requested: { label: "Requested", bg: "bg-amber-50", text: "text-amber-700" },
  failed: { label: "Failed", bg: "bg-red-50", text: "text-red-700" },
  cancelled: { label: "Cancelled", bg: "bg-gray-100", text: "text-gray-500" },
};

interface Props {
  onNavigate: (page: Page) => void;
  dashboard: DashboardSnapshot | null;
  isLoading: boolean;
  error: string | null;
  onRefresh: () => void;
}

export function Dashboard({ onNavigate, dashboard, isLoading, error, onRefresh }: Props) {
  const today = new Date().toLocaleDateString("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  });

  if (isLoading && !dashboard) {
    return (
      <div className="flex h-full items-center justify-center text-gray-400 text-sm">Loading...</div>
    );
  }

  if (error && !dashboard) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-red-600 mb-3">{error}</p>
          <button onClick={onRefresh} className="text-sm text-gray-500 hover:text-gray-900">
            Retry
          </button>
        </div>
      </div>
    );
  }

  const patient = dashboard?.patient;
  const latestCall = dashboard?.latestCall;
  const latestAnalysis = dashboard?.latestAnalysis;
  const nextCall = dashboard?.activeNextCallPlan;
  const screeningSchedule = dashboard?.screeningSchedule;
  const patientPeople = dashboard?.patientPeople ?? [];
  const recentMemoryBankEntries = dashboard?.recentMemoryBankEntries ?? [];
  const urgentFlags = (dashboard?.riskFlags ?? []).filter((f) => f.severity === "urgent");
  const callStatus = latestCall
    ? (statusConfig[latestCall.status] ?? { label: latestCall.status, bg: "bg-gray-100", text: "text-gray-500" })
    : null;
  const duration = formatDuration(latestCall?.startedAt, latestCall?.endedAt);

  return (
    <div className="app-page-enter mx-auto flex w-full max-w-7xl flex-col gap-4 px-4 py-8 sm:px-6 lg:px-8">
      <div>
        <p className="eyebrow mb-2">Care Dashboard</p>
        <div className="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950">
              {patient?.preferredName || patient?.displayName || "Patient overview"}
            </h1>
            <p className="mt-1 text-sm text-slate-500">{today}</p>
          </div>
          <button
            onClick={() => onNavigate("settings")}
            className="app-btn-secondary"
          >
            <Settings2 size={16} strokeWidth={2.1} />
            Edit profile and memory
          </button>
        </div>
      </div>

      {urgentFlags.length > 0 && (
        <div className="flex items-start gap-3 rounded-[28px] border border-red-200 bg-red-50/90 p-4 shadow-[0_18px_36px_rgba(239,68,68,0.12)]">
          <AlertTriangle size={15} className="text-red-500 flex-shrink-0 mt-0.5" strokeWidth={2} />
          <div>
            <p className="text-sm font-medium text-red-800">Urgent flag from last call</p>
            <p className="text-sm text-red-700 mt-0.5">
              {urgentFlags[0].whyItMatters ?? urgentFlags[0].flagType}
            </p>
          </div>
        </div>
      )}

      <div className="grid items-start gap-4 xl:grid-cols-5">
        {/* ── Left column (3/5) ── */}
        <div className="flex flex-col gap-4 xl:col-span-3">
          {/* Patient hero */}
          <div className="app-panel flex flex-col gap-5 overflow-hidden p-6 lg:flex-row lg:items-center">
            <div className="relative flex-shrink-0">
              <Avatar
                name={patient?.preferredName || patient?.displayName || "Patient"}
                imageUrl={patient?.profilePhotoDataUrl}
                size="xl"
                accent="sage"
              />
              <span
                className={`absolute bottom-1 right-1 h-4 w-4 rounded-full border-2 border-white ${
                  patient?.callingState === "active" ? "bg-green-400" : "bg-gray-300"
                }`}
              />
            </div>
            <div className="flex-1 min-w-0">
              <p className="eyebrow mb-2">Patient Snapshot</p>
              <h2 className="text-2xl font-semibold text-slate-950 mb-1">
                {patient?.displayName ?? "—"}
              </h2>
              <p className="text-base text-slate-500 mb-2">
                {[patient?.phoneE164, patient?.timezone].filter(Boolean).join(" · ") ||
                  "No contact info"}
              </p>
              {patient?.notes && (
                <p className="max-w-2xl text-base italic text-slate-600">&ldquo;{patient.notes}&rdquo;</p>
              )}
            </div>
            <div className="flex flex-shrink-0 flex-col gap-2 lg:items-end">
              <button
                onClick={() => onNavigate("schedule-call")}
                className="app-btn-primary"
              >
                <PhoneOutgoing size={15} strokeWidth={2.25} />
                Start Call
              </button>
              <button
                onClick={() => onNavigate("settings")}
                className="app-btn-ghost"
              >
                <Settings2 size={15} strokeWidth={2} />
                Care settings
              </button>
              {latestCall && (
                <p className="text-sm text-slate-500">
                  Last call: {formatCallTime(latestCall.startedAt ?? latestCall.requestedAt)}
                </p>
              )}
            </div>
          </div>

          {/* Last call */}
          <div className="app-panel p-5">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Clock size={15} className="text-gray-400" strokeWidth={1.75} />
                <span className="text-base font-semibold text-gray-900">Last Call</span>
              </div>
              {callStatus && (
                <span className={`px-2 py-0.5 text-sm font-medium rounded ${callStatus.bg} ${callStatus.text}`}>
                  {callStatus.label}
                </span>
              )}
            </div>
            {latestCall ? (
              <>
                <p className="text-sm text-gray-400 mb-3">
                  {[
                    formatCallTime(latestCall.startedAt ?? latestCall.requestedAt),
                    duration,
                    callTypeLabel(latestCall.callType),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
                {latestAnalysis?.result.caregiver_summary ? (
                  <p className="text-base text-gray-600 leading-relaxed">
                    {latestAnalysis.result.caregiver_summary}
                  </p>
                ) : (
                  <p className="text-sm text-gray-400 italic">No summary available yet.</p>
                )}
              </>
            ) : (
              <p className="text-sm text-gray-400 italic">No calls yet.</p>
            )}
          </div>

          {/* Stats */}
          <div className="app-panel grid gap-4 p-5 md:grid-cols-3 md:divide-x md:divide-slate-100">
            {[
              {
                label: "Recent calls",
                value: String(dashboard?.recentCalls?.length ?? 0),
                sub: "in history",
              },
              {
                label: "Calling state",
                value:
                  patient?.callingState === "active"
                    ? "Active"
                    : patient?.callingState === "paused"
                    ? "Paused"
                    : "—",
                sub: patient?.pauseReason ?? "calls enabled",
              },
              {
                label: "Consent",
                value:
                  dashboard?.consent.outboundCallStatus === "granted"
                    ? "Granted"
                    : (dashboard?.consent.outboundCallStatus ?? "—"),
                sub: "outbound calls",
              },
            ].map((s, i) => (
              <div key={s.label} className={i > 0 ? "md:pl-6" : "md:pr-6"}>
                <p className="text-sm text-slate-400 mb-1">{s.label}</p>
                <p className="text-2xl font-semibold text-slate-950 capitalize">{s.value}</p>
                <p className="text-sm text-slate-500 mt-0.5">{s.sub}</p>
              </div>
            ))}
          </div>

          {/* AI insight */}
          {latestAnalysis?.result.dashboard_summary && (
            <div className="app-panel flex items-center gap-4 p-5">
              <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl border border-sky-100 bg-sky-50">
                <Sparkles size={16} className="text-gray-400" strokeWidth={1.75} />
              </div>
              <div className="flex-1 min-w-0">
                <span className="text-xs font-medium text-gray-400 uppercase tracking-wide block mb-1">
                  AI Insight
                </span>
                <p className="text-base text-gray-700">{latestAnalysis.result.dashboard_summary}</p>
              </div>
              <button
                onClick={() => onNavigate("recent-calls")}
                className="flex-shrink-0 text-sm text-gray-400 hover:text-gray-700 transition-colors font-medium whitespace-nowrap"
              >
                View calls →
              </button>
            </div>
          )}
        </div>

        {/* ── Right column (2/5) ── */}
        <div className="flex flex-col gap-4 xl:col-span-2">
          {/* Next call */}
          <div className="app-panel p-5">
            <div className="flex items-center gap-2 mb-4">
              <CalendarDays size={15} className="text-gray-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-gray-900">Next Recommended Call</span>
            </div>
            {nextCall ? (
              <>
                <p className="text-2xl font-semibold text-gray-900 mb-0.5">
                  {callTypeLabel(nextCall.callType)}
                </p>
                <p className="text-base text-gray-400 mb-3">
                  {nextCall.plannedFor
                    ? new Date(nextCall.plannedFor).toLocaleString("en-US", {
                        month: "short",
                        day: "numeric",
                        hour: "numeric",
                        minute: "2-digit",
                      })
                    : nextCall.suggestedTimeNote ?? "Time TBD"}
                  {" · "}
                  {nextCall.durationMinutes}min
                </p>
                <p className="text-sm text-gray-600 mb-4 leading-relaxed">{nextCall.goal}</p>
                <span
                  className={`inline-block px-2 py-0.5 text-xs font-medium rounded mb-4 ${
                    nextCall.approvalStatus === "approved"
                      ? "bg-green-50 text-green-700"
                      : nextCall.approvalStatus === "pending_approval"
                      ? "bg-amber-50 text-amber-700"
                      : "bg-gray-100 text-gray-500"
                  }`}
                >
                  {nextCall.approvalStatus.replace(/_/g, " ")}
                </span>
                <button
                  onClick={() => onNavigate("schedule-call")}
                  className="w-full flex items-center justify-center gap-1.5 py-2.5 border border-gray-200 rounded-lg text-base text-gray-600 hover:bg-gray-50 transition-colors"
                >
                  Start call
                  <ChevronRight size={14} strokeWidth={2} />
                </button>
              </>
            ) : (
              <p className="text-sm text-gray-400 italic">No upcoming call scheduled.</p>
            )}

            {screeningSchedule && (
              <div className="mt-4 rounded-xl border border-violet-100 bg-violet-50 px-4 py-3">
                <p className="text-xs font-medium uppercase tracking-wide text-violet-500">
                  Recurring screening
                </p>
                <p className="mt-1 text-sm font-medium text-violet-900">
                  {screeningSchedule.enabled
                    ? `${screeningSchedule.cadence} on ${["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"][screeningSchedule.preferredWeekday]} at ${screeningSchedule.preferredLocalTime}`
                    : "Paused"}
                </p>
                {screeningSchedule.enabled && screeningSchedule.nextDueAt && (
                  <p className="mt-1 text-xs text-violet-700">
                    Next window: {formatDateTime(screeningSchedule.nextDueAt, screeningSchedule.timezone)}
                  </p>
                )}
              </div>
            )}
          </div>

          {/* Memory profile */}
          {patient && (
            <div className="app-panel p-5">
              <span className="text-base font-semibold text-gray-900 block mb-1">Memory Profile</span>
              <p className="text-sm text-gray-400 mb-4">
                Used to personalise reminiscence calls for {patient.preferredName || patient.displayName}
              </p>
              <div className="space-y-4">
                {patient.memoryProfile.likes.length > 0 && (
                  <div>
                    <div className="flex items-center gap-1.5 mb-2">
                      <Star size={12} className="text-gray-400" strokeWidth={1.75} />
                      <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Interests</span>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {patient.memoryProfile.likes.map((t) => (
                        <span key={t} className="text-xs bg-gray-50 border border-gray-100 rounded px-2 py-1 text-gray-600">{t}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.favoriteMusic.length > 0 && (
                  <div>
                    <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Favorite Music</span>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {patient.memoryProfile.favoriteMusic.map((item) => (
                        <span key={item} className="text-xs bg-blue-50 border border-blue-100 rounded px-2 py-1 text-blue-700">{item}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.favoriteShowsFilms.length > 0 && (
                  <div>
                    <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Shows & Films</span>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {patient.memoryProfile.favoriteShowsFilms.map((item) => (
                        <span key={item} className="text-xs bg-emerald-50 border border-emerald-100 rounded px-2 py-1 text-emerald-700">{item}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.significantPlaces.length > 0 && (
                  <div>
                    <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Significant Places</span>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {patient.memoryProfile.significantPlaces.map((item) => (
                        <span key={item} className="text-xs bg-amber-50 border border-amber-100 rounded px-2 py-1 text-amber-700">{item}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.lifeChapters.length > 0 && (
                  <div>
                    <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Life Chapters</span>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {patient.memoryProfile.lifeChapters.map((item) => (
                        <span key={item} className="text-xs bg-rose-50 border border-rose-100 rounded px-2 py-1 text-rose-700">{item}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.topicsToRevisit.length > 0 && (
                  <div>
                    <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Topics To Revisit</span>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {patient.memoryProfile.topicsToRevisit.map((item) => (
                        <span key={item} className="text-xs bg-violet-50 border border-violet-100 rounded px-2 py-1 text-violet-700">{item}</span>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.familyMembers.length > 0 && (
                  <div>
                    <div className="flex items-center gap-1.5 mb-2">
                      <Users size={12} className="text-gray-400" strokeWidth={1.75} />
                      <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Family & Friends</span>
                    </div>
                    <div className="space-y-1">
                      {patient.memoryProfile.familyMembers.map((m) => (
                        <p key={m.name} className="text-sm text-gray-600">
                          <span className="font-medium">{m.name}</span>
                          <span className="text-gray-400"> · {m.relation}</span>
                        </p>
                      ))}
                    </div>
                  </div>
                )}
                {patient.memoryProfile.reminiscenceNotes && (
                  <p className="text-sm text-gray-500 italic leading-relaxed">
                    "{patient.memoryProfile.reminiscenceNotes}"
                  </p>
                )}
                {patient.memoryProfile.likes.length === 0 &&
                  patient.memoryProfile.favoriteMusic.length === 0 &&
                  patient.memoryProfile.favoriteShowsFilms.length === 0 &&
                  patient.memoryProfile.significantPlaces.length === 0 &&
                  patient.memoryProfile.lifeChapters.length === 0 &&
                  patient.memoryProfile.topicsToRevisit.length === 0 &&
                  patient.memoryProfile.familyMembers.length === 0 &&
                  !patient.memoryProfile.reminiscenceNotes && (
                    <p className="text-sm text-gray-400 italic">No memory profile set up yet.</p>
                  )}
              </div>
            </div>
          )}

          <div className="app-panel p-5">
            <div className="flex items-center gap-2 mb-4">
              <BookOpen size={15} className="text-gray-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-gray-900">Memory Bank</span>
            </div>
            {recentMemoryBankEntries.length > 0 ? (
              <div className="space-y-3">
                {recentMemoryBankEntries.map((entry) => (
                  <div key={entry.id} className="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="text-sm font-medium text-gray-900">{entry.topic}</p>
                        <p className="mt-1 text-sm text-gray-600 leading-relaxed">{entry.summary}</p>
                      </div>
                      <span className="text-xs text-gray-400 whitespace-nowrap">
                        {formatDateTime(entry.occurredAt, patient?.timezone) ?? "Recorded"}
                      </span>
                    </div>
                    {entry.respondedWellTo.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-1.5">
                        {entry.respondedWellTo.map((item) => (
                          <span key={item} className="text-xs bg-white border border-gray-200 rounded px-2 py-1 text-gray-600">{item}</span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-400 italic">No memory-bank entries yet.</p>
            )}
          </div>

          <div className="app-panel p-5">
            <div className="flex items-center gap-2 mb-4">
              <Users size={15} className="text-gray-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-gray-900">People Learned From Calls</span>
            </div>
            {patientPeople.length > 0 ? (
              <div className="space-y-3">
                {patientPeople.map((person) => (
                  <div key={person.id} className="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <p className="text-sm font-medium text-gray-900">{person.name}</p>
                        <p className="text-xs text-gray-500">
                          {[person.relationship, person.context].filter(Boolean).join(" · ") || "Relationship pending review"}
                        </p>
                      </div>
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          person.safeToSuggestCall ? "bg-green-50 text-green-700" : "bg-amber-50 text-amber-700"
                        }`}
                      >
                        {person.safeToSuggestCall ? "Safe to suggest" : "Needs review"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-400 italic">No people have been extracted from calls yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
