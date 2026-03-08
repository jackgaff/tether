import { useEffect, useState, type ReactNode } from "react";
import { createDemoCheckIn, fetchCheckIns, fetchHealth, apiBaseUrl } from "./api/client";
import type { CheckIn, HealthSnapshot } from "./api/contracts";
import { SectionCard } from "./components/SectionCard";
import { StatusBadge } from "./components/StatusBadge";

const fallbackCheckIns: CheckIn[] = [
  {
    id: "demo-001",
    patientId: "patient-001",
    summary:
      "Morning check-in completed. The caller remembered breakfast and confirmed a ride for tomorrow's appointment.",
    status: "completed",
    agent: "analysis-agent",
    reminder: "Keep the appointment card by the front door.",
    recordedAt: new Date().toISOString()
  },
  {
    id: "demo-002",
    patientId: "patient-014",
    summary:
      "The caller repeated the bus question twice, so the next conversation should repeat transit details slowly and offer a short recap.",
    status: "needs_follow_up",
    agent: "safety-agent",
    reminder: "Review the bus route again before the clinic trip.",
    recordedAt: new Date(Date.now() - 1000 * 60 * 45).toISOString()
  }
];

const routeList = [
  "GET /",
  "GET /openapi.yaml",
  "GET /health",
  "GET /api/v1/check-ins",
  "POST /api/v1/check-ins"
];

