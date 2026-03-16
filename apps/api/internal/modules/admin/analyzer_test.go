package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type fakeConverseClient struct {
	outputs  []*bedrockruntime.ConverseOutput
	err      error
	requests []*bedrockruntime.ConverseInput
}

func (f *fakeConverseClient) Converse(_ context.Context, params *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.requests = append(f.requests, params)
	if len(f.outputs) == 0 {
		return nil, fmt.Errorf("no fake outputs configured")
	}
	index := len(f.requests) - 1
	if index < len(f.outputs) {
		return f.outputs[index], nil
	}
	return f.outputs[len(f.outputs)-1], nil
}

func TestBedrockAnalyzerReturnsValidationErrorForMalformedJSON(t *testing.T) {
	t.Parallel()

	client := &fakeConverseClient{
		outputs: []*bedrockruntime.ConverseOutput{
			converseTextOutput("not-json"),
		},
	}
	analyzer := NewBedrockAnalyzer(client, "amazon.nova-2-lite-v1:0")

	_, err := analyzer.Analyze(context.Background(), AnalysisPromptContext{
		CallTemplate: CallTemplate{
			AnalysisPromptTemplate: "return strict json",
		},
	})
	if err == nil {
		t.Fatal("expected malformed analysis output to fail")
	}
	if !isValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected one repair attempt after invalid output, got %d requests", len(client.requests))
	}
}

func TestBedrockAnalyzerRepairsMalformedJSON(t *testing.T) {
	t.Parallel()

	client := &fakeConverseClient{
		outputs: []*bedrockruntime.ConverseOutput{
			converseTextOutput("```json\nnot valid\n```"),
			converseTextOutput(`{"summary":"Call completed.","escalationLevel":"none","followUpIntent":{"requestedByPatient":false,"timeframeBucket":"unspecified","evidence":"","confidence":0.4},"checkIn":{"orientationStatus":"unknown","mealsStatus":"uncertain","fluidsStatus":"uncertain","socialContact":"unknown","remindersNoted":[],"reminderDeclined":false,"mood":"unknown","sleep":"unknown","memoryFlags":[],"deliriumWatch":false,"deliriumPotentialTriggers":[]}}`),
		},
	}
	analyzer := NewBedrockAnalyzer(client, "amazon.nova-2-lite-v1:0")

	payload, err := analyzer.Analyze(context.Background(), AnalysisPromptContext{
		CallTemplate: CallTemplate{
			AnalysisPromptTemplate: "return strict json",
		},
	})
	if err != nil {
		t.Fatalf("expected repair to succeed, got %v", err)
	}
	if payload.Summary != "Call completed." {
		t.Fatalf("expected repaired payload summary, got %#v", payload)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected two requests (initial + repair), got %d", len(client.requests))
	}
}

func TestBedrockAnalyzerDoesNotDuplicateAnalysisPromptInsideContextJSON(t *testing.T) {
	t.Parallel()

	const marker = "ANALYSIS_TEMPLATE_MARKER_SHOULD_NOT_BE_IN_USER_CONTEXT"
	client := &fakeConverseClient{
		outputs: []*bedrockruntime.ConverseOutput{
			converseTextOutput(`{"summary":"Call completed.","escalationLevel":"none","followUpIntent":{"requestedByPatient":false,"timeframeBucket":"unspecified","evidence":"","confidence":0.4},"checkIn":{"orientationStatus":"unknown","mealsStatus":"uncertain","fluidsStatus":"uncertain","socialContact":"unknown","remindersNoted":[],"reminderDeclined":false,"mood":"unknown","sleep":"unknown","memoryFlags":[],"deliriumWatch":false,"deliriumPotentialTriggers":[]}}`),
		},
	}
	analyzer := NewBedrockAnalyzer(client, "amazon.nova-2-lite-v1:0")

	_, err := analyzer.Analyze(context.Background(), AnalysisPromptContext{
		CallTemplate: CallTemplate{
			AnalysisPromptTemplate: marker,
		},
		Patient: Patient{
			DisplayName: "Pat Doe",
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(client.requests) == 0 {
		t.Fatal("expected at least one request")
	}

	first := client.requests[0]
	systemText := readSystemText(first)
	userText := readUserText(first)
	if !strings.Contains(systemText, marker) {
		t.Fatalf("expected system prompt to include marker, got %q", systemText)
	}
	if strings.Contains(userText, marker) {
		t.Fatalf("expected user context payload to exclude analysis template marker, got %q", userText)
	}
	if strings.Contains(userText, "analysisPromptTemplate") {
		t.Fatalf("expected user context payload to exclude analysisPromptTemplate, got %q", userText)
	}
}

func converseTextOutput(value string) *bedrockruntime.ConverseOutput {
	return &bedrockruntime.ConverseOutput{
		Output: &bedrocktypes.ConverseOutputMemberMessage{
			Value: bedrocktypes.Message{
				Content: []bedrocktypes.ContentBlock{
					&bedrocktypes.ContentBlockMemberText{Value: value},
				},
			},
		},
	}
}

func readSystemText(input *bedrockruntime.ConverseInput) string {
	var builder strings.Builder
	for _, block := range input.System {
		if textBlock, ok := block.(*bedrocktypes.SystemContentBlockMemberText); ok {
			builder.WriteString(textBlock.Value)
		}
	}
	return builder.String()
}

func readUserText(input *bedrockruntime.ConverseInput) string {
	var builder strings.Builder
	for _, message := range input.Messages {
		for _, block := range message.Content {
			if textBlock, ok := block.(*bedrocktypes.ContentBlockMemberText); ok {
				builder.WriteString(textBlock.Value)
			}
		}
	}
	return builder.String()
}
