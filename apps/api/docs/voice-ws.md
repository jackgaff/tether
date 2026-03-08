# Voice WebSocket Contract

`GET /api/v1/voice/sessions/{id}/stream?token=...` upgrades to the live browser voice stream.

Create the session first with `POST /api/v1/voice/sessions`. That request can include an optional `systemPrompt` string, which the backend injects into the Nova Sonic session before live audio begins. The backend currently rejects prompts larger than `40 KB`.

Rules:

- The `token` is short-lived, one-time-use, and bound to the session ID.
- The backend validates `Origin` against `ALLOWED_FRONTEND_ORIGINS`.
- The browser should stream mic audio at natural cadence in small chunks, roughly 32 ms per frame. Avoid batching large 250-500 ms blobs because it adds latency and hurts interruption handling.
- Binary frames are raw mono `pcm_s16le` audio at the negotiated input or output sample rate.
- JSON frames carry control messages and server events.

Client to server JSON:

```json
{ "type": "start_call" }
```

Send this once after `session_ready` when you want the assistant to open the call on its own.

```json
{ "type": "text_input", "text": "Can you repeat that more slowly?" }
```

```json
{ "type": "client_close" }
```

Server to client JSON:

```json
{
  "type": "session_ready",
  "voiceSessionId": "0195f0dc-0f1e-7ca3-a739-f8c6f95e0421",
  "sessionExpiresAt": "2026-03-08T17:02:39Z"
}
```

```json
{
  "type": "transcript_partial",
  "direction": "assistant",
  "text": "Let's slow down and go one step",
  "promptName": "0195f0dc-0f20-7b74-b327-0d720dce9dbb",
  "completionId": "cmp_001",
  "contentId": "cnt_001",
  "generationStage": "SPECULATIVE"
}
```

```json
{
  "type": "transcript_final",
  "sequenceNo": 3,
  "direction": "user",
  "modality": "audio",
  "text": "Could you remind me about tomorrow's ride?",
  "promptName": "0195f0dc-0f20-7b74-b327-0d720dce9dbb",
  "completionId": "cmp_001",
  "contentId": "cnt_002",
  "generationStage": "FINAL",
  "stopReason": "END_TURN",
  "occurredAt": "2026-03-08T16:54:17Z"
}
```

```json
{
  "type": "usage",
  "sequenceNo": 2,
  "promptName": "0195f0dc-0f20-7b74-b327-0d720dce9dbb",
  "completionId": "cmp_001",
  "deltas": {
    "inputSpeechTokens": 12,
    "inputTextTokens": 0,
    "outputSpeechTokens": 38,
    "outputTextTokens": 11
  },
  "totals": {
    "inputSpeechTokens": 45,
    "inputTextTokens": 0,
    "outputSpeechTokens": 76,
    "outputTextTokens": 25,
    "inputTokens": 45,
    "outputTokens": 101,
    "tokens": 146
  }
}
```

```json
{
  "type": "interrupted",
  "completionId": "cmp_001",
  "contentId": "cnt_003",
  "stopReason": "PARTIAL_TURN"
}
```

```json
{
  "type": "error",
  "code": "stream_error",
  "message": "voice session failed",
  "retryable": false
}
```

```json
{
  "type": "session_ended",
  "status": "completed",
  "stopReason": "END_TURN",
  "endedAt": "2026-03-08T16:56:12Z",
  "artifacts": {
    "jsonPath": "/Users/jackg/hack/nova-echoes/apps/api/testdata/voice-lab/0195f0dc-0f1e-7ca3-a739-f8c6f95e0421.json",
    "markdownPath": "/Users/jackg/hack/nova-echoes/apps/api/testdata/voice-lab/0195f0dc-0f1e-7ca3-a739-f8c6f95e0421.md"
  }
}
```

Finished prompt-lab sessions are exported to `VOICE_LAB_EXPORT_DIR` as both JSON and Markdown.
