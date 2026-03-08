package voice

import (
	"encoding/json"
	"time"
)

const (
	StatusAwaitingStream  = "awaiting_stream"
	StatusStreaming       = "streaming"
	StatusDisconnectGrace = "disconnect_grace"
	StatusCompleted       = "completed"
	StatusFailed          = "failed"
	StatusExpired         = "expired"
)

const (
	wsMessageTextInput         = "text_input"
	wsMessageClientClose       = "client_close"
	wsMessageSessionReady      = "session_ready"
	wsMessageTranscriptPartial = "transcript_partial"
	wsMessageTranscriptFinal   = "transcript_final"
	wsMessageUsage             = "usage"
	wsMessageInterrupted       = "interrupted"
	wsMessageError             = "error"
	wsMessageSessionEnded      = "session_ended"
)

const (
	maxSessionSeconds = 480
	drainSeconds      = 5
	streamTokenTTL    = 2 * time.Minute
)

type AudioConfig struct {
	Encoding     string `json:"encoding"`
	SampleRateHz int    `json:"sampleRateHz"`
	Channels     int    `json:"channels"`
}

type CreateSessionRequest struct {
	PatientID    string `json:"patientId"`
	VoiceID      string `json:"voiceId,omitempty"`
	SystemPrompt string `json:"systemPrompt,omitempty"`
}

type SessionDescriptor struct {
	ID                   string      `json:"id"`
	VoiceID              string      `json:"voiceId"`
	WebSocketPath        string      `json:"websocketPath"`
	StreamToken          string      `json:"streamToken"`
	StreamTokenExpiresAt time.Time   `json:"streamTokenExpiresAt"`
	AudioInput           AudioConfig `json:"audioInput"`
	AudioOutput          AudioConfig `json:"audioOutput"`
	DrainSeconds         int         `json:"drainSeconds"`
	MaxSessionSeconds    int         `json:"maxSessionSeconds"`
}

type ArtifactPaths struct {
	JSONPath     string `json:"jsonPath,omitempty"`
	MarkdownPath string `json:"markdownPath,omitempty"`
}

type SessionRecord struct {
	ID                       string
	PatientID                string
	Status                   string
	VoiceID                  string
	SystemPrompt             string
	InputSampleRateHz        int
	OutputSampleRateHz       int
	EndpointingSensitivity   string
	ModelID                  string
	AWSRegion                string
	BedrockRegion            string
	BedrockSessionID         string
	PromptName               string
	StreamTokenHash          []byte
	StreamTokenExpiresAt     time.Time
	StreamTokenConsumedAt    *time.Time
	ClientConnectedAt        *time.Time
	ClientDisconnectedAt     *time.Time
	DisconnectGraceExpiresAt *time.Time
	SessionExpiresAt         *time.Time
	LastActivityAt           time.Time
	StopReason               string
	FailureCode              string
	FailureMessage           string
	CreatedAt                time.Time
	EndedAt                  *time.Time
	UpdatedAt                time.Time
}

type SessionArtifact struct {
	Session     SessionRecord
	Status      string
	StopReason  string
	EndedAt     time.Time
	Transcripts []TranscriptTurn
	UsageEvents []UsageEvent
}

type TranscriptTurn struct {
	VoiceSessionID   string
	SequenceNo       int
	Direction        string
	Modality         string
	TranscriptText   string
	BedrockSessionID string
	PromptName       string
	CompletionID     string
	ContentID        string
	GenerationStage  string
	StopReason       string
	OccurredAt       time.Time
}

type UsageEvent struct {
	VoiceSessionID          string
	SequenceNo              int
	BedrockSessionID        string
	PromptName              string
	CompletionID            string
	InputSpeechTokensDelta  int
	InputTextTokensDelta    int
	OutputSpeechTokensDelta int
	OutputTextTokensDelta   int
	TotalInputSpeechTokens  int
	TotalInputTextTokens    int
	TotalOutputSpeechTokens int
	TotalOutputTextTokens   int
	TotalInputTokens        int
	TotalOutputTokens       int
	TotalTokens             int
	Payload                 json.RawMessage
	EmittedAt               time.Time
}

type clientMessage struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type outputContentState struct {
	Direction        string
	Modality         string
	GenerationStage  string
	BedrockSessionID string
	PromptName       string
	CompletionID     string
	ContentID        string
	Text             string
	OccurredAt       time.Time
}
