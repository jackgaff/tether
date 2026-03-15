package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"nova-echoes/api/internal/idgen"
)

type Store interface {
	CreateCaregiver(ctx context.Context, input CreateCaregiverRequest) (Caregiver, error)
	GetCaregiver(ctx context.Context, caregiverID string) (Caregiver, bool, error)
	UpdateCaregiver(ctx context.Context, caregiverID string, input UpdateCaregiverRequest) (Caregiver, error)
	CreatePatient(ctx context.Context, input CreatePatientRequest) (Patient, error)
	ListPatients(ctx context.Context) ([]Patient, error)
	GetPatient(ctx context.Context, patientID string) (Patient, bool, error)
	UpdatePatient(ctx context.Context, patientID string, input UpdatePatientRequest) (Patient, error)
	GetConsentState(ctx context.Context, patientID string) (ConsentState, bool, error)
	PutConsentState(ctx context.Context, patientID string, input UpdateConsentRequest, now time.Time) (ConsentState, error)
	SetPatientPause(ctx context.Context, patientID string, reason string, now time.Time) (Patient, error)
	ClearPatientPause(ctx context.Context, patientID string) (Patient, error)
	ListCallTemplates(ctx context.Context) ([]CallTemplate, error)
	GetCallTemplateByID(ctx context.Context, templateID string) (CallTemplate, bool, error)
	ResolveActiveCallTemplateByType(ctx context.Context, callType string) (CallTemplate, error)
	CreateCallRun(ctx context.Context, input CreateCallRunParams) (CallRun, error)
	GetCallRun(ctx context.Context, callRunID string) (CallRun, bool, error)
	ListRecentCallRuns(ctx context.Context, patientID string, limit int) ([]CallRun, error)
	ListTranscriptTurnsForCallRun(ctx context.Context, callRunID string) ([]CallTranscriptTurn, error)
	GetAnalysisRecord(ctx context.Context, callRunID string) (AnalysisRecord, bool, error)
	GetAnalysisPromptContext(ctx context.Context, callRunID string) (AnalysisPromptContext, error)
	SaveAnalysisResult(ctx context.Context, input SaveAnalysisResultInput) (AnalysisRecord, error)
	GetActiveNextCallPlan(ctx context.Context, patientID string) (NextCallPlan, bool, error)
	UpdateNextCallPlan(ctx context.Context, patientID string, input UpdateNextCallPlanStoreInput) (NextCallPlan, error)
	GetDashboard(ctx context.Context, patientID string) (DashboardSnapshot, error)
}

type PostgresStore struct {
	db *sql.DB
}

type CreateCallRunParams struct {
	PatientID    string
	CaregiverID  string
	CallTemplate CallTemplate
	Channel      string
	TriggerType  string
	RequestedAt  time.Time
}

type SaveAnalysisResultInput struct {
	CallRunID     string
	PatientID     string
	ModelID       string
	SchemaVersion string
	Result        AnalysisPayload
	RiskFlags     []RiskFlagSeed
	CreatedAt     time.Time
}

type RiskFlagSeed struct {
	FlagType      string
	Severity      string
	EvidenceQuote string
	WhyItMatters  string
	Confidence    float64
}

type UpdateNextCallPlanStoreInput struct {
	Action              string
	CallTemplate        *CallTemplate
	SuggestedTimeNote   string
	PlannedFor          *time.Time
	DurationMinutes     *int
	Goal                string
	Reason              string
	AdminUsername       string
	ApprovedCaregiverID string
	Now                 time.Time
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) CreateCaregiver(ctx context.Context, input CreateCaregiverRequest) (Caregiver, error) {
	id, err := idgen.New()
	if err != nil {
		return Caregiver{}, err
	}

	row := s.db.QueryRowContext(ctx, `
		insert into caregivers (
			id, display_name, email, phone_e164, timezone, updated_at
		) values ($1, $2, $3, $4, $5, now())
		returning id, display_name, email, phone_e164, timezone, created_at, updated_at
	`, id, strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.Email), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone))

	caregiver, err := scanCaregiver(row)
	if err != nil {
		return Caregiver{}, fmt.Errorf("create caregiver: %w", err)
	}

	return caregiver, nil
}

