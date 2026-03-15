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

func hydrateLegacyAnalysisPayload(payload *AnalysisPayload) {
	if payload == nil {
		return
	}

	if strings.TrimSpace(payload.DashboardSummary) == "" {
		payload.DashboardSummary = summarizeAnalysisForDashboard(*payload)
	}
	if strings.TrimSpace(payload.CaregiverSummary) == "" {
		payload.CaregiverSummary = summarizeAnalysisForCaregiver(*payload)
	}
	if payload.PatientState == nil {
		state := deriveLegacyPatientState(*payload)
		payload.PatientState = &state
	}

	for index := range payload.RiskFlags {
		if strings.TrimSpace(payload.RiskFlags[index].WhyItMatters) == "" {
			payload.RiskFlags[index].WhyItMatters = chooseString(
				payload.RiskFlags[index].Reason,
				chooseString(payload.RiskFlags[index].Evidence, payload.RiskFlags[index].FlagType),
			)
		}
	}
}

func hydrateLegacyRiskFlags(flags []RiskFlag) {
	for index := range flags {
		if strings.TrimSpace(flags[index].WhyItMatters) == "" {
			flags[index].WhyItMatters = chooseString(
				flags[index].Reason,
				chooseString(flags[index].Evidence, flags[index].FlagType),
			)
		}
	}
}

func deriveLegacyPatientState(payload AnalysisPayload) LegacyPatientState {
	orientation := "unclear"
	if hasLegacyKeyword(payload, "orientation", "confus", "repeat") {
		orientation = "mixed"
	}
	if payload.Screening != nil && payload.Screening.ScreeningCompletionStatus == ScreeningCompletionComplete && orientation == "unclear" {
		orientation = "mixed"
	}

	mood := "neutral"
	switch {
	case payload.EscalationLevel == EscalationCaregiverNow || payload.EscalationLevel == EscalationClinicalReview:
		mood = "distressed"
	case payload.Reminiscence != nil && len(payload.Reminiscence.DistressOrTriggerSignals) > 0:
		mood = "distressed"
	case payload.Reminiscence != nil && len(payload.Reminiscence.PositiveEngagementSignals) > 0:
		mood = "positive"
	case payload.CheckIn != nil && hasAnyMoodKeyword(payload.CheckIn.MoodSignals, "anxious", "worried", "sad", "upset"):
		mood = "anxious"
	case payload.CheckIn != nil && hasAnyMoodKeyword(payload.CheckIn.MoodSignals, "good", "calm", "happy", "positive", "steady"):
		mood = "positive"
	case strings.TrimSpace(payload.Summary) == "":
		mood = "unclear"
	}

	engagement := "medium"
	switch {
	case payload.Reminiscence != nil && len(payload.Reminiscence.PositiveEngagementSignals) > 0:
		engagement = "high"
	case payload.CheckIn != nil && payload.CheckIn.FollowUpRequestDetected:
		engagement = "high"
	case strings.TrimSpace(payload.Summary) == "" && len(payload.SalientEvidence) == 0:
		engagement = "low"
	}

	confidence := 0.5
	switch {
	case payload.FollowUpIntent.Confidence > 0:
		confidence = payload.FollowUpIntent.Confidence
	case len(payload.RiskFlags) > 0 && payload.RiskFlags[0].Confidence > 0:
		confidence = payload.RiskFlags[0].Confidence
	}

	return LegacyPatientState{
		Orientation: orientation,
		Mood:        mood,
		Engagement:  engagement,
		Confidence:  confidence,
	}
}

func hasLegacyKeyword(payload AnalysisPayload, keywords ...string) bool {
	for _, flag := range payload.RiskFlags {
		if containsAnySubstring(flag.FlagType, keywords...) || containsAnySubstring(flag.Reason, keywords...) || containsAnySubstring(flag.Evidence, keywords...) {
			return true
		}
	}
	if containsAnySubstring(payload.Summary, keywords...) || containsAnySubstring(payload.CaregiverReviewReason, keywords...) {
		return true
	}
	return false
}

func hasAnyMoodKeyword(values []string, keywords ...string) bool {
	for _, value := range values {
		if containsAnySubstring(value, keywords...) {
			return true
		}
	}
	return false
}

func containsAnySubstring(value string, needles ...string) bool {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	if lowerValue == "" {
		return false
	}
	for _, needle := range needles {
		if strings.Contains(lowerValue, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}

func normalizeRequestedCallTrigger(trigger string) string {
	switch strings.TrimSpace(trigger) {
	case "", CallTriggerCaregiverRequested, CallTriggerLegacyManual:
		return CallTriggerCaregiverRequested
	case CallTriggerFollowUpRecommendation, CallTriggerLegacyApprovedNextCall:
		return CallTriggerFollowUpRecommendation
	default:
		return strings.TrimSpace(trigger)
	}
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
