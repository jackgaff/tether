package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type converseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

type BedrockAnalyzer struct {
	client  converseAPI
	modelID string
}

type analysisPromptEnvelope struct {
	CallRun                  CallRun                `json:"callRun"`
	Patient                  Patient                `json:"patient"`
	Caregiver                Caregiver              `json:"caregiver"`
	CallTemplate             analysisPromptTemplate `json:"callTemplate"`
	ScreeningSchedule        *ScreeningSchedule     `json:"screeningSchedule,omitempty"`
	TranscriptTurns          []CallTranscriptTurn   `json:"transcriptTurns"`
	RecentAnalyses           []AnalysisPayload      `json:"recentAnalyses"`
	KnownPeopleFromProfile   []FamilyMember         `json:"knownPeopleFromProfile"`
	KnownTopicsFromProfile   []string               `json:"knownTopicsFromProfile"`
	KnownReminiscenceSignals analysisPromptSignals  `json:"knownReminiscenceSignals"`
}

type analysisPromptTemplate struct {
	ID                    string          `json:"id"`
	Slug                  string          `json:"slug"`
	DisplayName           string          `json:"displayName"`
	CallType              string          `json:"callType"`
	Description           string          `json:"description"`
	DurationMinutes       int             `json:"durationMinutes"`
	CallPromptVersion     string          `json:"callPromptVersion"`
	AnalysisPromptVersion string          `json:"analysisPromptVersion"`
	Checklist             json.RawMessage `json:"checklist"`
}

type analysisPromptSignals struct {
	DurablePeopleRule string `json:"durablePeopleRule"`
	DurableMemoryRule string `json:"durableMemoryRule"`
}

func NewBedrockAnalyzer(client converseAPI, modelID string) *BedrockAnalyzer {
	return &BedrockAnalyzer{
		client:  client,
		modelID: strings.TrimSpace(modelID),
	}
}

func (a *BedrockAnalyzer) Analyze(ctx context.Context, promptContext AnalysisPromptContext) (AnalysisPayload, error) {
	envelope := analysisPromptEnvelope{
		CallRun:   promptContext.CallRun,
		Patient:   promptContext.Patient,
		Caregiver: promptContext.Caregiver,
		CallTemplate: analysisPromptTemplate{
			ID:                    promptContext.CallTemplate.ID,
			Slug:                  promptContext.CallTemplate.Slug,
			DisplayName:           promptContext.CallTemplate.DisplayName,
			CallType:              promptContext.CallTemplate.CallType,
			Description:           promptContext.CallTemplate.Description,
			DurationMinutes:       promptContext.CallTemplate.DurationMinutes,
			CallPromptVersion:     promptContext.CallTemplate.CallPromptVersion,
			AnalysisPromptVersion: promptContext.CallTemplate.AnalysisPromptVersion,
			Checklist:             promptContext.CallTemplate.Checklist,
		},
		ScreeningSchedule:      promptContext.ScreeningSchedule,
		TranscriptTurns:        promptContext.TranscriptTurns,
		RecentAnalyses:         promptContext.RecentAnalyses,
		KnownPeopleFromProfile: promptContext.Patient.MemoryProfile.FamilyMembers,
		KnownTopicsFromProfile: promptContext.Patient.MemoryProfile.TopicsToRevisit,
		KnownReminiscenceSignals: analysisPromptSignals{
			DurablePeopleRule: "Only extract people when they are clearly identifiable and useful for future calls.",
			DurableMemoryRule: "Only propose durable memory updates when the transcript includes concrete, evidence-backed details.",
		},
	}

	contextJSON, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return AnalysisPayload{}, fmt.Errorf("marshal analysis prompt context: %w", err)
	}

	output, err := a.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(a.modelID),
		System: []bedrocktypes.SystemContentBlock{
			&bedrocktypes.SystemContentBlockMemberText{
				Value: promptContext.CallTemplate.AnalysisPromptTemplate,
			},
		},
		Messages: []bedrocktypes.Message{
			{
				Role: bedrocktypes.ConversationRoleUser,
				Content: []bedrocktypes.ContentBlock{
					&bedrocktypes.ContentBlockMemberText{
						Value: "Return JSON only for this completed call context:\n\n" + string(contextJSON),
					},
				},
			},
		},
		InferenceConfig: &bedrocktypes.InferenceConfiguration{
			MaxTokens:   aws.Int32(1800),
			Temperature: aws.Float32(0.1),
		},
	})
	if err != nil {
		return AnalysisPayload{}, fmt.Errorf("run Nova analysis: %w", err)
	}

	text, err := readConverseTextOutput(output)
	if err != nil {
		return AnalysisPayload{}, newValidationError(err.Error())
	}

	payload, parseErr := parseAnalysisPayload(text)
	if parseErr == nil {
		return payload, nil
	}

	repairedPayload, repairErr := a.repairAnalysisPayload(
		ctx,
		promptContext.CallTemplate.AnalysisPromptTemplate,
		string(contextJSON),
		text,
		parseErr,
	)
	if repairErr != nil {
		return AnalysisPayload{}, newValidationErrorf("decode analysis json: %v", parseErr)
	}

	return repairedPayload, nil
}

