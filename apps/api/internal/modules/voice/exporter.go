package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ArtifactExporter interface {
	Export(ctx context.Context, artifact SessionArtifact) (ArtifactPaths, error)
}

type NoopArtifactExporter struct{}

func NewNoopArtifactExporter() NoopArtifactExporter {
	return NoopArtifactExporter{}
}

func (NoopArtifactExporter) Export(context.Context, SessionArtifact) (ArtifactPaths, error) {
	return ArtifactPaths{}, nil
}

type FileArtifactExporter struct {
	dir string
}

func NewFileArtifactExporter(dir string) *FileArtifactExporter {
	return &FileArtifactExporter{dir: strings.TrimSpace(dir)}
}

func (e *FileArtifactExporter) Export(_ context.Context, artifact SessionArtifact) (ArtifactPaths, error) {
	if strings.TrimSpace(e.dir) == "" {
		return ArtifactPaths{}, nil
	}

	exportDir := e.dir
	if !filepath.IsAbs(exportDir) {
		absoluteDir, err := filepath.Abs(exportDir)
		if err != nil {
			return ArtifactPaths{}, fmt.Errorf("resolve artifact dir: %w", err)
		}
		exportDir = absoluteDir
	}

	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return ArtifactPaths{}, fmt.Errorf("create artifact dir: %w", err)
	}

	jsonPath := filepath.Join(exportDir, artifact.Session.ID+".json")
	markdownPath := filepath.Join(exportDir, artifact.Session.ID+".md")

	jsonBody, err := json.MarshalIndent(buildArtifactPayload(artifact), "", "  ")
	if err != nil {
		return ArtifactPaths{}, fmt.Errorf("marshal artifact json: %w", err)
	}
	jsonBody = append(jsonBody, '\n')

	if err := os.WriteFile(jsonPath, jsonBody, 0o644); err != nil {
		return ArtifactPaths{}, fmt.Errorf("write artifact json: %w", err)
	}

	if err := os.WriteFile(markdownPath, []byte(renderArtifactMarkdown(artifact)), 0o644); err != nil {
		return ArtifactPaths{}, fmt.Errorf("write artifact markdown: %w", err)
	}

	return ArtifactPaths{
		JSONPath:     jsonPath,
		MarkdownPath: markdownPath,
	}, nil
}

type artifactPayload struct {
	Session struct {
		ID                     string     `json:"id"`
		PatientID              string     `json:"patientId"`
		VoiceID                string     `json:"voiceId"`
		SystemPrompt           string     `json:"systemPrompt,omitempty"`
		Status                 string     `json:"status"`
		ModelID                string     `json:"modelId"`
		PromptName             string     `json:"promptName,omitempty"`
		BedrockSessionID       string     `json:"bedrockSessionId,omitempty"`
		StopReason             string     `json:"stopReason,omitempty"`
		FailureCode            string     `json:"failureCode,omitempty"`
		FailureMessage         string     `json:"failureMessage,omitempty"`
		InputSampleRateHz      int        `json:"inputSampleRateHz"`
		OutputSampleRateHz     int        `json:"outputSampleRateHz"`
		EndpointingSensitivity string     `json:"endpointingSensitivity"`
		CreatedAt              time.Time  `json:"createdAt"`
		SessionExpiresAt       *time.Time `json:"sessionExpiresAt,omitempty"`
		EndedAt                time.Time  `json:"endedAt"`
	} `json:"session"`
	Transcripts []artifactTranscript `json:"transcripts"`
	UsageEvents []artifactUsage      `json:"usageEvents"`
}

type artifactTranscript struct {
	SequenceNo       int       `json:"sequenceNo"`
	Direction        string    `json:"direction"`
	Modality         string    `json:"modality"`
	TranscriptText   string    `json:"transcriptText"`
	BedrockSessionID string    `json:"bedrockSessionId,omitempty"`
	PromptName       string    `json:"promptName,omitempty"`
	CompletionID     string    `json:"completionId,omitempty"`
	ContentID        string    `json:"contentId,omitempty"`
	GenerationStage  string    `json:"generationStage,omitempty"`
	StopReason       string    `json:"stopReason,omitempty"`
	OccurredAt       time.Time `json:"occurredAt"`
}

