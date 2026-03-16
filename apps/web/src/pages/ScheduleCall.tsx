import { useEffect, useMemo, useState } from "react";
import { CalendarDays, Clock3, PhoneOutgoing, Save } from "lucide-react";
import {
  createPatientCall,
  getScreeningSchedule,
  listCallTemplates,
  updateScreeningSchedule
} from "../api/admin";
import { LiveCallPanel } from "../components/LiveCallPanel";
import type {
  CallTemplate,
  Patient,
  ScreeningSchedule,
  ScreeningScheduleInput,
  VoiceSessionDescriptor
} from "../api/contracts";

const WEEKDAYS = [
  { value: 0, label: "Sunday" },
  { value: 1, label: "Monday" },
  { value: 2, label: "Tuesday" },
  { value: 3, label: "Wednesday" },
  { value: 4, label: "Thursday" },
  { value: 5, label: "Friday" },
  { value: 6, label: "Saturday" }
];

const KNOWN_TIMEZONES = [
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Anchorage",
  "Pacific/Honolulu",
  "Europe/London",
  "Europe/Paris",
  "Australia/Sydney"
];

function defaultSchedule(timezone: string): ScreeningScheduleInput {
  return {
    enabled: false,
    cadence: "weekly",
    timezone,
    preferredWeekday: 1,
    preferredLocalTime: "09:00"
  };
}

function formatPreviewDate(value: Date, timezone: string) {
  return new Intl.DateTimeFormat("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone: timezone
  }).format(value);
}

function buildUpcomingSchedulePreview(
  schedule: ScreeningScheduleInput,
  nextDueAt?: string,
  count = 4
) {
  if (!schedule.enabled) {
    return [];
  }

  const cadenceDays = schedule.cadence === "biweekly" ? 14 : 7;
  if (nextDueAt) {
    const first = new Date(nextDueAt);
    return Array.from({ length: count }, (_, index) => {
      const date = new Date(first);
      date.setDate(first.getDate() + index * cadenceDays);
      return date;
    });
  }

  const [hours, minutes] = schedule.preferredLocalTime.split(":").map((part) => Number(part));
  const now = new Date();
  const first = new Date(now);
  first.setHours(Number.isFinite(hours) ? hours : 9, Number.isFinite(minutes) ? minutes : 0, 0, 0);

  let daysUntil = (schedule.preferredWeekday - first.getDay() + 7) % 7;
  if (daysUntil === 0 && first <= now) {
    daysUntil = 7;
  }
  first.setDate(first.getDate() + daysUntil);

  return Array.from({ length: count }, (_, index) => {
    const date = new Date(first);
    date.setDate(first.getDate() + index * cadenceDays);
    return date;
  });
}

interface Props {
  patientId: string;
  patient: Patient | null;
  onCallStarted: () => void;
  onScheduleUpdated?: () => void;
}