func readConverseTextOutput(output *bedrockruntime.ConverseOutput) (string, error) {
	member, ok := output.Output.(*bedrocktypes.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected converse output type %T", output.Output)
	}

	var builder strings.Builder
	for _, block := range member.Value.Content {
		textBlock, ok := block.(*bedrocktypes.ContentBlockMemberText)
		if !ok {
			continue
		}
		builder.WriteString(textBlock.Value)
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", fmt.Errorf("analysis response did not contain text output")
	}

	return text, nil
}

func parseAnalysisPayload(raw string) (AnalysisPayload, error) {
	jsonBody, err := extractJSONObject(raw)
	if err != nil {
		return AnalysisPayload{}, err
	}

	var payload AnalysisPayload
	if err := json.Unmarshal([]byte(jsonBody), &payload); err != nil {
		return AnalysisPayload{}, err
	}
	normalizeAnalysisPayload(&payload)

	return payload, nil
}

func (a *BedrockAnalyzer) repairAnalysisPayload(
	ctx context.Context,
	systemPrompt string,
	contextJSON string,
	invalidOutput string,
	cause error,
) (AnalysisPayload, error) {
	repairRequest := strings.TrimSpace(`Your previous output was invalid JSON.
Return only one valid JSON object that matches the schema exactly.
Do not add markdown fences, commentary, or extra keys.

Validation error:
` + cause.Error() + `

Original context:
` + clipForPrompt(contextJSON, 32000) + `

Invalid output:
` + clipForPrompt(invalidOutput, 12000))

	output, err := a.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(a.modelID),
		System: []bedrocktypes.SystemContentBlock{
			&bedrocktypes.SystemContentBlockMemberText{
				Value: systemPrompt,
			},
		},
		Messages: []bedrocktypes.Message{
			{
				Role: bedrocktypes.ConversationRoleUser,
				Content: []bedrocktypes.ContentBlock{
					&bedrocktypes.ContentBlockMemberText{
						Value: repairRequest,
					},
				},
			},
		},
		InferenceConfig: &bedrocktypes.InferenceConfiguration{
			MaxTokens:   aws.Int32(1800),
			Temperature: aws.Float32(0.0),
		},
	})
	if err != nil {
		return AnalysisPayload{}, fmt.Errorf("run Nova analysis repair: %w", err)
	}

	text, err := readConverseTextOutput(output)
	if err != nil {
		return AnalysisPayload{}, err
	}

	payload, err := parseAnalysisPayload(text)
	if err != nil {
		return AnalysisPayload{}, err
	}

	return payload, nil
}

func clipForPrompt(value string, limit int) string {
	if limit <= 0 {
		return ""
	}

	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= limit {
		return trimmed
	}

	return trimmed[:limit] + "\n...[truncated]"
}

func extractJSONObject(raw string) (string, error) {
	trimmed := stripCodeFences(raw)
	start := strings.Index(trimmed, "{")
	if start == -1 {
		return "", fmt.Errorf("analysis response did not contain a JSON object")
	}

	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(trimmed); index++ {
		char := trimmed[index]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if char == '{' {
			depth++
			continue
		}
		if char == '}' {
			depth--
			if depth == 0 {
				return strings.TrimSpace(trimmed[start : index+1]), nil
			}
		}
	}

	return "", fmt.Errorf("analysis response did not contain a complete JSON object")
}

func stripCodeFences(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return trimmed
	}
	lines = lines[1:]
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		lines = lines[:len(lines)-1]
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