func (s *PostgresStore) GetCaregiver(ctx context.Context, caregiverID string) (Caregiver, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select id, display_name, email, phone_e164, timezone, created_at, updated_at
		from caregivers
		where id = $1
	`, strings.TrimSpace(caregiverID))

	caregiver, err := scanCaregiver(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Caregiver{}, false, nil
		}
		return Caregiver{}, false, fmt.Errorf("get caregiver: %w", err)
	}

	return caregiver, true, nil
}

func (s *PostgresStore) UpdateCaregiver(ctx context.Context, caregiverID string, input UpdateCaregiverRequest) (Caregiver, error) {
	row := s.db.QueryRowContext(ctx, `
		update caregivers
		set display_name = $2,
		    email = $3,
		    phone_e164 = $4,
		    timezone = $5,
		    updated_at = now()
		where id = $1
		returning id, display_name, email, phone_e164, timezone, created_at, updated_at
	`, strings.TrimSpace(caregiverID), strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.Email), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone))

	caregiver, err := scanCaregiver(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Caregiver{}, ErrCaregiverNotFound
		}
		return Caregiver{}, fmt.Errorf("update caregiver: %w", err)
	}

	return caregiver, nil
}

func (s *PostgresStore) CreatePatient(ctx context.Context, input CreatePatientRequest) (Patient, error) {
	patientID, err := idgen.New()
	if err != nil {
		return Patient{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Patient{}, fmt.Errorf("begin create patient tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
		insert into patients (
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
		returning
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
	`, patientID, strings.TrimSpace(input.PrimaryCaregiverID), strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.PreferredName), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone), nullableString(input.Notes), marshalStringList(input.RoutineAnchors), marshalStringList(input.FavoriteTopics), marshalStringList(input.CalmingCues), marshalStringList(input.TopicsToAvoid))

	patient, scanErr := scanPatient(row)
	if scanErr != nil {
		if isUniqueViolation(scanErr) {
			err = ErrPatientAlreadyAssigned
			return Patient{}, err
		}
		if isForeignKeyViolation(scanErr) {
			err = ErrCaregiverNotFound
			return Patient{}, err
		}
		err = fmt.Errorf("create patient: %w", scanErr)
		return Patient{}, err
	}

	if _, execErr := tx.ExecContext(ctx, `
		insert into patient_consent_state (
			patient_id,
			outbound_call_status,
			transcript_storage_status,
			updated_at
		) values ($1, 'pending', 'pending', now())
	`, patient.ID); execErr != nil {
		err = fmt.Errorf("create patient consent state: %w", execErr)
		return Patient{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return Patient{}, fmt.Errorf("commit create patient tx: %w", commitErr)
	}

	return patient, nil
}

func (s *PostgresStore) GetPatient(ctx context.Context, patientID string) (Patient, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
		from patients
		where id = $1
	`, strings.TrimSpace(patientID))

	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Patient{}, false, nil
		}
		return Patient{}, false, fmt.Errorf("get patient: %w", err)
	}

	return patient, true, nil
}

func (s *PostgresStore) ListPatients(ctx context.Context) ([]Patient, error) {
	rows, err := s.db.QueryContext(ctx, `
		select
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
		from patients
		order by created_at asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}
	defer rows.Close()

	var patients []Patient
	for rows.Next() {
		p, err := scanPatient(rows)
		if err != nil {
			return nil, fmt.Errorf("list patients scan: %w", err)
		}
		patients = append(patients, p)
	}
	return patients, rows.Err()
}

func (s *PostgresStore) UpdatePatient(ctx context.Context, patientID string, input UpdatePatientRequest) (Patient, error) {
	row := s.db.QueryRowContext(ctx, `
		update patients
		set primary_caregiver_id = $2,
		    display_name = $3,
		    preferred_name = $4,
		    phone_e164 = $5,
		    timezone = $6,
		    notes = $7,
		    routine_anchors = $8,
		    favorite_topics = $9,
		    calming_cues = $10,
		    topics_to_avoid = $11,
		    updated_at = now()
		where id = $1
		returning
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
	`, strings.TrimSpace(patientID), strings.TrimSpace(input.PrimaryCaregiverID), strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.PreferredName), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone), nullableString(input.Notes), marshalStringList(input.RoutineAnchors), marshalStringList(input.FavoriteTopics), marshalStringList(input.CalmingCues), marshalStringList(input.TopicsToAvoid))

	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Patient{}, ErrPatientNotFound
		}
		if isUniqueViolation(err) {
			return Patient{}, ErrPatientAlreadyAssigned
		}
		if isForeignKeyViolation(err) {
			return Patient{}, ErrCaregiverNotFound
		}
		return Patient{}, fmt.Errorf("update patient: %w", err)
	}

	return patient, nil
}

