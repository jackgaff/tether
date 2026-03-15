package admin

import (
	"fmt"
	"strings"
	"time"
)

func validateAnalysisPayload(callType string, payload AnalysisPayload) error {
	if strings.TrimSpace(payload.Summary) == "" {
		return newValidationError("analysis result summary is required")
	}
	if !contains(validEscalationLevels(), payload.EscalationLevel) {
		return newValidationError("analysis result escalationLevel is invalid")
	}
	if !contains(validTimeframeBuckets(), payload.FollowUpIntent.TimeframeBucket) {
		return newValidationError("analysis result followUpIntent.timeframeBucket is invalid")
	}
	if payload.FollowUpIntent.Confidence < 0 || payload.FollowUpIntent.Confidence > 1 {
		return newValidationError("analysis result followUpIntent.confidence must be between 0 and 1")
	}
	for _, flag := range payload.RiskFlags {
		if strings.TrimSpace(flag.FlagType) == "" {
			return newValidationError("analysis result riskFlags.flagType is required")
		}
		if !contains(validRiskSeverities(), flag.Severity) {
			return newValidationError("analysis result riskFlags.severity is invalid")
		}
		if flag.Confidence < 0 || flag.Confidence > 1 {
			return newValidationError("analysis result riskFlags.confidence must be between 0 and 1")
		}
	}
	if payload.NextCallRecommendation != nil {
		if !contains(validCallTypes(), payload.NextCallRecommendation.CallType) {
			return newValidationError("analysis result nextCallRecommendation.callType is invalid")
		}
		if !contains(validTimeframeBuckets(), payload.NextCallRecommendation.WindowBucket) {
			return newValidationError("analysis result nextCallRecommendation.windowBucket is invalid")
		}
		if strings.TrimSpace(payload.NextCallRecommendation.Goal) == "" {
			return newValidationError("analysis result nextCallRecommendation.goal is required")
		}
	}

	switch callType {
	case CallTypeScreening:
		if payload.Screening == nil {
			return newValidationError("analysis result screening payload is required for screening calls")
		}
		if !contains(validScreeningCompletionStatuses(), payload.Screening.ScreeningCompletionStatus) {
			return newValidationError("analysis result screening.screeningCompletionStatus is invalid")
		}
		if payload.Screening.ScreeningScoreInterpretation != "" && !contains(validScreeningInterpretations(), payload.Screening.ScreeningScoreInterpretation) {
			return newValidationError("analysis result screening.screeningScoreInterpretation is invalid")
		}
		if payload.Screening.SuggestedRescreenWindowBucket != "" && !contains(validTimeframeBuckets(), payload.Screening.SuggestedRescreenWindowBucket) {
			return newValidationError("analysis result screening.suggestedRescreenWindowBucket is invalid")
		}
	case CallTypeCheckIn:
		if payload.CheckIn == nil {
			return newValidationError("analysis result checkIn payload is required for check-in calls")
		}
	case CallTypeReminiscence:
		if payload.Reminiscence == nil {
			return newValidationError("analysis result reminiscence payload is required for reminiscence calls")
		}
	default:
		return newValidationError("analysis result callType is invalid")
	}

	return nil
}

func shouldCreateNextCallPlan(payload AnalysisPayload) bool {
	return payload.NextCallRecommendation != nil || payload.FollowUpIntent.RequestedByPatient
}

func summarizeAnalysisForDashboard(payload AnalysisPayload) string {
	return strings.TrimSpace(payload.Summary)
}

func summarizeAnalysisForCaregiver(payload AnalysisPayload) string {
	if strings.TrimSpace(payload.CaregiverReviewReason) != "" {
		return strings.TrimSpace(payload.CaregiverReviewReason)
	}
	return strings.TrimSpace(payload.Summary)
}

