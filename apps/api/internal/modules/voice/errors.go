package voice

import "errors"

var (
	ErrPatientIDRequired  = errors.New("patient id is required")
	ErrVoiceNotAllowed    = errors.New("voice id is not allowed")
	ErrSessionNotFound    = errors.New("voice session not found")
	ErrInvalidStreamToken = errors.New("invalid stream token")
	ErrTokenExpired       = errors.New("voice stream token expired")
	ErrStreamConsumed     = errors.New("voice stream token has already been used")
	ErrStreamOriginDenied = errors.New("websocket origin is not allowed")
)
