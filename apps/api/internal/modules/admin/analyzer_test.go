package admin

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type fakeConverseClient struct {
	output *bedrockruntime.ConverseOutput
	err    error
}

func (f fakeConverseClient) Converse(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

func TestBedrockAnalyzerReturnsValidationErrorForMalformedJSON(t *testing.T) {
	t.Parallel()

	analyzer := NewBedrockAnalyzer(fakeConverseClient{
		output: &bedrockruntime.ConverseOutput{
			Output: &bedrocktypes.ConverseOutputMemberMessage{
				Value: bedrocktypes.Message{
					Content: []bedrocktypes.ContentBlock{
						&bedrocktypes.ContentBlockMemberText{Value: "not-json"},
					},
				},
			},
		},
	}, "amazon.nova-2-lite-v1:0")

	_, err := analyzer.Analyze(context.Background(), AnalysisPromptContext{})
	if err == nil {
		t.Fatal("expected malformed analysis output to fail")
	}
	if !isValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}
