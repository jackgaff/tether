package preferences

import "time"

type Preference struct {
	PatientID      string     `json:"patientId"`
	DefaultVoiceID string     `json:"defaultVoiceId"`
	IsConfigured   bool       `json:"isConfigured"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

type UpdatePreferenceRequest struct {
	DefaultVoiceID string `json:"defaultVoiceId"`
}
