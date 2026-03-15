import { useState, useEffect, useCallback } from "react";
import { Radio, ChevronDown } from "lucide-react";
import { Sidebar } from "./components/Sidebar";
import { Dashboard } from "./pages/Dashboard";
import { Patients } from "./pages/Patients";
import { ScheduleCall } from "./pages/ScheduleCall";
import { RecentCalls } from "./pages/RecentCalls";
import { ApiSurface } from "./pages/ApiSurface";
import { getAdminSession, loginAdmin, getDashboard, listPatients } from "./api/admin";
import { useStoredString } from "./app/storage";
import { STORAGE_KEYS } from "./app/constants";
import type { AdminSession, DashboardSnapshot, Patient } from "./api/contracts";

export type Page = "dashboard" | "patients" | "schedule-call" | "recent-calls" | "api-surface";

export default function App() {
  const [currentPage, setCurrentPage] = useState<Page>("dashboard");
  const [collapsed, setCollapsed] = useState(false);

  // Auth
  const [session, setSession] = useState<AdminSession | null>(null);
  const [authChecked, setAuthChecked] = useState(false);
  const [loginForm, setLoginForm] = useState({ username: "", password: "" });
  const [loginError, setLoginError] = useState<string | null>(null);
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  // Patient selection
  const [patientId, setPatientId] = useStoredString(STORAGE_KEYS.patientId);
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

  // Load patient list after login
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

  if (!authChecked) {
    return (
      <div className="flex h-screen items-center justify-center text-gray-400 text-sm">
        Loading...
      </div>
    );
  }

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

  // Patient picker — shown on first load or when no patient selected
  if (!patientId) {
    return (
      <div className="flex h-screen items-center justify-center bg-[#f7f8fa]">
        <div className="bg-white border border-gray-200 rounded-2xl p-8 w-full max-w-sm">
          <div className="w-9 h-9 rounded-xl bg-gray-900 flex items-center justify-center mb-4">
            <Radio size={16} className="text-white" />
          </div>
          <h2 className="text-lg font-semibold text-gray-900 mb-1">Select a patient</h2>
          <p className="text-sm text-gray-400 mb-6">Choose the patient you'd like to manage.</p>

          {patientListLoading ? (
            <p className="text-sm text-gray-400">Loading patients...</p>
          ) : patientList.length === 0 ? (
            <p className="text-sm text-gray-400 italic">No patients found. Create one first.</p>
          ) : (
            <div className="space-y-2">
              {patientList.map((p) => (
                <button
                  key={p.id}
                  onClick={() => setPatientId(p.id)}
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
            </div>
          )}
        </div>
      </div>
    );
  }

  // Patient switcher in the sidebar header area (when multiple patients exist)
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
          patientList.length > 1 ? (
            <button
              onClick={() => {
                setPatientId("");
                setDashboard(null);
              }}
              className="w-full flex items-center gap-2 px-2.5 py-2 rounded-md text-xs text-gray-500 hover:bg-gray-50 hover:text-gray-800 transition-colors"
            >
              <span className="truncate flex-1 text-left">
                {currentPatient?.displayName ?? "Switch patient"}
              </span>
              <ChevronDown size={12} className="flex-shrink-0" />
            </button>
          ) : undefined
        }
      />
      <main className="flex-1 overflow-y-auto">{renderPage()}</main>
    </div>
  );
}