func (s *PostgresStore) GetConsentState(ctx context.Context, patientID string) (ConsentState, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select
			patient_id,
			outbound_call_status,
			transcript_storage_status,
			coalesce(granted_by_caregiver_id, ''),
			granted_at,
			revoked_at,
			coalesce(notes, ''),
			created_at,
			updated_at
		from patient_consent_state
		where patient_id = $1
	`, strings.TrimSpace(patientID))

	state, err := scanConsentState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConsentState{}, false, nil
		}
		return ConsentState{}, false, fmt.Errorf("get consent state: %w", err)
	}

	return state, true, nil
}

func (s *PostgresStore) PutConsentState(ctx context.Context, patientID string, input UpdateConsentRequest, now time.Time) (ConsentState, error) {
	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return ConsentState{}, err
	}
	if !ok {
		return ConsentState{}, ErrPatientNotFound
	}

	grantedBy := ""
	var grantedAt any
	var revokedAt any
	if input.OutboundCallStatus == ConsentStatusGranted || input.TranscriptStorageStatus == ConsentStatusGranted {
		grantedBy = patient.PrimaryCaregiverID
		grantedAt = now
	}
	if input.OutboundCallStatus == ConsentStatusRevoked || input.TranscriptStorageStatus == ConsentStatusRevoked {
		revokedAt = now
	}

	row := s.db.QueryRowContext(ctx, `
		update patient_consent_state
		set outbound_call_status = $2,
		    transcript_storage_status = $3,
		    granted_by_caregiver_id = nullif($4, ''),
		    granted_at = coalesce($5, granted_at),
		    revoked_at = coalesce($6, revoked_at),
		    notes = $7,
		    updated_at = $8
		where patient_id = $1
		returning
			patient_id,
			outbound_call_status,
			transcript_storage_status,
			coalesce(granted_by_caregiver_id, ''),
			granted_at,
			revoked_at,
			coalesce(notes, ''),
			created_at,
			updated_at
	`, strings.TrimSpace(patientID), input.OutboundCallStatus, input.TranscriptStorageStatus, grantedBy, grantedAt, revokedAt, nullableString(input.Notes), now)

	state, scanErr := scanConsentState(row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return ConsentState{}, ErrConsentStateNotFound
		}
		return ConsentState{}, fmt.Errorf("update consent state: %w", scanErr)
	}

	return state, nil
}

func (s *PostgresStore) SetPatientPause(ctx context.Context, patientID string, reason string, now time.Time) (Patient, error) {
	row := s.db.QueryRowContext(ctx, `
		update patients
		set calling_state = 'paused',
		    pause_reason = nullif($2, ''),
		    paused_at = $3,
		    updated_at = $3
		where id = $1
		returning
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
	`, strings.TrimSpace(patientID), strings.TrimSpace(reason), now)

	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Patient{}, ErrPatientNotFound
		}
		return Patient{}, fmt.Errorf("pause patient: %w", err)
	}

	return patient, nil
}

func (s *PostgresStore) ClearPatientPause(ctx context.Context, patientID string) (Patient, error) {
	row := s.db.QueryRowContext(ctx, `
		update patients
		set calling_state = 'active',
		    pause_reason = null,
		    paused_at = null,
		    updated_at = now()
		where id = $1
		returning
			id,
			primary_caregiver_id,
			display_name,
			preferred_name,
			phone_e164,
			timezone,
			notes,
			calling_state,
			pause_reason,
			paused_at,
			routine_anchors,
			favorite_topics,
			calming_cues,
			topics_to_avoid,
			created_at,
			updated_at
	`, strings.TrimSpace(patientID))

	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Patient{}, ErrPatientNotFound
		}
		return Patient{}, fmt.Errorf("clear patient pause: %w", err)
	}

	return patient, nil
}

func (s *PostgresStore) ListCallTemplates(ctx context.Context) ([]CallTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		select
			id,
			slug,
			display_name,
			call_type,
			description,
			duration_minutes,
			prompt_version,
			system_prompt_template,
			checklist_json,
			is_active,
			created_at,
			updated_at
		from call_templates
		where is_active = true
		order by display_name asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list call templates: %w", err)
	}
	defer rows.Close()

	var templates []CallTemplate
	for rows.Next() {
		template, scanErr := scanCallTemplate(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan call template: %w", scanErr)
		}
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate call templates: %w", err)
	}

	return templates, nil
}

func (s *PostgresStore) GetCallTemplateByID(ctx context.Context, templateID string) (CallTemplate, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select
			id,
			slug,
			display_name,
			call_type,
			description,
			duration_minutes,
			prompt_version,
			system_prompt_template,
			checklist_json,
			is_active,
			created_at,
			updated_at
		from call_templates
		where id = $1
	`, strings.TrimSpace(templateID))

	template, err := scanCallTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CallTemplate{}, false, nil
		}
		return CallTemplate{}, false, fmt.Errorf("get call template: %w", err)
	}

	return template, true, nil
}

func (s *PostgresStore) ResolveActiveCallTemplateByType(ctx context.Context, callType string) (CallTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		select
			id,
			slug,
			display_name,
			call_type,
			description,
			duration_minutes,
			prompt_version,
			system_prompt_template,
			checklist_json,
			is_active,
			created_at,
			updated_at
		from call_templates
		where call_type = $1 and is_active = true
	`, strings.TrimSpace(callType))
	if err != nil {
		return CallTemplate{}, fmt.Errorf("resolve call template by type: %w", err)
	}
	defer rows.Close()

	var matches []CallTemplate
	for rows.Next() {
		template, scanErr := scanCallTemplate(rows)
		if scanErr != nil {
			return CallTemplate{}, fmt.Errorf("scan call template by type: %w", scanErr)
		}
		matches = append(matches, template)
	}
	if err := rows.Err(); err != nil {
		return CallTemplate{}, fmt.Errorf("iterate call template by type: %w", err)
	}

	if len(matches) != 1 {
		return CallTemplate{}, ErrCallTemplateConflict
	}

	return matches[0], nil
}

func (s *PostgresStore) CreateCallRun(ctx context.Context, input CreateCallRunParams) (CallRun, error) {
	id, err := idgen.New()
	if err != nil {
		return CallRun{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CallRun{}, fmt.Errorf("begin create call run tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
		insert into call_runs (
			id,
			patient_id,
			caregiver_id,
			call_template_id,
			channel,
			trigger_type,
			status,
			requested_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, 'requested', $7, $7)
		returning
			id,
			patient_id,
			caregiver_id,
			call_template_id,
			$8,
			$9,
			$10,
			channel,
			trigger_type,
			status,
			coalesce(source_voice_session_id, ''),
			requested_at,
			started_at,
			ended_at,
			coalesce(stop_reason, ''),
			created_at,
			updated_at
	`, id, input.PatientID, input.CaregiverID, input.CallTemplate.ID, input.Channel, input.TriggerType, input.RequestedAt, input.CallTemplate.Slug, input.CallTemplate.DisplayName, input.CallTemplate.CallType)

	callRun, scanErr := scanCallRun(row)
	if scanErr != nil {
		err = fmt.Errorf("create call run: %w", scanErr)
		return CallRun{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return CallRun{}, fmt.Errorf("commit create call run tx: %w", commitErr)
	}

	return callRun, nil
}

