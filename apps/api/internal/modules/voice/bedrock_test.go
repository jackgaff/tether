package voice

import "testing"

func TestSplitTextInputChunksRespectsByteLimit(t *testing.T) {
	t.Parallel()

	text := "abcdEFGHijkl"
	chunks := splitTextInputChunks(text, 4)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	expected := []string{"abcd", "EFGH", "ijkl"}
	for index, chunk := range chunks {
		if chunk != expected[index] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", index, expected[index], chunk)
		}
	}
}

func TestSplitTextInputChunksPreservesUTF8Boundaries(t *testing.T) {
	t.Parallel()

	text := "aé世b"
	chunks := splitTextInputChunks(text, 4)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	expected := []string{"aé", "世b"}
	for index, chunk := range chunks {
		if chunk != expected[index] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", index, expected[index], chunk)
		}
	}
}

func TestBuildStartSessionEventsOnlyBootstrapsPromptState(t *testing.T) {
	t.Parallel()

	events := buildStartSessionEvents(StartLiveSessionInput{
		PromptName:         "prompt-001",
		VoiceID:            "matthew",
		SystemPrompt:       "Hello there.",
		InputSampleRateHz:  16000,
		OutputSampleRateHz: 24000,
	}, "system-001")

	eventNames := make([]string, 0, len(events))
	for _, event := range events {
		eventNames = append(eventNames, eventName(event))
	}

	expected := []string{
		"sessionStart",
		"promptStart",
		"contentStart",
		"textInput",
		"contentEnd",
	}

	if len(eventNames) != len(expected) {
		t.Fatalf("expected %d events, got %d (%v)", len(expected), len(eventNames), eventNames)
	}

	for index, name := range expected {
		if eventNames[index] != name {
			t.Fatalf("event %d mismatch: expected %q, got %q (%v)", index, name, eventNames[index], eventNames)
		}
	}

	systemPromptEvent, ok := events[2].(map[string]any)
	if !ok {
		t.Fatalf("expected system prompt event to be a map, got %T", events[2])
	}

	contentStart, ok := systemPromptEvent["contentStart"].(map[string]any)
	if !ok {
		t.Fatalf("expected system prompt event to be contentStart, got %#v", events[2])
	}

	if contentStart["contentName"] != "system-001" {
		t.Fatalf("expected system prompt content to start first, got %#v", contentStart)
	}

	if interactive, ok := contentStart["interactive"].(bool); !ok || !interactive {
		t.Fatalf("expected system prompt content to be interactive, got %#v", contentStart["interactive"])
	}
}

func TestNormalizeLiveVoiceModelID(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"us.amazon.nova-2-sonic-v1:0":     "amazon.nova-2-sonic-v1:0",
		"global.amazon.nova-2-sonic-v1:0": "amazon.nova-2-sonic-v1:0",
		"amazon.nova-2-sonic-v1:0":        "amazon.nova-2-sonic-v1:0",
		"amazon.nova-sonic-v1:0":          "amazon.nova-sonic-v1:0",
	}

	for input, expected := range cases {
		if actual := normalizeLiveVoiceModelID(input); actual != expected {
			t.Fatalf("normalizeLiveVoiceModelID(%q): expected %q, got %q", input, expected, actual)
		}
	}
}

func eventName(event any) string {
	payload, ok := event.(map[string]any)
	if !ok {
		return ""
	}

	for key := range payload {
		return key
	}

	return ""
}
