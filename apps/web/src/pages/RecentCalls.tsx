import { useEffect, useState } from "react";
import {
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  Bot,
  CalendarDays,
  CheckCircle2,
  Clock,
  Info,
  LoaderCircle,
  PhoneOutgoing,
  RefreshCcw,
  Sparkles,
  User
} from "lucide-react";
import { analyzeCall, getCall } from "../api/admin";
import type {
  AnalysisRecord,
  CallRun,
  CallRunDetail,
  CheckInAnalysis,
  ReminiscenceAnalysis,
  ScreeningAnalysis
} from "../api/contracts";

function formatCallTime(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit"
  });
}

function formatDuration(start?: string, end?: string) {
  if (!start || !end) return null;
  const ms = new Date(end).getTime() - new Date(start).getTime();
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}

function formatWindow(start?: string, end?: string) {
  if (!start) return null;
  const startLabel = formatCallTime(start);
  if (!end) return startLabel;
  return `${startLabel} to ${formatCallTime(end)}`;
}

function callTypeLabel(type: string) {
  return (
    {
      screening: "Screening",
      check_in: "Check-In",
      reminiscence: "Reminiscence",
      orientation: "Orientation",
      reminder: "Reminder",
      wellbeing: "Wellbeing"
    }[type] ?? type
  );
}

const statusConfig: Record<string, { label: string; Icon: typeof CheckCircle2; color: string }> = {
  scheduled: { label: "Scheduled", Icon: CalendarDays, color: "text-violet-500" },
  completed: { label: "Completed", Icon: CheckCircle2, color: "text-green-500" },
  in_progress: { label: "In Progress", Icon: Clock, color: "text-blue-500" },
  requested: { label: "Requested", Icon: Clock, color: "text-amber-500" },
  failed: { label: "Failed", Icon: AlertCircle, color: "text-red-500" },
  cancelled: { label: "Cancelled", Icon: AlertCircle, color: "text-gray-400" }
};

const escalationConfig: Record<string, { label: string; bg: string; text: string; border: string }> = {
  none: { label: "No escalation needed", bg: "bg-green-50", text: "text-green-700", border: "border-green-200" },
  caregiver_soon: { label: "Caregiver review suggested", bg: "bg-amber-50", text: "text-amber-700", border: "border-amber-200" },
  caregiver_now: { label: "Caregiver review needed now", bg: "bg-orange-50", text: "text-orange-700", border: "border-orange-200" },
  clinical_review: { label: "Clinical review recommended", bg: "bg-red-50", text: "text-red-700", border: "border-red-200" }
};

