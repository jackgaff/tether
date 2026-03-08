package checkins

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"nova-echoes/api/internal/idgen"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) List(ctx context.Context, patientID string) ([]CheckIn, error) {
	query := `
		select id, patient_id, summary, status, agent, reminder, recorded_at
		from check_ins
	`
	args := []any{}
	if trimmed := strings.TrimSpace(patientID); trimmed != "" {
		query += ` where patient_id = $1`
		args = append(args, trimmed)
	}

	query += ` order by recorded_at desc`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list check-ins: %w", err)
	}
	defer rows.Close()

	items := make([]CheckIn, 0)
	for rows.Next() {
		var (
			item     CheckIn
			reminder sql.NullString
		)

		if err := rows.Scan(
			&item.ID,
			&item.PatientID,
			&item.Summary,
			&item.Status,
			&item.Agent,
			&reminder,
			&item.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check-in: %w", err)
		}

		if reminder.Valid {
			item.Reminder = reminder.String
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check-ins: %w", err)
	}

	return items, nil
}

func (s *PostgresStore) Create(ctx context.Context, input CreateCheckInRequest) (CheckIn, error) {
	id, err := idgen.New()
	if err != nil {
		return CheckIn{}, err
	}

	checkIn := CheckIn{
		ID:         id,
		PatientID:  strings.TrimSpace(input.PatientID),
		Summary:    strings.TrimSpace(input.Summary),
		Status:     input.Status,
		Agent:      strings.TrimSpace(input.Agent),
		Reminder:   strings.TrimSpace(input.Reminder),
		RecordedAt: time.Now().UTC(),
	}

	if _, err := s.db.ExecContext(ctx, `
		insert into check_ins (
			id,
			patient_id,
			summary,
			status,
			agent,
			reminder,
			recorded_at,
			updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, now())
	`, checkIn.ID, checkIn.PatientID, checkIn.Summary, checkIn.Status, checkIn.Agent, nullableString(checkIn.Reminder), checkIn.RecordedAt); err != nil {
		return CheckIn{}, fmt.Errorf("create check-in: %w", err)
	}

	return checkIn, nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}
