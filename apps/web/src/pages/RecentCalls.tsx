import { useState } from "react";
import {
  ArrowLeft,
  CheckCircle2,
  AlertCircle,
  Bot,
  User,
  Clock,
  AlertTriangle,
  PhoneOutgoing,
  Sparkles,
  Info,
} from "lucide-react";
import { getCall } from "../api/admin";
import type { CallRun, AnalysisRecord, CallRunDetail } from "../api/contracts";

function formatCallTime(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatDuration(start?: string, end?: string) {
  if (!start || !end) return null;
  const ms = new Date(end).getTime() - new Date(start).getTime();
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}

const callTypeLabel: Record<string, string> = {
  screening: "Screening",
  check_in: "Check-In",
  reminiscence: "Reminiscence",
  orientation: "Orientation",
  reminder: "Reminder",
  wellbeing: "Wellbeing",
};

const statusConfig: Record<string, { label: string; Icon: typeof CheckCircle2; color: string }> = {
  completed: { label: "Completed", Icon: CheckCircle2, color: "text-green-500" },
  in_progress: { label: "In Progress", Icon: Clock, color: "text-blue-500" },
  requested: { label: "Requested", Icon: Clock, color: "text-amber-500" },
  failed: { label: "Failed", Icon: AlertCircle, color: "text-red-500" },
  cancelled: { label: "Cancelled", Icon: AlertCircle, color: "text-gray-400" },
};

const escalationConfig: Record<string, { label: string; bg: string; text: string; border: string }> = {
  none: { label: "No escalation needed", bg: "bg-green-50", text: "text-green-700", border: "border-green-200" },
  caregiver_soon: { label: "Caregiver review suggested", bg: "bg-amber-50", text: "text-amber-700", border: "border-amber-200" },
  caregiver_now: { label: "Caregiver review needed now", bg: "bg-orange-50", text: "text-orange-700", border: "border-orange-200" },
  clinical_review: { label: "Clinical review recommended", bg: "bg-red-50", text: "text-red-700", border: "border-red-200" },
};

const riskSeverityConfig: Record<string, { bg: string; text: string }> = {
  info: { bg: "bg-blue-50", text: "text-blue-700" },
  watch: { bg: "bg-amber-50", text: "text-amber-700" },
  urgent: { bg: "bg-red-50", text: "text-red-700" },
};

interface Props {
  recentCalls: CallRun[];
  latestAnalysis?: AnalysisRecord;
}

export function RecentCalls({ recentCalls, latestAnalysis }: Props) {
  const [selectedCallId, setSelectedCallId] = useState<string | null>(null);
  const [callDetail, setCallDetail] = useState<CallRunDetail | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function openCall(callId: string) {
    setSelectedCallId(callId);
    setCallDetail(null);
    setError(null);
    setIsLoading(true);
    try {
      const detail = await getCall(callId);
      setCallDetail(detail);
    } catch (err: any) {
      setError(err.message ?? "Failed to load call");
    } finally {
      setIsLoading(false);
    }
  }

  function closeCall() {
    setSelectedCallId(null);
    setCallDetail(null);
  }

  // ── Detail view ──────────────────────────────────────────────
  if (selectedCallId) {
    const call = recentCalls.find((c) => c.id === selectedCallId);
    const analysis =
      callDetail?.analysis ??
      (selectedCallId === recentCalls[0]?.id ? latestAnalysis : undefined);
    const result = analysis?.result;
    const escalation = result ? escalationConfig[result.escalationLevel] : null;

    return (
      <div className="p-8 max-w-3xl">
        <button
          onClick={closeCall}
          className="flex items-center gap-1.5 text-sm text-gray-400 hover:text-gray-700 transition-colors mb-6"
        >
          <ArrowLeft size={14} strokeWidth={2} />
          All calls
        </button>

        {call && (
          <div className="mb-6">
            <h1 className="text-xl font-semibold text-gray-900">
              {callTypeLabel[call.callType] ?? call.callType} Call
            </h1>
            <p className="text-sm text-gray-400 mt-0.5">
              {formatCallTime(call.startedAt ?? call.requestedAt)}
              {" · "}
              {formatDuration(call.startedAt, call.endedAt) ?? "Duration unknown"}
              {" · "}
              <span className="capitalize">{call.status}</span>
            </p>
          </div>
        )}

        {isLoading && <p className="text-sm text-gray-400 mb-4">Loading...</p>}
        {error && <p className="text-sm text-red-600 mb-4">{error}</p>}

        {result && (
          <div className="space-y-4 mb-4">
            {/* Escalation banner */}
            {escalation && result.escalationLevel !== "none" && (
              <div className={`flex items-start gap-3 rounded-xl border px-4 py-3 ${escalation.bg} ${escalation.border}`}>
                <AlertTriangle size={15} className={`${escalation.text} flex-shrink-0 mt-0.5`} strokeWidth={2} />
                <div>
                  <p className={`text-sm font-medium ${escalation.text}`}>{escalation.label}</p>
                  {result.caregiverReviewReason && (
                    <p className={`text-sm mt-0.5 ${escalation.text} opacity-80`}>{result.caregiverReviewReason}</p>
                  )}
                </div>
              </div>
            )}

            {/* AI summary */}
            <div className="bg-white border border-gray-200 rounded-2xl p-5">
              <div className="flex items-center gap-2 mb-3">
                <Sparkles size={14} className="text-gray-400" strokeWidth={1.75} />
                <h2 className="text-sm font-semibold text-gray-900">AI Summary</h2>
              </div>
              <p className="text-sm text-gray-600 leading-relaxed mb-4">
                {result.caregiver_summary ?? result.summary}
              </p>
              <div className="flex gap-3 flex-wrap">
                {[
                  { label: "Orientation", val: result.patient_state.orientation },
                  { label: "Mood", val: result.patient_state.mood },
                  { label: "Engagement", val: result.patient_state.engagement },
                ].map(({ label, val }) => (
                  <div key={label} className="bg-gray-50 rounded-lg px-3 py-2">
                    <p className="text-xs text-gray-400 mb-0.5">{label}</p>
                    <p className="text-sm font-medium text-gray-900 capitalize">{val}</p>
                  </div>
                ))}
              </div>
            </div>

            {/* Risk flags */}
            {result.riskFlags.length > 0 && (
              <div className="bg-white border border-gray-200 rounded-2xl p-5">
                <div className="flex items-center gap-2 mb-3">
                  <AlertTriangle size={14} className="text-gray-400" strokeWidth={1.75} />
                  <h2 className="text-sm font-semibold text-gray-900">Risk Flags</h2>
                </div>
                <div className="space-y-2">
                  {result.riskFlags.map((flag, i) => {
                    const sc = riskSeverityConfig[flag.severity] ?? riskSeverityConfig.info;
                    return (
                      <div key={i} className={`rounded-lg px-3 py-2.5 ${sc.bg}`}>
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className={`text-xs font-semibold uppercase tracking-wide ${sc.text}`}>
                            {flag.severity}
                          </span>
                          <span className={`text-xs font-medium ${sc.text}`}>{flag.flagType.replace(/_/g, " ")}</span>
                        </div>
                        {flag.whyItMatters && (
                          <p className={`text-xs ${sc.text} opacity-80`}>{flag.whyItMatters}</p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Call-type specific analysis */}
            {result.checkIn && (
              <CheckInDetail analysis={result.checkIn} followUp={result.followUpIntent?.requestedByPatient} />
            )}
            {result.screening && (
              <ScreeningDetail analysis={result.screening} />
            )}
            {result.reminiscence && (
              <ReminiscenceDetail analysis={result.reminiscence} />
            )}

            {/* Next call recommendation */}
            {result.nextCallRecommendation && (
              <div className="bg-white border border-gray-200 rounded-2xl p-5">
                <div className="flex items-center gap-2 mb-3">
                  <PhoneOutgoing size={14} className="text-gray-400" strokeWidth={1.75} />
                  <h2 className="text-sm font-semibold text-gray-900">Next Call Recommendation</h2>
                </div>
                <p className="text-sm font-medium text-gray-900 mb-1">
                  {callTypeLabel[result.nextCallRecommendation.callType] ?? result.nextCallRecommendation.callType} call
                </p>
                <p className="text-xs text-gray-400 mb-2 capitalize">
                  {result.nextCallRecommendation.windowBucket.replace(/_/g, " ")}
                </p>
                <p className="text-sm text-gray-600">{result.nextCallRecommendation.goal}</p>
              </div>
            )}
          </div>
        )}

        {/* No analysis yet */}
        {!result && !isLoading && call?.status === "completed" && (
          <div className="bg-gray-50 border border-gray-200 rounded-xl px-4 py-3 text-sm text-gray-500 mb-4 flex items-center gap-2">
            <Info size={14} className="text-gray-400 flex-shrink-0" strokeWidth={1.75} />
            Analysis not yet available for this call.
          </div>
        )}

        {/* Transcript */}
        {callDetail && callDetail.transcriptTurns.length > 0 && (
          <div className="bg-white border border-gray-200 rounded-2xl p-5">
            <h2 className="text-sm font-semibold text-gray-900 mb-4">Transcript</h2>
            <div className="space-y-4">
              {callDetail.transcriptTurns.map((turn) => (
                <div
                  key={turn.sequenceNo}
                  className={`flex gap-3 ${turn.direction === "user" ? "flex-row-reverse" : ""}`}
                >
                  <div
                    className={`w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0 ${
                      turn.direction === "assistant" ? "bg-gray-900" : "bg-gray-100"
                    }`}
                  >
                    {turn.direction === "assistant" ? (
                      <Bot size={14} className="text-white" strokeWidth={1.75} />
                    ) : (
                      <User size={14} className="text-gray-600" strokeWidth={1.75} />
                    )}
                  </div>
                  <div className={`flex-1 max-w-[80%] flex flex-col ${turn.direction === "user" ? "items-end" : ""}`}>
                    <div
                      className={`rounded-2xl px-4 py-2.5 ${
                        turn.direction === "assistant"
                          ? "bg-gray-100 text-gray-800"
                          : "bg-gray-900 text-white"
                      }`}
                    >
                      <p className="text-sm leading-relaxed">{turn.text}</p>
                    </div>
                    <p className="text-xs text-gray-400 mt-1 mx-1">
                      {new Date(turn.occurredAt).toLocaleTimeString("en-US", {
                        hour: "numeric",
                        minute: "2-digit",
                        second: "2-digit",
                      })}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {callDetail && callDetail.transcriptTurns.length === 0 && (
          <div className="bg-white border border-gray-200 rounded-2xl p-8 text-center">
            <p className="text-sm text-gray-400 italic">No transcript available for this call.</p>
          </div>
        )}
      </div>
    );
  }

  // ── List view ────────────────────────────────────────────────
  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Recent Calls</h1>
        <p className="mt-0.5 text-sm text-gray-400">
          {recentCalls.length} call{recentCalls.length !== 1 ? "s" : ""} in history
        </p>
      </div>

      {recentCalls.length === 0 ? (
        <div className="bg-white border border-gray-200 rounded-2xl p-12 text-center">
          <p className="text-sm text-gray-400 italic">No calls yet.</p>
        </div>
      ) : (
        <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100">
                {["Date", "Type", "Duration", "Status", ""].map((h) => (
                  <th
                    key={h}
                    className="text-left px-5 py-3.5 text-xs font-medium text-gray-400 uppercase tracking-wider"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {recentCalls.map((call) => {
                const status = statusConfig[call.status] ?? {
                  label: call.status,
                  Icon: CheckCircle2,
                  color: "text-gray-400",
                };
                const StatusIcon = status.Icon;
                return (
                  <tr
                    key={call.id}
                    className="border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors"
                  >
                    <td className="px-5 py-3.5 text-gray-600">
                      {formatCallTime(call.startedAt ?? call.requestedAt)}
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {callTypeLabel[call.callType] ?? call.callType}
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {formatDuration(call.startedAt, call.endedAt) ?? "—"}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-1.5">
                        <StatusIcon size={13} className={status.color} strokeWidth={2} />
                        <span className="text-gray-600">{status.label}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5">
                      <button
                        onClick={() => openCall(call.id)}
                        className="text-xs text-gray-500 font-medium hover:text-gray-900 transition-colors"
                      >
                        View →
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ── Call-type specific panels ─────────────────────────────────────
import type { CheckInAnalysis, ScreeningAnalysis, ReminiscenceAnalysis } from "../api/contracts";

function CheckInDetail({ analysis, followUp }: { analysis: CheckInAnalysis; followUp?: boolean }) {
  const items = [
    { label: "Day overview", val: analysis.reportedDayOverview },
    { label: "Food & hydration", val: analysis.foodAndHydration },
    { label: "Routine adherence", val: analysis.routineAdherence },
  ].filter((i) => i.val);

  const lists = [
    { label: "Mood signals", vals: analysis.moodSignals },
    { label: "Medications mentioned", vals: analysis.medicationMentions },
    { label: "Social contacts", vals: analysis.socialContactMentions },
  ].filter((i) => i.vals.length > 0);

  if (items.length === 0 && lists.length === 0 && !followUp) return null;

  return (
    <div className="bg-white border border-gray-200 rounded-2xl p-5">
      <h2 className="text-sm font-semibold text-gray-900 mb-3">Check-In Details</h2>
      <div className="space-y-3">
        {items.map(({ label, val }) => (
          <div key={label}>
            <p className="text-xs font-medium text-gray-400 mb-0.5">{label}</p>
            <p className="text-sm text-gray-700">{val}</p>
          </div>
        ))}
        {lists.map(({ label, vals }) => (
          <div key={label}>
            <p className="text-xs font-medium text-gray-400 mb-1">{label}</p>
            <div className="flex flex-wrap gap-1.5">
              {vals.map((v) => (
                <span key={v} className="text-xs bg-gray-50 border border-gray-100 rounded px-2 py-1 text-gray-600">{v}</span>
              ))}
            </div>
          </div>
        ))}
        {followUp && (
          <div className="flex items-center gap-2 bg-blue-50 rounded-lg px-3 py-2 text-sm text-blue-700">
            <PhoneOutgoing size={13} strokeWidth={2} className="flex-shrink-0" />
            Patient requested a follow-up call
          </div>
        )}
      </div>
    </div>
  );
}

function ScreeningDetail({ analysis }: { analysis: ScreeningAnalysis }) {
  const interpretationLabel: Record<string, string> = {
    routine_follow_up: "Routine follow-up",
    caregiver_review_suggested: "Caregiver review suggested",
    clinical_review_suggested: "Clinical review suggested",
    incomplete: "Incomplete",
  };

  const interpretationColor: Record<string, string> = {
    routine_follow_up: "text-green-700 bg-green-50",
    caregiver_review_suggested: "text-amber-700 bg-amber-50",
    clinical_review_suggested: "text-red-700 bg-red-50",
    incomplete: "text-gray-600 bg-gray-100",
  };

  return (
    <div className="bg-white border border-gray-200 rounded-2xl p-5">
      <h2 className="text-sm font-semibold text-gray-900 mb-3">Screening Results</h2>
      <div className="space-y-3">
        <div className="flex items-center gap-3">
          <div className="text-xs font-medium text-gray-400">Completion</div>
          <span className="text-xs capitalize font-medium text-gray-700 bg-gray-50 px-2 py-1 rounded">
            {analysis.screeningCompletionStatus}
          </span>
        </div>
        {analysis.screeningScoreRaw && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-0.5">Score</p>
            <p className="text-sm text-gray-700">{analysis.screeningScoreRaw}</p>
          </div>
        )}
        {analysis.screeningScoreInterpretation && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Interpretation</p>
            <span className={`text-xs font-medium px-2.5 py-1 rounded-full ${interpretationColor[analysis.screeningScoreInterpretation] ?? "text-gray-600 bg-gray-100"}`}>
              {interpretationLabel[analysis.screeningScoreInterpretation] ?? analysis.screeningScoreInterpretation}
            </span>
          </div>
        )}
        {analysis.screeningFlags.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Flags</p>
            <div className="flex flex-wrap gap-1.5">
              {analysis.screeningFlags.map((f) => (
                <span key={f} className="text-xs bg-amber-50 text-amber-700 border border-amber-100 rounded px-2 py-1">{f}</span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function ReminiscenceDetail({ analysis }: { analysis: ReminiscenceAnalysis }) {
  const sections = [
    { label: "Topics discussed", vals: analysis.topicsDiscussed },
    { label: "People mentioned", vals: analysis.peopleMentioned },
    { label: "Positive engagement", vals: analysis.positiveEngagementSignals },
    { label: "Distress signals", vals: analysis.distressOrTriggerSignals },
    { label: "Future topics", vals: analysis.futureReminiscenceCandidates },
  ].filter((s) => s.vals.length > 0);

  if (sections.length === 0) return null;

  return (
    <div className="bg-white border border-gray-200 rounded-2xl p-5">
      <h2 className="text-sm font-semibold text-gray-900 mb-3">Reminiscence Details</h2>
      <div className="space-y-3">
        {sections.map(({ label, vals }) => (
          <div key={label}>
            <p className="text-xs font-medium text-gray-400 mb-1">{label}</p>
            <div className="flex flex-wrap gap-1.5">
              {vals.map((v) => (
                <span key={v} className="text-xs bg-gray-50 border border-gray-100 rounded px-2 py-1 text-gray-600">{v}</span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
