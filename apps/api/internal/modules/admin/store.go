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
	ListCaregivers(ctx context.Context) ([]Caregiver, error)
	GetCaregiver(ctx context.Context, caregiverID string) (Caregiver, bool, error)
	UpdateCaregiver(ctx context.Context, caregiverID string, input UpdateCaregiverRequest) (Caregiver, error)
	CreatePatient(ctx context.Context, input CreatePatientRequest) (Patient, error)
	ListPatients(ctx context.Context) ([]Patient, error)
	GetPatient(ctx context.Context, patientID string) (Patient, bool, error)
	UpdatePatient(ctx context.Context, patientID string, input UpdatePatientRequest) (Patient, error)
	GetCallPromptContext(ctx context.Context, patientID string) (CallPromptContext, error)
	ListPatientPeople(ctx context.Context, patientID string) ([]PatientPerson, error)
	UpdatePatientPerson(ctx context.Context, patientID, personID string, input UpdatePatientPersonRequest) (PatientPerson, error)
	ListMemoryBankEntries(ctx context.Context, patientID string) ([]MemoryBankEntry, error)
	ListPatientReminders(ctx context.Context, patientID string) ([]Reminder, error)
	GetScreeningSchedule(ctx context.Context, patientID string) (ScreeningSchedule, bool, error)
	PutScreeningSchedule(ctx context.Context, patientID string, input ScreeningScheduleInput, now time.Time) (ScreeningSchedule, error)
	ListDueScreeningSchedules(ctx context.Context, now time.Time, limit int) ([]ScreeningSchedule, error)
	CreateScheduledScreeningCallRun(ctx context.Context, schedule ScreeningSchedule, now time.Time) (CallRun, bool, error)
	GetConsentState(ctx context.Context, patientID string) (ConsentState, bool, error)
	PutConsentState(ctx context.Context, patientID string, input UpdateConsentRequest, now time.Time) (ConsentState, error)
	SetPatientPause(ctx context.Context, patientID string, reason string, now time.Time) (Patient, error)
	ClearPatientPause(ctx context.Context, patientID string) (Patient, error)
	ListCallTemplates(ctx context.Context) ([]CallTemplate, error)
	GetCallTemplateByID(ctx context.Context, templateID string) (CallTemplate, bool, error)
	ResolveActiveCallTemplateByType(ctx context.Context, callType string) (CallTemplate, error)
	CreateCallRun(ctx context.Context, input CreateCallRunParams) (CallRun, error)
	MarkCallRunFailed(ctx context.Context, callRunID, stopReason string, endedAt time.Time) error
	GetCallRun(ctx context.Context, callRunID string) (CallRun, bool, error)
	ListRecentCallRuns(ctx context.Context, patientID string, limit int) ([]CallRun, error)
	ListTranscriptTurnsForCallRun(ctx context.Context, callRunID string) ([]CallTranscriptTurn, error)
	GetAnalysisJob(ctx context.Context, callRunID string) (AnalysisJob, bool, error)
	UpsertAnalysisJob(ctx context.Context, input UpsertAnalysisJobParams) (AnalysisJob, error)
	ClaimNextAnalysisJob(ctx context.Context, now time.Time) (AnalysisJob, bool, error)
	MarkAnalysisJobFailed(ctx context.Context, jobID, lastError string, now time.Time) error
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
	PatientID           string
	CaregiverID         string
	CallTemplate        CallTemplate
	CallType            string
	Channel             string
	TriggerType         string
	Status              string
	RequestedAt         time.Time
	ScheduleWindowStart *time.Time
	ScheduleWindowEnd   *time.Time
}

type UpsertAnalysisJobParams struct {
	CallRunID             string
	Force                 bool
	AnalysisPromptVersion string
	AnalysisSchemaVersion string
	ModelProvider         string
	ModelName             string
	Now                   time.Time
}

type SaveAnalysisResultInput struct {
	CallRunID             string
	PatientID             string
	PatientTimezone       string
	CallTemplateID        string
	CallType              string
	CallPromptVersion     string
	AnalysisPromptVersion string
	SchemaVersion         string
	ModelProvider         string
	ModelName             string
	Result                AnalysisPayload
	GeneratedAt           time.Time
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

func (s *PostgresStore) ListCaregivers(ctx context.Context) ([]Caregiver, error) {
	rows, err := s.db.QueryContext(ctx, `
		select id, display_name, email, phone_e164, timezone, created_at, updated_at
		from caregivers
		order by created_at asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list caregivers: %w", err)
	}
	defer rows.Close()

	var caregivers []Caregiver
	for rows.Next() {
		c, scanErr := scanCaregiver(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan caregiver: %w", scanErr)
		}
		caregivers = append(caregivers, c)
	}
	if caregivers == nil {
		caregivers = []Caregiver{}
	}
	return caregivers, rows.Err()
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

	if _, execErr := tx.ExecContext(ctx, `
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
	`, patientID, strings.TrimSpace(input.PrimaryCaregiverID), strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.PreferredName), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone), nullableString(input.Notes), marshalStringList(input.RoutineAnchors), marshalStringList(input.FavoriteTopics), marshalStringList(input.CalmingCues), marshalStringList(input.TopicsToAvoid)); execErr != nil {
		switch {
		case isForeignKeyViolation(execErr):
			err = ErrCaregiverNotFound
		default:
			err = fmt.Errorf("create patient: %w", execErr)
		}
		return Patient{}, err
	}

	if _, execErr := tx.ExecContext(ctx, `
		insert into patient_consent_state (
			patient_id,
			outbound_call_status,
			transcript_storage_status,
			updated_at
		) values ($1, 'pending', 'pending', now())
	`, patientID); execErr != nil {
		err = fmt.Errorf("create patient consent state: %w", execErr)
		return Patient{}, err
	}

	if err = upsertMemoryProfileTx(ctx, tx, patientID, input.MemoryProfile, input.ConversationGuidance); err != nil {
		return Patient{}, err
	}

	if _, execErr := tx.ExecContext(ctx, `
		insert into screening_schedules (
			patient_id,
			enabled,
			cadence,
			timezone,
			preferred_weekday,
			preferred_local_time,
			next_due_at,
			updated_at
		) values ($1, false, 'weekly', $2, 1, '09:00', null, now())
		on conflict (patient_id) do update
		set timezone = excluded.timezone,
		    updated_at = excluded.updated_at
	`, patientID, strings.TrimSpace(input.Timezone)); execErr != nil {
		err = fmt.Errorf("create screening schedule: %w", execErr)
		return Patient{}, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return Patient{}, fmt.Errorf("commit create patient tx: %w", commitErr)
	}

	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return Patient{}, err
	}
	if !ok {
		return Patient{}, ErrPatientNotFound
	}

	return patient, nil
}

func (s *PostgresStore) GetPatient(ctx context.Context, patientID string) (Patient, bool, error) {
	row := s.db.QueryRowContext(ctx, patientSelectByID, strings.TrimSpace(patientID))
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Patient{}, fmt.Errorf("begin update patient tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
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
		returning id
	`, strings.TrimSpace(patientID), strings.TrimSpace(input.PrimaryCaregiverID), strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.PreferredName), nullableString(input.PhoneE164), strings.TrimSpace(input.Timezone), nullableString(input.Notes), marshalStringList(input.RoutineAnchors), marshalStringList(input.FavoriteTopics), marshalStringList(input.CalmingCues), marshalStringList(input.TopicsToAvoid))

	var updatedID string
	if scanErr := row.Scan(&updatedID); scanErr != nil {
		switch {
		case errors.Is(scanErr, sql.ErrNoRows):
			return Patient{}, ErrPatientNotFound
		case isUniqueViolation(scanErr):
			return Patient{}, ErrPatientAlreadyAssigned
		case isForeignKeyViolation(scanErr):
			return Patient{}, ErrCaregiverNotFound
		default:
			return Patient{}, fmt.Errorf("update patient: %w", scanErr)
		}
	}

	if err = upsertMemoryProfileTx(ctx, tx, patientID, input.MemoryProfile, input.ConversationGuidance); err != nil {
		return Patient{}, err
	}

	if _, execErr := tx.ExecContext(ctx, `
		update screening_schedules
		set timezone = $2,
		    updated_at = now()
		where patient_id = $1
	`, patientID, strings.TrimSpace(input.Timezone)); execErr != nil {
		return Patient{}, fmt.Errorf("update screening schedule timezone: %w", execErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return Patient{}, fmt.Errorf("commit update patient tx: %w", commitErr)
	}

	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return Patient{}, err
	}
	if !ok {
		return Patient{}, ErrPatientNotFound
	}

	return patient, nil
}

