import { useState } from "react";
import { ArrowLeft, CheckCircle2, AlertCircle, Bot, User, Clock } from "lucide-react";
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
  orientation: "Orientation",
  reminder: "Reminder",
  wellbeing: "Wellbeing",
  reminiscence: "Reminiscence",
};

const statusConfig: Record<string, { label: string; Icon: typeof CheckCircle2; color: string }> = {
  completed: { label: "Completed", Icon: CheckCircle2, color: "text-green-500" },
  in_progress: { label: "In Progress", Icon: Clock, color: "text-blue-500" },
  requested: { label: "Requested", Icon: Clock, color: "text-amber-500" },
  failed: { label: "Failed", Icon: AlertCircle, color: "text-red-500" },
  cancelled: { label: "Cancelled", Icon: AlertCircle, color: "text-gray-400" },
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

  // ── Detail view ──
  if (selectedCallId) {
    const call = recentCalls.find((c) => c.id === selectedCallId);
    // Use the fetched analysis, or fall back to the dashboard's latestAnalysis if it matches
    const analysis =
      callDetail?.analysis ??
      (selectedCallId === recentCalls[0]?.id ? latestAnalysis : undefined);

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
              {callTypeLabel[call.callType] ?? call.callType}
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

        {analysis && (
          <div className="bg-white border border-gray-200 rounded-2xl p-5 mb-4">
            <h2 className="text-sm font-semibold text-gray-900 mb-2">AI Summary</h2>
            <p className="text-sm text-gray-600 leading-relaxed mb-4">
              {analysis.result.caregiver_summary}
            </p>
            <div className="flex gap-3 flex-wrap">
              {[
                { label: "Orientation", val: analysis.result.patient_state.orientation },
                { label: "Mood", val: analysis.result.patient_state.mood },
                { label: "Engagement", val: analysis.result.patient_state.engagement },
              ].map(({ label, val }) => (
                <div key={label} className="bg-gray-50 rounded-lg px-3 py-2">
                  <p className="text-xs text-gray-400 mb-0.5">{label}</p>
                  <p className="text-sm font-medium text-gray-900 capitalize">{val}</p>
                </div>
              ))}
            </div>
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

  // ── List view ──
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
