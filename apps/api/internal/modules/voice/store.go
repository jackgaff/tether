package voice

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository interface {
	CreateSession(ctx context.Context, session SessionRecord) error
	LinkCallRun(ctx context.Context, callRunID, patientID, sessionID string, now time.Time) error
	ConsumeAttachToken(ctx context.Context, sessionID string, tokenHash []byte, now time.Time) (SessionRecord, error)
	MarkSessionStreaming(ctx context.Context, sessionID, promptName string, sessionExpiresAt, now time.Time) error
	MarkCallRunInProgress(ctx context.Context, sessionID string, startedAt time.Time) error
	UpdateSessionMetadata(ctx context.Context, sessionID, bedrockSessionID, promptName string, sessionExpiresAt *time.Time, now time.Time) error
	MarkDisconnectGrace(ctx context.Context, sessionID string, disconnectedAt, graceExpiresAt time.Time) error
	MarkSessionEnded(ctx context.Context, sessionID, status, stopReason, failureCode, failureMessage string, endedAt time.Time) error
	MarkCallRunEnded(ctx context.Context, sessionID, status, stopReason string, endedAt time.Time) error
	TouchSession(ctx context.Context, sessionID string, now time.Time) error
	SaveTranscriptTurn(ctx context.Context, turn TranscriptTurn) error
	SaveUsageEvent(ctx context.Context, event UsageEvent) error
	ListTranscriptTurns(ctx context.Context, sessionID string) ([]TranscriptTurn, error)
	ListUsageEvents(ctx context.Context, sessionID string) ([]UsageEvent, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session SessionRecord) error {
	_, err := r.db.ExecContext(ctx, `
		insert into voice_sessions (
			id,
			patient_id,
			status,
			voice_id,
			system_prompt,
			input_sample_rate_hz,
			output_sample_rate_hz,
			endpointing_sensitivity,
			model_id,
			aws_region,
			bedrock_region,
			stream_token_hash,
			stream_token_expires_at,
			last_activity_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, session.ID, session.PatientID, session.Status, session.VoiceID, nullableString(session.SystemPrompt), session.InputSampleRateHz, session.OutputSampleRateHz,
		session.EndpointingSensitivity, session.ModelID, session.AWSRegion, session.BedrockRegion, session.StreamTokenHash,
		session.StreamTokenExpiresAt, session.LastActivityAt)
	if err != nil {
		return fmt.Errorf("create voice session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) LinkCallRun(ctx context.Context, callRunID, patientID, sessionID string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin call run link tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		storedPatientID string
		status          string
		sourceSessionID sql.NullString
	)
	row := tx.QueryRowContext(ctx, `
		select patient_id, status, source_voice_session_id
		from call_runs
		where id = $1
		for update
	`, callRunID)
	if scanErr := row.Scan(&storedPatientID, &status, &sourceSessionID); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			err = ErrCallRunNotFound
			return err
		}

		err = fmt.Errorf("load call run for voice session link: %w", scanErr)
		return err
	}

	if storedPatientID != patientID {
		err = ErrCallRunPatientMismatch
		return err
	}
	if sourceSessionID.Valid && strings.TrimSpace(sourceSessionID.String) != "" {
		err = ErrCallRunAlreadyLinked
		return err
	}
	if status != "requested" {
		err = ErrCallRunLinkInvalid
		return err
	}

	if _, execErr := tx.ExecContext(ctx, `
		update call_runs
		set source_voice_session_id = $2,
		    updated_at = $3
		where id = $1
	`, callRunID, sessionID, now); execErr != nil {
		err = fmt.Errorf("link call run to voice session: %w", execErr)
		return err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit call run link tx: %w", commitErr)
	}

	return nil
}

func (r *PostgresRepository) ConsumeAttachToken(ctx context.Context, sessionID string, tokenHash []byte, now time.Time) (SessionRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionRecord{}, fmt.Errorf("begin voice session attach tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
		select
			id,
			patient_id,
			status,
			voice_id,
			coalesce(system_prompt, ''),
			input_sample_rate_hz,
			output_sample_rate_hz,
			endpointing_sensitivity,
			model_id,
			aws_region,
			bedrock_region,
			stream_token_hash,
			stream_token_expires_at,
			stream_token_consumed_at,
			last_activity_at,
			created_at,
			updated_at
		from voice_sessions
		where id = $1
		for update
	`, sessionID)

	var (
		record              SessionRecord
		consumedAt          sql.NullTime
		tokenHashFromRecord []byte
	)
	if scanErr := row.Scan(
		&record.ID,
		&record.PatientID,
		&record.Status,
		&record.VoiceID,
		&record.SystemPrompt,
		&record.InputSampleRateHz,
		&record.OutputSampleRateHz,
		&record.EndpointingSensitivity,
		&record.ModelID,
		&record.AWSRegion,
		&record.BedrockRegion,
		&tokenHashFromRecord,
		&record.StreamTokenExpiresAt,
		&consumedAt,
		&record.LastActivityAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			err = ErrSessionNotFound
			return SessionRecord{}, err
		}

		err = fmt.Errorf("load voice session: %w", scanErr)
		return SessionRecord{}, err
	}

	if subtle.ConstantTimeCompare(tokenHashFromRecord, tokenHash) != 1 {
		err = ErrInvalidStreamToken
		return SessionRecord{}, err
	}

	if now.After(record.StreamTokenExpiresAt) {
		err = ErrTokenExpired
		return SessionRecord{}, err
	}

	if consumedAt.Valid {
		err = ErrStreamConsumed
		return SessionRecord{}, err
	}

	record.StreamTokenConsumedAt = &now
	record.ClientConnectedAt = &now
	record.LastActivityAt = now
	record.Status = StatusStreaming

	if _, execErr := tx.ExecContext(ctx, `
		update voice_sessions
		set status = $2,
		    stream_token_consumed_at = $3,
		    client_connected_at = $3,
		    last_activity_at = $3,
		    updated_at = $3
		where id = $1
	`, record.ID, StatusStreaming, now); execErr != nil {
		err = fmt.Errorf("consume stream token: %w", execErr)
		return SessionRecord{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		err = fmt.Errorf("commit voice session attach tx: %w", commitErr)
		return SessionRecord{}, err
	}

	return record, nil
}

func (r *PostgresRepository) MarkSessionStreaming(ctx context.Context, sessionID, promptName string, sessionExpiresAt, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update voice_sessions
		set status = $2,
		    prompt_name = $3,
		    session_expires_at = $4,
		    last_activity_at = $5,
		    updated_at = $5
		where id = $1
	`, sessionID, StatusStreaming, promptName, sessionExpiresAt, now)
	if err != nil {
		return fmt.Errorf("mark session streaming: %w", err)
	}

	return nil
}

func (r *PostgresRepository) MarkCallRunInProgress(ctx context.Context, sessionID string, startedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update call_runs
		set status = 'in_progress',
		    started_at = coalesce(started_at, $2),
		    updated_at = $2
		where source_voice_session_id = $1
		  and status = 'requested'
	`, sessionID, startedAt)
	if err != nil {
		return fmt.Errorf("mark call run in progress: %w", err)
	}

	return nil
}