func (s *PostgresStore) GetScreeningSchedule(ctx context.Context, patientID string) (ScreeningSchedule, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select patient_id, enabled, cadence, timezone, preferred_weekday, preferred_local_time, next_due_at, last_scheduled_window_start, last_scheduled_window_end, created_at, updated_at
		from screening_schedules
		where patient_id = $1
	`, strings.TrimSpace(patientID))

	schedule, err := scanScreeningSchedule(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScreeningSchedule{}, false, nil
		}
		return ScreeningSchedule{}, false, fmt.Errorf("get screening schedule: %w", err)
	}

	return schedule, true, nil
}

func (s *PostgresStore) PutScreeningSchedule(ctx context.Context, patientID string, input ScreeningScheduleInput, now time.Time) (ScreeningSchedule, error) {
	if _, ok, err := s.GetPatient(ctx, patientID); err != nil {
		return ScreeningSchedule{}, err
	} else if !ok {
		return ScreeningSchedule{}, ErrPatientNotFound
	}

	nextDueAt := computeNextDueAt(now, input.Timezone, input.PreferredWeekday, input.PreferredLocalTime, input.Cadence)
	if !input.Enabled {
		nextDueAt = nil
	}

	row := s.db.QueryRowContext(ctx, `
		insert into screening_schedules (
			patient_id,
			enabled,
			cadence,
			timezone,
			preferred_weekday,
			preferred_local_time,
			next_due_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8)
		on conflict (patient_id) do update
		set enabled = excluded.enabled,
		    cadence = excluded.cadence,
		    timezone = excluded.timezone,
		    preferred_weekday = excluded.preferred_weekday,
		    preferred_local_time = excluded.preferred_local_time,
		    next_due_at = excluded.next_due_at,
		    updated_at = excluded.updated_at
		returning patient_id, enabled, cadence, timezone, preferred_weekday, preferred_local_time, next_due_at, last_scheduled_window_start, last_scheduled_window_end, created_at, updated_at
	`, strings.TrimSpace(patientID), input.Enabled, input.Cadence, strings.TrimSpace(input.Timezone), input.PreferredWeekday, strings.TrimSpace(input.PreferredLocalTime), nextDueAt, now)

	schedule, err := scanScreeningSchedule(row)
	if err != nil {
		return ScreeningSchedule{}, fmt.Errorf("put screening schedule: %w", err)
	}

	return schedule, nil
}

func (s *PostgresStore) ListDueScreeningSchedules(ctx context.Context, now time.Time, limit int) ([]ScreeningSchedule, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		select patient_id, enabled, cadence, timezone, preferred_weekday, preferred_local_time, next_due_at, last_scheduled_window_start, last_scheduled_window_end, created_at, updated_at
		from screening_schedules
		where enabled = true
		  and next_due_at is not null
		  and next_due_at <= $1
		order by next_due_at asc
		limit $2
	`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list due screening schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]ScreeningSchedule, 0)
	for rows.Next() {
		schedule, scanErr := scanScreeningSchedule(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan due screening schedule: %w", scanErr)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due screening schedules: %w", err)
	}

	return schedules, nil
}

