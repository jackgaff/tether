import { useEffect, useRef, useState } from "react";
import { startMicrophoneStream, type MicrophoneStream } from "./audio";
import {
  apiBaseUrl,
  buildVoiceWebSocketUrl,
  createVoiceSession,
  fetchHealth,
  fetchLabConversations,
  fetchVoices
} from "./api/client";
import type {
  HealthSnapshot,
  LabConversation,
  VoiceOption,
  VoiceSessionDescriptor
} from "./api/contracts";

type LiveTurn = {
  id: string;
  direction: "user" | "assistant";
  modality: "audio" | "text";
  text: string;
  occurredAt?: string;
};

type RunState = "idle" | "starting" | "live" | "stopping" | "error";

const DEV_PATIENT_ID = "prompt-lab";
const DEFAULT_SYSTEM_PROMPT =
  "You are a gentle voice check-in assistant. Start the call yourself, greet the person warmly, speak briefly, ask one question at a time, and keep the pacing natural for spoken conversation.";

export default function App() {
  const [health, setHealth] = useState<HealthSnapshot | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [voices, setVoices] = useState<VoiceOption[]>([]);
  const [voicesError, setVoicesError] = useState<string | null>(null);
  const [history, setHistory] = useState<LabConversation[]>([]);
  const [historyError, setHistoryError] = useState<string | null>(null);
  const [selectedVoiceId, setSelectedVoiceId] = useState("");
  const [systemPrompt, setSystemPrompt] = useState(DEFAULT_SYSTEM_PROMPT);
  const [runState, setRunState] = useState<RunState>("idle");
  const [statusText, setStatusText] = useState("Ready to start a new test call.");
  const [errorText, setErrorText] = useState<string | null>(null);
  const [liveTurns, setLiveTurns] = useState<LiveTurn[]>([]);
  const [activeSession, setActiveSession] = useState<VoiceSessionDescriptor | null>(null);

  const socketRef = useRef<WebSocket | null>(null);
  const microphoneRef = useRef<MicrophoneStream | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const nextPlaybackTimeRef = useRef(0);
  const isStoppingRef = useRef(false);
  const expectedCloseRef = useRef(false);

  useEffect(() => {
    void loadBootData();

    return () => {
      void teardownLiveSession();
    };
  }, []);

  async function loadBootData() {
    await Promise.all([loadHealth(), loadVoices(), loadHistory()]);
  }

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
      setSelectedVoiceId((current) => {
        if (current && nextVoices.some((voice) => voice.id === current)) {
          return current;
        }

        return nextVoices.find((voice) => voice.isDefault)?.id ?? nextVoices[0]?.id ?? "";
      });
    } catch (error) {
      setVoices([]);
      setVoicesError(error instanceof Error ? error.message : "Unable to load voices.");
    }
  }

  async function loadHistory(preferredConversationId?: string) {
    setHistoryError(null);
    try {
      const nextHistory = await fetchLabConversations(20);
      setHistory(nextHistory);

      if (preferredConversationId && nextHistory.some((entry) => entry.id === preferredConversationId)) {
        setStatusText("Call saved. Ready for the next test run.");
      }
    } catch (error) {
      setHistory([]);
      setHistoryError(
        error instanceof Error ? error.message : "Unable to load saved prompt-lab conversations."
      );
    }
  }

  async function handleStart() {
    if (!selectedVoiceId || !systemPrompt.trim()) {
      return;
    }

    await teardownLiveSession();

    setRunState("starting");
    setErrorText(null);
    setStatusText("Creating voice session...");
    setLiveTurns([]);
    expectedCloseRef.current = false;

    try {
      const session = await createVoiceSession({
        patientId: DEV_PATIENT_ID,
        voiceId: selectedVoiceId,
        systemPrompt: systemPrompt.trim()
      });

      setActiveSession(session);
      connectToSession(session);
    } catch (error) {
      setActiveSession(null);
      setRunState("error");
      setErrorText(error instanceof Error ? error.message : "Unable to start the voice session.");
      setStatusText("Could not start the call.");
    }
  }

  function connectToSession(session: VoiceSessionDescriptor) {
    const socket = new WebSocket(buildVoiceWebSocketUrl(session.websocketPath, session.streamToken));
    socket.binaryType = "arraybuffer";

    socket.onopen = () => {
      setStatusText("Connecting to the live call...");
    };

    socket.onmessage = (event) => {
      void handleSocketMessage(session, event);
    };

    socket.onerror = () => {
      setRunState("error");
      setErrorText("The live socket reported an error.");
      setStatusText("The live call hit a socket error.");
    };

    socket.onclose = (event) => {
      socketRef.current = null;

      if (expectedCloseRef.current) {
        expectedCloseRef.current = false;
        return;
      }

      setActiveSession(null);
      setRunState("idle");
      setErrorText(event.reason || "The live socket closed unexpectedly.");
      setStatusText("Call closed.");
    };

    socketRef.current = socket;
  }

  async function handleSocketMessage(
    session: VoiceSessionDescriptor,
    event: MessageEvent<ArrayBuffer | Blob | string>
  ) {
    if (typeof event.data === "string") {
      await handleJsonEvent(session, JSON.parse(event.data) as Record<string, unknown>);
      return;
    }

    const audioBuffer =
      event.data instanceof Blob ? await event.data.arrayBuffer() : event.data;

    await playPcmChunk(audioBuffer, session.audioOutput.sampleRateHz);
  }

  async function handleJsonEvent(
    session: VoiceSessionDescriptor,
    payload: Record<string, unknown>
  ) {
    const type = typeof payload.type === "string" ? payload.type : "";

    switch (type) {
      case "session_ready":
        setRunState("live");
        setStatusText("Live. The assistant is starting the call.");
        await ensureMicrophone(session);
        socketRef.current?.send(JSON.stringify({ type: "start_call" }));
        break;
      case "transcript_final":
        appendLiveTurn({
          direction: payload.direction === "user" ? "user" : "assistant",
          modality: payload.modality === "text" ? "text" : "audio",
          text: typeof payload.text === "string" ? payload.text : "",
          occurredAt: typeof payload.occurredAt === "string" ? payload.occurredAt : undefined
        });
        break;
      case "interrupted":
        await closePlayback();
        break;
      case "error":
        setRunState("error");
        setErrorText(
          typeof payload.message === "string" ? payload.message : "The live call returned an error."
        );
        setStatusText("The call returned an error.");
        break;
      case "session_ended":
        isStoppingRef.current = false;
        expectedCloseRef.current = true;
        setActiveSession(null);
        setRunState("idle");
        await teardownLiveSession();
        setLiveTurns([]);
        await loadHistory(session.id);
        break;
      default:
        break;
    }
  }

  async function ensureMicrophone(session: VoiceSessionDescriptor) {
    if (microphoneRef.current) {
      return;
    }

    try {
      const socket = socketRef.current;
      if (!socket) {
        return;
      }

      microphoneRef.current = await startMicrophoneStream(socket, session.audioInput.sampleRateHz);
    } catch (error) {
      setRunState("error");
      setErrorText(
        error instanceof Error ? error.message : "Unable to start microphone capture."
      );
      setStatusText("Microphone capture failed.");
    }
  }

  function appendLiveTurn(turn: Omit<LiveTurn, "id">) {
    setLiveTurns((current) => [
      ...current,
      {
        id: crypto.randomUUID(),
        ...turn
      }
    ]);
  }

  async function handleStop() {
    const socket = socketRef.current;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      await teardownLiveSession();
      setActiveSession(null);
      setRunState("idle");
      setStatusText("Ready to start a new test call.");
      return;
    }

    isStoppingRef.current = true;
    expectedCloseRef.current = true;
    setRunState("stopping");
    setStatusText("Stopping the call and saving the transcript...");
    socket.send(JSON.stringify({ type: "client_close" }));
  }

  async function teardownLiveSession() {
    const microphone = microphoneRef.current;
    microphoneRef.current = null;
    if (microphone) {
      await microphone.stop();
    }

    const socket = socketRef.current;
    socketRef.current = null;
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
      expectedCloseRef.current = true;
      socket.close();
    }

    await closePlayback();
  }

  async function ensurePlaybackContext(): Promise<AudioContext | null> {
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
    const audioContext = await ensurePlaybackContext();
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

  async function closePlayback() {
    const current = audioContextRef.current;
    audioContextRef.current = null;
    nextPlaybackTimeRef.current = 0;

    if (current && current.state !== "closed") {
      await current.close();
    }
  }

  return (
    <main>
      <h1>Prompt Test</h1>
      <p>
        Pick a voice, paste a starting prompt, press start, and let the assistant open the call.
        Finished runs are saved below.
      </p>

      <p>
        <strong>API:</strong>{" "}
        {health ? `${health.service} (${health.env})` : healthError ?? "Unavailable"}
      </p>
      <p>
        <code>{apiBaseUrl}</code>
      </p>

      <section>
        <h2>New conversation</h2>

        <p>
          <label htmlFor="voice-select">Voice</label>
          <br />
          <select
            id="voice-select"
            value={selectedVoiceId}
            onChange={(event) => setSelectedVoiceId(event.target.value)}
            disabled={runState === "starting" || runState === "live" || runState === "stopping"}
          >
            {voices.map((voice) => (
              <option key={voice.id} value={voice.id}>
                {voice.displayName} ({voice.id}) · {voice.locale}
              </option>
            ))}
          </select>
        </p>

        <p>
          <label htmlFor="system-prompt">Starting prompt</label>
          <br />
          <textarea
            id="system-prompt"
            rows={8}
            value={systemPrompt}
            onChange={(event) => setSystemPrompt(event.target.value)}
            placeholder="Write the prompt you want to test..."
            disabled={runState === "starting" || runState === "live" || runState === "stopping"}
          />
        </p>

        <p>
          <button
            type="button"
            onClick={() => void handleStart()}
            disabled={
              !selectedVoiceId ||
              !systemPrompt.trim() ||
              runState === "starting" ||
              runState === "live" ||
              runState === "stopping"
            }
          >
            {runState === "starting" ? "Starting..." : "Start call"}
          </button>
          {" "}
          <button
            type="button"
            onClick={() => void handleStop()}
            disabled={!activeSession || (runState !== "live" && runState !== "stopping")}
          >
            {runState === "stopping" ? "Stopping..." : "Stop"}
          </button>
        </p>

        {voicesError ? <p>{voicesError}</p> : null}
        {errorText ? <p>{errorText}</p> : null}
        <p>{statusText}</p>
      </section>

      <section>
        <h2>Current conversation</h2>
        <p>{activeSession ? `${activeSession.voiceId} live` : "No active call"}</p>

        {liveTurns.length === 0 ? (
          <p>
            {activeSession
              ? "Waiting for the first spoken turn..."
              : "Press start to open a new voice session."}
          </p>
        ) : (
          <ol>
            {liveTurns.map((turn) => (
              <li key={turn.id}>
                <strong>{turn.direction === "assistant" ? "Assistant" : "You"}</strong>
                {" · "}
                {turn.modality}
                {turn.occurredAt ? (
                  <>
                    {" · "}
                    <time>{formatTime(turn.occurredAt)}</time>
                  </>
                ) : null}
                <div>{turn.text}</div>
              </li>
            ))}
          </ol>
        )}
      </section>

      <section>
        <h2>Previous conversations</h2>
        <p>{history.length} saved</p>

        {historyError ? <p>{historyError}</p> : null}
        {!historyError && history.length === 0 ? (
          <p>No saved prompt tests yet.</p>
        ) : (
          <div>
            {history.map((conversation) => (
              <details key={conversation.id}>
                <summary>
                  <strong>{conversation.voiceId}</strong>
                  {" · "}
                  {formatDateTime(conversation.endedAt)}
                  {" · "}
                  {conversation.status}
                </summary>

                {conversation.systemPrompt ? (
                  <>
                    <h3>Prompt</h3>
                    <pre>{conversation.systemPrompt}</pre>
                  </>
                ) : null}

                <div>
                  <h3>Transcript</h3>
                  {conversation.turns.length === 0 ? (
                    <p>No final turns were saved.</p>
                  ) : (
                    <ol>
                      {conversation.turns.map((turn) => (
                        <li key={`${conversation.id}-${turn.sequenceNo}`}>
                          <strong>{turn.direction === "assistant" ? "Assistant" : "You"}</strong>
                          {" · "}
                          {turn.modality}
                          {" · "}
                          <time>{formatTime(turn.occurredAt)}</time>
                          <div>{turn.text}</div>
                        </li>
                      ))}
                    </ol>
                  )}
                </div>
              </details>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString();
}

function formatTime(value: string): string {
  return new Date(value).toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit"
  });
}
