package admin

import (
	"errors"
	"fmt"
)

var (
	ErrCaregiverNotFound            = errors.New("caregiver not found")
	ErrPatientNotFound              = errors.New("patient not found")
	ErrConsentStateNotFound         = errors.New("patient consent state not found")
	ErrScreeningScheduleNotFound    = errors.New("screening schedule not found")
	ErrCallTemplateNotFound         = errors.New("call template not found")
	ErrCallTemplateConflict         = errors.New("call type must resolve to exactly one active template")
	ErrCallRunNotFound              = errors.New("call run not found")
	ErrCallRunNotCompleted          = errors.New("call run must be completed before analysis can run")
	ErrCallRunVoiceSessionMissing   = errors.New("call run must be linked to a voice session before analysis can run")
	ErrAnalysisNotFound             = errors.New("analysis result not found")
	ErrAnalysisJobNotFound          = errors.New("analysis job not found")
	ErrNextCallPlanNotFound         = errors.New("active next-call plan not found")
	ErrApprovedNextCallPlanRequired = errors.New("an approved next-call plan is required")
	ErrPatientConsentRequired       = errors.New("patient consent must be granted before starting a call")
	ErrPatientPaused                = errors.New("patient calling is currently paused")
	ErrPatientAlreadyAssigned       = errors.New("caregiver already has a patient assigned in this MVP")
	ErrAnalysisAlreadyRunning       = errors.New("analysis job is already running")
)

type validationError struct {
	message string
}

func (e validationError) Error() string {
	return e.message
}

func newValidationError(message string) error {
	return validationError{message: message}
}

func newValidationErrorf(format string, args ...any) error {
	return validationError{message: fmt.Sprintf(format, args...)}
}

func isValidationError(err error) bool {
	var target validationError
	return errors.As(err, &target)
}