func (s *PostgresStore) CreateScheduledScreeningCallRun(ctx context.Context, schedule ScreeningSchedule, now time.Time) (CallRun, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CallRun{}, false, fmt.Errorf("begin create scheduled call run tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, `
		select patient_id, enabled, cadence, timezone, preferred_weekday, preferred_local_time, next_due_at, last_scheduled_window_start, last_scheduled_window_end, created_at, updated_at
		from screening_schedules
		where patient_id = $1
		for update
	`, schedule.PatientID)

	lockedSchedule, scanErr := scanScreeningSchedule(row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return CallRun{}, false, ErrScreeningScheduleNotFound
		}
		return CallRun{}, false, fmt.Errorf("lock screening schedule: %w", scanErr)
	}
	if !lockedSchedule.Enabled || lockedSchedule.NextDueAt == nil || lockedSchedule.NextDueAt.After(now) {
		return CallRun{}, false, tx.Commit()
	}

	patient, ok, err := s.getPatientTx(ctx, tx, lockedSchedule.PatientID)
	if err != nil {
		return CallRun{}, false, err
	}
	if !ok {
		return CallRun{}, false, ErrPatientNotFound
	}
	if patient.CallingState == CallingStatePaused {
		return CallRun{}, false, tx.Commit()
	}

	consent, ok, err := s.getConsentStateTx(ctx, tx, lockedSchedule.PatientID)
	if err != nil {
		return CallRun{}, false, err
	}
	if !ok || consent.OutboundCallStatus != ConsentStatusGranted || consent.TranscriptStorageStatus != ConsentStatusGranted {
		return CallRun{}, false, tx.Commit()
	}

	template, err := s.resolveActiveCallTemplateByTypeTx(ctx, tx, CallTypeScreening)
	if err != nil {
		return CallRun{}, false, err
	}

	windowStart := lockedSchedule.NextDueAt.UTC()
	windowEnd := endOfScheduleWindow(windowStart, lockedSchedule.Cadence)
	callRunID, idErr := idgen.New()
	if idErr != nil {
		return CallRun{}, false, idErr
	}

	insertRow := tx.QueryRowContext(ctx, `
		insert into call_runs (
			id,
			patient_id,
			caregiver_id,
			call_template_id,
			call_type,
			channel,
			trigger_type,
			status,
			schedule_window_start,
			schedule_window_end,
			requested_at,
			updated_at
		) values ($1, $2, $3, $4, $5, 'connect', 'scheduled', 'scheduled', $6, $7, $8, $8)
		on conflict (patient_id, call_type, schedule_window_start, schedule_window_end) do nothing
		returning id
	`, callRunID, patient.ID, patient.PrimaryCaregiverID, template.ID, CallTypeScreening, windowStart, windowEnd, now)

	var insertedID string
	created := true
	if err := insertRow.Scan(&insertedID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return CallRun{}, false, fmt.Errorf("insert scheduled call run: %w", err)
		}
		created = false
	}

	nextDueAt := advanceScheduleDueAt(windowStart, lockedSchedule.Timezone, lockedSchedule.PreferredLocalTime, lockedSchedule.Cadence)
	if _, execErr := tx.ExecContext(ctx, `
		update screening_schedules
		set last_scheduled_window_start = $2,
		    last_scheduled_window_end = $3,
		    next_due_at = $4,
		    updated_at = $5
		where patient_id = $1
	`, lockedSchedule.PatientID, windowStart, windowEnd, nextDueAt, now); execErr != nil {
		return CallRun{}, false, fmt.Errorf("advance screening schedule: %w", execErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return CallRun{}, false, fmt.Errorf("commit scheduled call run tx: %w", commitErr)
	}

	if !created {
		return CallRun{}, false, nil
	}

	callRun, ok, err := s.GetCallRun(ctx, insertedID)
	if err != nil {
		return CallRun{}, false, err
	}
	if !ok {
		return CallRun{}, false, ErrCallRunNotFound
	}

	return callRun, true, nil
}

func (s *PostgresStore) GetConsentState(ctx context.Context, patientID string) (ConsentState, bool, error) {
	return s.getConsentStateQuery(ctx, s.db, patientID)
}

func (s *PostgresStore) PutConsentState(ctx context.Context, patientID string, input UpdateConsentRequest, now time.Time) (ConsentState, error) {
	row := s.db.QueryRowContext(ctx, `
		update patient_consent_state
		set outbound_call_status = $2,
		    transcript_storage_status = $3,
		    granted_by_caregiver_id = (
		    	select primary_caregiver_id from patients where id = $1
		    ),
		    granted_at = case when $2 = 'granted' and $3 = 'granted' then $4 else granted_at end,
		    revoked_at = case when $2 = 'revoked' or $3 = 'revoked' then $4 else revoked_at end,
		    notes = $5,
		    updated_at = $4
		where patient_id = $1
		returning patient_id, outbound_call_status, transcript_storage_status, coalesce(granted_by_caregiver_id, ''), granted_at, revoked_at, coalesce(notes, ''), created_at, updated_at
	`, strings.TrimSpace(patientID), input.OutboundCallStatus, input.TranscriptStorageStatus, now, strings.TrimSpace(input.Notes))

	state, err := scanConsentState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConsentState{}, ErrConsentStateNotFound
		}
		return ConsentState{}, fmt.Errorf("put consent state: %w", err)
	}

	return state, nil
}

func (s *PostgresStore) SetPatientPause(ctx context.Context, patientID string, reason string, now time.Time) (Patient, error) {
	if _, err := s.db.ExecContext(ctx, `
		update patients
		set calling_state = 'paused',
		    pause_reason = nullif($2, ''),
		    paused_at = $3,
		    updated_at = $3
		where id = $1
	`, strings.TrimSpace(patientID), strings.TrimSpace(reason), now); err != nil {
		return Patient{}, fmt.Errorf("pause patient: %w", err)
	}

	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return Patient{}, err
	}
	if !ok {
		return Patient{}, ErrPatientNotFound
	}
	return patient, nil
}

func (s *PostgresStore) ClearPatientPause(ctx context.Context, patientID string) (Patient, error) {
	if _, err := s.db.ExecContext(ctx, `
		update patients
		set calling_state = 'active',
		    pause_reason = null,
		    paused_at = null,
		    updated_at = now()
		where id = $1
	`, strings.TrimSpace(patientID)); err != nil {
		return Patient{}, fmt.Errorf("clear patient pause: %w", err)
	}

	patient, ok, err := s.GetPatient(ctx, patientID)
	if err != nil {
		return Patient{}, err
	}
	if !ok {
		return Patient{}, ErrPatientNotFound
	}
	return patient, nil
}

func (s *PostgresStore) ListCallTemplates(ctx context.Context) ([]CallTemplate, error) {
	rows, err := s.db.QueryContext(ctx, callTemplateSelectBase+` where is_active = true order by display_name asc`)
	if err != nil {
		return nil, fmt.Errorf("list call templates: %w", err)
	}
	defer rows.Close()

	templates := make([]CallTemplate, 0)
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
	row := s.db.QueryRowContext(ctx, callTemplateSelectBase+` where id = $1`, strings.TrimSpace(templateID))
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
	return s.resolveActiveCallTemplateByTypeQuery(ctx, s.db, callType)
}

func (s *PostgresStore) CreateCallRun(ctx context.Context, input CreateCallRunParams) (CallRun, error) {
	id, err := idgen.New()
	if err != nil {
		return CallRun{}, err
	}

	status := chooseString(input.Status, CallRunStatusRequested)
	row := s.db.QueryRowContext(ctx, `
		insert into call_runs (
			id,
			patient_id,
			caregiver_id,
			call_template_id,
			call_type,
			channel,
			trigger_type,
			status,
			schedule_window_start,
			schedule_window_end,
			requested_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		returning id, patient_id, caregiver_id, call_template_id, call_type, channel, trigger_type, status, coalesce(source_voice_session_id, ''), schedule_window_start, schedule_window_end, requested_at, started_at, ended_at, coalesce(stop_reason, ''), created_at, updated_at
	`, id, input.PatientID, input.CaregiverID, input.CallTemplate.ID, input.CallType, input.Channel, input.TriggerType, status, input.ScheduleWindowStart, input.ScheduleWindowEnd, input.RequestedAt)

	callRun, scanErr := scanCallRun(row)
	if scanErr != nil {
		if isUniqueViolation(scanErr) {
			return CallRun{}, ErrCallTemplateConflict
		}
		return CallRun{}, fmt.Errorf("create call run: %w", scanErr)
	}

	callRun.CallTemplateSlug = input.CallTemplate.Slug
	callRun.CallTemplateName = input.CallTemplate.DisplayName
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
		  and status in ('requested', 'scheduled')
	`, strings.TrimSpace(callRunID), endedAt, stopReason)
	if err != nil {
		return fmt.Errorf("mark call run failed: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetCallRun(ctx context.Context, callRunID string) (CallRun, bool, error) {
	row := s.db.QueryRowContext(ctx, callRunSelectByID, strings.TrimSpace(callRunID))
	callRun, err := scanCallRunWithTemplate(row)
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

	callRuns := make([]CallRun, 0)
	for rows.Next() {
		callRun, scanErr := scanCallRunWithTemplate(rows)
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
		select sequence_no, direction, coalesce(speaker_role, ''), modality, transcript_text, occurred_at, coalesce(stop_reason, '')
		from voice_transcript_turns
		where voice_session_id = $1
		order by sequence_no asc
	`, callRun.SourceVoiceSessionID)
	if err != nil {
		return nil, fmt.Errorf("list transcript turns: %w", err)
	}
	defer rows.Close()

	turns := make([]CallTranscriptTurn, 0)
	for rows.Next() {
		var turn CallTranscriptTurn
		if err := rows.Scan(&turn.SequenceNo, &turn.Direction, &turn.SpeakerRole, &turn.Modality, &turn.Text, &turn.OccurredAt, &turn.StopReason); err != nil {
			return nil, fmt.Errorf("scan transcript turn: %w", err)
		}
		turns = append(turns, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transcript turns: %w", err)
	}

	return turns, nil
}

func (s *PostgresStore) GetAnalysisJob(ctx context.Context, callRunID string) (AnalysisJob, bool, error) {
	row := s.db.QueryRowContext(ctx, analysisJobSelectBase+` where call_run_id = $1`, strings.TrimSpace(callRunID))
	job, err := scanAnalysisJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AnalysisJob{}, false, nil
		}
		return AnalysisJob{}, false, fmt.Errorf("get analysis job: %w", err)
	}

	return job, true, nil
}

func (s *PostgresStore) UpsertAnalysisJob(ctx context.Context, input UpsertAnalysisJobParams) (AnalysisJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AnalysisJob{}, fmt.Errorf("begin upsert analysis job tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(ctx, analysisJobSelectBase+` where call_run_id = $1 for update`, strings.TrimSpace(input.CallRunID))
	job, scanErr := scanAnalysisJob(row)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		return AnalysisJob{}, fmt.Errorf("load analysis job: %w", scanErr)
	}

	if scanErr == nil {
		if !input.Force && (job.Status == AnalysisJobStatusPending || job.Status == AnalysisJobStatusRunning || job.Status == AnalysisJobStatusSucceeded) {
			if commitErr := tx.Commit(); commitErr != nil {
				return AnalysisJob{}, fmt.Errorf("commit existing analysis job tx: %w", commitErr)
			}
			return job, nil
		}

		row = tx.QueryRowContext(ctx, `
			update analysis_jobs
			set status = 'pending',
			    last_error = null,
			    locked_at = null,
			    started_at = null,
			    finished_at = null,
			    analysis_prompt_version = $2,
			    analysis_schema_version = $3,
			    model_provider = $4,
			    model_name = $5,
			    updated_at = $6
			where id = $1
			returning id, call_run_id, status, attempt_count, coalesce(last_error, ''), locked_at, started_at, finished_at, analysis_prompt_version, analysis_schema_version, model_provider, model_name, created_at, updated_at
		`, job.ID, input.AnalysisPromptVersion, input.AnalysisSchemaVersion, input.ModelProvider, input.ModelName, input.Now)
		job, err = scanAnalysisJob(row)
		if err != nil {
			return AnalysisJob{}, fmt.Errorf("reset analysis job: %w", err)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return AnalysisJob{}, fmt.Errorf("commit reset analysis job tx: %w", commitErr)
		}
		return job, nil
	}

	jobID, idErr := idgen.New()
	if idErr != nil {
		return AnalysisJob{}, idErr
	}

	row = tx.QueryRowContext(ctx, `
		insert into analysis_jobs (
			id,
			call_run_id,
			status,
			analysis_prompt_version,
			analysis_schema_version,
			model_provider,
			model_name,
			updated_at
		) values ($1, $2, 'pending', $3, $4, $5, $6, $7)
		returning id, call_run_id, status, attempt_count, coalesce(last_error, ''), locked_at, started_at, finished_at, analysis_prompt_version, analysis_schema_version, model_provider, model_name, created_at, updated_at
	`, jobID, strings.TrimSpace(input.CallRunID), input.AnalysisPromptVersion, input.AnalysisSchemaVersion, input.ModelProvider, input.ModelName, input.Now)

	job, err = scanAnalysisJob(row)
	if err != nil {
		return AnalysisJob{}, fmt.Errorf("insert analysis job: %w", err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return AnalysisJob{}, fmt.Errorf("commit insert analysis job tx: %w", commitErr)
	}

	return job, nil
}

func (s *PostgresStore) ClaimNextAnalysisJob(ctx context.Context, now time.Time) (AnalysisJob, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		with next_job as (
			select id
			from analysis_jobs
			where status = 'pending'
			order by created_at asc
			for update skip locked
			limit 1
		)
		update analysis_jobs aj
		set status = 'running',
		    attempt_count = attempt_count + 1,
		    locked_at = $1,
		    started_at = coalesce(started_at, $1),
		    updated_at = $1
		from next_job
		where aj.id = next_job.id
		returning aj.id, aj.call_run_id, aj.status, aj.attempt_count, coalesce(aj.last_error, ''), aj.locked_at, aj.started_at, aj.finished_at, aj.analysis_prompt_version, aj.analysis_schema_version, aj.model_provider, aj.model_name, aj.created_at, aj.updated_at
	`, now)

	job, err := scanAnalysisJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AnalysisJob{}, false, nil
		}
		return AnalysisJob{}, false, fmt.Errorf("claim analysis job: %w", err)
	}

	return job, true, nil
}

func (s *PostgresStore) MarkAnalysisJobFailed(ctx context.Context, jobID, lastError string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		update analysis_jobs
		set status = 'failed',
		    last_error = $2,
		    locked_at = null,
		    finished_at = $3,
		    updated_at = $3
		where id = $1
	`, strings.TrimSpace(jobID), strings.TrimSpace(lastError), now)
	if err != nil {
		return fmt.Errorf("mark analysis job failed: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAnalysisRecord(ctx context.Context, callRunID string) (AnalysisRecord, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select id, call_run_id, coalesce(call_template_id, ''), model_id, model_provider, model_name, call_prompt_version, analysis_prompt_version, analysis_schema_version, generated_at, raw_result_json, created_at, updated_at
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
	if len(record.Result.RiskFlags) == 0 {
		record.Result.RiskFlags = make([]AnalysisRiskFlag, 0, len(riskFlags))
		for _, flag := range riskFlags {
			record.Result.RiskFlags = append(record.Result.RiskFlags, AnalysisRiskFlag{
				FlagType:     flag.FlagType,
				Severity:     flag.Severity,
				Evidence:     flag.Evidence,
				Reason:       flag.Reason,
				WhyItMatters: flag.WhyItMatters,
				Confidence:   flag.Confidence,
			})
		}
	}
	hydrateLegacyAnalysisPayload(&record.Result)

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

	schedule, ok, err := s.GetScreeningSchedule(ctx, patient.ID)
	if err != nil {
		return AnalysisPromptContext{}, err
	}

	recentAnalyses, err := s.listRecentAnalysisPayloads(ctx, patient.ID, callRunID, 5)
	if err != nil {
		return AnalysisPromptContext{}, err
	}

	contextValue := AnalysisPromptContext{
		CallRun:         callRun,
		Patient:         patient,
		Caregiver:       caregiver,
		CallTemplate:    callTemplate,
		TranscriptTurns: turns,
		RecentAnalyses:  recentAnalyses,
	}
	if ok {
		contextValue.ScreeningSchedule = &schedule
	}

	return contextValue, nil
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

	recommendedCallType := input.CallType
	windowBucket := TimeframeUnspecified
	goal := strings.TrimSpace(input.Result.CaregiverReviewReason)
	if goal == "" {
		goal = strings.TrimSpace(input.Result.Summary)
	}
	if input.Result.NextCallRecommendation != nil {
		recommendedCallType = input.Result.NextCallRecommendation.CallType
		windowBucket = input.Result.NextCallRecommendation.WindowBucket
		if strings.TrimSpace(input.Result.NextCallRecommendation.Goal) != "" {
			goal = strings.TrimSpace(input.Result.NextCallRecommendation.Goal)
		}
	} else if input.Result.FollowUpIntent.RequestedByPatient && input.Result.FollowUpIntent.TimeframeBucket != "" {
		windowBucket = input.Result.FollowUpIntent.TimeframeBucket
	}

	currentTemplate, ok, err := s.getCallTemplateByIDTx(ctx, tx, input.CallTemplateID)
	if err != nil {
		return AnalysisRecord{}, err
	}
	if !ok {
		return AnalysisRecord{}, ErrCallTemplateNotFound
	}

	recommendedTemplate := currentTemplate
	hasActiveRecommendedTemplate := currentTemplate.IsActive && currentTemplate.CallType == recommendedCallType
	if contains(activeCallTypes(), recommendedCallType) {
		template, resolveErr := s.resolveActiveCallTemplateByTypeTx(ctx, tx, recommendedCallType)
		if resolveErr == nil {
			recommendedTemplate = template
			hasActiveRecommendedTemplate = true
		}
	}

	row := tx.QueryRowContext(ctx, `
		insert into analysis_results (
			id,
			call_run_id,
			patient_id,
			call_template_id,
			model_id,
			model_provider,
			model_name,
			call_prompt_version,
			analysis_prompt_version,
			schema_version,
			analysis_schema_version,
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
			caregiver_review_reason,
			follow_up_requested_by_patient,
			follow_up_evidence,
			generated_at,
			updated_at
		) values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $11, $12, $13, 'unclear', 'unclear', 'medium', 0.0, $14, $15, $16, $17, $18, $19, $20, $21, $22, $22
		)
		on conflict (call_run_id) do update
		set call_template_id = excluded.call_template_id,
		    model_id = excluded.model_id,
		    model_provider = excluded.model_provider,
		    model_name = excluded.model_name,
		    call_prompt_version = excluded.call_prompt_version,
		    analysis_prompt_version = excluded.analysis_prompt_version,
		    schema_version = excluded.schema_version,
		    analysis_schema_version = excluded.analysis_schema_version,
		    raw_result_json = excluded.raw_result_json,
		    dashboard_summary = excluded.dashboard_summary,
		    caregiver_summary = excluded.caregiver_summary,
		    escalation_level = excluded.escalation_level,
		    recommended_call_type = excluded.recommended_call_type,
		    recommended_time_note = excluded.recommended_time_note,
		    recommended_duration_minutes = excluded.recommended_duration_minutes,
		    recommended_goal = excluded.recommended_goal,
		    caregiver_review_reason = excluded.caregiver_review_reason,
		    follow_up_requested_by_patient = excluded.follow_up_requested_by_patient,
		    follow_up_evidence = excluded.follow_up_evidence,
		    generated_at = excluded.generated_at,
		    updated_at = excluded.updated_at
		returning id
	`, analysisID, input.CallRunID, input.PatientID, nullableString(input.CallTemplateID), input.ModelName, input.ModelProvider, input.ModelName, input.CallPromptVersion, input.AnalysisPromptVersion, input.SchemaVersion, payload, summarizeAnalysisForDashboard(input.Result), summarizeAnalysisForCaregiver(input.Result), input.Result.EscalationLevel, recommendedCallType, nullableString(windowBucket), recommendedTemplate.DurationMinutes, goal, nullableString(input.Result.CaregiverReviewReason), input.Result.FollowUpIntent.RequestedByPatient, nullableString(input.Result.FollowUpIntent.Evidence), input.GeneratedAt)
	if scanErr := row.Scan(&analysisID); scanErr != nil {
		return AnalysisRecord{}, fmt.Errorf("upsert analysis result: %w", scanErr)
	}

	if _, execErr := tx.ExecContext(ctx, `delete from risk_flags where analysis_result_id = $1`, analysisID); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("delete existing risk flags: %w", execErr)
	}

	for _, flag := range input.Result.RiskFlags {
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
		`, riskID, analysisID, flag.FlagType, flag.Severity, nullableString(flag.Evidence), nullableString(flag.Reason), flag.Confidence); execErr != nil {
			return AnalysisRecord{}, fmt.Errorf("insert risk flag: %w", execErr)
		}
	}

	if err := s.materializeAnalysisSideEffectsTx(ctx, tx, input, analysisID); err != nil {
		return AnalysisRecord{}, err
	}

	if _, execErr := tx.ExecContext(ctx, `
		update next_call_plans
		set approval_status = 'superseded',
		    updated_at = $2
		where patient_id = $1
		  and approval_status in ('pending_approval', 'approved')
	`, input.PatientID, input.GeneratedAt); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("supersede active next call plans: %w", execErr)
	}

	if shouldCreateNextCallPlan(input.Result) && hasActiveRecommendedTemplate {
		windowStart, windowEnd, err := deriveSuggestedWindow(input.GeneratedAt, input.PatientTimezone, windowBucket)
		if err != nil {
			return AnalysisRecord{}, err
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
				suggested_window_start_at,
				suggested_window_end_at,
				duration_minutes,
				goal,
				follow_up_requested_by_patient,
				follow_up_evidence,
				caregiver_review_reason,
				approval_status,
				updated_at
			) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'pending_approval', $14)
		`, nextCallPlanID, input.PatientID, analysisID, recommendedTemplate.ID, recommendedCallType, nullableString(windowBucket), windowStart, windowEnd, recommendedTemplate.DurationMinutes, goal, input.Result.FollowUpIntent.RequestedByPatient, nullableString(input.Result.FollowUpIntent.Evidence), nullableString(input.Result.CaregiverReviewReason), input.GeneratedAt); execErr != nil {
			return AnalysisRecord{}, fmt.Errorf("insert next call plan: %w", execErr)
		}
	}

	if _, execErr := tx.ExecContext(ctx, `
		update analysis_jobs
		set status = 'succeeded',
		    last_error = null,
		    locked_at = null,
		    finished_at = $2,
		    updated_at = $2
		where call_run_id = $1
	`, input.CallRunID, input.GeneratedAt); execErr != nil {
		return AnalysisRecord{}, fmt.Errorf("mark analysis job succeeded: %w", execErr)
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
	durationMinutes := current.DurationMinutes
	if input.CallTemplate != nil {
		callTemplateID = input.CallTemplate.ID
		callType = input.CallTemplate.CallType
		callTemplateSlug = input.CallTemplate.Slug
		callTemplateName = input.CallTemplate.DisplayName
		durationMinutes = input.CallTemplate.DurationMinutes
	}
	if input.DurationMinutes != nil {
		durationMinutes = *input.DurationMinutes
	}

	suggestedTimeNote := chooseString(input.SuggestedTimeNote, current.SuggestedTimeNote)
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
			suggested_window_start_at,
			suggested_window_end_at,
			planned_for,
			duration_minutes,
			goal,
			follow_up_requested_by_patient,
			coalesce(follow_up_evidence, ''),
			coalesce(caregiver_review_reason, ''),
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

	activePlan, planFound, err := s.GetActiveNextCallPlan(ctx, patient.ID)
	if err != nil {
		return DashboardSnapshot{}, err
	}

	schedule, scheduleFound, err := s.GetScreeningSchedule(ctx, patient.ID)
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
	if planFound {
		dashboard.ActiveNextCallPlan = &activePlan
	}
	if scheduleFound {
		dashboard.ScreeningSchedule = &schedule
	}

	return dashboard, nil
}

func (s *PostgresStore) getConsentStateTx(ctx context.Context, tx *sql.Tx, patientID string) (ConsentState, bool, error) {
	return s.getConsentStateQuery(ctx, tx, patientID)
}

func (s *PostgresStore) getConsentStateQuery(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, patientID string) (ConsentState, bool, error) {
	row := queryer.QueryRowContext(ctx, `
		select patient_id, outbound_call_status, transcript_storage_status, coalesce(granted_by_caregiver_id, ''), granted_at, revoked_at, coalesce(notes, ''), created_at, updated_at
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