func (r *PostgresRepository) UpdateSessionMetadata(ctx context.Context, sessionID, bedrockSessionID, promptName string, sessionExpiresAt *time.Time, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update voice_sessions
		set bedrock_session_id = coalesce(nullif($2, ''), bedrock_session_id),
		    prompt_name = coalesce(nullif($3, ''), prompt_name),
		    session_expires_at = coalesce($4, session_expires_at),
		    last_activity_at = $5,
		    updated_at = $5
		where id = $1
	`, sessionID, bedrockSessionID, promptName, sessionExpiresAt, now)
	if err != nil {
		return fmt.Errorf("update session metadata: %w", err)
	}

	return nil
}

func (r *PostgresRepository) MarkDisconnectGrace(ctx context.Context, sessionID string, disconnectedAt, graceExpiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update voice_sessions
		set status = $2,
		    client_disconnected_at = $3,
		    disconnect_grace_expires_at = $4,
		    last_activity_at = $3,
		    updated_at = $3
		where id = $1
	`, sessionID, StatusDisconnectGrace, disconnectedAt, graceExpiresAt)
	if err != nil {
		return fmt.Errorf("mark disconnect grace: %w", err)
	}

	return nil
}

func (r *PostgresRepository) MarkSessionEnded(ctx context.Context, sessionID, status, stopReason, failureCode, failureMessage string, endedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update voice_sessions
		set status = $2,
		    stop_reason = nullif($3, ''),
		    failure_code = nullif($4, ''),
		    failure_message = nullif($5, ''),
		    ended_at = $6,
		    last_activity_at = $6,
		    updated_at = $6
		where id = $1
	`, sessionID, status, stopReason, failureCode, failureMessage, endedAt)
	if err != nil {
		return fmt.Errorf("mark session ended: %w", err)
	}

	return nil
}

func (r *PostgresRepository) MarkCallRunEnded(ctx context.Context, sessionID, status, stopReason string, endedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update call_runs
		set status = $2,
		    ended_at = $3,
		    stop_reason = nullif($4, ''),
		    updated_at = $3
		where source_voice_session_id = $1
		  and status in ('requested', 'in_progress')
	`, sessionID, status, endedAt, stopReason)
	if err != nil {
		return fmt.Errorf("mark call run ended: %w", err)
	}

	return nil
}

