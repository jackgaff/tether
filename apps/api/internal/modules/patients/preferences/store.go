package preferences

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Get(ctx context.Context, patientID string) (Preference, bool, error)
	Put(ctx context.Context, patientID string, defaultVoiceID string) (Preference, error)
	GetDefaultVoiceID(ctx context.Context, patientID string) (string, bool, error)
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Preference
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]Preference)}
}

func (s *MemoryStore) Get(_ context.Context, patientID string) (Preference, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	preference, ok := s.items[strings.TrimSpace(patientID)]
	return preference, ok, nil
}

func (s *MemoryStore) Put(_ context.Context, patientID string, defaultVoiceID string) (Preference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	preference := Preference{
		PatientID:      strings.TrimSpace(patientID),
		DefaultVoiceID: strings.TrimSpace(defaultVoiceID),
		IsConfigured:   true,
		UpdatedAt:      &now,
	}

	s.items[preference.PatientID] = preference

	return preference, nil
}

func (s *MemoryStore) GetDefaultVoiceID(ctx context.Context, patientID string) (string, bool, error) {
	preference, ok, err := s.Get(ctx, patientID)
	if err != nil || !ok {
		return "", ok, err
	}

	return preference.DefaultVoiceID, true, nil
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, patientID string) (Preference, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		select patient_id, default_voice_id, updated_at
		from patient_preferences
		where patient_id = $1
	`, strings.TrimSpace(patientID))

	var (
		preference Preference
		updatedAt  time.Time
	)
	if err := row.Scan(&preference.PatientID, &preference.DefaultVoiceID, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Preference{}, false, nil
		}

		return Preference{}, false, fmt.Errorf("get patient preferences: %w", err)
	}

	preference.IsConfigured = true
	preference.UpdatedAt = &updatedAt
	return preference, true, nil
}

func (s *PostgresStore) Put(ctx context.Context, patientID string, defaultVoiceID string) (Preference, error) {
	row := s.db.QueryRowContext(ctx, `
		insert into patient_preferences (
			patient_id,
			default_voice_id,
			updated_at
		) values ($1, $2, now())
		on conflict (patient_id) do update
		set default_voice_id = excluded.default_voice_id,
		    updated_at = now()
		returning patient_id, default_voice_id, updated_at
	`, strings.TrimSpace(patientID), strings.TrimSpace(defaultVoiceID))

	var (
		preference Preference
		updatedAt  time.Time
	)
	if err := row.Scan(&preference.PatientID, &preference.DefaultVoiceID, &updatedAt); err != nil {
		return Preference{}, fmt.Errorf("upsert patient preferences: %w", err)
	}

	preference.IsConfigured = true
	preference.UpdatedAt = &updatedAt
	return preference, nil
}

func (s *PostgresStore) GetDefaultVoiceID(ctx context.Context, patientID string) (string, bool, error) {
	preference, ok, err := s.Get(ctx, patientID)
	if err != nil || !ok {
		return "", ok, err
	}

	return preference.DefaultVoiceID, true, nil
}