func (s *PostgresStore) getPatientTx(ctx context.Context, tx *sql.Tx, patientID string) (Patient, bool, error) {
	row := tx.QueryRowContext(ctx, patientSelectByID, strings.TrimSpace(patientID))
	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Patient{}, false, nil
		}
		return Patient{}, false, fmt.Errorf("get patient tx: %w", err)
	}
	return patient, true, nil
}

func (s *PostgresStore) resolveActiveCallTemplateByTypeTx(ctx context.Context, tx *sql.Tx, callType string) (CallTemplate, error) {
	return s.resolveActiveCallTemplateByTypeQuery(ctx, tx, callType)
}

func (s *PostgresStore) resolveActiveCallTemplateByTypeQuery(ctx context.Context, queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, callType string) (CallTemplate, error) {
	rows, err := queryer.QueryContext(ctx, callTemplateSelectBase+` where call_type = $1 and is_active = true`, strings.TrimSpace(callType))
	if err != nil {
		return CallTemplate{}, fmt.Errorf("resolve call template by type: %w", err)
	}
	defer rows.Close()

	matches := make([]CallTemplate, 0, 2)
	for rows.Next() {
		template, scanErr := scanCallTemplate(rows)
		if scanErr != nil {
			return CallTemplate{}, fmt.Errorf("scan call template by type: %w", scanErr)
		}
		matches = append(matches, template)
	}
	if err := rows.Err(); err != nil {
		return CallTemplate{}, fmt.Errorf("iterate call templates by type: %w", err)
	}
	if len(matches) != 1 {
		return CallTemplate{}, ErrCallTemplateConflict
	}
	return matches[0], nil
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

	payloads := make([]AnalysisPayload, 0)
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

	flags := make([]RiskFlag, 0)
	for rows.Next() {
		var flag RiskFlag
		if err := rows.Scan(&flag.ID, &flag.AnalysisResultID, &flag.FlagType, &flag.Severity, &flag.Evidence, &flag.Reason, &flag.Confidence, &flag.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan risk flag: %w", err)
		}
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate risk flags: %w", err)
	}
	hydrateLegacyRiskFlags(flags)

	return flags, nil
}

