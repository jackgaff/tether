import { useEffect, useState, type FormEvent } from "react";
import { CALL_TYPES, CONSENT_STATUSES, STORAGE_KEYS } from "./app/constants";
import {
  buildNextCallInput,
  createCaregiverForm,
  createConsentForm,
  createNextCallForm,
  createPatientForm,
  formatError,
  getDefaultTimezone,
  patientFormToInput,
  patientToForm,
  type NextCallFormState
} from "./app/forms";
import { useStoredString } from "./app/storage";
import {
  analyzeCall,
  createCaregiver,
  createPatient,
  createPatientCall,
  getAdminSession,
  getCall,
  getCallAnalysis,
  getCaregiver,
  getConsent,
  getDashboard,
  getNextCall,
  getPatient,
  listCallTemplates,
  loginAdmin,
  logoutAdmin,
  pausePatient,
  unpausePatient,
  updateCaregiver,
  updateConsent,
  updateNextCall,
  updatePatient
} from "./api/admin";
import { ApiError, apiBaseUrl, fetchHealth } from "./api/client";
import type {
  AdminSession,
  AnalysisRecord,
  CallRunDetail,
  CallTemplate,
  Caregiver,
  CaregiverInput,
  ConsentInput,
  ConsentState,
  CreatePatientCallInput,
  DashboardSnapshot,
  HealthSnapshot,
  NextCallPlan,
  Patient,
  PatientInput,
  UpdateNextCallInput,
  VoiceSessionDescriptor
} from "./api/contracts";
import { ErrorText } from "./components/ErrorText";
import { JsonView } from "./components/JsonView";
import { LiveCallPanel } from "./components/LiveCallPanel";
import { Section } from "./components/Section";

