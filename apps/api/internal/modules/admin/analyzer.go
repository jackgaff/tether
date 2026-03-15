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

func NewBedrockAnalyzer(client converseAPI, modelID string) *BedrockAnalyzer {
	return &BedrockAnalyzer{
		client:  client,
		modelID: strings.TrimSpace(modelID),
	}
}

func (a *BedrockAnalyzer) Analyze(ctx context.Context, promptContext AnalysisPromptContext) (AnalysisPayload, error) {
	contextJSON, err := json.MarshalIndent(promptContext, "", "  ")
	if err != nil {
		return AnalysisPayload{}, fmt.Errorf("marshal analysis prompt context: %w", err)
	}

	output, err := a.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(a.modelID),
		System: []bedrocktypes.SystemContentBlock{
			&bedrocktypes.SystemContentBlockMemberText{
				Value: analysisSystemPrompt(promptContext.CallRun.CallType),
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

	jsonBody, err := extractJSONObject(text)
	if err != nil {
		return AnalysisPayload{}, newValidationError(err.Error())
	}

	var payload AnalysisPayload
	if err := json.Unmarshal([]byte(jsonBody), &payload); err != nil {
		return AnalysisPayload{}, newValidationErrorf("decode analysis json: %v", err)
	}

	return payload, nil
}

func analysisSystemPrompt(callType string) string {
	var specific string
	switch callType {
	case CallTypeScreening:
		specific = `
- Include a screening object.
- screening.screeningCompletionStatus must be complete, partial, or aborted.
- screening.screeningScoreInterpretation must be one of routine_follow_up, caregiver_review_suggested, clinical_review_suggested, or incomplete.
- Do not diagnose. Use observational language only.`
	case CallTypeCheckIn:
		specific = `
- Include a checkIn object.
- Capture food, hydration, mood, routine, social contact, and any explicit request for another check-in.
- followUpIntent.requestedByPatient should only be true when the transcript clearly supports it.`
	case CallTypeReminiscence:
		specific = `
- Include a reminiscence object.
- Focus on comfort, distress triggers, people/topics mentioned, and future supportive reminiscence ideas.
- Avoid framing memory changes as diagnosis or stage assessment.`
	default:
		specific = ""
	}

	return `You are Echo's structured extraction worker.
Return JSON only.

Rules:
- Be conservative and non-diagnostic.
- Ground every concern in transcript evidence.
- Never suggest medication changes.
- Do not invent exact dates or times.
- Use timeframe buckets only: same_day, tomorrow, few_days, next_week, two_weeks, unspecified.
- Use call types only: screening, check_in, reminiscence.
- Use escalation levels only: none, caregiver_soon, caregiver_now, clinical_review.
- Use risk severities only: info, watch, urgent.

Required shape:
{
  "summary": "",
  "salientEvidence": [{"quote": "", "reason": ""}],
  "riskFlags": [{"flagType": "", "severity": "info|watch|urgent", "evidence": "", "reason": "", "confidence": 0.0}],
  "escalationLevel": "none|caregiver_soon|caregiver_now|clinical_review",
  "caregiverReviewReason": "",
  "followUpIntent": {
    "requestedByPatient": false,
    "timeframeBucket": "same_day|tomorrow|few_days|next_week|two_weeks|unspecified",
    "evidence": "",
    "confidence": 0.0
  },
  "nextCallRecommendation": {
    "callType": "screening|check_in|reminiscence",
    "windowBucket": "same_day|tomorrow|few_days|next_week|two_weeks|unspecified",
    "goal": ""
  }
}
` + specific
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

func extractJSONObject(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("analysis response did not contain a JSON object")
	}

	return trimmed[start : end+1], nil
}