func upsertMemoryProfileTx(ctx context.Context, tx *sql.Tx, patientID string, profile MemoryProfile, guidance ConversationGuidance) error {
	if _, err := tx.ExecContext(ctx, `
		insert into patient_memory_profiles (
			patient_id,
			likes,
			family_members,
			life_events,
			reminiscence_notes,
			significant_places,
			life_chapters,
			favorite_music,
			favorite_shows_films,
			topics_to_revisit,
			preferred_greeting_style,
			calming_topics,
			upsetting_topics,
			hearing_or_pacing_notes,
			best_time_of_day,
			do_not_mention,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, now())
		on conflict (patient_id) do update
		set likes = excluded.likes,
		    family_members = excluded.family_members,
		    life_events = excluded.life_events,
		    reminiscence_notes = excluded.reminiscence_notes,
		    significant_places = excluded.significant_places,
		    life_chapters = excluded.life_chapters,
		    favorite_music = excluded.favorite_music,
		    favorite_shows_films = excluded.favorite_shows_films,
		    topics_to_revisit = excluded.topics_to_revisit,
		    preferred_greeting_style = excluded.preferred_greeting_style,
		    calming_topics = excluded.calming_topics,
		    upsetting_topics = excluded.upsetting_topics,
		    hearing_or_pacing_notes = excluded.hearing_or_pacing_notes,
		    best_time_of_day = excluded.best_time_of_day,
		    do_not_mention = excluded.do_not_mention,
		    updated_at = excluded.updated_at
	`, patientID, marshalStringList(profile.Likes), marshalJSON(profile.FamilyMembers), marshalJSON(profile.LifeEvents), nullableString(profile.ReminiscenceNotes), marshalStringList(profile.SignificantPlaces), marshalStringList(profile.LifeChapters), marshalStringList(profile.FavoriteMusic), marshalStringList(profile.FavoriteShowsFilms), marshalStringList(profile.TopicsToRevisit), nullableString(guidance.PreferredGreetingStyle), marshalStringList(guidance.CalmingTopics), marshalStringList(guidance.UpsettingTopics), nullableString(guidance.HearingOrPacingNotes), nullableString(guidance.BestTimeOfDay), marshalStringList(guidance.DoNotMention)); err != nil {
		return fmt.Errorf("upsert patient memory profile: %w", err)
	}
	return nil
}

