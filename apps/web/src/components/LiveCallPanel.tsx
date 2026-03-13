import { useEffect, useRef, useState } from "react";
import { startMicrophoneStream, type MicrophoneStream } from "../audio";
import { buildVoiceWebSocketUrl } from "../api/client";
import type { VoiceSessionDescriptor } from "../api/contracts";
import { ErrorText } from "./ErrorText";

type LiveTurn = {
  id: string;
  direction: "user" | "assistant";
  modality: "audio" | "text";
  text: string;
  occurredAt?: string;
  stopReason?: string;
};

type RunState = "idle" | "starting" | "live" | "stopping" | "error";

interface LiveCallPanelProps {
  voiceSession: VoiceSessionDescriptor | null;
  onSessionEnded: () => void | Promise<void>;
}

export function LiveCallPanel({
  voiceSession,
  onSessionEnded
}: LiveCallPanelProps) {
  const [runState, setRunState] = useState<RunState>("idle");
  const [statusText, setStatusText] = useState("No active browser call.");
  const [errorText, setErrorText] = useState<string | null>(null);
  const [turns, setTurns] = useState<LiveTurn[]>([]);

  const socketRef = useRef<WebSocket | null>(null);
  const microphoneRef = useRef<MicrophoneStream | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const nextPlaybackTimeRef = useRef(0);
  const expectedCloseRef = useRef(false);

  useEffect(() => {
    return () => {
      void teardownLiveSession();
    };
  }, []);

  useEffect(() => {
    if (!voiceSession) {
      void teardownLiveSession();
      setRunState("idle");
      setStatusText("No active browser call.");
      setErrorText(null);
      setTurns([]);
      return;
    }

    void teardownLiveSession();
    setRunState("starting");
    setStatusText("Connecting to the browser call...");
    setErrorText(null);
    setTurns([]);
    expectedCloseRef.current = false;

    const socket = new WebSocket(
      buildVoiceWebSocketUrl(voiceSession.websocketPath, voiceSession.streamToken)
    );
    socket.binaryType = "arraybuffer";

    socket.onopen = () => {
      setStatusText("Waiting for the call to become ready...");
    };

    socket.onmessage = (event) => {
      void handleSocketMessage(voiceSession, event);
    };

    socket.onerror = () => {
      void enterErrorState(
        "The browser call socket reported an error.",
        "The browser call hit a socket error.",
        true
      );
    };

    socket.onclose = (event) => {
      socketRef.current = null;

      if (expectedCloseRef.current) {
        expectedCloseRef.current = false;
        return;
      }

      void enterErrorState(
        event.reason || "The browser call socket closed unexpectedly.",
        "The browser call closed unexpectedly.",
        true
      );
    };

    socketRef.current = socket;
  }, [voiceSession?.id]);

  async function handleSocketMessage(
    session: VoiceSessionDescriptor,
    event: MessageEvent<ArrayBuffer | Blob | string>
  ) {
    if (typeof event.data === "string") {
      let payload: Record<string, unknown>;

      try {
        payload = JSON.parse(event.data) as Record<string, unknown>;
      } catch {
        await enterErrorState(
          "The browser call returned invalid JSON.",
          "The browser call returned invalid data.",
          true
        );
        return;
      }

      await handleJsonEvent(session, payload);
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
        if (!(await ensureMicrophone(session))) {
          return;
        }
        socketRef.current?.send(JSON.stringify({ type: "start_call" }));
        break;
      case "transcript_final":
        appendTurn({
          direction: payload.direction === "user" ? "user" : "assistant",
          modality: payload.modality === "text" ? "text" : "audio",
          text: typeof payload.text === "string" ? payload.text : "",
          occurredAt: typeof payload.occurredAt === "string" ? payload.occurredAt : undefined,
          stopReason: typeof payload.stopReason === "string" ? payload.stopReason : undefined
        });
        break;
      case "interrupted":
        await closePlayback();
        break;
      case "error":
        await enterErrorState(
          typeof payload.message === "string"
            ? payload.message
            : "The browser call returned an error.",
          "The browser call returned an error.",
          true
        );
        break;
      case "session_ended":
        expectedCloseRef.current = true;
        await teardownLiveSession();
        setRunState("idle");
        setStatusText("Call ended.");
        setErrorText(null);
        setTurns([]);
        await onSessionEnded();
        break;
      default:
        break;
    }
  }

  async function ensureMicrophone(session: VoiceSessionDescriptor): Promise<boolean> {
    if (microphoneRef.current) {
      return true;
    }

    const socket = socketRef.current;
    if (!socket) {
      return false;
    }

    try {
      microphoneRef.current = await startMicrophoneStream(
        socket,
        session.audioInput.sampleRateHz
      );
      return true;
    } catch (error) {
      await enterErrorState(
        error instanceof Error ? error.message : "Unable to start microphone capture.",
        "Microphone capture failed.",
        true
      );
      return false;
    }
  }

  function appendTurn(turn: Omit<LiveTurn, "id">) {
    const text = turn.text.trim();
    if (!text) {
      return;
    }

    setTurns((current) => [
      ...current,
      {
        id: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${current.length}`,
        ...turn,
        text
      }
    ]);
  }

  async function handleStop() {
    const socket = socketRef.current;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      expectedCloseRef.current = true;
      await teardownLiveSession();
      setRunState("idle");
      setStatusText("Call stopped.");
      setErrorText(null);
      setTurns([]);
      await onSessionEnded();
      return;
    }

    expectedCloseRef.current = true;
    setRunState("stopping");
    setStatusText("Stopping the browser call...");
    socket.send(JSON.stringify({ type: "client_close" }));
  }

  async function enterErrorState(message: string, status: string, closeSession: boolean) {
    if (closeSession) {
      expectedCloseRef.current = true;
      await teardownLiveSession();
    }

    setRunState("error");
    setErrorText(message);
    setStatusText(status);
  }

  async function teardownLiveSession() {
    const microphone = microphoneRef.current;
    microphoneRef.current = null;
    if (microphone) {
      await microphone.stop();
    }

    const socket = socketRef.current;
    socketRef.current = null;
    if (
      socket &&
      (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)
    ) {
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
    <div>
      <p>
        <strong>Status:</strong> {statusText}
      </p>
      {voiceSession ? (
        <p>
          <strong>Session ID:</strong> {voiceSession.id}
        </p>
      ) : null}
      <ErrorText message={errorText} />
      <button
        type="button"
        onClick={() => void handleStop()}
        disabled={!voiceSession || runState === "idle" || runState === "stopping"}
      >
        {runState === "stopping"
          ? "Stopping..."
          : runState === "error"
            ? "Reset browser call"
            : "Stop browser call"}
      </button>

      {turns.length === 0 ? (
        <p>No final transcript turns captured yet.</p>
      ) : (
        <ul className="transcript-list">
          {turns.map((turn) => (
            <li key={turn.id}>
              <strong>{turn.direction}</strong> ({turn.modality})
              {turn.occurredAt ? ` @ ${formatDateTime(turn.occurredAt)}` : ""}
              <div>{turn.text}</div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString();
}
