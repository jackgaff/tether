package admin

import "testing"

func TestNormalizeAnalysisPayloadCanonicalizesPartialCheckInValues(t *testing.T) {
	t.Parallel()

	payload := AnalysisPayload{
		Summary:         "Brief summary",
		EscalationLevel: "Clinical Review",
		FollowUpIntent: FollowUpIntent{
			TimeframeBucket: "few days",
			Confidence:      0.8,
		},
		NextCallRecommendation: &NextCallRecommendation{
			CallType:     "check-in",
			WindowBucket: "next week",
			Goal:         "Follow up gently.",
		},
		CheckIn: &CheckInAnalysis{
			OrientationStatus: "not discussed",
			MealsStatus:       "not discussed",
			FluidsStatus:      "mentioned",
			SocialContact:     "not discussed",
			Mood:              "neutral",
			Sleep:             "N/A",
			RemindersNoted: []ReminderNote{
				{Title: "  Follow up  ", Detail: " tomorrow "},
			},
		},
	}

	normalizeAnalysisPayload(&payload)

	if payload.EscalationLevel != EscalationClinicalReview {
		t.Fatalf("expected escalation level %q, got %q", EscalationClinicalReview, payload.EscalationLevel)
	}
	if payload.FollowUpIntent.TimeframeBucket != TimeframeFewDays {
		t.Fatalf("expected timeframe bucket %q, got %q", TimeframeFewDays, payload.FollowUpIntent.TimeframeBucket)
	}
	if payload.NextCallRecommendation == nil || payload.NextCallRecommendation.CallType != CallTypeCheckIn {
		t.Fatalf("expected normalized next call type %q, got %#v", CallTypeCheckIn, payload.NextCallRecommendation)
	}
	if payload.CheckIn.OrientationStatus != OrientationStatusUnknown {
		t.Fatalf("expected orientation %q, got %q", OrientationStatusUnknown, payload.CheckIn.OrientationStatus)
	}
	if payload.CheckIn.MealsStatus != CheckInCaptureUncertain {
		t.Fatalf("expected meals status %q, got %q", CheckInCaptureUncertain, payload.CheckIn.MealsStatus)
	}
	if payload.CheckIn.FluidsStatus != CheckInCaptureReported {
		t.Fatalf("expected fluids status %q, got %q", CheckInCaptureReported, payload.CheckIn.FluidsStatus)
	}
	if payload.CheckIn.SocialContact != SocialContactUnknown {
		t.Fatalf("expected social contact %q, got %q", SocialContactUnknown, payload.CheckIn.SocialContact)
	}
	if payload.CheckIn.Mood != CheckInMoodCalm {
		t.Fatalf("expected mood %q, got %q", CheckInMoodCalm, payload.CheckIn.Mood)
	}
	if payload.CheckIn.Sleep != SleepStatusUnknown {
		t.Fatalf("expected sleep %q, got %q", SleepStatusUnknown, payload.CheckIn.Sleep)
	}
	if payload.CheckIn.RemindersNoted[0].Title != "Follow up" || payload.CheckIn.RemindersNoted[0].Detail != "tomorrow" {
		t.Fatalf("expected trimmed reminder note, got %#v", payload.CheckIn.RemindersNoted[0])
	}
}

func TestValidateAnalysisPayloadAllowsUnknownPartialCheckInFields(t *testing.T) {
	t.Parallel()

	payload := AnalysisPayload{
		Summary:         "The call ended early before every section was covered.",
		EscalationLevel: EscalationNone,
		FollowUpIntent: FollowUpIntent{
			TimeframeBucket: TimeframeUnspecified,
			Confidence:      0.2,
		},
		CheckIn: &CheckInAnalysis{
			OrientationStatus:         OrientationStatusUnknown,
			MealsStatus:               CheckInCaptureUncertain,
			FluidsStatus:              CheckInCaptureUncertain,
			SocialContact:             SocialContactUnknown,
			RemindersNoted:            []ReminderNote{},
			ReminderDeclined:          false,
			Mood:                      CheckInMoodUnknown,
			Sleep:                     SleepStatusUnknown,
			MemoryFlags:               []string{},
			DeliriumWatch:             false,
			DeliriumPotentialTriggers: []string{},
		},
	}

	if err := validateAnalysisPayload(CallTypeCheckIn, payload); err != nil {
		t.Fatalf("expected payload to validate, got %v", err)
	}
}