const patientSelectBase = `
	select
		p.id,
		p.primary_caregiver_id,
		p.display_name,
		p.preferred_name,
		p.phone_e164,
		p.timezone,
		p.notes,
		p.calling_state,
		p.pause_reason,
		p.paused_at,
		p.routine_anchors,
		p.favorite_topics,
		p.calming_cues,
		p.topics_to_avoid,
		coalesce(pmp.likes, '[]'::jsonb),
		coalesce(pmp.family_members, '[]'::jsonb),
		coalesce(pmp.life_events, '[]'::jsonb),
		coalesce(pmp.reminiscence_notes, ''),
		coalesce(pmp.significant_places, '[]'::jsonb),
		coalesce(pmp.life_chapters, '[]'::jsonb),
		coalesce(pmp.favorite_music, '[]'::jsonb),
		coalesce(pmp.favorite_shows_films, '[]'::jsonb),
		coalesce(pmp.topics_to_revisit, '[]'::jsonb),
		coalesce(pmp.preferred_greeting_style, ''),
		coalesce(pmp.calming_topics, '[]'::jsonb),
		coalesce(pmp.upsetting_topics, '[]'::jsonb),
		coalesce(pmp.hearing_or_pacing_notes, ''),
		coalesce(pmp.best_time_of_day, ''),
		coalesce(pmp.do_not_mention, '[]'::jsonb),
		p.created_at,
		p.updated_at
	from patients p
	left join patient_memory_profiles pmp on pmp.patient_id = p.id
`

