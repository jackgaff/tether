import { useState, useEffect } from "react";
import { PhoneOutgoing, CheckCircle2 } from "lucide-react";
import { listCallTemplates, createPatientCall } from "../api/admin";
import type { CallTemplate, Patient } from "../api/contracts";

interface Props {
  patientId: string;
  patient: Patient | null;
  onCallStarted: () => void;
}

export function ScheduleCall({ patientId, patient, onCallStarted }: Props) {
  const [templates, setTemplates] = useState<CallTemplate[]>([]);
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(true);
  const [isStarting, setIsStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    listCallTemplates()
      .then((tpls) => {
        const active = tpls.filter((t) => t.isActive);
        setTemplates(active);
        if (active.length > 0) setSelectedTemplateId(active[0].id);
      })
      .catch((err) => setError(err.message ?? "Failed to load call templates"))
      .finally(() => setIsLoadingTemplates(false));
  }, []);

  async function handleStart() {
    if (!selectedTemplateId) return;
    setIsStarting(true);
    setError(null);
    try {
      await createPatientCall(patientId, {
        callTemplateId: selectedTemplateId,
        channel: "browser",
        triggerType: "manual",
      });
      setSuccess(true);
      setTimeout(onCallStarted, 1500);
    } catch (err: any) {
      setError(err.message ?? "Failed to start call");
    } finally {
      setIsStarting(false);
    }
  }

  if (success) {
    return (
      <div className="p-8 max-w-2xl flex flex-col items-center justify-center gap-4 min-h-64">
        <CheckCircle2 size={32} className="text-green-500" strokeWidth={1.75} />
        <p className="text-base font-medium text-gray-900">Call started successfully</p>
        <p className="text-sm text-gray-400">Redirecting to recent calls...</p>
      </div>
    );
  }

  const selectedTemplate = templates.find((t) => t.id === selectedTemplateId);
  const patientInitials = patient
    ? patient.displayName.split(" ").map((w) => w[0]).join("").slice(0, 2)
    : "—";

  return (
    <div className="p-8 max-w-2xl">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Start a Call</h1>
        <p className="mt-0.5 text-sm text-gray-400">
          Begin an outbound check-in call for{" "}
          {patient?.preferredName ?? patient?.displayName ?? "your patient"}
        </p>
      </div>

      <div className="space-y-4">
        {/* Patient */}
        <div className="bg-white border border-gray-200 rounded-xl p-5">
          <h2 className="text-sm font-semibold text-gray-900 mb-3">Patient</h2>
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
              <span className="text-xs font-medium text-gray-600">{patientInitials}</span>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900">{patient?.displayName ?? "—"}</p>
              <p className="text-xs text-gray-400">{patient?.phoneE164 ?? ""}</p>
            </div>
          </div>
        </div>

        {/* Call type */}
        <div className="bg-white border border-gray-200 rounded-xl p-5">
          <h2 className="text-sm font-semibold text-gray-900 mb-4">Call Type</h2>
          {isLoadingTemplates ? (
            <p className="text-sm text-gray-400">Loading templates...</p>
          ) : templates.length === 0 ? (
            <p className="text-sm text-gray-400 italic">No active call templates available.</p>
          ) : (
            <div className="space-y-2">
              {templates.map((t) => (
                <label
                  key={t.id}
                  className={`flex items-start gap-3 p-3.5 rounded-lg border transition-colors ${
                    selectedTemplateId === t.id
                      ? "border-gray-900 bg-gray-50"
                      : "border-gray-200 hover:bg-gray-50"
                  }`}
                >
                  <input
                    type="radio"
                    name="template"
                    value={t.id}
                    checked={selectedTemplateId === t.id}
                    onChange={() => setSelectedTemplateId(t.id)}
                    className="mt-0.5 flex-shrink-0"
                  />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{t.displayName}</p>
                    <p className="text-xs text-gray-400 mt-0.5">
                      {t.description} · {t.durationMinutes}min
                    </p>
                  </div>
                </label>
              ))}
            </div>
          )}
        </div>

        {selectedTemplate && (
          <div className="bg-amber-50 border border-amber-100 rounded-xl p-4">
            <p className="text-xs font-medium text-amber-800 mb-1">About this call type</p>
            <p className="text-sm text-amber-700">{selectedTemplate.description}</p>
          </div>
        )}

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          onClick={handleStart}
          disabled={isStarting || !selectedTemplateId || isLoadingTemplates}
          className="w-full flex items-center justify-center gap-2 py-2.5 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
        >
          <PhoneOutgoing size={15} strokeWidth={2.25} />
          {isStarting ? "Starting..." : "Start Call"}
        </button>
      </div>
    </div>
  );
}
