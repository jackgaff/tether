package voice

import "errors"

var (
	ErrPatientIDRequired      = errors.New("patient id is required")
	ErrVoiceNotAllowed        = errors.New("voice id is not allowed")
	ErrSystemPromptTooLarge   = errors.New("system prompt exceeds the 40 KB limit")
	ErrCallRunNotFound        = errors.New("call run not found")
	ErrCallRunPatientMismatch = errors.New("call run does not belong to the requested patient")
	ErrCallRunAlreadyLinked   = errors.New("call run is already linked to a voice session")
	ErrCallRunLinkInvalid     = errors.New("call run cannot be linked to a voice session")
	ErrSessionNotFound        = errors.New("voice session not found")
	ErrInvalidStreamToken     = errors.New("invalid stream token")
	ErrTokenExpired           = errors.New("voice stream token expired")
	ErrStreamConsumed         = errors.New("voice stream token has already been used")
	ErrStreamOriginDenied     = errors.New("websocket origin is not allowed")
)
