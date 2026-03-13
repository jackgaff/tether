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
				Value: `You are a dementia-support analysis agent.
Return JSON only.

Use this exact schema:
{
  "call_type_completed": "orientation|reminder|wellbeing|reminiscence",
  "patient_state": {
    "orientation": "good|mixed|poor|unclear",
    "mood": "positive|neutral|anxious|sad|distressed|unclear",
    "engagement": "high|medium|low",
    "confidence": 0.0
  },
  "signals": {
    "repetition": 0,
    "routine_adherence_issue": false,
    "sleep_concern": false,
    "nutrition_or_hydration_concern": false,
    "possible_safety_concern": false,
    "possible_bpsd_signals": [],
    "social_connection_need": false
  },
  "evidence": [
    {
      "quote": "",
      "why_it_matters": ""
    }
  ],
  "dashboard_summary": "",
  "caregiver_summary": "",
  "recommended_next_call": {
    "type": "orientation|reminder|wellbeing|reminiscence",
    "timing": "",
    "duration_minutes": 0,
    "goal": ""
  },
  "escalation_level": "none|caregiver_soon|caregiver_now|clinical_review",
  "uncertainties": []
}

Rules:
- Do not diagnose.
- Do not suggest medication changes.
- Be conservative.
- Ground concerns in transcript evidence.
- If evidence is weak, say so in uncertainties.`,
			},
		},
		Messages: []bedrocktypes.Message{
			{
				Role: bedrocktypes.ConversationRoleUser,
				Content: []bedrocktypes.ContentBlock{
					&bedrocktypes.ContentBlockMemberText{
						Value: "Analyze this completed call context and return JSON only:\n\n" + string(contextJSON),
					},
				},
			},
		},
		InferenceConfig: &bedrocktypes.InferenceConfiguration{
			MaxTokens:   aws.Int32(1400),
			Temperature: aws.Float32(0.2),
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
