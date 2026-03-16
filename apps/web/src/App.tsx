import { type FormEvent, useCallback, useEffect, useState } from "react";
import { LogOut, Radio, UserPlus } from "lucide-react";
import { Sidebar } from "./components/Sidebar";
import { Avatar } from "./components/Avatar";
import { Dashboard } from "./pages/Dashboard";
import { ScheduleCall } from "./pages/ScheduleCall";
import { RecentCalls } from "./pages/RecentCalls";
import { CreatePatient } from "./pages/CreatePatient";
import { PatientSettings } from "./pages/PatientSettings";
import {
  createCaregiver,
  getAdminSession,
  getDashboard,
  listCaregivers,
  listPatients,
  loginAdmin,
  updateCaregiver,
  updatePatient
} from "./api/admin";
import { useStoredString } from "./app/storage";
import { STORAGE_KEYS } from "./app/constants";
import type {
  AdminSession,
  CaregiverInput,
  DashboardSnapshot,
  Patient,
  PatientInput
} from "./api/contracts";

export type Page = "dashboard" | "schedule-call" | "recent-calls" | "settings";
type PreConsoleScreen = "picker" | "create-patient";

export default function App() {
  const [currentPage, setCurrentPage] = useState<Page>("dashboard");
  const [collapsed, setCollapsed] = useState(false);

  const [session, setSession] = useState<AdminSession | null>(null);
  const [authChecked, setAuthChecked] = useState(false);
  const [loginForm, setLoginForm] = useState({ username: "", password: "" });
  const [loginError, setLoginError] = useState<string | null>(null);
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  const [preScreen, setPreScreen] = useState<PreConsoleScreen>("picker");

  const [patientId, setPatientId] = useStoredString(STORAGE_KEYS.patientId);
  const [caregiverId, setCaregiverId] = useStoredString(STORAGE_KEYS.caregiverId);
  const [patientList, setPatientList] = useState<Patient[]>([]);
  const [patientListLoading, setPatientListLoading] = useState(false);
  const [caregiverBootstrapError, setCaregiverBootstrapError] = useState<string | null>(null);
  const [caregiverBootstrapState, setCaregiverBootstrapState] = useState<"idle" | "running" | "ready" | "failed">("idle");

  const [dashboard, setDashboard] = useState<DashboardSnapshot | null>(null);
  const [dashboardError, setDashboardError] = useState<string | null>(null);
  const [isDashboardLoading, setIsDashboardLoading] = useState(false);

  const loadPatients = useCallback(async () => {
    setPatientListLoading(true);
    try {
      const patients = await listPatients();
      setPatientList(patients);
    } catch {
      setPatientList([]);
    } finally {
      setPatientListLoading(false);
    }
  }, []);

  const fetchDashboard = useCallback(
    async (pid: string) => {
      setIsDashboardLoading(true);
      setDashboardError(null);
      try {
        const data = await getDashboard(pid);
        setDashboard(data);
        if (data.caregiver?.id) {
          setCaregiverId(data.caregiver.id);
        }
      } catch (error) {
        setDashboardError(error instanceof Error ? error.message : "Failed to load dashboard");
      } finally {
        setIsDashboardLoading(false);
      }
    },
    [setCaregiverId]
  );

  useEffect(() => {
    getAdminSession()
      .then(setSession)
      .catch(() => setSession(null))
      .finally(() => setAuthChecked(true));
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }
    void loadPatients();
  }, [session, loadPatients]);

  useEffect(() => {
    if (!session || patientListLoading || caregiverBootstrapState !== "idle") {
      return;
    }

    setCaregiverBootstrapState("running");
    setCaregiverBootstrapError(null);

    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "America/New_York";
    const usernameSlug = session.username
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "demo-admin";
    const caregiverEmail = `${usernameSlug}@local.tether.test`;

    const caregiverInput: CaregiverInput = {
      displayName:
        session.username
          .trim()
          .split(/[\s-_]+/)
          .filter(Boolean)
          .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
          .join(" ") || "Demo Caregiver",
      email: caregiverEmail,
      phoneE164: "",
      timezone
    };

    listCaregivers()
      .then((caregivers) => {
        const storedCaregiver = caregiverId
          ? caregivers.find((caregiver) => caregiver.id === caregiverId)
          : null;
        if (caregiverId && !storedCaregiver) {
          setCaregiverId("");
        }
        if (storedCaregiver) {
          setCaregiverBootstrapState("ready");
          return;
        }

        const matchingCaregiver = caregivers.find(
          (caregiver) => caregiver.email.trim().toLowerCase() === caregiverEmail
        );
        if (matchingCaregiver) {
          setCaregiverId(matchingCaregiver.id);
          setCaregiverBootstrapState("ready");
          return;
        }
        if (patientList.length > 0) {
          setCaregiverBootstrapState("failed");
          setCaregiverBootstrapError(
            "Select an existing patient to load its caregiver profile for this demo."
          );
          return;
        }
        return createCaregiver(caregiverInput).then((caregiver) => {
          setCaregiverId(caregiver.id);
          setCaregiverBootstrapState("ready");
        });
      })
      .catch((error) => {
        setCaregiverBootstrapState("failed");
        setCaregiverBootstrapError(
          error instanceof Error ? error.message : "Could not prepare the caregiver profile."
        );
      });
  }, [session, caregiverId, patientListLoading, caregiverBootstrapState, patientList.length, setCaregiverId]);

  useEffect(() => {
    if (session && patientId) {
      void fetchDashboard(patientId);
    }
  }, [session, patientId, fetchDashboard]);

  const retryCaregiverBootstrap = useCallback(() => {
    setCaregiverBootstrapError(null);
    setCaregiverBootstrapState("idle");
  }, []);

  const refreshCaregiverBootstrap = useCallback(() => {
    setCaregiverId("");
    setCaregiverBootstrapError(null);
    setCaregiverBootstrapState("idle");
  }, [setCaregiverId]);

  const canRetryCaregiverBootstrap =
    caregiverBootstrapState === "failed" && patientList.length === 0;

  async function handleLogin(event: FormEvent) {
    event.preventDefault();
    setIsLoggingIn(true);
    setLoginError(null);
    try {
      const nextSession = await loginAdmin(loginForm);
      setSession(nextSession);
      setCaregiverBootstrapError(null);
      setCaregiverBootstrapState("idle");
    } catch (error) {
      setLoginError(error instanceof Error ? error.message : "Login failed");
    } finally {
      setIsLoggingIn(false);
    }
  }

  function selectPatient(id: string) {
    setPatientId(id);
    setCurrentPage("dashboard");
  }

  async function handleSavePatient(input: PatientInput) {
    if (!patientId) {
      throw new Error("No patient selected.");
    }
    const updated = await updatePatient(patientId, input);
    setPatientList((current) => current.map((patient) => (patient.id === updated.id ? updated : patient)));
    await fetchDashboard(patientId);
  }

  async function handleSaveCaregiver(input: CaregiverInput) {
    const activeCaregiverId = dashboard?.caregiver.id || caregiverId;
    if (!activeCaregiverId) {
      throw new Error("Caregiver profile is not ready yet.");
    }
    await updateCaregiver(activeCaregiverId, input);
    if (patientId) {
      await fetchDashboard(patientId);
    }
  }

  const currentPatient =
    patientList.find((patient) => patient.id === patientId) ?? dashboard?.patient ?? null;

  if (!authChecked) {
    return (
      <div className="flex h-screen items-center justify-center text-sm text-slate-500">
        Loading caregiver workspace...
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex min-h-screen items-center justify-center px-4 py-10">
        <div className="app-panel grid w-full max-w-5xl overflow-hidden lg:grid-cols-[1.1fr_0.9fr]">
          <section className="border-b border-white/70 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.14),_transparent_38%),linear-gradient(180deg,_rgba(255,255,255,0.95),_rgba(245,247,250,0.88))] p-8 md:p-10 lg:border-b-0 lg:border-r">
            <div className="mb-10 flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-950 shadow-[0_18px_35px_rgba(15,23,42,0.18)]">
              <Radio size={18} className="text-white" />
            </div>
            <p className="eyebrow mb-2">Tether</p>
            <h1 className="max-w-xl text-4xl font-semibold tracking-tight text-slate-950">
              A calmer caregiver console for memory-forward check-ins
            </h1>
            <p className="mt-4 max-w-xl text-base leading-7 text-slate-600">
              Review the last call, polish the memory bank, and keep patient context ready before the next conversation begins.
            </p>
            <div className="mt-10 grid gap-4 sm:grid-cols-3">
              <HeroStat label="Profiles" value="Patient + caregiver" />
              <HeroStat label="Insights" value="Call analysis" />
              <HeroStat label="Actions" value="Memory + people edits" />
            </div>
          </section>
          <section className="bg-white/86 p-8 md:p-10">
            <p className="eyebrow mb-2">Sign In</p>
            <h2 className="text-2xl font-semibold text-slate-950">Open the caregiver workspace</h2>
            <p className="mt-2 text-sm leading-6 text-slate-500">
              Use the local demo credentials configured for the app. Your session stays in the browser.
            </p>
            <form onSubmit={handleLogin} className="mt-8 space-y-4">
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-800">Username</span>
                <input
                  type="text"
                  value={loginForm.username}
                  onChange={(event) =>
                    setLoginForm((current) => ({ ...current, username: event.target.value }))
                  }
                  className={authFieldClass}
                  autoFocus
                />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-800">Password</span>
                <input
                  type="password"
                  value={loginForm.password}
                  onChange={(event) =>
                    setLoginForm((current) => ({ ...current, password: event.target.value }))
                  }
                  className={authFieldClass}
                />
              </label>
              {loginError && (
                <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
                  {loginError}
                </div>
              )}
              <button type="submit" disabled={isLoggingIn} className="app-btn-primary w-full justify-center">
                {isLoggingIn ? "Signing in..." : "Sign in"}
              </button>
            </form>
          </section>
        </div>
      </div>
    );
  }

  if (!patientId) {
    if (preScreen === "create-patient") {
      return (
        <div className="min-h-screen overflow-y-auto">
          <CreatePatient
            caregiverId={caregiverId}
            caregiverBootstrapError={caregiverBootstrapError}
            canRetryCaregiverBootstrap={canRetryCaregiverBootstrap}
            onRetryCaregiverBootstrap={retryCaregiverBootstrap}
            onInvalidCaregiverReference={refreshCaregiverBootstrap}
            onCreated={(patient) => {
              setPatientList((current) => [...current, patient]);
              selectPatient(patient.id);
            }}
            onCancel={() => setPreScreen("picker")}
          />
        </div>
      );
    }

    return (
      <div className="flex min-h-screen items-center justify-center px-4 py-10">
        <div className="app-panel w-full max-w-4xl overflow-hidden">
          <div className="grid gap-8 p-8 md:grid-cols-[1fr_1.1fr] md:p-10">
            <section>
              <div className="mb-10 flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-950 shadow-[0_18px_35px_rgba(15,23,42,0.18)]">
                <Radio size={18} className="text-white" />
              </div>
              <p className="eyebrow mb-2">Patient Selection</p>
              <h2 className="text-3xl font-semibold tracking-tight text-slate-950">
                Choose who you want to support today
              </h2>
              <p className="mt-4 text-sm leading-7 text-slate-500">
                Open an existing dashboard or create a new profile with photo, memory cues, and caregiver context in one pass.
              </p>
              {caregiverBootstrapError && (
                <div className="mt-6 rounded-[28px] border border-rose-200 bg-rose-50/90 px-5 py-4 text-sm text-rose-700">
                  <p>{caregiverBootstrapError}</p>
                  {canRetryCaregiverBootstrap && (
                    <button
                      type="button"
                      onClick={retryCaregiverBootstrap}
                      className="mt-3 font-semibold text-rose-800 underline underline-offset-4"
                    >
                      Retry caregiver setup
                    </button>
                  )}
                </div>
              )}
            </section>
            <section className="space-y-4">
              {patientListLoading ? (
                <div className="app-panel-muted px-5 py-6 text-sm text-slate-500">Loading patients…</div>
              ) : (
                <>
                  {patientList.map((patient) => (
                    <button
                      key={patient.id}
                      type="button"
                      onClick={() => selectPatient(patient.id)}
                      className="app-panel-muted flex w-full items-center justify-between gap-4 p-4 text-left transition-transform hover:-translate-y-0.5"
                    >
                      <div className="flex items-center gap-4">
                        <Avatar
                          name={patient.preferredName || patient.displayName}
                          imageUrl={patient.profilePhotoDataUrl}
                          size="md"
                          accent="sky"
                        />
                        <div>
                          <p className="text-base font-semibold text-slate-950">{patient.displayName}</p>
                          <p className="mt-1 text-sm text-slate-500">
                            {[patient.phoneE164, patient.timezone].filter(Boolean).join(" · ") || "No contact details yet"}
                          </p>
                        </div>
                      </div>
                      <span
                        className={[
                          "rounded-full px-3 py-1 text-xs font-semibold",
                          patient.callingState === "active"
                            ? "bg-emerald-50 text-emerald-700"
                            : "bg-slate-100 text-slate-500"
                        ].join(" ")}
                      >
                        {patient.callingState === "active" ? "Calls active" : "Paused"}
                      </span>
                    </button>
                  ))}
                  {patientList.length === 0 && (
                    <div className="app-panel-muted px-5 py-7 text-center">
                      <p className="text-base font-semibold text-slate-800">No patients yet</p>
                      <p className="mt-2 text-sm text-slate-500">
                        Start by creating a patient profile with care notes, memory cues, and a photo.
                      </p>
                    </div>
                  )}
                  <button
                    type="button"
                    onClick={() => setPreScreen("create-patient")}
                    className="app-btn-secondary w-full justify-center"
                  >
                    <UserPlus size={15} strokeWidth={2.1} />
                    Add a new patient
                  </button>
                </>
              )}
            </section>
          </div>
        </div>
      </div>
    );
  }

  function renderPage() {
    switch (currentPage) {
      case "dashboard":
        return (
          <Dashboard
            onNavigate={setCurrentPage}
            dashboard={dashboard}
            isLoading={isDashboardLoading}
            error={dashboardError}
            onRefresh={() => void fetchDashboard(patientId)}
          />
        );
      case "schedule-call":
        return (
          <ScheduleCall
            patientId={patientId}
            patient={dashboard?.patient ?? null}
            onScheduleUpdated={() => {
              void fetchDashboard(patientId);
            }}
            onCallStarted={() => {
              void fetchDashboard(patientId);
              setCurrentPage("recent-calls");
            }}
          />
        );
      case "recent-calls":
        return (
          <RecentCalls
            recentCalls={dashboard?.recentCalls ?? []}
            latestAnalysis={dashboard?.latestAnalysis}
          />
        );
      case "settings":
        return (
          <PatientSettings
            patientId={patientId}
            patient={dashboard?.patient ?? currentPatient}
            caregiver={dashboard?.caregiver ?? null}
            onSavePatient={handleSavePatient}
            onSaveCaregiver={handleSaveCaregiver}
            onRefresh={async () => {
              await loadPatients();
              await fetchDashboard(patientId);
            }}
          />
        );
    }
  }

  return (
    <div className="flex h-screen overflow-hidden bg-transparent">
      <Sidebar
        currentPage={currentPage}
        onNavigate={setCurrentPage}
        collapsed={collapsed}
        onToggleCollapse={() => setCollapsed((current) => !current)}
        patientSwitcher={
          !collapsed ? (
            <div className="flex items-center justify-between gap-3 rounded-[24px] bg-slate-50/80 p-3">
              <div className="flex min-w-0 items-center gap-3">
                <Avatar
                  name={currentPatient?.preferredName || currentPatient?.displayName || "Patient"}
                  imageUrl={currentPatient?.profilePhotoDataUrl}
                  size="sm"
                  accent="sage"
                />
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-slate-900">
                    {currentPatient?.displayName ?? "Patient"}
                  </p>
                  <p className="truncate text-xs text-slate-500">
                    {currentPatient?.timezone ?? "Care profile"}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => {
                  setPatientId("");
                  setDashboard(null);
                  setPreScreen("picker");
                  setCurrentPage("dashboard");
                }}
                className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium text-slate-500 transition-colors hover:bg-white hover:text-slate-900"
              >
                <LogOut size={12} strokeWidth={2} />
                Change
              </button>
            </div>
          ) : undefined
        }
      />
      <main className="flex-1 overflow-y-auto bg-transparent">{renderPage()}</main>
    </div>
  );
}

function HeroStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[22px] border border-slate-200 bg-white/86 px-4 py-3 shadow-[0_12px_24px_rgba(15,23,42,0.05)]">
      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-400">{label}</p>
      <p className="mt-1 text-sm font-semibold text-slate-900">{value}</p>
    </div>
  );
}

const authFieldClass =
  "w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 shadow-[0_10px_24px_rgba(15,23,42,0.04)] outline-none transition focus:border-slate-300 focus:ring-4 focus:ring-sky-100";
