package checkins

import "time"

type Status string

const (
	StatusScheduled     Status = "scheduled"
	StatusCompleted     Status = "completed"
	StatusNeedsFollowUp Status = "needs_follow_up"
)

type CheckIn struct {
	ID         string    `json:"id"`
	PatientID  string    `json:"patientId"`
	Summary    string    `json:"summary"`
	Status     Status    `json:"status"`
	Agent      string    `json:"agent"`
	Reminder   string    `json:"reminder,omitempty"`
	RecordedAt time.Time `json:"recordedAt"`
}

type CreateCheckInRequest struct {
	PatientID string `json:"patientId"`
	Summary   string `json:"summary"`
	Status    Status `json:"status"`
	Agent     string `json:"agent"`
	Reminder  string `json:"reminder"`
}
