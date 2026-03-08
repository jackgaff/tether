package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"nova-echoes/api/db"
	"nova-echoes/api/internal/modules/checkins"
	"nova-echoes/api/internal/modules/patients/preferences"
	"nova-echoes/api/internal/modules/voice"
)

func TestPostgresPersistence(t *testing.T) {
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := db.Open(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	if err := db.ResetForTest(ctx, database); err != nil {
		t.Fatalf("db.ResetForTest: %v", err)
	}

	preferenceStore := preferences.NewPostgresStore(database)
	if _, err := preferenceStore.Put(ctx, "patient-001", "tiffany"); err != nil {
		t.Fatalf("preferenceStore.Put: %v", err)
	}

	preference, ok, err := preferenceStore.Get(ctx, "patient-001")
	if err != nil {
		t.Fatalf("preferenceStore.Get: %v", err)
	}
	if !ok || preference.DefaultVoiceID != "tiffany" {
		t.Fatalf("expected stored preference, got %+v (ok=%v)", preference, ok)
	}

	checkInStore := checkins.NewPostgresStore(database)
	createdCheckIn, err := checkInStore.Create(ctx, checkins.CreateCheckInRequest{
		PatientID: "patient-001",
		Summary:   "Caller completed a check-in.",
		Status:    checkins.StatusCompleted,
		Agent:     "analysis-agent",
		Reminder:  "Keep tomorrow's card by the door.",
	})
	if err != nil {
		t.Fatalf("checkInStore.Create: %v", err)
	}

	checkIns, err := checkInStore.List(ctx, "patient-001")
	if err != nil {
		t.Fatalf("checkInStore.List: %v", err)
	}
	if len(checkIns) != 1 || checkIns[0].ID != createdCheckIn.ID {
		t.Fatalf("expected one stored check-in, got %+v", checkIns)
	}

	repo := voice.NewPostgresRepository(database)
	now := time.Now().UTC()
	session := voice.SessionRecord{
		ID:                     "session-001",
		PatientID:              "patient-001",
		Status:                 voice.StatusAwaitingStream,
		VoiceID:                "tiffany",
		InputSampleRateHz:      16000,
		OutputSampleRateHz:     24000,
		EndpointingSensitivity: "LOW",
		ModelID:                "amazon.nova-2-sonic-v1:0",
		AWSRegion:              "us-east-1",
		BedrockRegion:          "us-east-1",
		StreamTokenHash:        []byte("hash"),
		StreamTokenExpiresAt:   now.Add(time.Minute),
		LastActivityAt:         now,
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("repo.CreateSession: %v", err)
	}

	consumed, err := repo.ConsumeAttachToken(ctx, session.ID, []byte("hash"), now)
	if err != nil {
		t.Fatalf("repo.ConsumeAttachToken: %v", err)
	}
	if consumed.ID != session.ID {
		t.Fatalf("expected consumed session %q, got %q", session.ID, consumed.ID)
	}

	expiresAt := now.Add(8 * time.Minute)
	if err := repo.MarkSessionStreaming(ctx, session.ID, "prompt-001", expiresAt, now); err != nil {
		t.Fatalf("repo.MarkSessionStreaming: %v", err)
	}

	if err := repo.UpdateSessionMetadata(ctx, session.ID, "bedrock-session-001", "prompt-001", &expiresAt, now); err != nil {
		t.Fatalf("repo.UpdateSessionMetadata: %v", err)
	}

	if err := repo.SaveTranscriptTurn(ctx, voice.TranscriptTurn{
		VoiceSessionID:   session.ID,
		SequenceNo:       1,
		Direction:        "assistant",
		Modality:         "audio",
		TranscriptText:   "Let's take this one step at a time.",
		BedrockSessionID: "bedrock-session-001",
		PromptName:       "prompt-001",
		CompletionID:     "completion-001",
		ContentID:        "content-001",
		GenerationStage:  "FINAL",
		StopReason:       "END_TURN",
		OccurredAt:       now,
	}); err != nil {
		t.Fatalf("repo.SaveTranscriptTurn: %v", err)
	}

	rawPayload, err := json.Marshal(map[string]any{"event": map[string]any{"usageEvent": true}})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	if err := repo.SaveUsageEvent(ctx, voice.UsageEvent{
		VoiceSessionID:          session.ID,
		SequenceNo:              1,
		BedrockSessionID:        "bedrock-session-001",
		PromptName:              "prompt-001",
		CompletionID:            "completion-001",
		OutputSpeechTokensDelta: 10,
		OutputTextTokensDelta:   4,
		TotalOutputSpeechTokens: 10,
		TotalOutputTextTokens:   4,
		TotalOutputTokens:       14,
		TotalTokens:             14,
		Payload:                 rawPayload,
		EmittedAt:               now,
	}); err != nil {
		t.Fatalf("repo.SaveUsageEvent: %v", err)
	}

	if err := repo.MarkSessionEnded(ctx, session.ID, voice.StatusCompleted, "END_TURN", "", "", now); err != nil {
		t.Fatalf("repo.MarkSessionEnded: %v", err)
	}

	assertCount(t, database, "voice_sessions", 1)
	assertCount(t, database, "voice_transcript_turns", 1)
	assertCount(t, database, "voice_usage_events", 1)
	assertCount(t, database, "patient_preferences", 1)
	assertCount(t, database, "check_ins", 1)
}

func assertCount(t *testing.T, database *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := database.QueryRow(`select count(*) from ` + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}

	if count != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, count)
	}
}