func computeNextDueAt(reference time.Time, timezone string, preferredWeekday int, preferredLocalTime string, cadence string) *time.Time {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil
	}

	hour, minute, parseErr := parseLocalClock(strings.TrimSpace(preferredLocalTime))
	if parseErr != nil {
		return nil
	}

	local := reference.In(location)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, location)

	daysUntil := (preferredWeekday - int(candidate.Weekday()) + 7) % 7
	if daysUntil == 0 && !candidate.After(local) {
		daysUntil = 7
	}
	candidate = candidate.AddDate(0, 0, daysUntil)

	next := candidate.UTC()
	return &next
}

func advanceScheduleDueAt(windowStart time.Time, timezone string, preferredLocalTime string, cadence string) *time.Time {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil
	}

	hour, minute, parseErr := parseLocalClock(strings.TrimSpace(preferredLocalTime))
	if parseErr != nil {
		return nil
	}

	localStart := windowStart.In(location)
	days := 7
	if cadence == CadenceBiweekly {
		days = 14
	}

	nextLocal := time.Date(
		localStart.Year(),
		localStart.Month(),
		localStart.Day(),
		hour,
		minute,
		0,
		0,
		location,
	).AddDate(0, 0, days)

	next := nextLocal.UTC()
	return &next
}

func endOfScheduleWindow(windowStart time.Time, cadence string) time.Time {
	days := 7
	if cadence == CadenceBiweekly {
		days = 14
	}
	return windowStart.AddDate(0, 0, days)
}

func deriveSuggestedWindow(reference time.Time, timezone string, bucket string) (*time.Time, *time.Time, error) {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil, nil, fmt.Errorf("load patient timezone: %w", err)
	}

	local := reference.In(location)
	startOfDay := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	var start time.Time
	var end time.Time

	switch bucket {
	case "", TimeframeUnspecified:
		return nil, nil, nil
	case TimeframeSameDay:
		start = local
		end = startOfDay.Add(24*time.Hour - time.Second)
	case TimeframeTomorrow:
		start = time.Date(startOfDay.AddDate(0, 0, 1).Year(), startOfDay.AddDate(0, 0, 1).Month(), startOfDay.AddDate(0, 0, 1).Day(), 9, 0, 0, 0, location)
		end = time.Date(start.Year(), start.Month(), start.Day(), 18, 0, 0, 0, location)
	case TimeframeFewDays:
		start = time.Date(startOfDay.AddDate(0, 0, 2).Year(), startOfDay.AddDate(0, 0, 2).Month(), startOfDay.AddDate(0, 0, 2).Day(), 9, 0, 0, 0, location)
		end = time.Date(startOfDay.AddDate(0, 0, 4).Year(), startOfDay.AddDate(0, 0, 4).Month(), startOfDay.AddDate(0, 0, 4).Day(), 18, 0, 0, 0, location)
	case TimeframeNextWeek:
		start = time.Date(startOfDay.AddDate(0, 0, 5).Year(), startOfDay.AddDate(0, 0, 5).Month(), startOfDay.AddDate(0, 0, 5).Day(), 9, 0, 0, 0, location)
		end = time.Date(startOfDay.AddDate(0, 0, 9).Year(), startOfDay.AddDate(0, 0, 9).Month(), startOfDay.AddDate(0, 0, 9).Day(), 18, 0, 0, 0, location)
	case TimeframeTwoWeeks:
		start = time.Date(startOfDay.AddDate(0, 0, 10).Year(), startOfDay.AddDate(0, 0, 10).Month(), startOfDay.AddDate(0, 0, 10).Day(), 9, 0, 0, 0, location)
		end = time.Date(startOfDay.AddDate(0, 0, 16).Year(), startOfDay.AddDate(0, 0, 16).Month(), startOfDay.AddDate(0, 0, 16).Day(), 18, 0, 0, 0, location)
	default:
		return nil, nil, newValidationError("window bucket is invalid")
	}

	startUTC := start.UTC()
	endUTC := end.UTC()
	return &startUTC, &endUTC, nil
}

func parseLocalClock(value string) (int, int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, fmt.Errorf("preferredLocalTime must be HH:MM")
	}
	return parsed.Hour(), parsed.Minute(), nil
}