const patientSelectByID = patientSelectBase + `
	where p.id = $1
`

const callTemplateSelectBase = `
	select
		id,
		slug,
		display_name,
		call_type,
		description,
		duration_minutes,
		coalesce(prompt_version, call_prompt_version),
		call_prompt_version,
		system_prompt_template,
		analysis_prompt_version,
		analysis_prompt_template,
		checklist_json,
		is_active,
		created_at,
		updated_at
	from call_templates
`

const callRunSelectColumns = `
	select
		cr.id,
		cr.patient_id,
		cr.caregiver_id,
		cr.call_template_id,
		ct.slug,
		ct.display_name,
		cr.call_type,
		cr.channel,
		cr.trigger_type,
		cr.status,
		coalesce(cr.source_voice_session_id, ''),
		cr.schedule_window_start,
		cr.schedule_window_end,
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

const analysisJobSelectBase = `
	select id, call_run_id, status, attempt_count, coalesce(last_error, ''), locked_at, started_at, finished_at, analysis_prompt_version, analysis_schema_version, model_provider, model_name, created_at, updated_at
	from analysis_jobs
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
		ncp.suggested_window_start_at,
		ncp.suggested_window_end_at,
		ncp.planned_for,
		ncp.duration_minutes,
		ncp.goal,
		ncp.follow_up_requested_by_patient,
		coalesce(ncp.follow_up_evidence, ''),
		coalesce(ncp.caregiver_review_reason, ''),
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
		patient                Patient
		phone                  sql.NullString
		notes                  sql.NullString
		pauseReason            sql.NullString
		pausedAt               sql.NullTime
		routineAnchors         []byte
		favoriteTopics         []byte
		calmingCues            []byte
		topicsToAvoid          []byte
		likes                  []byte
		familyMembers          []byte
		lifeEvents             []byte
		reminiscenceNotes      string
		significantPlaces      []byte
		lifeChapters           []byte
		favoriteMusic          []byte
		favoriteShowsFilms     []byte
		topicsToRevisit        []byte
		preferredGreetingStyle string
		calmingTopics          []byte
		upsettingTopics        []byte
		hearingOrPacingNotes   string
		bestTimeOfDay          string
		doNotMention           []byte
	)
	if err := row.Scan(
		&patient.ID,
		&patient.PrimaryCaregiverID,
		&patient.DisplayName,
		&patient.PreferredName,
		&phone,
		&patient.Timezone,
		&notes,
		&patient.CallingState,
		&pauseReason,
		&pausedAt,
		&routineAnchors,
		&favoriteTopics,
		&calmingCues,
		&topicsToAvoid,
		&likes,
		&familyMembers,
		&lifeEvents,
		&reminiscenceNotes,
		&significantPlaces,
		&lifeChapters,
		&favoriteMusic,
		&favoriteShowsFilms,
		&topicsToRevisit,
		&preferredGreetingStyle,
		&calmingTopics,
		&upsettingTopics,
		&hearingOrPacingNotes,
		&bestTimeOfDay,
		&doNotMention,
		&patient.CreatedAt,
		&patient.UpdatedAt,
	); err != nil {
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
	patient.MemoryProfile = MemoryProfile{
		Likes:              parseStringList(likes),
		FamilyMembers:      parseJSONSlice[FamilyMember](familyMembers),
		LifeEvents:         parseJSONSlice[LifeEvent](lifeEvents),
		ReminiscenceNotes:  strings.TrimSpace(reminiscenceNotes),
		SignificantPlaces:  parseStringList(significantPlaces),
		LifeChapters:       parseStringList(lifeChapters),
		FavoriteMusic:      parseStringList(favoriteMusic),
		FavoriteShowsFilms: parseStringList(favoriteShowsFilms),
		TopicsToRevisit:    parseStringList(topicsToRevisit),
	}
	patient.ConversationGuidance = ConversationGuidance{
		PreferredGreetingStyle: strings.TrimSpace(preferredGreetingStyle),
		CalmingTopics:          parseStringList(calmingTopics),
		UpsettingTopics:        parseStringList(upsettingTopics),
		HearingOrPacingNotes:   strings.TrimSpace(hearingOrPacingNotes),
		BestTimeOfDay:          strings.TrimSpace(bestTimeOfDay),
		DoNotMention:           parseStringList(doNotMention),
	}
	return patient, nil
}