const riskSeverityConfig: Record<string, { bg: string; text: string }> = {
  info: { bg: "bg-blue-50", text: "text-blue-700" },
  watch: { bg: "bg-amber-50", text: "text-amber-700" },
  urgent: { bg: "bg-red-50", text: "text-red-700" }
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
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisError, setAnalysisError] = useState<string | null>(null);
  const [analysisAttemptedCallId, setAnalysisAttemptedCallId] = useState<string | null>(null);

  async function openCall(callId: string) {
    setSelectedCallId(callId);
    setCallDetail(null);
    setError(null);
    setAnalysisError(null);
    setAnalysisAttemptedCallId(null);
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
    setAnalysisError(null);
    setAnalysisAttemptedCallId(null);
  }

  async function runAnalysis(callId: string, force: boolean) {
    setIsAnalyzing(true);
    setAnalysisError(null);

    try {
      const analysis = await analyzeCall(callId, { force });
      setCallDetail((current) => {
        if (!current || current.callRun.id !== callId) {
          return current;
        }
        return {
          ...current,
          analysis,
          analysisJob: current.analysisJob
            ? {
                ...current.analysisJob,
                status: "succeeded",
                finishedAt: analysis.generatedAt
              }
            : current.analysisJob
        };
      });
    } catch (err: any) {
      setAnalysisError(err.message ?? "Failed to analyze the call.");
      try {
        const refreshed = await getCall(callId);
        setCallDetail(refreshed);
      } catch {
        // Keep the current detail and surface the original analysis error.
      }
    } finally {
      setIsAnalyzing(false);
    }
  }

  useEffect(() => {
    if (
      !selectedCallId ||
      !callDetail ||
      callDetail.callRun.status !== "completed" ||
      callDetail.analysis ||
      analysisAttemptedCallId === selectedCallId ||
      callDetail.analysisJob?.status === "failed"
    ) {
      return;
    }

    setAnalysisAttemptedCallId(selectedCallId);
    void runAnalysis(selectedCallId, false);
  }, [
    selectedCallId,
    callDetail,
    analysisAttemptedCallId
  ]);

  if (selectedCallId) {
    const call = recentCalls.find((item) => item.id === selectedCallId);
    const analysis =
      callDetail?.analysis ??
      (selectedCallId === recentCalls[0]?.id ? latestAnalysis : undefined);
    const result = analysis?.result;
    const escalation = result ? escalationConfig[result.escalationLevel] : null;
    const jobStatus = callDetail?.analysisJob?.status;
    const showAutoAnalysisState = call?.status === "completed" && !result;

    return (
      <div className="p-8 max-w-3xl mx-auto">
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
              {callTypeLabel(call.callType)} Call
            </h1>
            <p className="text-sm text-gray-400 mt-0.5">
              {call.status === "scheduled"
                ? formatWindow(call.scheduleWindowStart, call.scheduleWindowEnd) ?? formatCallTime(call.requestedAt)
                : formatCallTime(call.startedAt ?? call.requestedAt)}
              {" · "}
              {call.status === "scheduled"
                ? "Awaiting scheduled run"
                : formatDuration(call.startedAt, call.endedAt) ?? "Duration unknown"}
              {" · "}
              <span className="capitalize">{call.status.replace(/_/g, " ")}</span>
            </p>
          </div>
        )}

        {isLoading && <p className="text-sm text-gray-400 mb-4">Loading...</p>}
        {error && <p className="text-sm text-red-600 mb-4">{error}</p>}

        {result && (
          <div className="space-y-4 mb-4">
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
                  { label: "Orientation", val: result.patient_state?.orientation ?? "unknown" },
                  { label: "Mood", val: result.patient_state?.mood ?? "unknown" },
                  { label: "Engagement", val: result.patient_state?.engagement ?? "unknown" }
                ].map(({ label, val }) => (
                  <div key={label} className="bg-gray-50 rounded-lg px-3 py-2">
                    <p className="text-xs text-gray-400 mb-0.5">{label}</p>
                    <p className="text-sm font-medium text-gray-900 capitalize">{val.replace(/_/g, " ")}</p>
                  </div>
                ))}
              </div>
            </div>

            {result.riskFlags.length > 0 && (
              <div className="bg-white border border-gray-200 rounded-2xl p-5">
                <div className="flex items-center gap-2 mb-3">
                  <AlertTriangle size={14} className="text-gray-400" strokeWidth={1.75} />
                  <h2 className="text-sm font-semibold text-gray-900">Risk Flags</h2>
                </div>
                <div className="space-y-2">
                  {result.riskFlags.map((flag, index) => {
                    const config = riskSeverityConfig[flag.severity] ?? riskSeverityConfig.info;
                    return (
                      <div key={`${flag.flagType}-${index}`} className={`rounded-lg px-3 py-2.5 ${config.bg}`}>
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className={`text-xs font-semibold uppercase tracking-wide ${config.text}`}>
                            {flag.severity}
                          </span>
                          <span className={`text-xs font-medium ${config.text}`}>
                            {flag.flagType.replace(/_/g, " ")}
                          </span>
                        </div>
                        {(flag.whyItMatters || flag.reason) && (
                          <p className={`text-xs ${config.text} opacity-80`}>
                            {flag.whyItMatters ?? flag.reason}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {result.checkIn && (
              <CheckInDetail analysis={result.checkIn} followUp={result.followUpIntent.requestedByPatient} />
            )}
            {result.screening && <ScreeningDetail analysis={result.screening} />}
            {result.reminiscence && <ReminiscenceDetail analysis={result.reminiscence} />}

            {result.nextCallRecommendation && (
              <div className="bg-white border border-gray-200 rounded-2xl p-5">
                <div className="flex items-center gap-2 mb-3">
                  <PhoneOutgoing size={14} className="text-gray-400" strokeWidth={1.75} />
                  <h2 className="text-sm font-semibold text-gray-900">Next Call Recommendation</h2>
                </div>
                <p className="text-sm font-medium text-gray-900 mb-1">
                  {callTypeLabel(result.nextCallRecommendation.callType)} call
                </p>
                <p className="text-xs text-gray-400 mb-2 capitalize">
                  {result.nextCallRecommendation.windowBucket.replace(/_/g, " ")}
                </p>
                <p className="text-sm text-gray-600">{result.nextCallRecommendation.goal}</p>
              </div>
            )}
          </div>
        )}

        {showAutoAnalysisState && !isLoading && (
          <div className="mb-4 rounded-xl border border-gray-200 bg-gray-50 px-4 py-3">
            {isAnalyzing || jobStatus === "pending" || jobStatus === "running" ? (
              <div className="flex items-center gap-2 text-sm text-gray-600">
                <LoaderCircle size={14} className="animate-spin text-gray-400" strokeWidth={1.75} />
                Generating analysis for this call...
              </div>
            ) : analysisError || jobStatus === "failed" ? (
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-2 text-sm text-red-700">
                  <AlertCircle size={14} className="text-red-500 flex-shrink-0 mt-0.5" strokeWidth={1.75} />
                  <div>
                    <p className="font-medium">Analysis failed for this call.</p>
                    <p className="mt-0.5">{analysisError ?? callDetail?.analysisJob?.lastError ?? "Please try again."}</p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => void runAnalysis(selectedCallId, true)}
                  className="flex-shrink-0 inline-flex items-center gap-1.5 rounded-lg border border-red-200 bg-white px-3 py-2 text-xs font-medium text-red-700 hover:bg-red-50"
                >
                  <RefreshCcw size={12} strokeWidth={2} />
                  Retry analysis
                </button>
              </div>
            ) : (
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-center gap-2 text-sm text-gray-500">
                  <Info size={14} className="text-gray-400 flex-shrink-0" strokeWidth={1.75} />
                  Analysis not yet available for this call.
                </div>
                <button
                  type="button"
                  onClick={() => void runAnalysis(selectedCallId, false)}
                  className="flex-shrink-0 inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-medium text-gray-700 hover:bg-gray-50"
                >
                  <Sparkles size={12} strokeWidth={2} />
                  Run analysis
                </button>
              </div>
            )}
          </div>
        )}

        {call?.status === "scheduled" && (
          <div className="mb-4 rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-700">
            This call has been scheduled and will remain here until the outbound workflow picks it up.
          </div>
        )}

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
                        second: "2-digit"
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
                {["Date", "Type", "Duration", "Status", ""].map((header) => (
                  <th
                    key={header}
                    className="text-left px-5 py-3.5 text-xs font-medium text-gray-400 uppercase tracking-wider"
                  >
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {recentCalls.map((call) => {
                const status = statusConfig[call.status] ?? {
                  label: call.status,
                  Icon: CheckCircle2,
                  color: "text-gray-400"
                };
                const StatusIcon = status.Icon;
                return (
                  <tr
                    key={call.id}
                    className="border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors"
                  >
                    <td className="px-5 py-3.5 text-gray-600">
                      {call.status === "scheduled"
                        ? formatWindow(call.scheduleWindowStart, call.scheduleWindowEnd) ?? formatCallTime(call.requestedAt)
                        : formatCallTime(call.startedAt ?? call.requestedAt)}
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {callTypeLabel(call.callType)}
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {call.status === "scheduled"
                        ? "Scheduled window"
                        : formatDuration(call.startedAt, call.endedAt) ?? "—"}
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-1.5">
                        <StatusIcon size={13} className={status.color} strokeWidth={2} />
                        <span className="text-gray-600">{status.label}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5">
                      <button
                        onClick={() => void openCall(call.id)}
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

function DetailRow({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div>
      <p className="text-xs font-medium text-gray-400 mb-0.5">{label}</p>
      <p className="text-sm text-gray-700">{value}</p>
    </div>
  );
}

function PillList({ values, tone = "gray" }: { values: string[]; tone?: "gray" | "amber" | "green" }) {
  if (values.length === 0) return null;
  const styles =
    tone === "amber"
      ? "bg-amber-50 text-amber-700 border-amber-100"
      : tone === "green"
      ? "bg-green-50 text-green-700 border-green-100"
      : "bg-gray-50 text-gray-600 border-gray-100";
  return (
    <div className="flex flex-wrap gap-1.5">
      {values.map((value) => (
        <span key={value} className={`text-xs border rounded px-2 py-1 ${styles}`}>
          {value}
        </span>
      ))}
    </div>
  );
}

function CheckInDetail({ analysis, followUp }: { analysis: CheckInAnalysis; followUp?: boolean }) {
  return (
    <div className="bg-white border border-gray-200 rounded-2xl p-5">
      <h2 className="text-sm font-semibold text-gray-900 mb-3">Check-In Details</h2>
      <div className="space-y-3">
        <DetailRow
          label="Orientation"
          value={[analysis.orientationStatus.replace(/_/g, " "), analysis.orientationNotes].filter(Boolean).join(" · ")}
        />
        <DetailRow
          label="Meals"
          value={[analysis.mealsStatus.replace(/_/g, " "), analysis.mealsDetail].filter(Boolean).join(" · ")}
        />
        <DetailRow
          label="Fluids"
          value={[analysis.fluidsStatus.replace(/_/g, " "), analysis.fluidsDetail].filter(Boolean).join(" · ")}
        />
        <DetailRow label="Activity" value={analysis.activityDetail} />
        <DetailRow
          label="Social contact"
          value={[analysis.socialContact.replace(/_/g, " "), analysis.socialContactDetail].filter(Boolean).join(" · ")}
        />
        <DetailRow
          label="Mood"
          value={[analysis.mood.replace(/_/g, " "), analysis.moodNotes].filter(Boolean).join(" · ")}
        />
        <DetailRow
          label="Sleep"
          value={[analysis.sleep.replace(/_/g, " "), analysis.sleepNotes].filter(Boolean).join(" · ")}
        />
        {analysis.remindersNoted.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Reminders noted</p>
            <div className="space-y-2">
              {analysis.remindersNoted.map((reminder) => (
                <div key={`${reminder.title}-${reminder.detail ?? ""}`} className="rounded-lg border border-gray-100 bg-gray-50 px-3 py-2">
                  <p className="text-sm font-medium text-gray-800">{reminder.title}</p>
                  {reminder.detail && <p className="text-xs text-gray-500 mt-0.5">{reminder.detail}</p>}
                </div>
              ))}
            </div>
          </div>
        )}
        {analysis.reminderDeclined && (
          <div className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700">
            Reminder declined{analysis.reminderDeclinedTopic ? `: ${analysis.reminderDeclinedTopic}` : "."}
          </div>
        )}
        {analysis.memoryFlags.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Memory flags</p>
            <PillList values={analysis.memoryFlags} tone="amber" />
          </div>
        )}
        {analysis.deliriumWatch && (
          <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
            <p className="font-medium">Delirium watch noted</p>
            {analysis.deliriumWatchNotes && <p className="mt-0.5">{analysis.deliriumWatchNotes}</p>}
          </div>
        )}
        {analysis.deliriumPotentialTriggers.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Potential triggers</p>
            <PillList values={analysis.deliriumPotentialTriggers} tone="amber" />
          </div>
        )}
        {followUp && (
          <div className="flex items-center gap-2 bg-blue-50 rounded-lg px-3 py-2 text-sm text-blue-700">
            <PhoneOutgoing size={13} strokeWidth={2} className="flex-shrink-0" />
            Patient requested a follow-up call.
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
    incomplete: "Incomplete"
  };

  const interpretationColor: Record<string, string> = {
    routine_follow_up: "text-green-700 bg-green-50",
    caregiver_review_suggested: "text-amber-700 bg-amber-50",
    clinical_review_suggested: "text-red-700 bg-red-50",
    incomplete: "text-gray-600 bg-gray-100"
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
        {analysis.screeningScoreRaw && <DetailRow label="Score" value={analysis.screeningScoreRaw} />}
        {analysis.screeningScoreInterpretation && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Interpretation</p>
            <span className={`text-xs font-medium px-2.5 py-1 rounded-full ${interpretationColor[analysis.screeningScoreInterpretation] ?? "text-gray-600 bg-gray-100"}`}>
              {interpretationLabel[analysis.screeningScoreInterpretation] ?? analysis.screeningScoreInterpretation}
            </span>
          </div>
        )}
        {analysis.screeningItemsAdministered.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Items administered</p>
            <PillList values={analysis.screeningItemsAdministered} />
          </div>
        )}
        {analysis.screeningFlags.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Flags</p>
            <PillList values={analysis.screeningFlags} tone="amber" />
          </div>
        )}
      </div>
    </div>
  );
}

function ReminiscenceDetail({ analysis }: { analysis: ReminiscenceAnalysis }) {
  return (
    <div className="bg-white border border-gray-200 rounded-2xl p-5">
      <h2 className="text-sm font-semibold text-gray-900 mb-3">Reminiscence Details</h2>
      <div className="space-y-3">
        <DetailRow label="Topic" value={analysis.topic} />
        <DetailRow label="Summary" value={analysis.summary} />
        <DetailRow label="Emotional tone" value={analysis.emotionalTone} />
        {analysis.mentionedPeople.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">People mentioned</p>
            <div className="space-y-2">
              {analysis.mentionedPeople.map((person) => (
                <div
                  key={`${person.name}-${person.relationship ?? ""}-${person.context ?? ""}`}
                  className="rounded-lg border border-gray-100 bg-gray-50 px-3 py-2"
                >
                  <p className="text-sm font-medium text-gray-800">{person.name}</p>
                  {person.relationship && <p className="text-xs text-gray-500 mt-0.5">{person.relationship}</p>}
                  {person.context && <p className="text-xs text-gray-500 mt-0.5">{person.context}</p>}
                </div>
              ))}
            </div>
          </div>
        )}
        {analysis.mentionedPlaces.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Places mentioned</p>
            <PillList values={analysis.mentionedPlaces} />
          </div>
        )}
        {analysis.mentionedMusic.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Music mentioned</p>
            <PillList values={analysis.mentionedMusic} tone="green" />
          </div>
        )}
        {analysis.mentionedShowsFilms.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Shows & films</p>
            <PillList values={analysis.mentionedShowsFilms} />
          </div>
        )}
        {analysis.lifeChapters.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Life chapters</p>
            <PillList values={analysis.lifeChapters} />
          </div>
        )}
        {analysis.respondedWellTo.length > 0 && (
          <div>
            <p className="text-xs font-medium text-gray-400 mb-1">Responded well to</p>
            <PillList values={analysis.respondedWellTo} tone="green" />
          </div>
        )}
        {(analysis.anchorOffered || analysis.anchorDetail || analysis.anchorType) && (
          <div className="rounded-lg bg-blue-50 px-3 py-2 text-sm text-blue-700">
            <p className="font-medium">Anchor support</p>
            <p className="mt-0.5">
              {analysis.anchorOffered ? "Anchor offered" : "No anchor offered"}
              {analysis.anchorType ? ` · ${analysis.anchorType.replace(/_/g, " ")}` : ""}
              {analysis.anchorAccepted ? " · accepted" : analysis.anchorOffered ? " · declined" : ""}
            </p>
            {analysis.anchorDetail && <p className="mt-0.5">{analysis.anchorDetail}</p>}
          </div>
        )}
        <DetailRow label="Suggested follow-up" value={analysis.suggestedFollowUp} />
      </div>
    </div>
  );
}