type artifactUsage struct {
	SequenceNo              int             `json:"sequenceNo"`
	BedrockSessionID        string          `json:"bedrockSessionId,omitempty"`
	PromptName              string          `json:"promptName,omitempty"`
	CompletionID            string          `json:"completionId,omitempty"`
	InputSpeechTokensDelta  int             `json:"inputSpeechTokensDelta"`
	InputTextTokensDelta    int             `json:"inputTextTokensDelta"`
	OutputSpeechTokensDelta int             `json:"outputSpeechTokensDelta"`
	OutputTextTokensDelta   int             `json:"outputTextTokensDelta"`
	TotalInputSpeechTokens  int             `json:"totalInputSpeechTokens"`
	TotalInputTextTokens    int             `json:"totalInputTextTokens"`
	TotalOutputSpeechTokens int             `json:"totalOutputSpeechTokens"`
	TotalOutputTextTokens   int             `json:"totalOutputTextTokens"`
	TotalInputTokens        int             `json:"totalInputTokens"`
	TotalOutputTokens       int             `json:"totalOutputTokens"`
	TotalTokens             int             `json:"totalTokens"`
	Payload                 json.RawMessage `json:"payload,omitempty"`
	EmittedAt               time.Time       `json:"emittedAt"`
}

func buildArtifactPayload(artifact SessionArtifact) artifactPayload {
	transcripts := sortedTranscripts(artifact.Transcripts)
	usageEvents := sortedUsageEvents(artifact.UsageEvents)

	payload := artifactPayload{}
	payload.Session.ID = artifact.Session.ID
	payload.Session.PatientID = artifact.Session.PatientID
	payload.Session.VoiceID = artifact.Session.VoiceID
	payload.Session.SystemPrompt = artifact.Session.SystemPrompt
	payload.Session.Status = artifact.Status
	payload.Session.ModelID = artifact.Session.ModelID
	payload.Session.PromptName = artifact.Session.PromptName
	payload.Session.BedrockSessionID = artifact.Session.BedrockSessionID
	payload.Session.StopReason = artifact.StopReason
	payload.Session.FailureCode = artifact.Session.FailureCode
	payload.Session.FailureMessage = artifact.Session.FailureMessage
	payload.Session.InputSampleRateHz = artifact.Session.InputSampleRateHz
	payload.Session.OutputSampleRateHz = artifact.Session.OutputSampleRateHz
	payload.Session.EndpointingSensitivity = artifact.Session.EndpointingSensitivity
	payload.Session.CreatedAt = artifact.Session.CreatedAt
	payload.Session.SessionExpiresAt = artifact.Session.SessionExpiresAt
	payload.Session.EndedAt = artifact.EndedAt

	payload.Transcripts = make([]artifactTranscript, 0, len(transcripts))
	for _, transcript := range transcripts {
		payload.Transcripts = append(payload.Transcripts, artifactTranscript{
			SequenceNo:       transcript.SequenceNo,
			Direction:        transcript.Direction,
			Modality:         transcript.Modality,
			TranscriptText:   transcript.TranscriptText,
			BedrockSessionID: transcript.BedrockSessionID,
			PromptName:       transcript.PromptName,
			CompletionID:     transcript.CompletionID,
			ContentID:        transcript.ContentID,
			GenerationStage:  transcript.GenerationStage,
			StopReason:       transcript.StopReason,
			OccurredAt:       transcript.OccurredAt,
		})
	}

	payload.UsageEvents = make([]artifactUsage, 0, len(usageEvents))
	for _, usage := range usageEvents {
		payload.UsageEvents = append(payload.UsageEvents, artifactUsage{
			SequenceNo:              usage.SequenceNo,
			BedrockSessionID:        usage.BedrockSessionID,
			PromptName:              usage.PromptName,
			CompletionID:            usage.CompletionID,
			InputSpeechTokensDelta:  usage.InputSpeechTokensDelta,
			InputTextTokensDelta:    usage.InputTextTokensDelta,
			OutputSpeechTokensDelta: usage.OutputSpeechTokensDelta,
			OutputTextTokensDelta:   usage.OutputTextTokensDelta,
			TotalInputSpeechTokens:  usage.TotalInputSpeechTokens,
			TotalInputTextTokens:    usage.TotalInputTextTokens,
			TotalOutputSpeechTokens: usage.TotalOutputSpeechTokens,
			TotalOutputTextTokens:   usage.TotalOutputTextTokens,
			TotalInputTokens:        usage.TotalInputTokens,
			TotalOutputTokens:       usage.TotalOutputTokens,
			TotalTokens:             usage.TotalTokens,
			Payload:                 usage.Payload,
			EmittedAt:               usage.EmittedAt,
		})
	}

	return payload
}