func scanScreeningSchedule(row scanner) (ScreeningSchedule, error) {
	var (
		schedule                 ScreeningSchedule
		nextDueAt                sql.NullTime
		lastScheduledWindowStart sql.NullTime
		lastScheduledWindowEnd   sql.NullTime
	)
	if err := row.Scan(&schedule.PatientID, &schedule.Enabled, &schedule.Cadence, &schedule.Timezone, &schedule.PreferredWeekday, &schedule.PreferredLocalTime, &nextDueAt, &lastScheduledWindowStart, &lastScheduledWindowEnd, &schedule.CreatedAt, &schedule.UpdatedAt); err != nil {
		return ScreeningSchedule{}, err
	}
	if nextDueAt.Valid {
		schedule.NextDueAt = &nextDueAt.Time
	}
	if lastScheduledWindowStart.Valid {
		schedule.LastScheduledWindowStart = &lastScheduledWindowStart.Time
	}
	if lastScheduledWindowEnd.Valid {
		schedule.LastScheduledWindowEnd = &lastScheduledWindowEnd.Time
	}
	return schedule, nil
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
	if err := row.Scan(&template.ID, &template.Slug, &template.DisplayName, &template.CallType, &template.Description, &template.DurationMinutes, &template.PromptVersion, &template.CallPromptVersion, &template.SystemPromptTemplate, &template.AnalysisPromptVersion, &template.AnalysisPromptTemplate, &checklist, &template.IsActive, &template.CreatedAt, &template.UpdatedAt); err != nil {
		return CallTemplate{}, err
	}
	template.Checklist = append(template.Checklist[:0], checklist...)
	if template.PromptVersion == "" {
		template.PromptVersion = template.CallPromptVersion
	}
	return template, nil
}

func scanCallRun(row scanner) (CallRun, error) {
	var (
		callRun             CallRun
		scheduleWindowStart sql.NullTime
		scheduleWindowEnd   sql.NullTime
		startedAt           sql.NullTime
		endedAt             sql.NullTime
	)
	if err := row.Scan(&callRun.ID, &callRun.PatientID, &callRun.CaregiverID, &callRun.CallTemplateID, &callRun.CallType, &callRun.Channel, &callRun.TriggerType, &callRun.Status, &callRun.SourceVoiceSessionID, &scheduleWindowStart, &scheduleWindowEnd, &callRun.RequestedAt, &startedAt, &endedAt, &callRun.StopReason, &callRun.CreatedAt, &callRun.UpdatedAt); err != nil {
		return CallRun{}, err
	}
	if scheduleWindowStart.Valid {
		callRun.ScheduleWindowStart = &scheduleWindowStart.Time
	}
	if scheduleWindowEnd.Valid {
		callRun.ScheduleWindowEnd = &scheduleWindowEnd.Time
	}
	if startedAt.Valid {
		callRun.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		callRun.EndedAt = &endedAt.Time
	}
	return callRun, nil
}

func scanCallRunWithTemplate(row scanner) (CallRun, error) {
	var (
		callRun             CallRun
		scheduleWindowStart sql.NullTime
		scheduleWindowEnd   sql.NullTime
		startedAt           sql.NullTime
		endedAt             sql.NullTime
	)
	if err := row.Scan(&callRun.ID, &callRun.PatientID, &callRun.CaregiverID, &callRun.CallTemplateID, &callRun.CallTemplateSlug, &callRun.CallTemplateName, &callRun.CallType, &callRun.Channel, &callRun.TriggerType, &callRun.Status, &callRun.SourceVoiceSessionID, &scheduleWindowStart, &scheduleWindowEnd, &callRun.RequestedAt, &startedAt, &endedAt, &callRun.StopReason, &callRun.CreatedAt, &callRun.UpdatedAt); err != nil {
		return CallRun{}, err
	}
	if scheduleWindowStart.Valid {
		callRun.ScheduleWindowStart = &scheduleWindowStart.Time
	}
	if scheduleWindowEnd.Valid {
		callRun.ScheduleWindowEnd = &scheduleWindowEnd.Time
	}
	if startedAt.Valid {
		callRun.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		callRun.EndedAt = &endedAt.Time
	}
	return callRun, nil
}

func scanAnalysisJob(row scanner) (AnalysisJob, error) {
	var (
		job        AnalysisJob
		lockedAt   sql.NullTime
		startedAt  sql.NullTime
		finishedAt sql.NullTime
	)
	if err := row.Scan(&job.ID, &job.CallRunID, &job.Status, &job.AttemptCount, &job.LastError, &lockedAt, &startedAt, &finishedAt, &job.AnalysisPromptVersion, &job.SchemaVersion, &job.ModelProvider, &job.ModelName, &job.CreatedAt, &job.UpdatedAt); err != nil {
		return AnalysisJob{}, err
	}
	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	return job, nil
}

func scanAnalysisRecord(row scanner) (AnalysisRecord, error) {
	var (
		record AnalysisRecord
		raw    json.RawMessage
	)
	if err := row.Scan(&record.ID, &record.CallRunID, &record.CallTemplateID, &record.ModelID, &record.ModelProvider, &record.ModelName, &record.CallPromptVersion, &record.AnalysisPromptVersion, &record.SchemaVersion, &record.GeneratedAt, &raw, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return AnalysisRecord{}, err
	}
	if err := json.Unmarshal(raw, &record.Result); err != nil {
		return AnalysisRecord{}, fmt.Errorf("decode analysis result json: %w", err)
	}
	return record, nil
}

func scanNextCallPlan(row scanner) (NextCallPlan, error) {
	var (
		plan                   NextCallPlan
		suggestedWindowStartAt sql.NullTime
		suggestedWindowEndAt   sql.NullTime
		plannedFor             sql.NullTime
		approvedAt             sql.NullTime
		rejectedAt             sql.NullTime
	)
	if err := row.Scan(&plan.ID, &plan.PatientID, &plan.SourceAnalysisResultID, &plan.CallTemplateID, &plan.CallTemplateSlug, &plan.CallTemplateName, &plan.CallType, &plan.SuggestedTimeNote, &suggestedWindowStartAt, &suggestedWindowEndAt, &plannedFor, &plan.DurationMinutes, &plan.Goal, &plan.FollowUpRequestedByPatient, &plan.FollowUpEvidence, &plan.CaregiverReviewReason, &plan.ApprovalStatus, &plan.ApprovedByCaregiverID, &plan.ApprovedByAdminUsername, &approvedAt, &plan.RejectionReason, &rejectedAt, &plan.ExecutedCallRunID, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return NextCallPlan{}, err
	}
	if suggestedWindowStartAt.Valid {
		plan.SuggestedWindowStartAt = &suggestedWindowStartAt.Time
	}
	if suggestedWindowEndAt.Valid {
		plan.SuggestedWindowEndAt = &suggestedWindowEndAt.Time
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

func marshalJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
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

func parseJSONSlice[T any](raw []byte) []T {
	if len(raw) == 0 {
		return []T{}
	}
	var values []T
	if err := json.Unmarshal(raw, &values); err != nil {
		return []T{}
	}
	return values
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
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
