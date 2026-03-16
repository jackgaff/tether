package admin

import (
	"fmt"
	"strings"
	"time"
)

func normalizeAnalysisPayload(payload *AnalysisPayload) {
	if payload == nil {
		return
	}

	payload.EscalationLevel = normalizeEnumValue(payload.EscalationLevel, map[string]string{
		"caregiver_soon":  EscalationCaregiverSoon,
		"clinical_review": EscalationClinicalReview,
	}, "")
	payload.FollowUpIntent.TimeframeBucket = normalizeTimeframeBucket(payload.FollowUpIntent.TimeframeBucket)

	if payload.NextCallRecommendation != nil {
		payload.NextCallRecommendation.CallType = normalizeCallType(payload.NextCallRecommendation.CallType)
		payload.NextCallRecommendation.WindowBucket = normalizeTimeframeBucket(payload.NextCallRecommendation.WindowBucket)
	}

	for index := range payload.RiskFlags {
		payload.RiskFlags[index].Severity = normalizeEnumValue(payload.RiskFlags[index].Severity, map[string]string{
			"warning": "watch",
		}, "")
		payload.RiskFlags[index].FlagType = strings.TrimSpace(payload.RiskFlags[index].FlagType)
		payload.RiskFlags[index].Evidence = strings.TrimSpace(payload.RiskFlags[index].Evidence)
		payload.RiskFlags[index].Reason = strings.TrimSpace(payload.RiskFlags[index].Reason)
		payload.RiskFlags[index].WhyItMatters = strings.TrimSpace(payload.RiskFlags[index].WhyItMatters)
	}

	if payload.CheckIn != nil {
		payload.CheckIn.OrientationStatus = normalizeEnumValue(payload.CheckIn.OrientationStatus, map[string]string{
			"mildly_confused": OrientationStatusMildlyConfused,
			"mild_confused":   OrientationStatusMildlyConfused,
			"mild_confusion":  OrientationStatusMildlyConfused,
			"confused":        OrientationStatusMildlyConfused,
		}, OrientationStatusUnknown)
		payload.CheckIn.MealsStatus = normalizeEnumValue(payload.CheckIn.MealsStatus, map[string]string{
			"not recalled":   CheckInCaptureNotRecalled,
			"not_recalled":   CheckInCaptureNotRecalled,
			"not remembered": CheckInCaptureNotRecalled,
			"mentioned":      CheckInCaptureReported,
		}, CheckInCaptureUncertain)
		payload.CheckIn.FluidsStatus = normalizeEnumValue(payload.CheckIn.FluidsStatus, map[string]string{
			"not recalled":   CheckInCaptureNotRecalled,
			"not_recalled":   CheckInCaptureNotRecalled,
			"not remembered": CheckInCaptureNotRecalled,
			"mentioned":      CheckInCaptureReported,
		}, CheckInCaptureUncertain)
		payload.CheckIn.SocialContact = normalizeEnumValue(payload.CheckIn.SocialContact, map[string]string{
			"none":          SocialContactNo,
			"not_discussed": SocialContactUnknown,
		}, SocialContactUnknown)
		payload.CheckIn.Mood = normalizeEnumValue(payload.CheckIn.Mood, map[string]string{
			"neutral": CheckInMoodCalm,
		}, CheckInMoodUnknown)
		payload.CheckIn.Sleep = normalizeEnumValue(payload.CheckIn.Sleep, nil, SleepStatusUnknown)

		for index := range payload.CheckIn.MentionedPeople {
			payload.CheckIn.MentionedPeople[index].Name = strings.TrimSpace(payload.CheckIn.MentionedPeople[index].Name)
			payload.CheckIn.MentionedPeople[index].Relationship = strings.TrimSpace(payload.CheckIn.MentionedPeople[index].Relationship)
			payload.CheckIn.MentionedPeople[index].Context = strings.TrimSpace(payload.CheckIn.MentionedPeople[index].Context)
		}
		for index := range payload.CheckIn.RemindersNoted {
			payload.CheckIn.RemindersNoted[index].Title = strings.TrimSpace(payload.CheckIn.RemindersNoted[index].Title)
			payload.CheckIn.RemindersNoted[index].Detail = strings.TrimSpace(payload.CheckIn.RemindersNoted[index].Detail)
		}
		for index := range payload.CheckIn.MemoryFlags {
			payload.CheckIn.MemoryFlags[index] = strings.TrimSpace(payload.CheckIn.MemoryFlags[index])
		}
		for index := range payload.CheckIn.DeliriumPotentialTriggers {
			payload.CheckIn.DeliriumPotentialTriggers[index] = strings.TrimSpace(payload.CheckIn.DeliriumPotentialTriggers[index])
		}
	}

	if payload.Reminiscence != nil {
		payload.Reminiscence.AnchorType = normalizeEnumValue(payload.Reminiscence.AnchorType, map[string]string{
			"show":      AnchorTypeShowFilm,
			"film":      AnchorTypeShowFilm,
			"show_film": AnchorTypeShowFilm,
		}, AnchorTypeNone)
		for index := range payload.Reminiscence.MentionedPeople {
			payload.Reminiscence.MentionedPeople[index].Name = strings.TrimSpace(payload.Reminiscence.MentionedPeople[index].Name)
			payload.Reminiscence.MentionedPeople[index].Relationship = strings.TrimSpace(payload.Reminiscence.MentionedPeople[index].Relationship)
			payload.Reminiscence.MentionedPeople[index].Context = strings.TrimSpace(payload.Reminiscence.MentionedPeople[index].Context)
		}
	}
}

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
		if !contains(validOrientationStatuses(), payload.CheckIn.OrientationStatus) {
			return newValidationError("analysis result checkIn.orientationStatus is invalid")
		}
		if !contains(validCheckInCaptureStatuses(), payload.CheckIn.MealsStatus) {
			return newValidationError("analysis result checkIn.mealsStatus is invalid")
		}
		if !contains(validCheckInCaptureStatuses(), payload.CheckIn.FluidsStatus) {
			return newValidationError("analysis result checkIn.fluidsStatus is invalid")
		}
		if !contains(validSocialContactStatuses(), payload.CheckIn.SocialContact) {
			return newValidationError("analysis result checkIn.socialContact is invalid")
		}
		if !contains(validCheckInMoods(), payload.CheckIn.Mood) {
			return newValidationError("analysis result checkIn.mood is invalid")
		}
		if !contains(validSleepStatuses(), payload.CheckIn.Sleep) {
			return newValidationError("analysis result checkIn.sleep is invalid")
		}
		for _, person := range payload.CheckIn.MentionedPeople {
			if strings.TrimSpace(person.Name) == "" {
				return newValidationError("analysis result checkIn.mentionedPeople.name is required")
			}
		}
		for _, reminder := range payload.CheckIn.RemindersNoted {
			if strings.TrimSpace(reminder.Title) == "" && strings.TrimSpace(reminder.Detail) == "" {
				return newValidationError("analysis result checkIn.remindersNoted entries must include a title or detail")
			}
		}
	case CallTypeReminiscence:
		if payload.Reminiscence == nil {
			return newValidationError("analysis result reminiscence payload is required for reminiscence calls")
		}
		if payload.Reminiscence.AnchorType != "" && !contains(validAnchorTypes(), payload.Reminiscence.AnchorType) {
			return newValidationError("analysis result reminiscence.anchorType is invalid")
		}
		for _, person := range payload.Reminiscence.MentionedPeople {
			if strings.TrimSpace(person.Name) == "" {
				return newValidationError("analysis result reminiscence.mentionedPeople.name is required")
			}
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
	if payload.CheckIn != nil && strings.TrimSpace(payload.CheckIn.CaregiverSummary) != "" {
		return strings.TrimSpace(payload.CheckIn.CaregiverSummary)
	}
	if payload.Reminiscence != nil && strings.TrimSpace(payload.Reminiscence.CaregiverSummary) != "" {
		return strings.TrimSpace(payload.Reminiscence.CaregiverSummary)
	}
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
	if payload.CheckIn != nil && contains([]string{OrientationStatusMildlyConfused, OrientationStatusDisoriented}, payload.CheckIn.OrientationStatus) {
		orientation = "mixed"
	}
	if orientation == "unclear" && hasLegacyKeyword(payload, "orientation", "confus", "repeat") {
		orientation = "mixed"
	}
	if payload.Screening != nil && payload.Screening.ScreeningCompletionStatus == ScreeningCompletionComplete && orientation == "unclear" {
		orientation = "mixed"
	}

	mood := "neutral"
	switch {
	case payload.EscalationLevel == EscalationCaregiverNow || payload.EscalationLevel == EscalationClinicalReview:
		mood = "distressed"
	case payload.CheckIn != nil && payload.CheckIn.Mood == CheckInMoodDistressed:
		mood = "distressed"
	case payload.Reminiscence != nil && containsAnySubstring(payload.Reminiscence.EmotionalTone, "joy", "warm", "animated", "tender", "reflective"):
		mood = "positive"
	case payload.CheckIn != nil && payload.CheckIn.Mood == CheckInMoodWithdrawn:
		mood = "anxious"
	case payload.CheckIn != nil && payload.CheckIn.Mood == CheckInMoodCalm:
		mood = "positive"
	case strings.TrimSpace(payload.Summary) == "":
		mood = "unclear"
	}

	engagement := "medium"
	switch {
	case payload.Reminiscence != nil && len(payload.Reminiscence.RespondedWellTo) > 0:
		engagement = "high"
	case payload.CheckIn != nil && (len(payload.CheckIn.RemindersNoted) > 0 || payload.FollowUpIntent.RequestedByPatient):
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

func normalizeCallType(raw string) string {
	return normalizeEnumValue(raw, map[string]string{
		"checkin":     CallTypeCheckIn,
		"check_in":    CallTypeCheckIn,
		"reminisce":   CallTypeReminiscence,
		"reminiscing": CallTypeReminiscence,
	}, "")
}

func normalizeTimeframeBucket(raw string) string {
	return normalizeEnumValue(raw, map[string]string{
		"same_day":  TimeframeSameDay,
		"few_days":  TimeframeFewDays,
		"next_week": TimeframeNextWeek,
		"two_weeks": TimeframeTwoWeeks,
	}, "")
}

func normalizeEnumValue(raw string, aliases map[string]string, unknownFallback string) string {
	normalized := normalizeToken(raw)
	if normalized == "" {
		return unknownFallback
	}
	if aliases != nil {
		if canonical, ok := aliases[normalized]; ok {
			return canonical
		}
	}
	if isUnknownToken(normalized) {
		return unknownFallback
	}
	return normalized
}

func normalizeToken(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	trimmed = strings.Trim(trimmed, `"'`)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "_", " ", "_", "/", "_")
	trimmed = replacer.Replace(trimmed)
	for strings.Contains(trimmed, "__") {
		trimmed = strings.ReplaceAll(trimmed, "__", "_")
	}
	return strings.Trim(trimmed, "_")
}

func isUnknownToken(value string) bool {
	switch value {
	case "", "unknown", "unclear", "not_discussed", "not_mentioned", "not_asked", "not_assessed", "not_reached", "not_covered", "n_a", "na":
		return true
	default:
		return false
	}
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