func renderArtifactMarkdown(artifact SessionArtifact) string {
	transcripts := sortedTranscripts(artifact.Transcripts)
	usageEvents := sortedUsageEvents(artifact.UsageEvents)

	var builder strings.Builder

	builder.WriteString("# Voice Prompt Lab Session\n\n")
	builder.WriteString(fmt.Sprintf("- Session ID: `%s`\n", artifact.Session.ID))
	builder.WriteString(fmt.Sprintf("- Patient ID: `%s`\n", artifact.Session.PatientID))
	builder.WriteString(fmt.Sprintf("- Voice ID: `%s`\n", artifact.Session.VoiceID))
	builder.WriteString(fmt.Sprintf("- Status: `%s`\n", artifact.Status))
	if artifact.StopReason != "" {
		builder.WriteString(fmt.Sprintf("- Stop reason: `%s`\n", artifact.StopReason))
	}
	builder.WriteString(fmt.Sprintf("- Input sample rate: `%d`\n", artifact.Session.InputSampleRateHz))
	builder.WriteString(fmt.Sprintf("- Output sample rate: `%d`\n", artifact.Session.OutputSampleRateHz))
	builder.WriteString(fmt.Sprintf("- Endpointing sensitivity: `%s`\n", artifact.Session.EndpointingSensitivity))
	builder.WriteString(fmt.Sprintf("- Created at: `%s`\n", artifact.Session.CreatedAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Ended at: `%s`\n", artifact.EndedAt.Format(time.RFC3339)))

	if strings.TrimSpace(artifact.Session.SystemPrompt) != "" {
		builder.WriteString("\n## System Prompt\n\n")
		builder.WriteString("```text\n")
		builder.WriteString(strings.TrimSpace(artifact.Session.SystemPrompt))
		builder.WriteString("\n```\n")
	}

	builder.WriteString("\n## Transcript\n\n")
	if len(transcripts) == 0 {
		builder.WriteString("_No FINAL transcript turns were captured._\n")
	} else {
		for _, transcript := range transcripts {
			builder.WriteString(fmt.Sprintf("### %d. %s (%s)\n\n", transcript.SequenceNo, titleCase(transcript.Direction), transcript.Modality))
			builder.WriteString(strings.TrimSpace(transcript.TranscriptText))
			builder.WriteString("\n\n")
			builder.WriteString(fmt.Sprintf("- Occurred at: `%s`\n", transcript.OccurredAt.Format(time.RFC3339)))
			if transcript.GenerationStage != "" {
				builder.WriteString(fmt.Sprintf("- Generation stage: `%s`\n", transcript.GenerationStage))
			}
			if transcript.StopReason != "" {
				builder.WriteString(fmt.Sprintf("- Stop reason: `%s`\n", transcript.StopReason))
			}
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n## Usage\n\n")
	if len(usageEvents) == 0 {
		builder.WriteString("_No usage events were captured._\n")
	} else {
		last := usageEvents[len(usageEvents)-1]
		builder.WriteString(fmt.Sprintf("- Total tokens: `%d`\n", last.TotalTokens))
		builder.WriteString(fmt.Sprintf("- Input tokens: `%d`\n", last.TotalInputTokens))
		builder.WriteString(fmt.Sprintf("- Output tokens: `%d`\n", last.TotalOutputTokens))
		builder.WriteString(fmt.Sprintf("- Total input speech tokens: `%d`\n", last.TotalInputSpeechTokens))
		builder.WriteString(fmt.Sprintf("- Total output speech tokens: `%d`\n", last.TotalOutputSpeechTokens))
		builder.WriteString(fmt.Sprintf("- Total input text tokens: `%d`\n", last.TotalInputTextTokens))
		builder.WriteString(fmt.Sprintf("- Total output text tokens: `%d`\n", last.TotalOutputTextTokens))
	}

	return builder.String()
}

func sortedTranscripts(items []TranscriptTurn) []TranscriptTurn {
	sorted := append([]TranscriptTurn(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].SequenceNo < sorted[j].SequenceNo
	})
	return sorted
}

func sortedUsageEvents(items []UsageEvent) []UsageEvent {
	sorted := append([]UsageEvent(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].SequenceNo < sorted[j].SequenceNo
	})
	return sorted
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	return strings.ToUpper(value[:1]) + value[1:]
}
