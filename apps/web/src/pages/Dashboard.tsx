import {
  AlertTriangle,
  BookOpen,
  CalendarDays,
  ChevronRight,
  Clock,
  PhoneOutgoing,
  Settings2,
  Sparkles,
  Star,
  Users,
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
    timeZone: timezone,
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
      <div className="flex h-full items-center justify-center text-sm text-gray-400">Loading...</div>
    );
  }

  if (error && !dashboard) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="mb-3 text-sm text-red-600">{error}</p>
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
  const urgentFlags = (dashboard?.riskFlags ?? []).filter((flag) => flag.severity === "urgent");
  const callStatus = latestCall
    ? (statusConfig[latestCall.status] ?? {
        label: latestCall.status,
        bg: "bg-gray-100",
        text: "text-gray-500",
      })
    : null;
  const duration = formatDuration(latestCall?.startedAt, latestCall?.endedAt);
  const patientName = patient?.preferredName || patient?.displayName || "Patient overview";
  const patientMeta = [patient?.phoneE164, patient?.timezone].filter(Boolean).join(" · ");
  const memoryProfile = patient?.memoryProfile;
  const memoryProfileSections = memoryProfile
    ? [
        { label: "Interests", items: memoryProfile.likes, icon: Star },
        { label: "Favorite Music", items: memoryProfile.favoriteMusic },
        { label: "Shows & Films", items: memoryProfile.favoriteShowsFilms },
        { label: "Significant Places", items: memoryProfile.significantPlaces },
        { label: "Life Chapters", items: memoryProfile.lifeChapters },
        { label: "Topics To Revisit", items: memoryProfile.topicsToRevisit },
      ].filter((section) => section.items.length > 0)
    : [];
  const hasMemoryProfile = Boolean(
    memoryProfile &&
      (
        memoryProfileSections.length > 0 ||
        memoryProfile.familyMembers.length > 0 ||
        memoryProfile.reminiscenceNotes
      )
  );

  return (
    <div className="app-page-enter mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-8 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-sm font-medium text-slate-500">Caregiver overview</p>
          <h1 className="text-3xl font-semibold tracking-tight text-slate-950">{patientName}</h1>
          <p className="mt-1 text-sm text-slate-500">{today}</p>
        </div>
        <button
          onClick={() => onNavigate("settings")}
          className="app-btn-secondary self-start lg:self-auto"
        >
          <Settings2 size={16} strokeWidth={2.1} />
          Open settings
        </button>
      </div>

      {urgentFlags.length > 0 && (
        <div className="flex items-start gap-3 rounded-[24px] border border-red-200 bg-red-50/90 p-4">
          <AlertTriangle size={15} className="mt-0.5 flex-shrink-0 text-red-500" strokeWidth={2} />
          <div>
            <p className="text-sm font-medium text-red-800">Urgent flag from last call</p>
            <p className="mt-0.5 text-sm text-red-700">
              {urgentFlags[0].whyItMatters ?? urgentFlags[0].flagType}
            </p>
          </div>
        </div>
      )}

      <div className="grid items-start gap-4 xl:grid-cols-5">
        <div className="flex flex-col gap-4 xl:col-span-3">
          <div className="app-panel flex flex-col gap-4 overflow-hidden p-5 lg:flex-row lg:items-center">
            <div className="relative flex-shrink-0">
              <Avatar
                name={patientName}
                imageUrl={patient?.profilePhotoDataUrl}
                size="lg"
                accent="sage"
              />
              <span
                className={`absolute bottom-1 right-1 h-4 w-4 rounded-full border-2 border-white ${
                  patient?.callingState === "active" ? "bg-green-400" : "bg-gray-300"
                }`}
              />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-2xl font-semibold text-slate-950">{patient?.displayName ?? "—"}</h2>
                {patient?.callingState && (
                  <span className="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium capitalize text-slate-600">
                    {patient.callingState.replace(/_/g, " ")}
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-slate-500">{patientMeta || "No contact info yet"}</p>
              {patient?.notes && (
                <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">{patient.notes}</p>
              )}
            </div>
            <div className="flex flex-shrink-0 flex-col gap-2 lg:items-end">
              <button onClick={() => onNavigate("schedule-call")} className="app-btn-primary">
                <PhoneOutgoing size={15} strokeWidth={2.25} />
                Start Call
              </button>
              {latestCall && (
                <p className="text-sm text-slate-500">
                  Last call: {formatCallTime(latestCall.startedAt ?? latestCall.requestedAt)}
                </p>
              )}
            </div>
          </div>

          <div className="app-panel-muted p-5">
            <div className="mb-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Clock size={15} className="text-slate-400" strokeWidth={1.75} />
                <span className="text-base font-semibold text-slate-950">Last call</span>
              </div>
              {callStatus && (
                <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${callStatus.bg} ${callStatus.text}`}>
                  {callStatus.label}
                </span>
              )}
            </div>
            {latestCall ? (
              <>
                <p className="mb-3 text-sm text-slate-500">
                  {[
                    formatCallTime(latestCall.startedAt ?? latestCall.requestedAt),
                    duration,
                    callTypeLabel(latestCall.callType),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
                {latestAnalysis?.result.caregiver_summary ? (
                  <p className="text-base leading-relaxed text-slate-700">
                    {latestAnalysis.result.caregiver_summary}
                  </p>
                ) : (
                  <p className="text-sm italic text-slate-400">No summary available yet.</p>
                )}
              </>
            ) : (
              <p className="text-sm italic text-slate-400">No calls yet.</p>
            )}
          </div>

          <div className="grid gap-3 md:grid-cols-3">
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
            ].map((stat) => (
              <div key={stat.label} className="app-panel-muted px-4 py-4">
                <p className="mb-1 text-sm text-slate-400">{stat.label}</p>
                <p className="text-2xl font-semibold capitalize text-slate-950">{stat.value}</p>
                <p className="mt-0.5 text-sm text-slate-500">{stat.sub}</p>
              </div>
            ))}
          </div>

          {latestAnalysis?.result.dashboard_summary && (
            <div className="app-panel-muted flex items-center gap-4 p-5">
              <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl border border-slate-200 bg-white/80">
                <Sparkles size={16} className="text-slate-400" strokeWidth={1.75} />
              </div>
              <div className="min-w-0 flex-1">
                <span className="mb-1 block text-sm font-medium text-slate-500">AI insight</span>
                <p className="text-base text-slate-700">{latestAnalysis.result.dashboard_summary}</p>
              </div>
              <button
                onClick={() => onNavigate("recent-calls")}
                className="flex-shrink-0 whitespace-nowrap text-sm font-medium text-slate-500 transition-colors hover:text-slate-900"
              >
                View calls
              </button>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-4 xl:col-span-2">
          <div className="app-panel p-5">
            <div className="mb-4 flex items-center gap-2">
              <CalendarDays size={15} className="text-slate-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-slate-950">Next recommended call</span>
            </div>
            {nextCall ? (
              <>
                <p className="mb-0.5 text-2xl font-semibold text-slate-950">
                  {callTypeLabel(nextCall.callType)}
                </p>
                <p className="mb-3 text-base text-slate-500">
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
                <p className="mb-4 text-sm leading-relaxed text-slate-600">{nextCall.goal}</p>
                <span
                  className={`mb-4 inline-block rounded-full px-2.5 py-1 text-xs font-medium ${
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
                  className="app-btn-secondary w-full justify-center"
                >
                  Start call
                  <ChevronRight size={14} strokeWidth={2} />
                </button>
              </>
            ) : (
              <p className="text-sm italic text-slate-400">No upcoming call scheduled.</p>
            )}

            {screeningSchedule && (
              <div className="mt-4 rounded-[22px] border border-slate-200 bg-slate-50/90 px-4 py-3">
                <p className="text-xs font-medium uppercase tracking-[0.14em] text-slate-400">
                  Recurring screening
                </p>
                <p className="mt-1 text-sm font-medium text-slate-900">
                  {screeningSchedule.enabled
                    ? `${screeningSchedule.cadence} on ${["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"][screeningSchedule.preferredWeekday]} at ${screeningSchedule.preferredLocalTime}`
                    : "Paused"}
                </p>
                {screeningSchedule.enabled && screeningSchedule.nextDueAt && (
                  <p className="mt-1 text-xs text-slate-500">
                    Next window:{" "}
                    {formatDateTime(screeningSchedule.nextDueAt, screeningSchedule.timezone)}
                  </p>
                )}
              </div>
            )}
          </div>

          {patient && (
            <div className="app-panel-muted p-5">
              <span className="mb-1 block text-base font-semibold text-slate-950">Memory profile</span>
              <p className="mb-4 text-sm text-slate-500">
                Used to personalise reminiscence calls for {patient.preferredName || patient.displayName}
              </p>
              <div className="space-y-4">
                {memoryProfileSections.map((section) => {
                  const SectionIcon = section.icon;
                  return (
                    <div key={section.label}>
                      <div className="mb-2 flex items-center gap-1.5">
                        {SectionIcon ? (
                          <SectionIcon size={12} className="text-slate-400" strokeWidth={1.75} />
                        ) : null}
                        <span className="text-xs font-medium uppercase tracking-[0.14em] text-slate-400">
                          {section.label}
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {section.items.map((item) => (
                          <span
                            key={item}
                            className="rounded-full border border-slate-200 bg-white/85 px-2.5 py-1 text-xs text-slate-700"
                          >
                            {item}
                          </span>
                        ))}
                      </div>
                    </div>
                  );
                })}

                {memoryProfile?.familyMembers.length ? (
                  <div>
                    <div className="mb-2 flex items-center gap-1.5">
                      <Users size={12} className="text-slate-400" strokeWidth={1.75} />
                      <span className="text-xs font-medium uppercase tracking-[0.14em] text-slate-400">
                        Family & Friends
                      </span>
                    </div>
                    <div className="space-y-2">
                      {memoryProfile.familyMembers.map((member) => (
                        <div
                          key={member.name}
                          className="rounded-2xl border border-slate-200 bg-white/85 px-3 py-2"
                        >
                          <p className="text-sm font-medium text-slate-900">{member.name}</p>
                          <p className="text-sm text-slate-500">{member.relation}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}

                {memoryProfile?.reminiscenceNotes && (
                  <p className="rounded-2xl border border-slate-200 bg-white/85 px-4 py-3 text-sm italic leading-relaxed text-slate-600">
                    {memoryProfile.reminiscenceNotes}
                  </p>
                )}

                {!hasMemoryProfile && (
                  <p className="text-sm italic text-slate-400">No memory profile set up yet.</p>
                )}
              </div>
            </div>
          )}

          <div className="app-panel-muted p-5">
            <div className="mb-4 flex items-center gap-2">
              <BookOpen size={15} className="text-slate-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-slate-950">Memory bank</span>
            </div>
            {recentMemoryBankEntries.length > 0 ? (
              <div className="divide-y divide-slate-100">
                {recentMemoryBankEntries.map((entry) => (
                  <div key={entry.id} className="py-3 first:pt-0 last:pb-0">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="text-sm font-medium text-slate-900">{entry.topic}</p>
                        <p className="mt-1 text-sm leading-relaxed text-slate-600">{entry.summary}</p>
                      </div>
                      <span className="whitespace-nowrap text-xs text-slate-400">
                        {formatDateTime(entry.occurredAt, patient?.timezone) ?? "Recorded"}
                      </span>
                    </div>
                    {entry.respondedWellTo.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-2">
                        {entry.respondedWellTo.map((item) => (
                          <span
                            key={item}
                            className="rounded-full border border-slate-200 bg-white/85 px-2.5 py-1 text-xs text-slate-600"
                          >
                            {item}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm italic text-slate-400">No memory-bank entries yet.</p>
            )}
          </div>

          <div className="app-panel-muted p-5">
            <div className="mb-4 flex items-center gap-2">
              <Users size={15} className="text-slate-400" strokeWidth={1.75} />
              <span className="text-base font-semibold text-slate-950">People learned from calls</span>
            </div>
            {patientPeople.length > 0 ? (
              <div className="divide-y divide-slate-100">
                {patientPeople.map((person) => (
                  <div key={person.id} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                    <div>
                      <p className="text-sm font-medium text-slate-900">{person.name}</p>
                      <p className="text-xs text-slate-500">
                        {[person.relationship, person.context].filter(Boolean).join(" · ") ||
                          "Relationship pending review"}
                      </p>
                    </div>
                    <span
                      className={`rounded-full px-2.5 py-1 text-xs font-medium ${
                        person.safeToSuggestCall ? "bg-green-50 text-green-700" : "bg-amber-50 text-amber-700"
                      }`}
                    >
                      {person.safeToSuggestCall ? "Safe to suggest" : "Needs review"}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm italic text-slate-400">No people have been extracted from calls yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
