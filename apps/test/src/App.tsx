import { useEffect, useMemo, useRef, useState } from "react";
import {
  apiBaseUrl,
  buildVoiceWebSocketUrl,
  createVoiceSession,
  fetchHealth,
  fetchPatientPreferences,
  fetchVoices,
  savePatientPreferences
} from "./api/client";
import type {
  ArtifactPaths,
  HealthSnapshot,
  PatientPreference,
  VoiceOption,
  VoiceSessionDescriptor
} from "./api/contracts";

type ConnectionState = "idle" | "connecting" | "open" | "closed" | "error";

type TranscriptEntry = {
  id: string;
  direction: "user" | "assistant";
  modality: "audio" | "text";
  text: string;
  occurredAt?: string;
  generationStage?: string;
  stopReason?: string;
};

type EventEntry = {
  id: string;
  level: "info" | "event" | "error";
  label: string;
  detail: string;
};

const promptPresets = [
  {
    label: "Gentle Check-in",
    value:
      "You are a gentle check-in assistant for older adults. Speak slowly, ask one question at a time, confirm important details back to the caller, and keep every response concise."
  },
  {
    label: "Memory Support",
    value:
      "You are a calm memory-support companion. Repeat dates, names, and locations carefully, offer brief recaps after each answer, and avoid long explanations."
  },
  {
    label: "Caregiver Update",
    value:
      "You are preparing for a caregiver-facing check-in. Clarify what the caller remembers, ask for missing facts one at a time, and flag uncertainty explicitly instead of guessing."
  }
];