export default function App() {
  const defaultTimezone = getDefaultTimezone();

  const [health, setHealth] = useState<HealthSnapshot | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);

  const [session, setSession] = useState<AdminSession | null>(null);
  const [sessionForm, setSessionForm] = useState({
    username: "",
    password: ""
  });
  const [sessionMessage, setSessionMessage] = useState("Checking admin session...");
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [isSessionBusy, setIsSessionBusy] = useState(false);

  const [caregiverId, setCaregiverId] = useStoredString(STORAGE_KEYS.caregiverId);
  const [caregiver, setCaregiver] = useState<Caregiver | null>(null);
  const [caregiverForm, setCaregiverForm] = useState<CaregiverInput>(
    createCaregiverForm(defaultTimezone)
  );
  const [caregiverMessage, setCaregiverMessage] = useState("No caregiver loaded.");
  const [caregiverError, setCaregiverError] = useState<string | null>(null);
  const [isCaregiverBusy, setIsCaregiverBusy] = useState(false);

  const [patientId, setPatientId] = useStoredString(STORAGE_KEYS.patientId);
  const [patient, setPatient] = useState<Patient | null>(null);
  const [patientForm, setPatientForm] = useState(() =>
    createPatientForm(defaultTimezone, caregiverId)
  );
  const [patientMessage, setPatientMessage] = useState("No patient loaded.");
  const [patientError, setPatientError] = useState<string | null>(null);
  const [isPatientBusy, setIsPatientBusy] = useState(false);

  const [consent, setConsent] = useState<ConsentState | null>(null);
  const [consentForm, setConsentForm] = useState<ConsentInput>(createConsentForm());
  const [pauseReason, setPauseReason] = useState("");
  const [consentMessage, setConsentMessage] = useState("No consent loaded.");
  const [consentError, setConsentError] = useState<string | null>(null);
  const [isConsentBusy, setIsConsentBusy] = useState(false);

  const [templates, setTemplates] = useState<CallTemplate[]>([]);
  const [templatesMessage, setTemplatesMessage] = useState("Log in to load call templates.");
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [isTemplatesBusy, setIsTemplatesBusy] = useState(false);

  const [dashboard, setDashboard] = useState<DashboardSnapshot | null>(null);
  const [dashboardMessage, setDashboardMessage] = useState("No dashboard loaded.");
  const [dashboardError, setDashboardError] = useState<string | null>(null);
  const [isDashboardBusy, setIsDashboardBusy] = useState(false);

  const [callId, setCallId] = useStoredString(STORAGE_KEYS.callId);
  const [callForm, setCallForm] = useState({
    callType: "orientation" as (typeof CALL_TYPES)[number],
    callTemplateId: ""
  });
  const [activeVoiceSession, setActiveVoiceSession] = useState<VoiceSessionDescriptor | null>(
    null
  );
  const [callDetail, setCallDetail] = useState<CallRunDetail | null>(null);
  const [callMessage, setCallMessage] = useState("No call loaded.");
  const [callError, setCallError] = useState<string | null>(null);
  const [isCallBusy, setIsCallBusy] = useState(false);

  const [analysis, setAnalysis] = useState<AnalysisRecord | null>(null);
  const [analysisMessage, setAnalysisMessage] = useState("No analysis loaded.");
  const [analysisError, setAnalysisError] = useState<string | null>(null);
  const [isAnalysisBusy, setIsAnalysisBusy] = useState(false);

  const [nextCallPlan, setNextCallPlan] = useState<NextCallPlan | null>(null);
  const [nextCallForm, setNextCallForm] = useState<NextCallFormState>(
    createNextCallForm()
  );
  const [nextCallMessage, setNextCallMessage] = useState("No next-call plan loaded.");
  const [nextCallError, setNextCallError] = useState<string | null>(null);
  const [isNextCallBusy, setIsNextCallBusy] = useState(false);
  const isEditingNextCall = nextCallForm.action === "edit";
  const canEditNextCallReason =
    nextCallForm.action === "edit" ||
    nextCallForm.action === "reject" ||
    nextCallForm.action === "cancel";

  useEffect(() => {
    void loadHealth();
    void refreshSession(true);
  }, []);

  async function loadHealth() {
    setHealthError(null);

    try {
      setHealth(await fetchHealth());
    } catch (error) {
      setHealth(null);
      setHealthError(formatError(error));
    }
  }

  async function refreshSession(isBoot = false) {
    setIsSessionBusy(true);
    setSessionError(null);

    try {
      const nextSession = await getAdminSession();
      setSession(nextSession);
      setSessionMessage(`Logged in as ${nextSession.username}.`);
      await loadTemplates(true);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setSession(null);
        setSessionMessage("Logged out.");
        setTemplates([]);
        setTemplatesMessage("Log in to load call templates.");
        setTemplatesError(null);
        if (!isBoot) {
          setSessionError(null);
        }
      } else {
        setSession(null);
        setSessionError(formatError(error));
        setSessionMessage("Could not verify the admin session.");
      }
    } finally {
      setIsSessionBusy(false);
    }
  }

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSessionBusy(true);
    setSessionError(null);

    try {
      const nextSession = await loginAdmin(sessionForm);
      setSession(nextSession);
      setSessionMessage(`Logged in as ${nextSession.username}.`);
      await loadTemplates(true);
    } catch (error) {
      setSession(null);
      setSessionError(formatError(error));
      setSessionMessage("Login failed.");
    } finally {
      setIsSessionBusy(false);
    }
  }

  async function handleLogout() {
    setIsSessionBusy(true);
    setSessionError(null);

    try {
      await logoutAdmin();
      setSession(null);
      setSessionMessage("Logged out.");
      setTemplates([]);
      setTemplatesMessage("Log in to load call templates.");
      setActiveVoiceSession(null);
    } catch (error) {
      setSessionError(formatError(error));
      setSessionMessage("Logout failed.");
    } finally {
      setIsSessionBusy(false);
    }
  }

  async function loadTemplates(silentUnauthorized = false) {
    setIsTemplatesBusy(true);
    setTemplatesError(null);

    try {
      const nextTemplates = await listCallTemplates();
      setTemplates(nextTemplates);
      setTemplatesMessage(`Loaded ${nextTemplates.length} call template(s).`);
    } catch (error) {
      if (silentUnauthorized && error instanceof ApiError && error.status === 401) {
        setTemplates([]);
        setTemplatesMessage("Log in to load call templates.");
        setTemplatesError(null);
      } else {
        setTemplates([]);
        setTemplatesMessage("Could not load call templates.");
        setTemplatesError(formatError(error));
      }
    } finally {
      setIsTemplatesBusy(false);
    }
  }

  function applyCaregiver(nextCaregiver: Caregiver) {
    setCaregiver(nextCaregiver);
    setCaregiverId(nextCaregiver.id);
    setCaregiverForm({
      displayName: nextCaregiver.displayName,
      email: nextCaregiver.email,
      phoneE164: nextCaregiver.phoneE164 ?? "",
      timezone: nextCaregiver.timezone
    });
    setPatientForm((current) => ({
      ...current,
      primaryCaregiverId: nextCaregiver.id
    }));
  }

  function applyPatient(nextPatient: Patient) {
    setPatient(nextPatient);
    setPatientId(nextPatient.id);
    setPatientForm(patientToForm(nextPatient));
    setPauseReason(nextPatient.pauseReason ?? "");
  }

  function applyConsent(nextConsent: ConsentState) {
    setConsent(nextConsent);
    setConsentForm({
      outboundCallStatus: nextConsent.outboundCallStatus,
      transcriptStorageStatus: nextConsent.transcriptStorageStatus,
      notes: nextConsent.notes ?? ""
    });
  }

  function applyCallDetail(nextCallDetail: CallRunDetail) {
    setCallDetail(nextCallDetail);
    setCallId(nextCallDetail.callRun.id);
    if (nextCallDetail.analysis) {
      setAnalysis(nextCallDetail.analysis);
      setAnalysisMessage(`Loaded saved analysis for ${nextCallDetail.callRun.id}.`);
      setAnalysisError(null);
      return;
    }

    clearAnalysisState("No saved analysis found for this call.");
  }

  function applyNextCallPlan(nextPlan: NextCallPlan) {
    setNextCallPlan(nextPlan);
    setNextCallForm(createNextCallForm(nextPlan));
    setNextCallError(null);
  }

  function clearAnalysisState(message = "No analysis loaded.") {
    setAnalysis(null);
    setAnalysisMessage(message);
    setAnalysisError(null);
  }

  function clearNextCallState(message = "No next-call plan loaded.") {
    setNextCallPlan(null);
    setNextCallForm(createNextCallForm());
    setNextCallMessage(message);
    setNextCallError(null);
  }

  async function handleCreateCaregiver(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsCaregiverBusy(true);
    setCaregiverError(null);

    try {
      const nextCaregiver = await createCaregiver(caregiverForm);
      applyCaregiver(nextCaregiver);
      setCaregiverMessage(`Created caregiver ${nextCaregiver.id}.`);
    } catch (error) {
      setCaregiverError(formatError(error));
      setCaregiverMessage("Could not create caregiver.");
    } finally {
      setIsCaregiverBusy(false);
    }
  }

  async function handleLoadCaregiver() {
    if (!caregiverId.trim()) {
      setCaregiverError("Enter a caregiver ID first.");
      return;
    }

    setIsCaregiverBusy(true);
    setCaregiverError(null);

    try {
      const nextCaregiver = await getCaregiver(caregiverId.trim());
      applyCaregiver(nextCaregiver);
      setCaregiverMessage(`Loaded caregiver ${nextCaregiver.id}.`);
    } catch (error) {
      setCaregiverError(formatError(error));
      setCaregiverMessage("Could not load caregiver.");
    } finally {
      setIsCaregiverBusy(false);
    }
  }

  async function handleUpdateCaregiver(event?: FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    if (!caregiverId.trim()) {
      setCaregiverError("Enter or create a caregiver ID first.");
      return;
    }

    setIsCaregiverBusy(true);
    setCaregiverError(null);

    try {
      const nextCaregiver = await updateCaregiver(caregiverId.trim(), caregiverForm);
      applyCaregiver(nextCaregiver);
      setCaregiverMessage(`Updated caregiver ${nextCaregiver.id}.`);
    } catch (error) {
      setCaregiverError(formatError(error));
      setCaregiverMessage("Could not update caregiver.");
    } finally {
      setIsCaregiverBusy(false);
    }
  }

  async function handleCreatePatient(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsPatientBusy(true);
    setPatientError(null);

    try {
      const nextPatient = await createPatient(patientFormToInput(patientForm));
      applyPatient(nextPatient);
      setPatientMessage(`Created patient ${nextPatient.id}.`);
    } catch (error) {
      setPatientError(formatError(error));
      setPatientMessage("Could not create patient.");
    } finally {
      setIsPatientBusy(false);
    }
  }

  async function handleLoadPatient() {
    if (!patientId.trim()) {
      setPatientError("Enter a patient ID first.");
      return;
    }

    setIsPatientBusy(true);
    setPatientError(null);

    try {
      const nextPatient = await getPatient(patientId.trim());
      applyPatient(nextPatient);
      setPatientMessage(`Loaded patient ${nextPatient.id}.`);
    } catch (error) {
      setPatientError(formatError(error));
      setPatientMessage("Could not load patient.");
    } finally {
      setIsPatientBusy(false);
    }
  }

  async function handleUpdatePatient(event?: FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    if (!patientId.trim()) {
      setPatientError("Enter or create a patient ID first.");
      return;
    }

    setIsPatientBusy(true);
    setPatientError(null);

    try {
      const nextPatient = await updatePatient(patientId.trim(), patientFormToInput(patientForm));
      applyPatient(nextPatient);
      setPatientMessage(`Updated patient ${nextPatient.id}.`);
    } catch (error) {
      setPatientError(formatError(error));
      setPatientMessage("Could not update patient.");
    } finally {
      setIsPatientBusy(false);
    }
  }

  async function handleLoadConsent() {
    if (!patientId.trim()) {
      setConsentError("Enter a patient ID first.");
      return;
    }

    setIsConsentBusy(true);
    setConsentError(null);

    try {
      const nextConsent = await getConsent(patientId.trim());
      applyConsent(nextConsent);
      setConsentMessage(`Loaded consent for patient ${nextConsent.patientId}.`);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setConsent(null);
        setConsentMessage("No consent state found yet.");
        setConsentError(null);
      } else {
        setConsentError(formatError(error));
        setConsentMessage("Could not load consent.");
      }
    } finally {
      setIsConsentBusy(false);
    }
  }

  async function handleUpdateConsent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!patientId.trim()) {
      setConsentError("Enter a patient ID first.");
      return;
    }

    setIsConsentBusy(true);
    setConsentError(null);

    try {
      const nextConsent = await updateConsent(patientId.trim(), consentForm);
      applyConsent(nextConsent);
      setConsentMessage("Updated consent state.");
    } catch (error) {
      setConsentError(formatError(error));
      setConsentMessage("Could not update consent.");
    } finally {
      setIsConsentBusy(false);
    }
  }

  async function handlePausePatient(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!patientId.trim()) {
      setConsentError("Enter a patient ID first.");
      return;
    }

    setIsConsentBusy(true);
    setConsentError(null);

    try {
      const nextPatient = await pausePatient(patientId.trim(), { reason: pauseReason });
      applyPatient(nextPatient);
      setConsentMessage("Paused outgoing calls for the patient.");
    } catch (error) {
      setConsentError(formatError(error));
      setConsentMessage("Could not pause patient.");
    } finally {
      setIsConsentBusy(false);
    }
  }

  async function handleUnpausePatient() {
    if (!patientId.trim()) {
      setConsentError("Enter a patient ID first.");
      return;
    }

    setIsConsentBusy(true);
    setConsentError(null);

    try {
      const nextPatient = await unpausePatient(patientId.trim());
      applyPatient(nextPatient);
      setConsentMessage("Resumed outgoing calls for the patient.");
    } catch (error) {
      setConsentError(formatError(error));
      setConsentMessage("Could not unpause patient.");
    } finally {
      setIsConsentBusy(false);
    }
  }

  async function handleLoadDashboard() {
    if (!patientId.trim()) {
      setDashboardError("Enter a patient ID first.");
      return;
    }

    setIsDashboardBusy(true);
    setDashboardError(null);

    try {
      const nextDashboard = await getDashboard(patientId.trim());
      setDashboard(nextDashboard);
      applyCaregiver(nextDashboard.caregiver);
      applyPatient(nextDashboard.patient);
      applyConsent(nextDashboard.consent);
      if (nextDashboard.latestAnalysis) {
        setAnalysis(nextDashboard.latestAnalysis);
        setAnalysisMessage(`Loaded latest analysis for patient ${nextDashboard.patient.id}.`);
        setAnalysisError(null);
      } else {
        clearAnalysisState("No analysis is available for this patient yet.");
      }
      if (nextDashboard.activeNextCallPlan) {
        applyNextCallPlan(nextDashboard.activeNextCallPlan);
        setNextCallMessage(`Loaded next-call plan ${nextDashboard.activeNextCallPlan.id}.`);
      } else {
        clearNextCallState("No active next-call plan found for this patient.");
      }
      setDashboardMessage(`Loaded dashboard for patient ${nextDashboard.patient.id}.`);
    } catch (error) {
      setDashboardError(formatError(error));
      setDashboardMessage("Could not load dashboard.");
    } finally {
      setIsDashboardBusy(false);
    }
  }

  async function handleCreateCall(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!patientId.trim()) {
      setCallError("Enter a patient ID first.");
      return;
    }

    const input: CreatePatientCallInput = {
      channel: "browser"
    };
    if (callForm.callTemplateId.trim()) {
      input.callTemplateId = callForm.callTemplateId.trim();
    } else {
      input.callType = callForm.callType;
    }

    setIsCallBusy(true);
    setCallError(null);

    try {
      const created = await createPatientCall(patientId.trim(), input);
      setCallId(created.callRun.id);
      setCallDetail({
        callRun: created.callRun,
        transcriptTurns: [],
        analysis: undefined
      });
      clearAnalysisState("No analysis loaded for the new call.");
      setActiveVoiceSession(created.voiceSession ?? null);
      setCallMessage(`Created call run ${created.callRun.id}.`);
    } catch (error) {
      setCallError(formatError(error));
      setCallMessage("Could not create the browser call.");
    } finally {
      setIsCallBusy(false);
    }
  }

  async function handleLoadCall() {
    if (!callId.trim()) {
      setCallError("Enter a call ID first.");
      return;
    }

    setIsCallBusy(true);
    setCallError(null);

    try {
      const nextCallDetail = await getCall(callId.trim());
      applyCallDetail(nextCallDetail);
      setCallMessage(`Loaded call ${nextCallDetail.callRun.id}.`);
    } catch (error) {
      setCallError(formatError(error));
      setCallMessage("Could not load call detail.");
    } finally {
      setIsCallBusy(false);
    }
  }

  async function handleAnalyzeCall() {
    if (!callId.trim()) {
      setAnalysisError("Enter a call ID first.");
      return;
    }

    setIsAnalysisBusy(true);
    setAnalysisError(null);

    try {
      const nextAnalysis = await analyzeCall(callId.trim());
      setAnalysis(nextAnalysis);
      setAnalysisMessage(`Analyzed call ${nextAnalysis.callRunId}.`);
      if (patientId.trim()) {
        await Promise.all([handleLoadDashboard(), handleLoadNextCall()]);
      }
    } catch (error) {
      setAnalysisError(formatError(error));
      setAnalysisMessage("Could not analyze the call.");
    } finally {
      setIsAnalysisBusy(false);
    }
  }

  async function handleLoadAnalysis() {
    if (!callId.trim()) {
      setAnalysisError("Enter a call ID first.");
      return;
    }

    setIsAnalysisBusy(true);
    setAnalysisError(null);

    try {
      const nextAnalysis = await getCallAnalysis(callId.trim());
      setAnalysis(nextAnalysis);
      setAnalysisMessage(`Loaded saved analysis for ${nextAnalysis.callRunId}.`);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        clearAnalysisState("No saved analysis found yet.");
      } else {
        setAnalysisError(formatError(error));
        setAnalysisMessage("Could not load saved analysis.");
      }
    } finally {
      setIsAnalysisBusy(false);
    }
  }

  async function handleLoadNextCall() {
    if (!patientId.trim()) {
      setNextCallError("Enter a patient ID first.");
      return;
    }

    setIsNextCallBusy(true);
    setNextCallError(null);

    try {
      const nextPlan = await getNextCall(patientId.trim());
      applyNextCallPlan(nextPlan);
      setNextCallMessage(`Loaded next-call plan ${nextPlan.id}.`);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        clearNextCallState("No active next-call plan found.");
      } else {
        setNextCallError(formatError(error));
        setNextCallMessage("Could not load the next-call plan.");
      }
    } finally {
      setIsNextCallBusy(false);
    }
  }

  async function handleUpdateNextCall(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!patientId.trim()) {
      setNextCallError("Enter a patient ID first.");
      return;
    }

    const input = buildNextCallInput(nextCallForm);
    if (input instanceof Error) {
      setNextCallError(input.message);
      return;
    }

    setIsNextCallBusy(true);
    setNextCallError(null);

    try {
      const nextPlan = await updateNextCall(patientId.trim(), input);
      applyNextCallPlan(nextPlan);
      setNextCallMessage(`Updated next-call plan ${nextPlan.id}.`);
    } catch (error) {
      setNextCallError(formatError(error));
      setNextCallMessage("Could not update the next-call plan.");
    } finally {
      setIsNextCallBusy(false);
    }
  }

  async function handleLiveSessionEnded() {
    setActiveVoiceSession(null);
    if (callId.trim()) {
      await handleLoadCall();
    }
  }

  return (
    <main className="app-shell">
      <h1>Nova Echoes Minimal Admin UI</h1>
      <p>
        <strong>API base URL:</strong> {apiBaseUrl}
      </p>

      <Section
        title="Health"
        actions={
          <button type="button" onClick={() => void loadHealth()}>
            Refresh health
          </button>
        }
      >
        <ErrorText message={healthError} />
        <JsonView value={health} emptyLabel="No health data loaded yet." />
      </Section>

      <Section
        title="Session"
        actions={
          <div className="button-row">
            <button
              type="button"
              onClick={() => void refreshSession(false)}
              disabled={isSessionBusy}
            >
              Check session
            </button>
            <button
              type="button"
              onClick={() => void handleLogout()}
              disabled={isSessionBusy}
            >
              Logout
            </button>
          </div>
        }
      >
        <p>{sessionMessage}</p>
        <ErrorText message={sessionError} />
        <form onSubmit={handleLogin}>
          <div className="field-grid">
            <label>
              Username
              <input
                type="text"
                value={sessionForm.username}
                onChange={(event) =>
                  setSessionForm((current) => ({
                    ...current,
                    username: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={sessionForm.password}
                onChange={(event) =>
                  setSessionForm((current) => ({
                    ...current,
                    password: event.target.value
                  }))
                }
              />
            </label>
          </div>
          <button type="submit" disabled={isSessionBusy}>
            {isSessionBusy ? "Working..." : "Login"}
          </button>
        </form>
        <JsonView value={session} emptyLabel="No active session." />
      </Section>

      <Section title="Caregiver">
        <p>{caregiverMessage}</p>
        <ErrorText message={caregiverError} />
        <label>
          Caregiver ID
          <input
            type="text"
            value={caregiverId}
            onChange={(event) => setCaregiverId(event.target.value)}
          />
        </label>
        <div className="button-row">
          <button type="button" onClick={() => void handleLoadCaregiver()} disabled={isCaregiverBusy}>
            Load caregiver
          </button>
        </div>
        <form onSubmit={handleCreateCaregiver}>
          <div className="field-grid">
            <label>
              Display name
              <input
                type="text"
                value={caregiverForm.displayName}
                onChange={(event) =>
                  setCaregiverForm((current) => ({
                    ...current,
                    displayName: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Email
              <input
                type="email"
                value={caregiverForm.email}
                onChange={(event) =>
                  setCaregiverForm((current) => ({
                    ...current,
                    email: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Phone E.164
              <input
                type="text"
                value={caregiverForm.phoneE164}
                onChange={(event) =>
                  setCaregiverForm((current) => ({
                    ...current,
                    phoneE164: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Timezone
              <input
                type="text"
                value={caregiverForm.timezone}
                onChange={(event) =>
                  setCaregiverForm((current) => ({
                    ...current,
                    timezone: event.target.value
                  }))
                }
              />
            </label>
          </div>
          <div className="button-row">
            <button type="submit" disabled={isCaregiverBusy}>
              Create caregiver
            </button>
            <button
              type="button"
              onClick={() => void handleUpdateCaregiver()}
              disabled={isCaregiverBusy}
            >
              Update caregiver
            </button>
          </div>
        </form>
        <JsonView value={caregiver} emptyLabel="No caregiver record." />
      </Section>

      <Section title="Patient">
        <p>{patientMessage}</p>
        <ErrorText message={patientError} />
        <label>
          Patient ID
          <input
            type="text"
            value={patientId}
            onChange={(event) => setPatientId(event.target.value)}
          />
        </label>
        <div className="button-row">
          <button type="button" onClick={() => void handleLoadPatient()} disabled={isPatientBusy}>
            Load patient
          </button>
        </div>
        <form onSubmit={handleCreatePatient}>
          <div className="field-grid">
            <label>
              Primary caregiver ID
              <input
                type="text"
                value={patientForm.primaryCaregiverId}
                onChange={(event) =>
                  setPatientForm((current) => ({
                    ...current,
                    primaryCaregiverId: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Display name
              <input
                type="text"
                value={patientForm.displayName}
                onChange={(event) =>
                  setPatientForm((current) => ({
                    ...current,
                    displayName: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Preferred name
              <input
                type="text"
                value={patientForm.preferredName}
                onChange={(event) =>
                  setPatientForm((current) => ({
                    ...current,
                    preferredName: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Phone E.164
              <input
                type="text"
                value={patientForm.phoneE164}
                onChange={(event) =>
                  setPatientForm((current) => ({
                    ...current,
                    phoneE164: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Timezone
              <input
                type="text"
                value={patientForm.timezone}
                onChange={(event) =>
                  setPatientForm((current) => ({
                    ...current,
                    timezone: event.target.value
                  }))
                }
              />
            </label>
          </div>

          <label>
            Notes
            <textarea
              rows={3}
              value={patientForm.notes}
              onChange={(event) =>
                setPatientForm((current) => ({
                  ...current,
                  notes: event.target.value
                }))
              }
            />
          </label>
          <label>
            Routine anchors (comma or newline separated)
            <textarea
              rows={2}
              value={patientForm.routineAnchors}
              onChange={(event) =>
                setPatientForm((current) => ({
                  ...current,
                  routineAnchors: event.target.value
                }))
              }
            />
          </label>
          <label>
            Favorite topics (comma or newline separated)
            <textarea
              rows={2}
              value={patientForm.favoriteTopics}
              onChange={(event) =>
                setPatientForm((current) => ({
                  ...current,
                  favoriteTopics: event.target.value
                }))
              }
            />
          </label>
          <label>
            Calming cues (comma or newline separated)
            <textarea
              rows={2}
              value={patientForm.calmingCues}
              onChange={(event) =>
                setPatientForm((current) => ({
                  ...current,
                  calmingCues: event.target.value
                }))
              }
            />
          </label>
          <label>
            Topics to avoid (comma or newline separated)
            <textarea
              rows={2}
              value={patientForm.topicsToAvoid}
              onChange={(event) =>
                setPatientForm((current) => ({
                  ...current,
                  topicsToAvoid: event.target.value
                }))
              }
            />
          </label>
          <div className="button-row">
            <button type="submit" disabled={isPatientBusy}>
              Create patient
            </button>
            <button
              type="button"
              onClick={() => void handleUpdatePatient()}
              disabled={isPatientBusy}
            >
              Update patient
            </button>
          </div>
        </form>
        <JsonView value={patient} emptyLabel="No patient record." />
      </Section>

      <Section title="Consent and Pause">
        <p>{consentMessage}</p>
        <ErrorText message={consentError} />
        <div className="button-row">
          <button type="button" onClick={() => void handleLoadConsent()} disabled={isConsentBusy}>
            Load consent
          </button>
          <button type="button" onClick={() => void handleUnpausePatient()} disabled={isConsentBusy}>
            Unpause patient
          </button>
        </div>
        <form onSubmit={handleUpdateConsent}>
          <div className="field-grid">
            <label>
              Outbound call status
              <select
                value={consentForm.outboundCallStatus}
                onChange={(event) =>
                  setConsentForm((current) => ({
                    ...current,
                    outboundCallStatus: event.target.value as ConsentInput["outboundCallStatus"]
                  }))
                }
              >
                {CONSENT_STATUSES.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Transcript storage status
              <select
                value={consentForm.transcriptStorageStatus}
                onChange={(event) =>
                  setConsentForm((current) => ({
                    ...current,
                    transcriptStorageStatus: event.target.value as ConsentInput["transcriptStorageStatus"]
                  }))
                }
              >
                {CONSENT_STATUSES.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <label>
            Consent notes
            <textarea
              rows={3}
              value={consentForm.notes}
              onChange={(event) =>
                setConsentForm((current) => ({
                  ...current,
                  notes: event.target.value
                }))
              }
            />
          </label>
          <button type="submit" disabled={isConsentBusy}>
            Update consent
          </button>
        </form>
        <form onSubmit={handlePausePatient}>
          <label>
            Pause reason
            <input
              type="text"
              value={pauseReason}
              onChange={(event) => setPauseReason(event.target.value)}
            />
          </label>
          <button type="submit" disabled={isConsentBusy}>
            Pause patient
          </button>
        </form>
        <JsonView value={{ consent, patient }} emptyLabel="No consent or patient pause state." />
      </Section>

      <Section
        title="Dashboard"
        actions={
          <button
            type="button"
            onClick={() => void handleLoadDashboard()}
            disabled={isDashboardBusy}
          >
            Fetch dashboard
          </button>
        }
      >
        <p>{dashboardMessage}</p>
        <ErrorText message={dashboardError} />
        {dashboard ? (
          <ul>
            <li>Patient: {dashboard.patient.displayName}</li>
            <li>Calling state: {dashboard.patient.callingState}</li>
            <li>Latest call status: {dashboard.latestCall?.status ?? "none"}</li>
            <li>Risk flags: {dashboard.riskFlags.length}</li>
            <li>
              Active next-call status: {dashboard.activeNextCallPlan?.approvalStatus ?? "none"}
            </li>
          </ul>
        ) : null}
        <JsonView value={dashboard} emptyLabel="No dashboard payload." />
      </Section>

      <Section
        title="Call"
        actions={
          <div className="button-row">
            <button type="button" onClick={() => void loadTemplates(false)} disabled={isTemplatesBusy}>
              Refresh templates
            </button>
            <button type="button" onClick={() => void handleLoadCall()} disabled={isCallBusy}>
              Load call
            </button>
          </div>
        }
      >
        <p>{templatesMessage}</p>
        <ErrorText message={templatesError} />
        <JsonView value={templates} emptyLabel="No call templates loaded." />

        <p>{callMessage}</p>
        <ErrorText message={callError} />
        <label>
          Call ID
          <input
            type="text"
            value={callId}
            onChange={(event) => setCallId(event.target.value)}
          />
        </label>
        <form onSubmit={handleCreateCall}>
          <div className="field-grid">
            <label>
              Call type
              <select
                value={callForm.callType}
                onChange={(event) =>
                  setCallForm((current) => ({
                    ...current,
                    callType: event.target.value as (typeof CALL_TYPES)[number]
                  }))
                }
              >
                {CALL_TYPES.map((callType) => (
                  <option key={callType} value={callType}>
                    {callType}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Optional call template ID override
              <input
                type="text"
                value={callForm.callTemplateId}
                onChange={(event) =>
                  setCallForm((current) => ({
                    ...current,
                    callTemplateId: event.target.value
                  }))
                }
              />
            </label>
          </div>
          <p>Channel is fixed to browser for this minimal UI.</p>
          <button type="submit" disabled={isCallBusy}>
            Create browser call
          </button>
        </form>
        <JsonView value={callDetail?.callRun ?? callDetail} emptyLabel="No call loaded." />
      </Section>

      <Section title="Live Browser Call">
        <LiveCallPanel
          voiceSession={activeVoiceSession}
          onSessionEnded={() => void handleLiveSessionEnded()}
        />
      </Section>

      <Section
        title="Analysis"
        actions={
          <div className="button-row">
            <button
              type="button"
              onClick={() => void handleAnalyzeCall()}
              disabled={isAnalysisBusy}
            >
              Analyze call
            </button>
            <button
              type="button"
              onClick={() => void handleLoadAnalysis()}
              disabled={isAnalysisBusy}
            >
              Refresh analysis
            </button>
          </div>
        }
      >
        <p>{analysisMessage}</p>
        <ErrorText message={analysisError} />
        {analysis ? (
          <ul>
            {analysis.riskFlags.map((riskFlag) => (
              <li key={riskFlag.id}>
                {riskFlag.flagType}: {riskFlag.severity} ({riskFlag.confidence})
              </li>
            ))}
          </ul>
        ) : null}
        <JsonView value={analysis} emptyLabel="No analysis payload." />
        <JsonView
          value={callDetail?.transcriptTurns}
          emptyLabel="No transcript turns loaded in call detail."
        />
      </Section>

      <Section
        title="Next Call"
        actions={
          <button
            type="button"
            onClick={() => void handleLoadNextCall()}
            disabled={isNextCallBusy}
          >
            Fetch next-call plan
          </button>
        }
      >
        <p>{nextCallMessage}</p>
        <ErrorText message={nextCallError} />
        <form onSubmit={handleUpdateNextCall}>
          <div className="field-grid">
            <label>
              Action
              <select
                value={nextCallForm.action}
                onChange={(event) =>
                  setNextCallForm((current) => ({
                    ...current,
                    action: event.target.value as NextCallFormState["action"]
                  }))
                }
              >
                <option value="approve">approve</option>
                <option value="edit">edit</option>
                <option value="reject">reject</option>
                <option value="cancel">cancel</option>
              </select>
            </label>
            <label>
              Call template ID
              <input
                type="text"
                value={nextCallForm.callTemplateId}
                disabled={!isEditingNextCall}
                onChange={(event) =>
                  setNextCallForm((current) => ({
                    ...current,
                    callTemplateId: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Suggested time note
              <input
                type="text"
                value={nextCallForm.suggestedTimeNote}
                disabled={!isEditingNextCall}
                onChange={(event) =>
                  setNextCallForm((current) => ({
                    ...current,
                    suggestedTimeNote: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Planned for (RFC3339)
              <input
                type="text"
                value={nextCallForm.plannedFor}
                disabled={!isEditingNextCall}
                onChange={(event) =>
                  setNextCallForm((current) => ({
                    ...current,
                    plannedFor: event.target.value
                  }))
                }
              />
            </label>
            <label>
              Duration minutes
              <input
                type="text"
                value={nextCallForm.durationMinutes}
                disabled={!isEditingNextCall}
                onChange={(event) =>
                  setNextCallForm((current) => ({
                    ...current,
                    durationMinutes: event.target.value
                  }))
                }
              />
            </label>
          </div>
          <label>
            Goal
            <textarea
              rows={2}
              value={nextCallForm.goal}
              disabled={!isEditingNextCall}
              onChange={(event) =>
                setNextCallForm((current) => ({
                  ...current,
                  goal: event.target.value
                }))
              }
            />
          </label>
          <label>
            Reason
            <textarea
              rows={2}
              value={nextCallForm.reason}
              disabled={!canEditNextCallReason}
              onChange={(event) =>
                setNextCallForm((current) => ({
                  ...current,
                  reason: event.target.value
                }))
              }
            />
          </label>
          {!isEditingNextCall ? (
            <p>Select the `edit` action to change plan fields.</p>
          ) : null}
          <button type="submit" disabled={isNextCallBusy}>
            Update next-call plan
          </button>
        </form>
        <JsonView value={nextCallPlan} emptyLabel="No next-call plan loaded." />
      </Section>
    </main>
  );
}