export function ScheduleCall({ patientId, patient, onCallStarted, onScheduleUpdated }: Props) {
  const [templates, setTemplates] = useState<CallTemplate[]>([]);
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(true);
  const [isStarting, setIsStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [voiceSession, setVoiceSession] = useState<VoiceSessionDescriptor | null>(null);

  const [screeningSchedule, setScreeningSchedule] = useState<ScreeningSchedule | null>(null);
  const [scheduleForm, setScheduleForm] = useState<ScreeningScheduleInput>(
    defaultSchedule(patient?.timezone ?? "America/New_York")
  );
  const [isLoadingSchedule, setIsLoadingSchedule] = useState(true);
  const [isSavingSchedule, setIsSavingSchedule] = useState(false);
  const [scheduleError, setScheduleError] = useState<string | null>(null);
  const [scheduleSuccess, setScheduleSuccess] = useState<string | null>(null);

  useEffect(() => {
    listCallTemplates()
      .then((tpls) => {
        const active = tpls.filter((t) => t.isActive);
        setTemplates(active);
        if (active.length > 0) {
          setSelectedTemplateId((current) => current || active[0].id);
        }
      })
      .catch((err) => setError(err.message ?? "Failed to load call templates"))
      .finally(() => setIsLoadingTemplates(false));
  }, []);

  useEffect(() => {
    setIsLoadingSchedule(true);
    setScheduleError(null);

    getScreeningSchedule(patientId)
      .then((schedule) => {
        setScreeningSchedule(schedule);
        setScheduleForm({
          enabled: schedule.enabled,
          cadence: schedule.cadence,
          timezone: schedule.timezone,
          preferredWeekday: schedule.preferredWeekday,
          preferredLocalTime: schedule.preferredLocalTime
        });
      })
      .catch((err) => {
        setScheduleError(err.message ?? "Failed to load the screening schedule.");
        setScheduleForm(defaultSchedule(patient?.timezone ?? "America/New_York"));
      })
      .finally(() => setIsLoadingSchedule(false));
  }, [patientId, patient?.timezone]);

  const selectedTemplate = templates.find((t) => t.id === selectedTemplateId);
  const patientInitials = patient
    ? patient.displayName
        .split(" ")
        .map((word) => word[0])
        .join("")
        .slice(0, 2)
    : "—";

  const timezoneOptions = useMemo(() => {
    const items = new Set(KNOWN_TIMEZONES);
    if (scheduleForm.timezone) {
      items.add(scheduleForm.timezone);
    }
    if (patient?.timezone) {
      items.add(patient.timezone);
    }
    return Array.from(items);
  }, [scheduleForm.timezone, patient?.timezone]);

  const screeningPreview = useMemo(
    () => buildUpcomingSchedulePreview(scheduleForm, screeningSchedule?.nextDueAt),
    [scheduleForm, screeningSchedule?.nextDueAt]
  );

  async function handleStart() {
    if (!selectedTemplateId) return;
    setIsStarting(true);
    setError(null);
    try {
      const response = await createPatientCall(patientId, {
        callTemplateId: selectedTemplateId,
        channel: "browser",
        triggerType: "manual"
      });
      if (response.voiceSession) {
        setVoiceSession(response.voiceSession);
      } else {
        onCallStarted();
      }
    } catch (err: any) {
      setError(err.message ?? "Failed to start call");
    } finally {
      setIsStarting(false);
    }
  }

  async function handleSaveSchedule() {
    setIsSavingSchedule(true);
    setScheduleError(null);
    setScheduleSuccess(null);
    try {
      const saved = await updateScreeningSchedule(patientId, scheduleForm);
      setScreeningSchedule(saved);
      setScheduleForm({
        enabled: saved.enabled,
        cadence: saved.cadence,
        timezone: saved.timezone,
        preferredWeekday: saved.preferredWeekday,
        preferredLocalTime: saved.preferredLocalTime
      });
      setScheduleSuccess(
        saved.enabled
          ? "Recurring screening schedule saved."
          : "Recurring screening schedule paused."
      );
      onScheduleUpdated?.();
    } catch (err: any) {
      setScheduleError(err.message ?? "Failed to save the screening schedule.");
    } finally {
      setIsSavingSchedule(false);
    }
  }

  if (voiceSession) {
    return (
      <div className="p-8 max-w-2xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold text-gray-900">Live Call</h1>
          <p className="mt-0.5 text-sm text-gray-400">
            Browser call in progress with {patient?.preferredName ?? patient?.displayName ?? "patient"}
          </p>
        </div>
        <LiveCallPanel voiceSession={voiceSession} onSessionEnded={onCallStarted} />
      </div>
    );
  }

  return (
    <div className="p-8 max-w-5xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Calls & Schedule</h1>
        <p className="mt-0.5 text-sm text-gray-400">
          Start a live browser call or manage the recurring screening schedule for{" "}
          {patient?.preferredName ?? patient?.displayName ?? "your patient"}.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h2 className="text-sm font-semibold text-gray-900 mb-3">Patient</h2>
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
                <span className="text-xs font-medium text-gray-600">{patientInitials}</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-900">{patient?.displayName ?? "—"}</p>
                <p className="text-xs text-gray-400">
                  {[patient?.phoneE164, patient?.timezone].filter(Boolean).join(" · ")}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <div className="flex items-center gap-2 mb-4">
              <PhoneOutgoing size={15} className="text-gray-400" strokeWidth={1.75} />
              <h2 className="text-sm font-semibold text-gray-900">Start Live Call</h2>
            </div>

            {isLoadingTemplates ? (
              <p className="text-sm text-gray-400">Loading templates...</p>
            ) : templates.length === 0 ? (
              <p className="text-sm text-gray-400 italic">No active call templates available.</p>
            ) : (
              <div className="space-y-2">
                {templates.map((template) => (
                  <label
                    key={template.id}
                    className={`flex items-start gap-3 rounded-lg border p-3.5 transition-colors ${
                      selectedTemplateId === template.id
                        ? "border-gray-900 bg-gray-50"
                        : "border-gray-200 hover:bg-gray-50"
                    }`}
                  >
                    <input
                      type="radio"
                      name="template"
                      value={template.id}
                      checked={selectedTemplateId === template.id}
                      onChange={() => setSelectedTemplateId(template.id)}
                      className="mt-0.5 flex-shrink-0"
                    />
                    <div>
                      <p className="text-sm font-medium text-gray-900">{template.displayName}</p>
                      <p className="text-xs text-gray-400 mt-0.5">
                        {template.description} · {template.durationMinutes}min
                      </p>
                    </div>
                  </label>
                ))}
              </div>
            )}

            {selectedTemplate && (
              <div className="mt-4 rounded-xl border border-amber-100 bg-amber-50 p-4">
                <p className="text-xs font-medium text-amber-800 mb-1">About this call type</p>
                <p className="text-sm text-amber-700">{selectedTemplate.description}</p>
              </div>
            )}

            {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

            <button
              onClick={handleStart}
              disabled={isStarting || !selectedTemplateId || isLoadingTemplates}
              className="mt-4 w-full flex items-center justify-center gap-2 py-2.5 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
            >
              <PhoneOutgoing size={15} strokeWidth={2.25} />
              {isStarting ? "Starting..." : "Start Browser Call"}
            </button>
          </div>
        </div>

        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <div className="flex items-center gap-2 mb-4">
              <CalendarDays size={15} className="text-gray-400" strokeWidth={1.75} />
              <h2 className="text-sm font-semibold text-gray-900">Recurring Screening Schedule</h2>
            </div>

            {isLoadingSchedule ? (
              <p className="text-sm text-gray-400">Loading screening schedule...</p>
            ) : (
              <div className="space-y-4">
                <label className="flex items-center gap-3 rounded-lg border border-gray-200 p-3">
                  <input
                    type="checkbox"
                    checked={scheduleForm.enabled}
                    onChange={(event) =>
                      setScheduleForm((current) => ({
                        ...current,
                        enabled: event.target.checked
                      }))
                    }
                  />
                  <div>
                    <p className="text-sm font-medium text-gray-900">Enable recurring screening calls</p>
                    <p className="text-xs text-gray-400">
                      The backend will create scheduled screening call runs using this cadence.
                    </p>
                  </div>
                </label>

                <div className="grid gap-3 sm:grid-cols-2">
                  <label className="block">
                    <span className="block text-xs font-medium text-gray-500 mb-1.5">Cadence</span>
                    <select
                      value={scheduleForm.cadence}
                      onChange={(event) =>
                        setScheduleForm((current) => ({
                          ...current,
                          cadence: event.target.value as ScreeningScheduleInput["cadence"]
                        }))
                      }
                      className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
                    >
                      <option value="weekly">Weekly</option>
                      <option value="biweekly">Biweekly</option>
                    </select>
                  </label>

                  <label className="block">
                    <span className="block text-xs font-medium text-gray-500 mb-1.5">Weekday</span>
                    <select
                      value={scheduleForm.preferredWeekday}
                      onChange={(event) =>
                        setScheduleForm((current) => ({
                          ...current,
                          preferredWeekday: Number(event.target.value)
                        }))
                      }
                      className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
                    >
                      {WEEKDAYS.map((weekday) => (
                        <option key={weekday.value} value={weekday.value}>
                          {weekday.label}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="block">
                    <span className="block text-xs font-medium text-gray-500 mb-1.5">Local time</span>
                    <input
                      type="time"
                      value={scheduleForm.preferredLocalTime}
                      onChange={(event) =>
                        setScheduleForm((current) => ({
                          ...current,
                          preferredLocalTime: event.target.value
                        }))
                      }
                      className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
                    />
                  </label>

                  <label className="block">
                    <span className="block text-xs font-medium text-gray-500 mb-1.5">Timezone</span>
                    <select
                      value={scheduleForm.timezone}
                      onChange={(event) =>
                        setScheduleForm((current) => ({
                          ...current,
                          timezone: event.target.value
                        }))
                      }
                      className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
                    >
                      {timezoneOptions.map((timezone) => (
                        <option key={timezone} value={timezone}>
                          {timezone}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>

                {scheduleError && <p className="text-sm text-red-600">{scheduleError}</p>}
                {scheduleSuccess && <p className="text-sm text-green-600">{scheduleSuccess}</p>}

                <button
                  type="button"
                  onClick={handleSaveSchedule}
                  disabled={isSavingSchedule}
                  className="w-full flex items-center justify-center gap-2 rounded-lg border border-gray-200 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                >
                  <Save size={15} strokeWidth={2} />
                  {isSavingSchedule ? "Saving..." : "Save Schedule"}
                </button>
              </div>
            )}
          </div>

          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <div className="flex items-center gap-2 mb-4">
              <Clock3 size={15} className="text-gray-400" strokeWidth={1.75} />
              <h2 className="text-sm font-semibold text-gray-900">Calendar Preview</h2>
            </div>

            {!scheduleForm.enabled ? (
              <p className="text-sm text-gray-400 italic">
                Turn on recurring screening calls to preview the next scheduled windows.
              </p>
            ) : screeningPreview.length === 0 ? (
              <p className="text-sm text-gray-400 italic">No scheduled windows yet.</p>
            ) : (
              <div className="grid gap-3">
                {screeningPreview.map((date, index) => (
                  <div key={`${date.toISOString()}-${index}`} className="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3">
                    <p className="text-xs font-medium uppercase tracking-wide text-gray-400">
                      {index === 0 ? "Next screening" : `Upcoming #${index + 1}`}
                    </p>
                    <p className="mt-1 text-sm font-medium text-gray-900">
                      {formatPreviewDate(date, scheduleForm.timezone)}
                    </p>
                  </div>
                ))}
              </div>
            )}

            {screeningSchedule?.lastScheduledWindowStart && (
              <div className="mt-4 rounded-xl border border-blue-100 bg-blue-50 px-4 py-3">
                <p className="text-xs font-medium uppercase tracking-wide text-blue-500">Last scheduled window</p>
                <p className="mt-1 text-sm text-blue-800">
                  {formatPreviewDate(new Date(screeningSchedule.lastScheduledWindowStart), scheduleForm.timezone)}
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