export default function App() {
  const [health, setHealth] = useState<HealthSnapshot | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [voices, setVoices] = useState<VoiceOption[]>([]);
  const [voicesError, setVoicesError] = useState<string | null>(null);
  const [patientId, setPatientId] = useState("patient-001");
  const [preference, setPreference] = useState<PatientPreference | null>(null);
  const [preferenceDraft, setPreferenceDraft] = useState("");
  const [preferenceError, setPreferenceError] = useState<string | null>(null);
  const [sessionVoiceId, setSessionVoiceId] = useState("");
  const [systemPrompt, setSystemPrompt] = useState(promptPresets[0].value);
  const [textInput, setTextInput] = useState("");
  const [session, setSession] = useState<VoiceSessionDescriptor | null>(null);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>("idle");
  const [transcripts, setTranscripts] = useState<TranscriptEntry[]>([]);
  const [events, setEvents] = useState<EventEntry[]>([]);
  const [artifacts, setArtifacts] = useState<ArtifactPaths | null>(null);
  const [audioStats, setAudioStats] = useState({ chunks: 0, bytes: 0 });
  const [isSavingPreference, setIsSavingPreference] = useState(false);
  const [isStartingSession, setIsStartingSession] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const nextPlaybackTimeRef = useRef(0);

  useEffect(() => {
    void loadHealth();
    void loadVoices();
  }, []);

  useEffect(() => {
    void loadPreference(patientId);
  }, [patientId]);

  useEffect(() => {
    return () => {
      disconnect();
      void closeAudioContext();
    };
  }, []);

  const selectedVoiceSummary = useMemo(() => {
    const resolvedVoiceId = sessionVoiceId || preference?.defaultVoiceId || "";
    const selected = voices.find((voice) => voice.id === resolvedVoiceId);
    if (!selected) {
      return resolvedVoiceId || "No voice selected";
    }

    return `${selected.displayName} (${selected.id})`;
  }, [preference?.defaultVoiceId, sessionVoiceId, voices]);

  async function loadHealth() {
    setHealthError(null);
    try {
      setHealth(await fetchHealth());
    } catch (error) {
      setHealth(null);
      setHealthError(error instanceof Error ? error.message : "Unable to reach the API.");
    }
  }

  async function loadVoices() {
    setVoicesError(null);
    try {
      const nextVoices = await fetchVoices();
      setVoices(nextVoices);
      const fallbackVoice = nextVoices.find((voice) => voice.isDefault)?.id ?? nextVoices[0]?.id ?? "";
      setPreferenceDraft((current) => current || fallbackVoice);
    } catch (error) {
      setVoices([]);
      setVoicesError(error instanceof Error ? error.message : "Unable to load voices.");
    }
  }

  async function loadPreference(nextPatientId: string) {
    const trimmedPatientId = nextPatientId.trim();
    if (!trimmedPatientId) {
      setPreference(null);
      return;
    }

    setPreferenceError(null);
    try {
      const nextPreference = await fetchPatientPreferences(trimmedPatientId);
      setPreference(nextPreference);
      setPreferenceDraft(nextPreference.defaultVoiceId);
    } catch (error) {
      setPreference(null);
      setPreferenceError(
        error instanceof Error ? error.message : "Unable to load patient preferences."
      );
    }
  }

  async function handleSavePreference() {
    setIsSavingPreference(true);
    setPreferenceError(null);

    try {
      const saved = await savePatientPreferences(patientId.trim(), {
        defaultVoiceId: preferenceDraft
      });
      setPreference(saved);
      appendEvent("info", "Saved default voice", `${saved.patientId} -> ${saved.defaultVoiceId}`);
    } catch (error) {
      setPreferenceError(
        error instanceof Error ? error.message : "Unable to save patient preferences."
      );
    } finally {
      setIsSavingPreference(false);
    }
  }

  async function handleStartSession() {
    setIsStartingSession(true);
    setSessionError(null);
    setArtifacts(null);
    setTranscripts([]);
    setEvents([]);
    setAudioStats({ chunks: 0, bytes: 0 });

    try {
      await disconnect();
      await closeAudioContext();

      const created = await createVoiceSession({
        patientId: patientId.trim(),
        voiceId: sessionVoiceId || undefined,
        systemPrompt: systemPrompt.trim() || undefined
      });

      setSession(created);
      appendEvent("info", "Voice session created", `${created.id} -> ${created.voiceId}`);
      await connect(created);
    } catch (error) {
      setSession(null);
      setConnectionState("error");
      setSessionError(error instanceof Error ? error.message : "Unable to create a voice session.");
    } finally {
      setIsStartingSession(false);
    }
  }

  async function handleSendText() {
    const trimmed = textInput.trim();
    const socket = wsRef.current;
    if (!trimmed || !socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    socket.send(JSON.stringify({ type: "text_input", text: trimmed }));
    setTextInput("");
    appendTranscript({
      direction: "user",
      modality: "text",
      text: trimmed,
      generationStage: "FINAL",
      occurredAt: new Date().toISOString()
    });
  }

  async function handleCloseSession() {
    const socket = wsRef.current;
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: "client_close" }));
      return;
    }

    await disconnect();
  }

  async function connect(nextSession: VoiceSessionDescriptor) {
    setConnectionState("connecting");

    const socket = new WebSocket(
      buildVoiceWebSocketUrl(nextSession.websocketPath, nextSession.streamToken)
    );
    socket.binaryType = "arraybuffer";

    socket.onopen = () => {
      setConnectionState("open");
      appendEvent("info", "WebSocket connected", nextSession.websocketPath);
    };

    socket.onmessage = (event) => {
      void handleSocketMessage(event, nextSession);
    };

    socket.onerror = () => {
      setConnectionState("error");
      appendEvent("error", "WebSocket error", "The live session reported a socket error.");
    };

    socket.onclose = (event) => {
      setConnectionState("closed");
      appendEvent("info", "WebSocket closed", `${event.code} ${event.reason || ""}`.trim());
      if (wsRef.current === socket) {
        wsRef.current = null;
      }
    };

    wsRef.current = socket;
  }

  async function handleSocketMessage(
    event: MessageEvent<ArrayBuffer | Blob | string>,
    activeSession: VoiceSessionDescriptor
  ) {
    if (typeof event.data === "string") {
      handleJsonEvent(JSON.parse(event.data) as Record<string, unknown>);
      return;
    }

    const audioBuffer =
      event.data instanceof Blob ? await event.data.arrayBuffer() : event.data;

    setAudioStats((current) => ({
      chunks: current.chunks + 1,
      bytes: current.bytes + audioBuffer.byteLength
    }));

    appendEvent(
      "event",
      "Audio chunk",
      `${audioBuffer.byteLength.toLocaleString()} bytes at ${activeSession.audioOutput.sampleRateHz} Hz`
    );

    await playPcmChunk(audioBuffer, activeSession.audioOutput.sampleRateHz);
  }

  function handleJsonEvent(payload: Record<string, unknown>) {
    const type = typeof payload.type === "string" ? payload.type : "unknown";

    switch (type) {
      case "session_ready":
        appendEvent("info", "Session ready", JSON.stringify(payload));
        break;
      case "transcript_partial":
        appendEvent(
          "event",
          "Partial transcript",
          `${payload.direction ?? "assistant"}: ${payload.text ?? ""}`
        );
        break;
      case "transcript_final":
        appendTranscript({
          direction: payload.direction === "user" ? "user" : "assistant",
          modality: payload.modality === "text" ? "text" : "audio",
          text: typeof payload.text === "string" ? payload.text : "",
          occurredAt: typeof payload.occurredAt === "string" ? payload.occurredAt : undefined,
          generationStage:
            typeof payload.generationStage === "string" ? payload.generationStage : undefined,
          stopReason: typeof payload.stopReason === "string" ? payload.stopReason : undefined
        });
        break;
      case "usage":
        appendEvent("event", "Usage", JSON.stringify(payload));
        break;
      case "interrupted":
        appendEvent("event", "Interruption", JSON.stringify(payload));
        void closeAudioContext();
        break;
      case "error":
        appendEvent(
          "error",
          typeof payload.code === "string" ? payload.code : "error",
          typeof payload.message === "string" ? payload.message : JSON.stringify(payload)
        );
        break;
      case "session_ended":
        appendEvent("info", "Session ended", JSON.stringify(payload));
        if (payload.artifacts && typeof payload.artifacts === "object") {
          const nextArtifacts = payload.artifacts as Record<string, unknown>;
          setArtifacts({
            jsonPath:
              typeof nextArtifacts.jsonPath === "string" ? nextArtifacts.jsonPath : undefined,
            markdownPath:
              typeof nextArtifacts.markdownPath === "string"
                ? nextArtifacts.markdownPath
                : undefined
          });
        }
        void disconnect();
        break;
      default:
        appendEvent("event", "Socket event", JSON.stringify(payload));
    }
  }

  function appendTranscript(entry: Omit<TranscriptEntry, "id">) {
    setTranscripts((current) => [
      {
        id: crypto.randomUUID(),
        ...entry
      },
      ...current
    ]);
  }

  function appendEvent(level: EventEntry["level"], label: string, detail: string) {
    setEvents((current) => [
      {
        id: crypto.randomUUID(),
        level,
        label,
        detail
      },
      ...current
    ].slice(0, 40));
  }

  async function ensureAudioContext(): Promise<AudioContext | null> {
    if (typeof window === "undefined" || typeof window.AudioContext === "undefined") {
      return null;
    }

    if (!audioContextRef.current || audioContextRef.current.state === "closed") {
      audioContextRef.current = new window.AudioContext();
    }

    if (audioContextRef.current.state === "suspended") {
      await audioContextRef.current.resume();
    }

    return audioContextRef.current;
  }

  async function playPcmChunk(buffer: ArrayBuffer, sampleRateHz: number) {
    const audioContext = await ensureAudioContext();
    if (!audioContext) {
      return;
    }

    const samples = new Int16Array(buffer);
    const channelData = new Float32Array(samples.length);
    for (let index = 0; index < samples.length; index += 1) {
      channelData[index] = samples[index] / 32768;
    }

    const audioBuffer = audioContext.createBuffer(1, channelData.length, sampleRateHz);
    audioBuffer.copyToChannel(channelData, 0);

    const source = audioContext.createBufferSource();
    source.buffer = audioBuffer;
    source.connect(audioContext.destination);

    const startAt = Math.max(audioContext.currentTime, nextPlaybackTimeRef.current);
    source.start(startAt);
    nextPlaybackTimeRef.current = startAt + audioBuffer.duration;
  }

  async function closeAudioContext() {
    const current = audioContextRef.current;
    audioContextRef.current = null;
    nextPlaybackTimeRef.current = 0;
    if (current && current.state !== "closed") {
      await current.close();
    }
  }

  async function disconnect() {
    const socket = wsRef.current;
    wsRef.current = null;
    if (!socket) {
      return;
    }

    if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
      socket.close();
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8 lg:py-10">
      <section className="rounded-[2rem] border border-amber-200/80 bg-white/85 p-6 shadow-[0_30px_120px_rgba(17,24,39,0.12)] backdrop-blur lg:p-8">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl">
            <p className="text-xs font-semibold uppercase tracking-[0.3em] text-amber-700">
              Dev App
            </p>
            <h1 className="mt-4 max-w-2xl text-4xl font-semibold tracking-tight text-slate-950 sm:text-5xl">
              Prompt Lab for Nova voice session testing
            </h1>
            <p className="mt-4 max-w-3xl text-base leading-7 text-slate-600">
              This app is isolated from the main frontend and exists just for prompt,
              voice, and transcript iteration. Start with a system prompt, choose the
              active voice, then send text turns over the same live WebSocket session.
            </p>
          </div>

          <div className="rounded-3xl border border-slate-200 bg-slate-950 px-5 py-4 text-sm text-slate-100 shadow-lg">
            <p className="font-medium text-white">API base URL</p>
            <p className="mt-2 break-all font-mono text-slate-300">{apiBaseUrl}</p>
            <p className="mt-3 text-xs text-slate-400">
              Saved artifacts land in <code>apps/api/testdata/voice-lab</code>.
            </p>
          </div>
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <Panel title="Setup">
          <div className="grid gap-4">
            <Field label="Patient ID">
              <input
                type="text"
                value={patientId}
                onChange={(event) => setPatientId(event.target.value)}
                className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition focus:border-amber-400"
              />
            </Field>

            <Field label="Patient default voice">
              <div className="flex flex-col gap-3 md:flex-row">
                <select
                  value={preferenceDraft}
                  onChange={(event) => setPreferenceDraft(event.target.value)}
                  className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition focus:border-amber-400"
                >
                  {voices.map((voice) => (
                    <option key={voice.id} value={voice.id}>
                      {voice.displayName} ({voice.id}) {voice.polyglot ? "polyglot" : ""}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={handleSavePreference}
                  disabled={isSavingPreference || !patientId.trim() || !preferenceDraft}
                  className="rounded-full bg-slate-950 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
                >
                  {isSavingPreference ? "Saving..." : "Save default"}
                </button>
              </div>
            </Field>

            <InfoBox title="Current default">
              {preference?.isConfigured
                ? `${preference.patientId} -> ${preference.defaultVoiceId}`
                : "No saved patient preference yet. The app default voice will be used."}
            </InfoBox>

            <Field label="Session voice">
              <select
                value={sessionVoiceId}
                onChange={(event) => setSessionVoiceId(event.target.value)}
                className="w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition focus:border-amber-400"
              >
                <option value="">Use patient default</option>
                {voices.map((voice) => (
                  <option key={voice.id} value={voice.id}>
                    {voice.displayName} ({voice.id}) {voice.polyglot ? "polyglot" : ""}
                  </option>
                ))}
              </select>
            </Field>

            <InfoBox title="Resolved voice for the next session">{selectedVoiceSummary}</InfoBox>

            <Field label="System prompt">
              <div className="mb-3 flex flex-wrap gap-2">
                {promptPresets.map((preset) => (
                  <button
                    key={preset.label}
                    type="button"
                    onClick={() => setSystemPrompt(preset.value)}
                    className="rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-amber-300 hover:text-amber-800"
                  >
                    {preset.label}
                  </button>
                ))}
              </div>
              <textarea
                value={systemPrompt}
                onChange={(event) => setSystemPrompt(event.target.value)}
                rows={10}
                className="w-full rounded-3xl border border-slate-200 bg-white px-4 py-4 text-sm leading-6 text-slate-900 outline-none transition focus:border-amber-400"
              />
            </Field>

            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                onClick={handleStartSession}
                disabled={isStartingSession || !patientId.trim()}
                className="rounded-full bg-amber-500 px-5 py-3 text-sm font-semibold text-slate-950 transition hover:bg-amber-400 disabled:cursor-not-allowed disabled:bg-amber-200"
              >
                {isStartingSession ? "Starting..." : "Start session"}
              </button>
              <button
                type="button"
                onClick={handleCloseSession}
                className="rounded-full border border-slate-200 bg-white px-5 py-3 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:text-slate-950"
              >
                Close session
              </button>
            </div>

            {sessionError ? <Alert tone="error" body={sessionError} /> : null}
            {preferenceError ? <Alert tone="error" body={preferenceError} /> : null}
            {voicesError ? <Alert tone="error" body={voicesError} /> : null}
          </div>
        </Panel>

        <Panel title="Session">
          <div className="grid gap-4">
            <InfoGrid
              items={[
                ["Connection", connectionState],
                ["API health", health ? health.status : healthError ?? "unavailable"],
                ["Session ID", session?.id ?? "Not started"],
                ["Negotiated output", session ? `${session.audioOutput.sampleRateHz} Hz` : "n/a"],
                ["Audio chunks", `${audioStats.chunks}`],
                ["Audio bytes", audioStats.bytes.toLocaleString()]
              ]}
            />

            <Field label="Send text into the live session">
              <div className="flex flex-col gap-3">
                <textarea
                  value={textInput}
                  onChange={(event) => setTextInput(event.target.value)}
                  rows={4}
                  placeholder="Type a test turn. This uses the same live voice session instead of a separate REST text endpoint."
                  className="w-full rounded-3xl border border-slate-200 bg-white px-4 py-4 text-sm leading-6 text-slate-900 outline-none transition focus:border-amber-400"
                />
                <div className="flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={handleSendText}
                    disabled={connectionState !== "open" || !textInput.trim()}
                    className="rounded-full bg-slate-950 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
                  >
                    Send text turn
                  </button>
                  <button
                    type="button"
                    onClick={loadHealth}
                    className="rounded-full border border-slate-200 bg-white px-5 py-3 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:text-slate-950"
                  >
                    Refresh API status
                  </button>
                </div>
              </div>
            </Field>

            <InfoBox title="Artifacts">
              {artifacts?.jsonPath || artifacts?.markdownPath ? (
                <div className="space-y-2 font-mono text-xs text-slate-700">
                  <p>{artifacts.jsonPath ?? "No JSON artifact path"}</p>
                  <p>{artifacts.markdownPath ?? "No Markdown artifact path"}</p>
                </div>
              ) : (
                "Artifact paths will appear here after session shutdown."
              )}
            </InfoBox>
          </div>
        </Panel>
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <Panel title="FINAL transcripts">
          <div className="grid gap-3">
            {transcripts.length === 0 ? (
              <EmptyState body="No FINAL transcript turns yet. Start a session and send text or audio." />
            ) : (
              transcripts.map((item) => (
                <article
                  key={item.id}
                  className="rounded-3xl border border-slate-200 bg-slate-50/80 p-4"
                >
                  <div className="flex flex-wrap items-center gap-2 text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">
                    <span>{item.direction}</span>
                    <span>{item.modality}</span>
                    {item.generationStage ? <span>{item.generationStage}</span> : null}
                  </div>
                  <p className="mt-3 text-sm leading-7 text-slate-900">{item.text}</p>
                  <p className="mt-3 text-xs text-slate-500">
                    {item.occurredAt ?? "No timestamp"} {item.stopReason ? `• ${item.stopReason}` : ""}
                  </p>
                </article>
              ))
            )}
          </div>
        </Panel>

        <Panel title="Event log">
          <div className="grid gap-3">
            {events.length === 0 ? (
              <EmptyState body="Socket events, usage, and lifecycle messages show up here." />
            ) : (
              events.map((event) => (
                <article
                  key={event.id}
                  className="rounded-3xl border border-slate-200 bg-white p-4"
                >
                  <div className="flex items-center justify-between gap-4">
                    <p className="text-sm font-semibold text-slate-900">{event.label}</p>
                    <span
                      className={`rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.22em] ${
                        event.level === "error"
                          ? "bg-rose-100 text-rose-700"
                          : event.level === "info"
                            ? "bg-amber-100 text-amber-800"
                            : "bg-sky-100 text-sky-800"
                      }`}
                    >
                      {event.level}
                    </span>
                  </div>
                  <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs leading-6 text-slate-600">
                    {event.detail}
                  </pre>
                </article>
              ))
            )}
          </div>
        </Panel>
      </div>
    </main>
  );
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[2rem] border border-slate-200/80 bg-white/85 p-6 shadow-[0_20px_80px_rgba(15,23,42,0.08)] backdrop-blur lg:p-7">
      <div className="mb-5">
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">Prompt Lab</p>
        <h2 className="mt-2 text-2xl font-semibold text-slate-950">{title}</h2>
      </div>
      {children}
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block text-sm font-medium text-slate-700">
      <span>{label}</span>
      <div className="mt-2">{children}</div>
    </label>
  );
}

function InfoBox({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-3xl border border-slate-200 bg-slate-50/90 px-4 py-4 text-sm text-slate-600">
      <p className="font-semibold text-slate-900">{title}</p>
      <div className="mt-2 leading-6">{children}</div>
    </div>
  );
}

function InfoGrid({ items }: { items: Array<[string, string]> }) {
  return (
    <dl className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {items.map(([label, value]) => (
        <div key={label} className="rounded-3xl border border-slate-200 bg-slate-50/90 px-4 py-4">
          <dt className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-500">
            {label}
          </dt>
          <dd className="mt-2 break-all text-sm font-medium text-slate-900">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function Alert({ tone, body }: { tone: "error"; body: string }) {
  return (
    <div
      className={`rounded-3xl border px-4 py-3 text-sm ${
        tone === "error"
          ? "border-rose-200 bg-rose-50 text-rose-700"
          : "border-slate-200 bg-slate-50 text-slate-700"
      }`}
    >
      {body}
    </div>
  );
}

function EmptyState({ body }: { body: string }) {
  return (
    <div className="rounded-3xl border border-dashed border-slate-300 bg-white px-4 py-6 text-sm text-slate-500">
      {body}
    </div>
  );
}
