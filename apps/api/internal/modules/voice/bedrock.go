package voice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"tether/api/internal/idgen"
)

type LiveSession interface {
	SendAudio(ctx context.Context, audio []byte) error
	SendText(ctx context.Context, text string) error
	EndConversation(ctx context.Context) error
	Events() <-chan LiveSessionEvent
	Close() error
}

type LiveSessionEvent struct {
	Payload []byte
	Err     error
}

type LiveSessionStarter interface {
	StartSession(ctx context.Context, input StartLiveSessionInput) (LiveSession, error)
}

type StartLiveSessionInput struct {
	ModelID                string
	VoiceID                string
	SystemPrompt           string
	PromptName             string
	AudioContentName       string
	InputSampleRateHz      int
	OutputSampleRateHz     int
	EndpointingSensitivity string
}

type bedrockInvoker interface {
	InvokeModelWithBidirectionalStream(ctx context.Context, params *bedrockruntime.InvokeModelWithBidirectionalStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithBidirectionalStreamOutput, error)
}

type BedrockAdapter struct {
	client bedrockInvoker
}

const maxTextInputEventBytes = 1024

func NewBedrockAdapter(client bedrockInvoker) *BedrockAdapter {
	return &BedrockAdapter{client: client}
}

func (a *BedrockAdapter) StartSession(ctx context.Context, input StartLiveSessionInput) (LiveSession, error) {
	modelID := normalizeLiveVoiceModelID(input.ModelID)
	if modelID != input.ModelID {
		log.Printf("normalizing live voice model id from %q to %q for bidirectional stream", input.ModelID, modelID)
	}

	output, err := a.client.InvokeModelWithBidirectionalStream(ctx, &bedrockruntime.InvokeModelWithBidirectionalStreamInput{
		ModelId: aws.String(modelID),
	})
	if err != nil {
		return nil, fmt.Errorf("invoke bidirectional stream: %w", err)
	}

	session := &bedrockLiveSession{
		stream:           output.GetStream(),
		events:           make(chan LiveSessionEvent, 64),
		promptName:       input.PromptName,
		audioContentName: input.AudioContentName,
		inputSampleRate:  input.InputSampleRateHz,
	}

	go session.readLoop()

	systemPromptContentName := ""
	if strings.TrimSpace(input.SystemPrompt) != "" {
		var idErr error
		systemPromptContentName, idErr = idgen.New()
		if idErr != nil {
			_ = session.Close()
			return nil, idErr
		}
	}

	for _, payload := range buildStartSessionEvents(input, systemPromptContentName) {
		if err := session.sendEvent(ctx, payload); err != nil {
			_ = session.Close()
			return nil, err
		}
	}

	return session, nil
}

type bedrockLiveSession struct {
	stream           *bedrockruntime.InvokeModelWithBidirectionalStreamEventStream
	events           chan LiveSessionEvent
	promptName       string
	audioContentName string
	inputSampleRate  int
	audioStarted     bool
	closeOnce        sync.Once
	endOnce          sync.Once
	sendMu           sync.Mutex
}

func (s *bedrockLiveSession) SendAudio(ctx context.Context, audio []byte) error {
	if len(audio) == 0 {
		return nil
	}

	events := make([]any, 0, 2)

	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if !s.audioStarted {
		s.audioStarted = true
		events = append(events, map[string]any{
			"contentStart": map[string]any{
				"promptName":  s.promptName,
				"contentName": s.audioContentName,
				"type":        "AUDIO",
				"interactive": true,
				"role":        "USER",
				"audioInputConfiguration": map[string]any{
					"mediaType":       "audio/lpcm",
					"sampleRateHertz": s.inputSampleRate,
					"sampleSizeBits":  16,
					"channelCount":    1,
					"audioType":       "SPEECH",
					"encoding":        "base64",
				},
			},
		})
	}

	events = append(events, map[string]any{
		"audioInput": map[string]any{
			"promptName":  s.promptName,
			"contentName": s.audioContentName,
			"content":     base64.StdEncoding.EncodeToString(audio),
		},
	})

	return s.sendEventsLocked(ctx, events)
}

func (s *bedrockLiveSession) SendText(ctx context.Context, text string) error {
	contentName, err := idgen.New()
	if err != nil {
		return err
	}

	events := []any{
		map[string]any{
			"contentStart": map[string]any{
				"promptName":  s.promptName,
				"contentName": contentName,
				"role":        "USER",
				"type":        "TEXT",
				"interactive": true,
				"textInputConfiguration": map[string]any{
					"mediaType": "text/plain",
				},
			},
		},
	}

	for _, event := range events {
		if err := s.sendEvent(ctx, event); err != nil {
			return err
		}
	}

	if err := s.sendTextChunks(ctx, s.promptName, contentName, text); err != nil {
		return err
	}

	if err := s.sendEvent(ctx, map[string]any{
		"contentEnd": map[string]any{
			"promptName":  s.promptName,
			"contentName": contentName,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *bedrockLiveSession) EndConversation(ctx context.Context) error {
	var sendErr error
	s.endOnce.Do(func() {
		s.sendMu.Lock()
		defer s.sendMu.Unlock()

		events := make([]any, 0, 3)

		if s.audioStarted {
			events = append(events, map[string]any{
				"contentEnd": map[string]any{
					"promptName":  s.promptName,
					"contentName": s.audioContentName,
				},
			})
		}

		events = append(events,
			map[string]any{
				"promptEnd": map[string]any{
					"promptName": s.promptName,
				},
			},
			map[string]any{
				"sessionEnd": map[string]any{},
			},
		)

		if err := s.sendEventsLocked(ctx, events); err != nil {
			sendErr = err
		}
	})

	return sendErr
}

func (s *bedrockLiveSession) Events() <-chan LiveSessionEvent {
	return s.events
}

func (s *bedrockLiveSession) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		closeErr = s.stream.Close()
	})
	return closeErr
}