func (s *PostgresStore) MarkCallRunFailed(ctx context.Context, callRunID, stopReason string, endedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		update call_runs
		set status = 'failed',
		    ended_at = $2,
		    stop_reason = nullif($3, ''),
		    updated_at = $2
		where id = $1
		  and status = 'requested'
	`, strings.TrimSpace(callRunID), endedAt, stopReason)
	if err != nil {
		return fmt.Errorf("mark call run failed: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetCallRun(ctx context.Context, callRunID string) (CallRun, bool, error) {
	row := s.db.QueryRowContext(ctx, callRunSelectByID, strings.TrimSpace(callRunID))
	callRun, err := scanCallRun(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CallRun{}, false, nil
		}
		return CallRun{}, false, fmt.Errorf("get call run: %w", err)
	}
	return callRun, true, nil
}

func (s *PostgresStore) ListRecentCallRuns(ctx context.Context, patientID string, limit int) ([]CallRun, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, callRunListByPatientQuery, strings.TrimSpace(patientID), limit)
	if err != nil {
		return nil, fmt.Errorf("list recent call runs: %w", err)
	}
	defer rows.Close()

	var callRuns []CallRun
	for rows.Next() {
		callRun, scanErr := scanCallRun(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan call run: %w", scanErr)
		}
		callRuns = append(callRuns, callRun)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate call runs: %w", err)
	}

	return callRuns, nil
}

func (s *PostgresStore) ListTranscriptTurnsForCallRun(ctx context.Context, callRunID string) ([]CallTranscriptTurn, error) {
	callRun, ok, err := s.GetCallRun(ctx, callRunID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCallRunNotFound
	}
	if strings.TrimSpace(callRun.SourceVoiceSessionID) == "" {
		return []CallTranscriptTurn{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		select sequence_no, direction, modality, transcript_text, occurred_at, coalesce(stop_reason, '')
		from voice_transcript_turns
		where voice_session_id = $1
		order by sequence_no asc
	`, callRun.SourceVoiceSessionID)
	if err != nil {
		return nil, fmt.Errorf("list transcript turns: %w", err)
	}
	defer rows.Close()

	var turns []CallTranscriptTurn
	for rows.Next() {
		var (
			turn       CallTranscriptTurn
			stopReason string
		)
		if err := rows.Scan(&turn.SequenceNo, &turn.Direction, &turn.Modality, &turn.Text, &turn.OccurredAt, &stopReason); err != nil {
			return nil, fmt.Errorf("scan transcript turn: %w", err)
		}
		turn.StopReason = strings.TrimSpace(stopReason)
		turns = append(turns, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transcript turns: %w", err)
	}

	return turns, nil
}

