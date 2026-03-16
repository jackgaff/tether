import {
  PhoneOutgoing,
  Clock,
  CalendarDays,
  Sparkles,
  ChevronRight,
  Star,
  Users,
  AlertTriangle,
} from "lucide-react";
import type { Page } from "../App";
import type { DashboardSnapshot } from "../api/contracts";

function initials(name: string) {
  return name.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase();
}

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
  const urgentFlags = (dashboard?.riskFlags ?? []).filter((f) => f.severity === "urgent");
  const callStatus = latestCall
    ? (statusConfig[latestCall.status] ?? { label: latestCall.status, bg: "bg-gray-100", text: "text-gray-500" })
    : null;
  const duration = formatDuration(latestCall?.startedAt, latestCall?.endedAt);

  return (
    <div className="p-8">
      <p className="text-sm text-gray-400 font-medium mb-6 tracking-widest uppercase">{today}</p>

      {urgentFlags.length > 0 && (
        <div className="mb-4 flex items-start gap-3 bg-red-50 border border-red-200 rounded-xl p-4">
          <AlertTriangle size={15} className="text-red-500 flex-shrink-0 mt-0.5" strokeWidth={2} />
          <div>
            <p className="text-sm font-medium text-red-800">Urgent flag from last call</p>
            <p className="text-sm text-red-700 mt-0.5">
              {urgentFlags[0].whyItMatters ?? urgentFlags[0].flagType}
            </p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-5 gap-4 items-start">
        {/* ── Left column (3/5) ── */}
        <div className="col-span-3 flex flex-col gap-4">
          {/* Patient hero */}
          <div className="bg-white border border-gray-200 rounded-2xl p-6 flex items-center gap-5">
            <div className="relative flex-shrink-0">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center ring-4 ring-green-200">
                <span className="text-xl font-semibold text-gray-500">
                  {patient ? initials(patient.displayName) : "—"}
                </span>
              </div>
              <span
                className={`absolute bottom-0.5 right-0.5 w-3.5 h-3.5 rounded-full border-2 border-white ${
                  patient?.callingState === "active" ? "bg-green-400" : "bg-gray-300"
                }`}
              />
            </div>
            <div className="flex-1 min-w-0">
              <h1 className="text-xl font-semibold text-gray-900 mb-0.5">
                {patient?.displayName ?? "—"}
              </h1>
              <p className="text-base text-gray-400 mb-1.5">
                {[patient?.phoneE164, patient?.timezone].filter(Boolean).join(" · ") ||
                  "No contact info"}
              </p>
              {patient?.notes && (
                <p className="text-base text-gray-500 italic">&ldquo;{patient.notes}&rdquo;</p>
              )}
            </div>
            <div className="flex-shrink-0 flex flex-col items-end gap-2">
              <button
                onClick={() => onNavigate("schedule-call")}
                className="flex items-center gap-2 px-4 py-2 bg-gray-900 text-white text-base font-medium rounded-lg hover:bg-gray-700 transition-colors"
              >
                <PhoneOutgoing size={15} strokeWidth={2.25} />
                Start Call
              </button>
              {latestCall && (
                <p className="text-sm text-gray-400">
                  Last call: {formatCallTime(latestCall.startedAt ?? latestCall.requestedAt)}
                </p>
              )}
            </div>
          </div>

          {/* Last call */}
          <div className="bg-white border border-gray-200 rounded-2xl p-5">
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
          <div className="bg-white border border-gray-200 rounded-2xl p-5 grid grid-cols-3 divide-x divide-gray-100">
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
              <div key={s.label} className={i > 0 ? "pl-6" : "pr-6"}>
                <p className="text-sm text-gray-400 mb-1">{s.label}</p>
                <p className="text-2xl font-semibold text-gray-900 capitalize">{s.value}</p>
                <p className="text-sm text-gray-400 mt-0.5">{s.sub}</p>
              </div>
            ))}
          </div>

          {/* AI insight */}
          {latestAnalysis?.result.dashboard_summary && (
            <div className="bg-white border border-gray-200 rounded-2xl p-5 flex items-center gap-4">
              <div className="w-10 h-10 rounded-xl bg-gray-50 border border-gray-100 flex items-center justify-center flex-shrink-0">
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
        <div className="col-span-2 flex flex-col gap-4">
          {/* Next call */}
          <div className="bg-white border border-gray-200 rounded-2xl p-5">
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
          </div>

          {/* Memory profile */}
          {patient && (
            <div className="bg-white border border-gray-200 rounded-2xl p-5">
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
                  patient.memoryProfile.familyMembers.length === 0 &&
                  !patient.memoryProfile.reminiscenceNotes && (
                    <p className="text-sm text-gray-400 italic">No memory profile set up yet.</p>
                  )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