func (r *PostgresRepository) TouchSession(ctx context.Context, sessionID string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update voice_sessions
		set last_activity_at = $2,
		    updated_at = $2
		where id = $1
	`, sessionID, now)
	if err != nil {
		return fmt.Errorf("touch voice session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) SaveTranscriptTurn(ctx context.Context, turn TranscriptTurn) error {
	_, err := r.db.ExecContext(ctx, `
		insert into voice_transcript_turns (
			voice_session_id,
			sequence_no,
			direction,
			modality,
			transcript_text,
			bedrock_session_id,
			prompt_name,
			completion_id,
			content_id,
			generation_stage,
			stop_reason,
			occurred_at
		) values ($1, $2, $3, $4, $5, nullif($6, ''), nullif($7, ''), nullif($8, ''), nullif($9, ''), nullif($10, ''), nullif($11, ''), $12)
	`, turn.VoiceSessionID, turn.SequenceNo, turn.Direction, turn.Modality, turn.TranscriptText, turn.BedrockSessionID, turn.PromptName,
		turn.CompletionID, turn.ContentID, turn.GenerationStage, turn.StopReason, turn.OccurredAt)
	if err != nil {
		return fmt.Errorf("save transcript turn: %w", err)
	}

	return nil
}

func (r *PostgresRepository) SaveUsageEvent(ctx context.Context, event UsageEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal usage payload: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		insert into voice_usage_events (
			voice_session_id,
			sequence_no,
			bedrock_session_id,
			prompt_name,
			completion_id,
			input_speech_tokens_delta,
			input_text_tokens_delta,
			output_speech_tokens_delta,
			output_text_tokens_delta,
			total_input_speech_tokens,
			total_input_text_tokens,
			total_output_speech_tokens,
			total_output_text_tokens,
			total_input_tokens,
			total_output_tokens,
			total_tokens,
			payload,
			emitted_at
		) values ($1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17::jsonb, $18)
	`, event.VoiceSessionID, event.SequenceNo, event.BedrockSessionID, event.PromptName, event.CompletionID,
		event.InputSpeechTokensDelta, event.InputTextTokensDelta, event.OutputSpeechTokensDelta, event.OutputTextTokensDelta,
		event.TotalInputSpeechTokens, event.TotalInputTextTokens, event.TotalOutputSpeechTokens, event.TotalOutputTextTokens,
		event.TotalInputTokens, event.TotalOutputTokens, event.TotalTokens, string(payload), event.EmittedAt)
	if err != nil {
		return fmt.Errorf("save usage event: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListTranscriptTurns(ctx context.Context, sessionID string) ([]TranscriptTurn, error) {
	rows, err := r.db.QueryContext(ctx, `
		select
			voice_session_id,
			sequence_no,
			direction,
			modality,
			transcript_text,
			coalesce(bedrock_session_id, ''),
			coalesce(prompt_name, ''),
			coalesce(completion_id, ''),
			coalesce(content_id, ''),
			coalesce(generation_stage, ''),
			coalesce(stop_reason, ''),
			occurred_at
		from voice_transcript_turns
		where voice_session_id = $1
		order by sequence_no asc
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list transcript turns: %w", err)
	}
	defer rows.Close()

	turns := make([]TranscriptTurn, 0)
	for rows.Next() {
		var turn TranscriptTurn
		if scanErr := rows.Scan(
			&turn.VoiceSessionID,
			&turn.SequenceNo,
			&turn.Direction,
			&turn.Modality,
			&turn.TranscriptText,
			&turn.BedrockSessionID,
			&turn.PromptName,
			&turn.CompletionID,
			&turn.ContentID,
			&turn.GenerationStage,
			&turn.StopReason,
			&turn.OccurredAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan transcript turn: %w", scanErr)
		}
		turns = append(turns, turn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transcript turns: %w", err)
	}

	return turns, nil
}

func (r *PostgresRepository) ListUsageEvents(ctx context.Context, sessionID string) ([]UsageEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		select
			voice_session_id,
			sequence_no,
			coalesce(bedrock_session_id, ''),
			coalesce(prompt_name, ''),
			coalesce(completion_id, ''),
			input_speech_tokens_delta,
			input_text_tokens_delta,
			output_speech_tokens_delta,
			output_text_tokens_delta,
			total_input_speech_tokens,
			total_input_text_tokens,
			total_output_speech_tokens,
			total_output_text_tokens,
			total_input_tokens,
			total_output_tokens,
			total_tokens,
			payload,
			emitted_at
		from voice_usage_events
		where voice_session_id = $1
		order by sequence_no asc
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list usage events: %w", err)
	}
	defer rows.Close()

	events := make([]UsageEvent, 0)
	for rows.Next() {
		var event UsageEvent
		if scanErr := rows.Scan(
			&event.VoiceSessionID,
			&event.SequenceNo,
			&event.BedrockSessionID,
			&event.PromptName,
			&event.CompletionID,
			&event.InputSpeechTokensDelta,
			&event.InputTextTokensDelta,
			&event.OutputSpeechTokensDelta,
			&event.OutputTextTokensDelta,
			&event.TotalInputSpeechTokens,
			&event.TotalInputTextTokens,
			&event.TotalOutputSpeechTokens,
			&event.TotalOutputTextTokens,
			&event.TotalInputTokens,
			&event.TotalOutputTokens,
			&event.TotalTokens,
			&event.Payload,
			&event.EmittedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan usage event: %w", scanErr)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage events: %w", err)
	}

	return events, nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}