func (s *PostgresStore) GetAnalysisRecord(ctx context.Context, callRunID string) (AnalysisRecord, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select
			id,
			call_run_id,
			model_id,
			schema_version,
			raw_result_json,
			created_at,
			updated_at
		from analysis_results
		where call_run_id = $1
	`, strings.TrimSpace(callRunID))

	record, err := scanAnalysisRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AnalysisRecord{}, false, nil
		}
		return AnalysisRecord{}, false, fmt.Errorf("get analysis record: %w", err)
	}

	riskFlags, err := s.listRiskFlags(ctx, record.ID)
	if err != nil {
		return AnalysisRecord{}, false, err
	}
	record.RiskFlags = riskFlags

	return record, true, nil
}

func (s *PostgresStore) GetAnalysisPromptContext(ctx context.Context, callRunID string) (AnalysisPromptContext, error) {
	callRun, ok, err := s.GetCallRun(ctx, callRunID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}
	if !ok {
		return AnalysisPromptContext{}, ErrCallRunNotFound
	}
	if callRun.Status != CallRunStatusCompleted {
		return AnalysisPromptContext{}, ErrCallRunNotCompleted
	}
	if strings.TrimSpace(callRun.SourceVoiceSessionID) == "" {
		return AnalysisPromptContext{}, ErrCallRunVoiceSessionMissing
	}

	patient, ok, err := s.GetPatient(ctx, callRun.PatientID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}
	if !ok {
		return AnalysisPromptContext{}, ErrPatientNotFound
	}

	caregiver, ok, err := s.GetCaregiver(ctx, patient.PrimaryCaregiverID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}
	if !ok {
		return AnalysisPromptContext{}, ErrCaregiverNotFound
	}

	callTemplate, ok, err := s.GetCallTemplateByID(ctx, callRun.CallTemplateID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}
	if !ok {
		return AnalysisPromptContext{}, ErrCallTemplateNotFound
	}

	turns, err := s.ListTranscriptTurnsForCallRun(ctx, callRunID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}

	recentAnalyses, err := s.listRecentAnalysisPayloads(ctx, patient.ID, callRunID, 5)
	if err != nil {
		return AnalysisPromptContext{}, err
	}

	return AnalysisPromptContext{
		CallRun:         callRun,
		Patient:         patient,
		Caregiver:       caregiver,
		CallTemplate:    callTemplate,
		TranscriptTurns: turns,
		RecentAnalyses:  recentAnalyses,
	}, nil
}

func (s *PostgresStore) SaveAnalysisResult(ctx context.Context, input SaveAnalysisResultInput) (AnalysisRecord, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AnalysisRecord{}, fmt.Errorf("begin save analysis tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	analysisID, err := idgen.New()
	if err != nil {
		return AnalysisRecord{}, err
	}

	payload, err := json.Marshal(input.Result)
	if err != nil {
		return AnalysisRecord{}, fmt.Errorf("marshal analysis payload: %w", err)
	}

	templateRow := tx.QueryRowContext(ctx, `
		select
			id,
			slug,
			display_name,
			call_type,
			description,
			duration_minutes,
			prompt_version,
			system_prompt_template,
			checklist_json,
			is_active,
			created_at,
			updated_at
		from call_templates
		where call_type = $1 and is_active = true
	`, input.Result.RecommendedNextCall.Type)

	nextTemplate, scanErr := scanCallTemplate(templateRow)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return AnalysisRecord{}, ErrCallTemplateConflict
		}
		return AnalysisRecord{}, fmt.Errorf("resolve recommended next call template: %w", scanErr)
	}

	row := tx.QueryRowContext(ctx, `
		insert into analysis_results (
			id,
			call_run_id,
			patient_id,
			model_id,
			schema_version,
			raw_result_json,
			dashboard_summary,
			caregiver_summary,
			orientation,
			mood,
			engagement,
			confidence,
			escalation_level,
			recommended_call_type,
			recommended_time_note,
			recommended_duration_minutes,
			recommended_goal,
			updated_at
		) values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		on conflict (call_run_id) do update
		set model_id = excluded.model_id,
		    schema_version = excluded.schema_version,
		    raw_result_json = excluded.raw_result_json,
		    dashboard_summary = excluded.dashboard_summary,
		    caregiver_summary = excluded.caregiver_summary,
		    orientation = excluded.orientation,
		    mood = excluded.mood,
		    engagement = excluded.engagement,
		    confidence = excluded.confidence,
		    escalation_level = excluded.escalation_level,
		    recommended_call_type = excluded.recommended_call_type,
		    recommended_time_note = excluded.recommended_time_note,
		    recommended_duration_minutes = excluded.recommended_duration_minutes,
		    recommended_goal = excluded.recommended_goal,
		    updated_at = excluded.updated_at
		returning id
	`, analysisID, input.CallRunID, input.PatientID, input.ModelID, input.SchemaVersion, payload, input.Result.DashboardSummary, input.Result.CaregiverSummary, input.Result.PatientState.Orientation, input.Result.PatientState.Mood, input.Result.PatientState.Engagement, input.Result.PatientState.Confidence, input.Result.EscalationLevel, input.Result.RecommendedNextCall.Type, nullableString(input.Result.RecommendedNextCall.Timing), input.Result.RecommendedNextCall.DurationMinutes, input.Result.RecommendedNextCall.Goal, input.CreatedAt)

	if scanErr := row.Scan(&analysisID); scanErr != nil {
		return AnalysisRecord{}, fmt.Errorf("upsert analysis result: %w", scanErr)
	}

	if _, execErr := tx.ExecContext(ctx, `delete from risk_flags where analysis_result_id = $1`, analysisID); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("delete existing risk flags: %w", execErr)
	}

	for _, flag := range input.RiskFlags {
		riskID, idErr := idgen.New()
		if idErr != nil {
			return AnalysisRecord{}, idErr
		}
		if _, execErr := tx.ExecContext(ctx, `
			insert into risk_flags (
				id,
				analysis_result_id,
				flag_type,
				severity,
				evidence_quote,
				why_it_matters,
				confidence
			) values ($1, $2, $3, $4, $5, $6, $7)
		`, riskID, analysisID, flag.FlagType, flag.Severity, nullableString(flag.EvidenceQuote), nullableString(flag.WhyItMatters), flag.Confidence); execErr != nil {
			return AnalysisRecord{}, fmt.Errorf("insert risk flag: %w", execErr)
		}
	}

	if _, execErr := tx.ExecContext(ctx, `
		update next_call_plans
		set approval_status = 'superseded',
		    updated_at = $2
		where patient_id = $1
		  and approval_status in ('pending_approval', 'approved')
	`, input.PatientID, input.CreatedAt); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("supersede active next call plans: %w", execErr)
	}

	nextCallPlanID, idErr := idgen.New()
	if idErr != nil {
		return AnalysisRecord{}, idErr
	}
	if _, execErr := tx.ExecContext(ctx, `
		insert into next_call_plans (
			id,
			patient_id,
			source_analysis_result_id,
			call_template_id,
			call_type,
			suggested_time_note,
			duration_minutes,
			goal,
			approval_status,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, 'pending_approval', $9)
	`, nextCallPlanID, input.PatientID, analysisID, nextTemplate.ID, input.Result.RecommendedNextCall.Type, nullableString(input.Result.RecommendedNextCall.Timing), input.Result.RecommendedNextCall.DurationMinutes, input.Result.RecommendedNextCall.Goal, input.CreatedAt); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("insert next call plan: %w", execErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return AnalysisRecord{}, fmt.Errorf("commit save analysis tx: %w", commitErr)
	}

	record, ok, err := s.GetAnalysisRecord(ctx, input.CallRunID)
	if err != nil {
		return AnalysisRecord{}, err
	}
	if !ok {
		return AnalysisRecord{}, ErrAnalysisNotFound
	}

	return record, nil
}

func (s *PostgresStore) GetActiveNextCallPlan(ctx context.Context, patientID string) (NextCallPlan, bool, error) {
	row := s.db.QueryRowContext(ctx, nextCallPlanSelectBase+`
		where ncp.patient_id = $1
		  and ncp.approval_status in ('pending_approval', 'approved')
		order by ncp.updated_at desc
		limit 1
	`, strings.TrimSpace(patientID))

	plan, err := scanNextCallPlan(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NextCallPlan{}, false, nil
		}
		return NextCallPlan{}, false, fmt.Errorf("get active next call plan: %w", err)
	}

	return plan, true, nil
}

func (s *PostgresStore) UpdateNextCallPlan(ctx context.Context, patientID string, input UpdateNextCallPlanStoreInput) (NextCallPlan, error) {
	current, ok, err := s.GetActiveNextCallPlan(ctx, patientID)
	if err != nil {
		return NextCallPlan{}, err
	}
	if !ok {
		return NextCallPlan{}, ErrNextCallPlanNotFound
	}

	callTemplateID := current.CallTemplateID
	callType := current.CallType
	callTemplateSlug := current.CallTemplateSlug
	callTemplateName := current.CallTemplateName
	if input.CallTemplate != nil {
		callTemplateID = input.CallTemplate.ID
		callType = input.CallTemplate.CallType
		callTemplateSlug = input.CallTemplate.Slug
		callTemplateName = input.CallTemplate.DisplayName
	}

	suggestedTimeNote := chooseString(input.SuggestedTimeNote, current.SuggestedTimeNote)
	durationMinutes := current.DurationMinutes
	if input.DurationMinutes != nil {
		durationMinutes = *input.DurationMinutes
	}
	goal := chooseString(input.Goal, current.Goal)
	plannedFor := current.PlannedFor
	if input.PlannedFor != nil {
		plannedFor = input.PlannedFor
	}

	status := current.ApprovalStatus
	approvedByCaregiverID := current.ApprovedByCaregiverID
	approvedByAdminUsername := current.ApprovedByAdminUsername
	approvedAt := current.ApprovedAt
	rejectionReason := current.RejectionReason
	rejectedAt := current.RejectedAt

	switch input.Action {
	case NextCallActionApprove:
		status = NextCallStatusApproved
		approvedByCaregiverID = input.ApprovedCaregiverID
		approvedByAdminUsername = input.AdminUsername
		approvedAt = &input.Now
		rejectionReason = ""
		rejectedAt = nil
	case NextCallActionEdit:
	case NextCallActionReject:
		status = NextCallStatusRejected
		rejectionReason = strings.TrimSpace(input.Reason)
		rejectedAt = &input.Now
	case NextCallActionCancel:
		status = NextCallStatusCancelled
	default:
		return NextCallPlan{}, fmt.Errorf("unsupported next-call plan action %q", input.Action)
	}

	row := s.db.QueryRowContext(ctx, `
		update next_call_plans
		set call_template_id = $2,
		    call_type = $3,
		    suggested_time_note = $4,
		    planned_for = $5,
		    duration_minutes = $6,
		    goal = $7,
		    approval_status = $8,
		    approved_by_caregiver_id = nullif($9, ''),
		    approved_by_admin_username = nullif($10, ''),
		    approved_at = $11,
		    rejection_reason = nullif($12, ''),
		    rejected_at = $13,
		    updated_at = $14
		where id = $1
		returning
			id,
			patient_id,
			source_analysis_result_id,
			call_template_id,
			$15,
			$16,
			call_type,
			coalesce(suggested_time_note, ''),
			planned_for,
			duration_minutes,
			goal,
			approval_status,
			coalesce(approved_by_caregiver_id, ''),
			coalesce(approved_by_admin_username, ''),
			approved_at,
			coalesce(rejection_reason, ''),
			rejected_at,
			coalesce(executed_call_run_id, ''),
			created_at,
			updated_at
	`, current.ID, callTemplateID, callType, nullableString(suggestedTimeNote), plannedFor, durationMinutes, goal, status, approvedByCaregiverID, approvedByAdminUsername, approvedAt, rejectionReason, rejectedAt, input.Now, callTemplateSlug, callTemplateName)

	plan, err := scanNextCallPlan(row)
	if err != nil {
		return NextCallPlan{}, fmt.Errorf("update next call plan: %w", err)
	}

	return plan, nil
}

func (s *PostgresStore) GetDashboard(ctx context.Context, patientID string) (DashboardSnapshot, error) {
	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	if !ok {
		return DashboardSnapshot{}, ErrPatientNotFound
	}

	caregiver, ok, err := s.GetCaregiver(ctx, patient.PrimaryCaregiverID)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	if !ok {
		return DashboardSnapshot{}, ErrCaregiverNotFound
	}

	consent, ok, err := s.GetConsentState(ctx, patient.ID)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	if !ok {
		return DashboardSnapshot{}, ErrConsentStateNotFound
	}

	recentCalls, err := s.ListRecentCallRuns(ctx, patient.ID, 10)
	if err != nil {
		return DashboardSnapshot{}, err
	}

	var latestCall *CallRun
	var latestAnalysis *AnalysisRecord
	if len(recentCalls) > 0 {
		latestCall = &recentCalls[0]
		record, ok, analysisErr := s.GetAnalysisRecord(ctx, recentCalls[0].ID)
		if analysisErr != nil {
			return DashboardSnapshot{}, analysisErr
		}
		if ok {
			latestAnalysis = &record
		}
	}

	activePlan, ok, err := s.GetActiveNextCallPlan(ctx, patient.ID)
	if err != nil {
		return DashboardSnapshot{}, err
	}

	riskFlags := []RiskFlag{}
	if latestAnalysis != nil {
		riskFlags = append(riskFlags, latestAnalysis.RiskFlags...)
	}

	dashboard := DashboardSnapshot{
		Patient:        patient,
		Caregiver:      caregiver,
		Consent:        consent,
		LatestCall:     latestCall,
		RecentCalls:    recentCalls,
		LatestAnalysis: latestAnalysis,
		RiskFlags:      riskFlags,
	}
	if ok {
		dashboard.ActiveNextCallPlan = &activePlan
	}

	return dashboard, nil
}

func (s *PostgresStore) listRecentAnalysisPayloads(ctx context.Context, patientID string, excludeCallRunID string, limit int) ([]AnalysisPayload, error) {
	rows, err := s.db.QueryContext(ctx, `
		select raw_result_json
		from analysis_results
		where patient_id = $1
		  and call_run_id <> $2
		order by created_at desc
		limit $3
	`, patientID, excludeCallRunID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent analyses: %w", err)
	}
	defer rows.Close()

	var payloads []AnalysisPayload
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan recent analysis payload: %w", err)
		}
		var payload AnalysisPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode recent analysis payload: %w", err)
		}
		payloads = append(payloads, payload)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent analyses: %w", err)
	}

	return payloads, nil
}

func (s *PostgresStore) listRiskFlags(ctx context.Context, analysisResultID string) ([]RiskFlag, error) {
	rows, err := s.db.QueryContext(ctx, `
		select id, analysis_result_id, flag_type, severity, coalesce(evidence_quote, ''), coalesce(why_it_matters, ''), confidence, created_at
		from risk_flags
		where analysis_result_id = $1
		order by created_at asc, id asc
	`, analysisResultID)
	if err != nil {
		return nil, fmt.Errorf("list risk flags: %w", err)
	}
	defer rows.Close()

	var flags []RiskFlag
	for rows.Next() {
		var flag RiskFlag
		if err := rows.Scan(&flag.ID, &flag.AnalysisResultID, &flag.FlagType, &flag.Severity, &flag.EvidenceQuote, &flag.WhyItMatters, &flag.Confidence, &flag.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan risk flag: %w", err)
		}
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate risk flags: %w", err)
	}

	return flags, nil
}

const callRunSelectColumns = `
	select
		cr.id,
		cr.patient_id,
		cr.caregiver_id,
		cr.call_template_id,
		ct.slug,
		ct.display_name,
		ct.call_type,
		cr.channel,
		cr.trigger_type,
		cr.status,
		coalesce(cr.source_voice_session_id, ''),
		cr.requested_at,
		cr.started_at,
		cr.ended_at,
		coalesce(cr.stop_reason, ''),
		cr.created_at,
		cr.updated_at
	from call_runs cr
	join call_templates ct on ct.id = cr.call_template_id
`

const callRunSelectByID = callRunSelectColumns + `
	where cr.id = $1
`

const callRunListByPatientQuery = callRunSelectColumns + `
	where cr.patient_id = $1
	order by cr.requested_at desc
	limit $2
`

const nextCallPlanSelectBase = `
	select
		ncp.id,
		ncp.patient_id,
		ncp.source_analysis_result_id,
		ncp.call_template_id,
		ct.slug,
		ct.display_name,
		ncp.call_type,
		coalesce(ncp.suggested_time_note, ''),
		ncp.planned_for,
		ncp.duration_minutes,
		ncp.goal,
		ncp.approval_status,
		coalesce(ncp.approved_by_caregiver_id, ''),
		coalesce(ncp.approved_by_admin_username, ''),
		ncp.approved_at,
		coalesce(ncp.rejection_reason, ''),
		ncp.rejected_at,
		coalesce(ncp.executed_call_run_id, ''),
		ncp.created_at,
		ncp.updated_at
	from next_call_plans ncp
	join call_templates ct on ct.id = ncp.call_template_id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanCaregiver(row scanner) (Caregiver, error) {
	var caregiver Caregiver
	var phone sql.NullString
	if err := row.Scan(&caregiver.ID, &caregiver.DisplayName, &caregiver.Email, &phone, &caregiver.Timezone, &caregiver.CreatedAt, &caregiver.UpdatedAt); err != nil {
		return Caregiver{}, err
	}
	if phone.Valid {
		caregiver.PhoneE164 = phone.String
	}
	return caregiver, nil
}

func scanPatient(row scanner) (Patient, error) {
	var (
		patient        Patient
		phone          sql.NullString
		notes          sql.NullString
		pauseReason    sql.NullString
		pausedAt       sql.NullTime
		routineAnchors []byte
		favoriteTopics []byte
		calmingCues    []byte
		topicsToAvoid  []byte
	)
	if err := row.Scan(&patient.ID, &patient.PrimaryCaregiverID, &patient.DisplayName, &patient.PreferredName, &phone, &patient.Timezone, &notes, &patient.CallingState, &pauseReason, &pausedAt, &routineAnchors, &favoriteTopics, &calmingCues, &topicsToAvoid, &patient.CreatedAt, &patient.UpdatedAt); err != nil {
		return Patient{}, err
	}
	if phone.Valid {
		patient.PhoneE164 = phone.String
	}
	if notes.Valid {
		patient.Notes = notes.String
	}
	if pauseReason.Valid {
		patient.PauseReason = pauseReason.String
	}
	if pausedAt.Valid {
		patient.PausedAt = &pausedAt.Time
	}
	patient.RoutineAnchors = parseStringList(routineAnchors)
	patient.FavoriteTopics = parseStringList(favoriteTopics)
	patient.CalmingCues = parseStringList(calmingCues)
	patient.TopicsToAvoid = parseStringList(topicsToAvoid)
	return patient, nil
}

func scanConsentState(row scanner) (ConsentState, error) {
	var (
		state     ConsentState
		grantedAt sql.NullTime
		revokedAt sql.NullTime
	)
	if err := row.Scan(&state.PatientID, &state.OutboundCallStatus, &state.TranscriptStorageStatus, &state.GrantedByCaregiverID, &grantedAt, &revokedAt, &state.Notes, &state.CreatedAt, &state.UpdatedAt); err != nil {
		return ConsentState{}, err
	}
	if grantedAt.Valid {
		state.GrantedAt = &grantedAt.Time
	}
	if revokedAt.Valid {
		state.RevokedAt = &revokedAt.Time
	}
	return state, nil
}

func scanCallTemplate(row scanner) (CallTemplate, error) {
	var template CallTemplate
	var checklist []byte
	if err := row.Scan(&template.ID, &template.Slug, &template.DisplayName, &template.CallType, &template.Description, &template.DurationMinutes, &template.PromptVersion, &template.SystemPromptTemplate, &checklist, &template.IsActive, &template.CreatedAt, &template.UpdatedAt); err != nil {
		return CallTemplate{}, err
	}
	template.Checklist = append(template.Checklist[:0], checklist...)
	return template, nil
}

func scanCallRun(row scanner) (CallRun, error) {
	var (
		callRun   CallRun
		startedAt sql.NullTime
		endedAt   sql.NullTime
	)
	if err := row.Scan(&callRun.ID, &callRun.PatientID, &callRun.CaregiverID, &callRun.CallTemplateID, &callRun.CallTemplateSlug, &callRun.CallTemplateName, &callRun.CallType, &callRun.Channel, &callRun.TriggerType, &callRun.Status, &callRun.SourceVoiceSessionID, &callRun.RequestedAt, &startedAt, &endedAt, &callRun.StopReason, &callRun.CreatedAt, &callRun.UpdatedAt); err != nil {
		return CallRun{}, err
	}
	if startedAt.Valid {
		callRun.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		callRun.EndedAt = &endedAt.Time
	}
	return callRun, nil
}

func scanAnalysisRecord(row scanner) (AnalysisRecord, error) {
	var (
		record AnalysisRecord
		raw    json.RawMessage
	)
	if err := row.Scan(&record.ID, &record.CallRunID, &record.ModelID, &record.SchemaVersion, &raw, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return AnalysisRecord{}, err
	}
	if err := json.Unmarshal(raw, &record.Result); err != nil {
		return AnalysisRecord{}, fmt.Errorf("decode analysis result json: %w", err)
	}
	return record, nil
}

func scanNextCallPlan(row scanner) (NextCallPlan, error) {
	var (
		plan       NextCallPlan
		plannedFor sql.NullTime
		approvedAt sql.NullTime
		rejectedAt sql.NullTime
	)
	if err := row.Scan(&plan.ID, &plan.PatientID, &plan.SourceAnalysisResultID, &plan.CallTemplateID, &plan.CallTemplateSlug, &plan.CallTemplateName, &plan.CallType, &plan.SuggestedTimeNote, &plannedFor, &plan.DurationMinutes, &plan.Goal, &plan.ApprovalStatus, &plan.ApprovedByCaregiverID, &plan.ApprovedByAdminUsername, &approvedAt, &plan.RejectionReason, &rejectedAt, &plan.ExecutedCallRunID, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return NextCallPlan{}, err
	}
	if plannedFor.Valid {
		plan.PlannedFor = &plannedFor.Time
	}
	if approvedAt.Valid {
		plan.ApprovedAt = &approvedAt.Time
	}
	if rejectedAt.Valid {
		plan.RejectedAt = &rejectedAt.Time
	}
	return plan, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func marshalStringList(values []string) []byte {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	payload, _ := json.Marshal(normalized)
	return payload
}

func parseStringList(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}
	return values
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func chooseString(candidate, fallback string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
