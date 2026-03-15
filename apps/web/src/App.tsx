import { useState, useEffect, useCallback } from "react";
import { Radio, UserPlus, LogOut } from "lucide-react";
import { Sidebar } from "./components/Sidebar";
import { Dashboard } from "./pages/Dashboard";
import { Patients } from "./pages/Patients";
import { ScheduleCall } from "./pages/ScheduleCall";
import { RecentCalls } from "./pages/RecentCalls";
import { ApiSurface } from "./pages/ApiSurface";
import { CreatePatient } from "./pages/CreatePatient";
import { getAdminSession, loginAdmin, getDashboard, listPatients } from "./api/admin";
import { useStoredString } from "./app/storage";
import { STORAGE_KEYS } from "./app/constants";
import type { AdminSession, DashboardSnapshot, Patient } from "./api/contracts";

export type Page = "dashboard" | "patients" | "schedule-call" | "recent-calls" | "api-surface";

// Screens outside the main console
type PreConsoleScreen = "picker" | "create-patient";

export default function App() {
  const [currentPage, setCurrentPage] = useState<Page>("dashboard");
  const [collapsed, setCollapsed] = useState(false);

  // Auth
  const [session, setSession] = useState<AdminSession | null>(null);
  const [authChecked, setAuthChecked] = useState(false);
  const [loginForm, setLoginForm] = useState({ username: "", password: "" });
  const [loginError, setLoginError] = useState<string | null>(null);
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  // Pre-console screen
  const [preScreen, setPreScreen] = useState<PreConsoleScreen>("picker");

  // Patient selection
  const [patientId, setPatientId] = useStoredString(STORAGE_KEYS.patientId);
  const [caregiverId, setCaregiverId] = useStoredString(STORAGE_KEYS.caregiverId);
  const [patientList, setPatientList] = useState<Patient[]>([]);
  const [patientListLoading, setPatientListLoading] = useState(false);

  // Dashboard data
  const [dashboard, setDashboard] = useState<DashboardSnapshot | null>(null);
  const [dashboardError, setDashboardError] = useState<string | null>(null);
  const [isDashboardLoading, setIsDashboardLoading] = useState(false);

  useEffect(() => {
    getAdminSession()
      .then(setSession)
      .catch(() => setSession(null))
      .finally(() => setAuthChecked(true));
  }, []);

  useEffect(() => {
    if (!session) return;
    setPatientListLoading(true);
    listPatients()
      .then(setPatientList)
      .catch(() => setPatientList([]))
      .finally(() => setPatientListLoading(false));
  }, [session]);

  const fetchDashboard = useCallback(async (pid: string) => {
    setIsDashboardLoading(true);
    setDashboardError(null);
    try {
      const data = await getDashboard(pid);
      setDashboard(data);
      if (data.caregiver?.id) setCaregiverId(data.caregiver.id);
    } catch (err: any) {
      setDashboardError(err.message ?? "Failed to load dashboard");
    } finally {
      setIsDashboardLoading(false);
    }
  }, []);

  useEffect(() => {
    if (session && patientId) {
      fetchDashboard(patientId);
    }
  }, [session, patientId, fetchDashboard]);

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setIsLoggingIn(true);
    setLoginError(null);
    try {
      const s = await loginAdmin(loginForm);
      setSession(s);
    } catch (err: any) {
      setLoginError(err.message ?? "Login failed");
    } finally {
      setIsLoggingIn(false);
    }
  }

  function selectPatient(id: string) {
    setPatientId(id);
    setCurrentPage("dashboard");
  }

  // ── Loading ──────────────────────────────────────────────────────
  if (!authChecked) {
    return (
      <div className="flex h-screen items-center justify-center text-gray-400 text-sm">
        Loading...
      </div>
    );
  }

  // ── Login ────────────────────────────────────────────────────────
  if (!session) {
    return (
      <div className="flex h-screen items-center justify-center bg-[#f7f8fa]">
        <div className="bg-white border border-gray-200 rounded-2xl p-8 w-full max-w-sm">
          <div className="mb-6">
            <div className="w-9 h-9 rounded-xl bg-gray-900 flex items-center justify-center mb-4">
              <Radio size={16} className="text-white" />
            </div>
            <h1 className="text-xl font-semibold text-gray-900">Nova Echoes</h1>
            <p className="text-sm text-gray-400 mt-1">Sign in to your caregiver account</p>
          </div>
          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1.5">Username</label>
              <input
                type="text"
                value={loginForm.username}
                onChange={(e) => setLoginForm((f) => ({ ...f, username: e.target.value }))}
                className="w-full px-3 py-2 text-sm bg-white border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1.5">Password</label>
              <input
                type="password"
                value={loginForm.password}
                onChange={(e) => setLoginForm((f) => ({ ...f, password: e.target.value }))}
                className="w-full px-3 py-2 text-sm bg-white border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
              />
            </div>
            {loginError && <p className="text-sm text-red-600">{loginError}</p>}
            <button
              type="submit"
              disabled={isLoggingIn}
              className="w-full py-2.5 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50"
            >
              {isLoggingIn ? "Signing in..." : "Sign in"}
            </button>
          </form>
        </div>
      </div>
    );
  }

  // ── Patient picker / Add patient (pre-console screens) ───────────
  if (!patientId) {
    if (preScreen === "create-patient") {
      return (
        <div className="min-h-screen bg-[#f7f8fa] overflow-y-auto">
          <CreatePatient
            caregiverId={caregiverId}
            onCreated={(patient) => {
              setPatientList((prev) => [...prev, patient]);
              selectPatient(patient.id);
            }}
            onCancel={() => setPreScreen("picker")}
          />
        </div>
      );
    }

    // Patient picker
    return (
      <div className="flex h-screen items-center justify-center bg-[#f7f8fa]">
        <div className="bg-white border border-gray-200 rounded-2xl p-8 w-full max-w-sm">
          <div className="w-9 h-9 rounded-xl bg-gray-900 flex items-center justify-center mb-4">
            <Radio size={16} className="text-white" />
          </div>
          <h2 className="text-lg font-semibold text-gray-900 mb-1">Select a patient</h2>
          <p className="text-sm text-gray-400 mb-6">
            Choose the patient you'd like to manage.
          </p>

          {patientListLoading ? (
            <p className="text-sm text-gray-400">Loading...</p>
          ) : (
            <div className="space-y-2">
              {patientList.map((p) => (
                <button
                  key={p.id}
                  onClick={() => selectPatient(p.id)}
                  className="w-full flex items-center justify-between px-4 py-3 bg-white border border-gray-200 rounded-xl hover:border-gray-900 hover:bg-gray-50 transition-colors text-left"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-900">{p.displayName}</p>
                    <p className="text-xs text-gray-400 mt-0.5">{p.phoneE164 ?? "No phone"}</p>
                  </div>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                      p.callingState === "active"
                        ? "bg-green-50 text-green-700"
                        : "bg-gray-100 text-gray-500"
                    }`}
                  >
                    {p.callingState}
                  </span>
                </button>
              ))}

              {patientList.length === 0 && (
                <p className="text-sm text-gray-400 italic text-center py-2">No patients yet.</p>
              )}

              <button
                onClick={() => setPreScreen("create-patient")}
                className="w-full flex items-center justify-center gap-2 py-2.5 border border-dashed border-gray-300 rounded-xl text-sm text-gray-500 hover:border-gray-900 hover:text-gray-900 transition-colors mt-1"
              >
                <UserPlus size={14} strokeWidth={2} />
                Add a new patient
              </button>
            </div>
          )}
        </div>
      </div>
    );
  }

  // ── Console ──────────────────────────────────────────────────────
  const currentPatient = patientList.find((p) => p.id === patientId);

  function renderPage() {
    switch (currentPage) {
      case "dashboard":
        return (
          <Dashboard
            onNavigate={setCurrentPage}
            dashboard={dashboard}
            isLoading={isDashboardLoading}
            error={dashboardError}
            onRefresh={() => fetchDashboard(patientId)}
          />
        );
      case "patients":
        return (
          <Patients
            patient={dashboard?.patient ?? null}
            onScheduleCall={() => setCurrentPage("schedule-call")}
          />
        );
      case "schedule-call":
        return (
          <ScheduleCall
            patientId={patientId}
            patient={dashboard?.patient ?? null}
            onCallStarted={() => {
              fetchDashboard(patientId);
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
      case "api-surface":
        return <ApiSurface />;
    }
  }

  return (
    <div className="flex h-screen overflow-hidden bg-[#f7f8fa]">
      <Sidebar
        currentPage={currentPage}
        onNavigate={setCurrentPage}
        collapsed={collapsed}
        onToggleCollapse={() => setCollapsed(!collapsed)}
        patientSwitcher={
          !collapsed ? (
            <div className="px-2.5 py-2">
              <p className="text-xs text-gray-400 truncate mb-1">
                {currentPatient?.displayName ?? "Patient"}
              </p>
              <button
                onClick={() => {
                  setPatientId("");
                  setDashboard(null);
                  setPreScreen("picker");
                  setCurrentPage("dashboard");
                }}
                className="flex items-center gap-1.5 text-xs text-gray-400 hover:text-gray-700 transition-colors"
              >
                <LogOut size={11} strokeWidth={2} />
                Change patient
              </button>
            </div>
          ) : undefined
        }
      />
      <main className="flex-1 overflow-y-auto">{renderPage()}</main>
    </div>
  );
}