func (s *bedrockLiveSession) readLoop() {
	defer close(s.events)

	for event := range s.stream.Events() {
		switch chunk := event.(type) {
		case *bedrocktypes.InvokeModelWithBidirectionalStreamOutputMemberChunk:
			s.events <- LiveSessionEvent{Payload: chunk.Value.Bytes}
		}
	}

	if err := s.stream.Err(); err != nil {
		s.events <- LiveSessionEvent{Err: err}
	}
}

func (s *bedrockLiveSession) sendEvent(ctx context.Context, event any) error {
	return s.sendEvents(ctx, []any{event})
}

func (s *bedrockLiveSession) sendEvents(ctx context.Context, events []any) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	return s.sendEventsLocked(ctx, events)
}

func (s *bedrockLiveSession) sendEventsLocked(ctx context.Context, events []any) error {
	for _, event := range events {
		payload, err := json.Marshal(map[string]any{"event": event})
		if err != nil {
			return fmt.Errorf("marshal bedrock event: %w", err)
		}

		if err := s.stream.Send(ctx, &bedrocktypes.InvokeModelWithBidirectionalStreamInputMemberChunk{
			Value: bedrocktypes.BidirectionalInputPayloadPart{
				Bytes: payload,
			},
		}); err != nil {
			return fmt.Errorf("send bedrock event: %w", err)
		}
	}

	return nil
}

func (s *bedrockLiveSession) sendTextChunks(ctx context.Context, promptName, contentName, text string) error {
	for _, chunk := range splitTextInputChunks(text, maxTextInputEventBytes) {
		if err := s.sendEvent(ctx, map[string]any{
			"textInput": map[string]any{
				"promptName":  promptName,
				"contentName": contentName,
				"content":     chunk,
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func buildStartSessionEvents(input StartLiveSessionInput, systemPromptContentName string) []any {
	events := []any{
		map[string]any{
			"sessionStart": map[string]any{
				"inferenceConfiguration": map[string]any{
					"maxTokens":   1024,
					"topP":        0.9,
					"temperature": 0.7,
				},
				"turnDetectionConfiguration": map[string]any{
					"endpointingSensitivity": input.EndpointingSensitivity,
				},
			},
		},
		map[string]any{
			"promptStart": map[string]any{
				"promptName": input.PromptName,
				"textOutputConfiguration": map[string]any{
					"mediaType": "text/plain",
				},
				"audioOutputConfiguration": map[string]any{
					"mediaType":       "audio/lpcm",
					"sampleRateHertz": input.OutputSampleRateHz,
					"sampleSizeBits":  16,
					"channelCount":    1,
					"voiceId":         input.VoiceID,
					"encoding":        "base64",
					"audioType":       "SPEECH",
				},
			},
		},
	}

	if systemPrompt := strings.TrimSpace(input.SystemPrompt); systemPrompt != "" {
		events = append(events, map[string]any{
			"contentStart": map[string]any{
				"promptName":  input.PromptName,
				"contentName": systemPromptContentName,
				"type":        "TEXT",
				"interactive": true,
				"role":        "SYSTEM",
				"textInputConfiguration": map[string]any{
					"mediaType": "text/plain",
				},
			},
		})

		for _, chunk := range splitTextInputChunks(systemPrompt, maxTextInputEventBytes) {
			events = append(events, map[string]any{
				"textInput": map[string]any{
					"promptName":  input.PromptName,
					"contentName": systemPromptContentName,
					"content":     chunk,
				},
			})
		}

		events = append(events, map[string]any{
			"contentEnd": map[string]any{
				"promptName":  input.PromptName,
				"contentName": systemPromptContentName,
			},
		})
	}

	return events
}

func splitTextInputChunks(text string, maxBytes int) []string {
	if text == "" {
		return nil
	}

	if maxBytes <= 0 {
		return []string{text}
	}

	chunks := make([]string, 0, 1)
	start := 0
	currentBytes := 0

	for index, runeValue := range text {
		runeBytes := utf8.RuneLen(runeValue)
		if runeBytes < 0 {
			runeBytes = 1
		}

		if currentBytes > 0 && currentBytes+runeBytes > maxBytes {
			chunks = append(chunks, text[start:index])
			start = index
			currentBytes = 0
		}

		currentBytes += runeBytes
	}

	chunks = append(chunks, text[start:])
	return chunks
}

func normalizeLiveVoiceModelID(modelID string) string {
	trimmed := strings.TrimSpace(modelID)

	switch trimmed {
	case "us.amazon.nova-2-sonic-v1:0", "global.amazon.nova-2-sonic-v1:0":
		return "amazon.nova-2-sonic-v1:0"
	default:
		return trimmed
	}
}