export default function App() {
  const [health, setHealth] = useState<HealthSnapshot | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [checkIns, setCheckIns] = useState<CheckIn[]>([]);
  const [checkInsError, setCheckInsError] = useState<string | null>(null);
  const [isLoadingHealth, setIsLoadingHealth] = useState(true);
  const [isLoadingCheckIns, setIsLoadingCheckIns] = useState(true);
  const [isCreatingCheckIn, setIsCreatingCheckIn] = useState(false);

  useEffect(() => {
    void hydrate();
  }, []);

  async function hydrate() {
    await Promise.all([loadHealth(), loadCheckIns()]);
  }

  async function loadHealth() {
    setIsLoadingHealth(true);
    setHealthError(null);

    try {
      setHealth(await fetchHealth());
    } catch (error) {
      setHealth(null);
      setHealthError(
        error instanceof Error ? error.message : "Unable to reach the backend."
      );
    } finally {
      setIsLoadingHealth(false);
    }
  }

  async function loadCheckIns() {
    setIsLoadingCheckIns(true);
    setCheckInsError(null);

    try {
      setCheckIns(await fetchCheckIns());
    } catch (error) {
      setCheckIns([]);
      setCheckInsError(
        error instanceof Error ? error.message : "Unable to load check-ins."
      );
    } finally {
      setIsLoadingCheckIns(false);
    }
  }

  async function handleCreateDemoCheckIn() {
    setIsCreatingCheckIn(true);
    setCheckInsError(null);

    try {
      const created = await createDemoCheckIn();
      setCheckIns((current) => [created, ...current]);
    } catch (error) {
      setCheckInsError(
        error instanceof Error ? error.message : "Unable to create a demo check-in."
      );
    } finally {
      setIsCreatingCheckIn(false);
    }
  }

  const timeline = checkIns.length > 0 ? checkIns : fallbackCheckIns;

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8 lg:py-10">
      <section className="rounded-[2rem] border border-slate-200/80 bg-white/80 p-6 shadow-[0_30px_120px_rgba(15,23,42,0.12)] backdrop-blur lg:p-8">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl">
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-sky-700">
              Bun + React + Tailwind starter
            </p>
            <h1 className="mt-4 max-w-2xl text-4xl font-semibold tracking-tight text-slate-950 sm:text-5xl">
              A minimal frontend scaffold with a clear path to the Go API.
            </h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-slate-600">
              This starter is intentionally small: one typed API layer, one clean
              dashboard, and a structure that is easy to extend as you add call
              orchestration, summaries, and caregiver views.
            </p>
          </div>

          <div className="rounded-2xl border border-slate-200 bg-slate-950 px-5 py-4 text-sm text-slate-100 shadow-lg">
            <p className="font-medium text-white">API base URL</p>
            <p className="mt-2 break-all font-mono text-slate-300">{apiBaseUrl}</p>
          </div>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <SectionCard
          eyebrow="Backend"
          title="Service health"
          action={
            <button
              type="button"
              onClick={loadHealth}
              className="inline-flex items-center rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:text-slate-950"
            >
              Refresh
            </button>
          }
        >
          {isLoadingHealth ? (
            <p className="text-sm text-slate-500">Checking the Go API...</p>
          ) : health ? (
            <dl className="grid gap-4 sm:grid-cols-2">
              <Metric label="Status" value={<StatusBadge value={health.status} />} />
              <Metric label="Service" value={health.service} />
              <Metric label="Environment" value={health.env} />
              <Metric label="Auth mode" value={health.authMode} />
              <Metric
                label="Database URL"
                value={health.databaseURLConfigured ? "Configured" : "Not configured"}
              />
              <Metric label="Reported at" value={formatDateTime(health.time)} />
            </dl>
          ) : (
            <Notice
              title="The frontend is ready, but the backend is not responding yet."
              body={healthError ?? "Start the Go server to connect the dashboard."}
            />
          )}
        </SectionCard>

        <SectionCard eyebrow="Contract" title="API surface">
          <div className="space-y-4">
            <ul className="space-y-3 text-sm text-slate-600">
              {routeList.map((route) => (
                <li
                  key={route}
                  className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-slate-700"
                >
                  {route}
                </li>
              ))}
            </ul>
            <p className="text-sm leading-6 text-slate-500">
              The frontend reads <code className="rounded bg-slate-100 px-1.5 py-0.5">VITE_API_BASE_URL</code>{" "}
              from the shared root <code className="rounded bg-slate-100 px-1.5 py-0.5">.env</code>.
              The OpenAPI source is served at <code className="rounded bg-slate-100 px-1.5 py-0.5">/openapi.yaml</code>.
              Keep browser-facing routes public and reserve internal API keys for
              server-to-server traffic.
            </p>
          </div>
        </SectionCard>
      </div>

      <SectionCard
        eyebrow="Check-ins"
        title="Recent summaries"
        action={
          <button
            type="button"
            onClick={handleCreateDemoCheckIn}
            disabled={isCreatingCheckIn}
            className="inline-flex items-center rounded-full bg-slate-950 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {isCreatingCheckIn ? "Creating..." : "Create demo check-in"}
          </button>
        }
      >
        <div className="mb-5 flex flex-col gap-2 text-sm text-slate-500 sm:flex-row sm:items-center sm:justify-between">
          <p>
            Typed requests live in <code className="rounded bg-slate-100 px-1.5 py-0.5">src/api</code>, so
            backend contract changes stay isolated from the UI.
          </p>
          <p>{timeline.length} visible record(s)</p>
        </div>

        {checkInsError ? (
          <div className="mb-5">
            <Notice title="Live check-ins are unavailable." body={checkInsError} />
          </div>
        ) : null}

        {isLoadingCheckIns ? (
          <p className="text-sm text-slate-500">Loading recent check-ins...</p>
        ) : (
          <div className="grid gap-4">
            {timeline.map((item) => (
              <article
                key={item.id}
                className="rounded-3xl border border-slate-200 bg-slate-50/80 p-5"
              >
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">
                      {item.patientId}
                    </p>
                    <p className="mt-3 text-base font-medium text-slate-950">
                      {item.summary}
                    </p>
                  </div>
                  <StatusBadge value={item.status} />
                </div>

                <dl className="mt-5 grid gap-4 border-t border-slate-200 pt-4 text-sm sm:grid-cols-3">
                  <Metric label="Agent" value={item.agent} />
                  <Metric label="Recorded" value={formatDateTime(item.recordedAt)} />
                  <Metric label="Reminder" value={item.reminder ?? "No reminder captured"} />
                </dl>
              </article>
            ))}
          </div>
        )}
      </SectionCard>
    </main>
  );
}

function Metric({
  label,
  value
}: {
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3">
      <dt className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">
        {label}
      </dt>
      <dd className="mt-2 text-sm font-medium text-slate-900">{value}</dd>
    </div>
  );
}

function Notice({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
      <p className="font-medium">{title}</p>
      <p className="mt-1 text-amber-800">{body}</p>
    </div>
  );
}

function formatDateTime(value: string) {
  const parsed = new Date(value);

  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleString([], {
    dateStyle: "medium",
    timeStyle: "short"
  });
}
