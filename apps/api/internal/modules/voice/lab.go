package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultLabConversationLimit = 20

type LabConversation struct {
	ID           string                `json:"id"`
	VoiceID      string                `json:"voiceId"`
	Status       string                `json:"status"`
	SystemPrompt string                `json:"systemPrompt,omitempty"`
	StopReason   string                `json:"stopReason,omitempty"`
	CreatedAt    time.Time             `json:"createdAt"`
	EndedAt      time.Time             `json:"endedAt"`
	JSONPath     string                `json:"jsonPath,omitempty"`
	MarkdownPath string                `json:"markdownPath,omitempty"`
	Turns        []LabConversationTurn `json:"turns"`
}

type LabConversationTurn struct {
	SequenceNo int       `json:"sequenceNo"`
	Direction  string    `json:"direction"`
	Modality   string    `json:"modality"`
	Text       string    `json:"text"`
	OccurredAt time.Time `json:"occurredAt"`
	StopReason string    `json:"stopReason,omitempty"`
}

func (s *Service) ListLabConversations(ctx context.Context, limit int) ([]LabConversation, error) {
	_ = ctx

	if limit <= 0 {
		limit = defaultLabConversationLimit
	}

	exportDir := strings.TrimSpace(s.cfg.VoiceLabExportDir)
	if exportDir == "" {
		return []LabConversation{}, nil
	}

	entries, err := os.ReadDir(exportDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LabConversation{}, nil
		}

		return nil, fmt.Errorf("read voice lab export dir: %w", err)
	}

	conversations := make([]LabConversation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		conversation, readErr := readLabConversation(filepath.Join(exportDir, entry.Name()))
		if readErr != nil {
			continue
		}

		conversations = append(conversations, conversation)
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].EndedAt.After(conversations[j].EndedAt)
	})

	if len(conversations) > limit {
		conversations = conversations[:limit]
	}

	return conversations, nil
}

func parseConversationLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLabConversationLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be a positive integer")
	}

	if limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}

	if limit > 100 {
		return 100, nil
	}

	return limit, nil
}

func readLabConversation(jsonPath string) (LabConversation, error) {
	body, err := os.ReadFile(jsonPath)
	if err != nil {
		return LabConversation{}, fmt.Errorf("read artifact json: %w", err)
	}

	var payload artifactPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return LabConversation{}, fmt.Errorf("decode artifact json: %w", err)
	}

	conversation := LabConversation{
		ID:           payload.Session.ID,
		VoiceID:      payload.Session.VoiceID,
		Status:       payload.Session.Status,
		SystemPrompt: strings.TrimSpace(payload.Session.SystemPrompt),
		StopReason:   payload.Session.StopReason,
		CreatedAt:    payload.Session.CreatedAt,
		EndedAt:      payload.Session.EndedAt,
		JSONPath:     jsonPath,
		MarkdownPath: strings.TrimSuffix(jsonPath, filepath.Ext(jsonPath)) + ".md",
		Turns:        make([]LabConversationTurn, 0, len(payload.Transcripts)),
	}

	for _, turn := range payload.Transcripts {
		conversation.Turns = append(conversation.Turns, LabConversationTurn{
			SequenceNo: turn.SequenceNo,
			Direction:  turn.Direction,
			Modality:   turn.Modality,
			Text:       turn.TranscriptText,
			OccurredAt: turn.OccurredAt,
			StopReason: turn.StopReason,
		})
	}

	if _, err := os.Stat(conversation.MarkdownPath); err != nil {
		conversation.MarkdownPath = ""
	}

	return conversation, nil
}
