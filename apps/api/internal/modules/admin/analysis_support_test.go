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
			MentionedPeople: []MentionedPerson{
				{Name: " Pat ", Relationship: " friend ", Context: " talked recently "},
			},
			Mood:  "neutral",
			Sleep: "N/A",
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
	if payload.CheckIn.MentionedPeople[0].Name != "Pat" || payload.CheckIn.MentionedPeople[0].Relationship != "friend" || payload.CheckIn.MentionedPeople[0].Context != "talked recently" {
		t.Fatalf("expected trimmed mentioned person, got %#v", payload.CheckIn.MentionedPeople[0])
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

func TestNormalizeAnalysisPayloadDedupesDurableLists(t *testing.T) {
	t.Parallel()

	payload := AnalysisPayload{
		Summary: "  Recap  ",
		FollowUpIntent: FollowUpIntent{
			TimeframeBucket: TimeframeUnspecified,
		},
		CheckIn: &CheckInAnalysis{
			MemoryFlags: []string{" repeat question ", "repeat question", ""},
			MentionedPeople: []MentionedPerson{
				{Name: " Pat ", Relationship: "friend", Context: "school"},
				{Name: "pat", Relationship: "friend", Context: "school"},
				{Name: " "},
			},
			RemindersNoted: []ReminderNote{
				{Title: " Call Pat ", Detail: " tomorrow "},
				{Title: "call pat", Detail: "tomorrow"},
				{Title: "", Detail: ""},
			},
			DeliriumPotentialTriggers: []string{" dehydration ", "dehydration"},
		},
		Reminiscence: &ReminiscenceAnalysis{
			RespondedWellTo: []string{"music", " music "},
		},
	}

	normalizeAnalysisPayload(&payload)

	if payload.Summary != "Recap" {
		t.Fatalf("expected trimmed summary, got %q", payload.Summary)
	}
	if len(payload.CheckIn.MemoryFlags) != 1 || payload.CheckIn.MemoryFlags[0] != "repeat question" {
		t.Fatalf("expected deduped memory flags, got %#v", payload.CheckIn.MemoryFlags)
	}
	if len(payload.CheckIn.MentionedPeople) != 1 || payload.CheckIn.MentionedPeople[0].Name != "Pat" {
		t.Fatalf("expected deduped/trimmed people, got %#v", payload.CheckIn.MentionedPeople)
	}
	if len(payload.CheckIn.RemindersNoted) != 1 || payload.CheckIn.RemindersNoted[0].Title != "Call Pat" {
		t.Fatalf("expected deduped reminders, got %#v", payload.CheckIn.RemindersNoted)
	}
	if len(payload.CheckIn.DeliriumPotentialTriggers) != 1 || payload.CheckIn.DeliriumPotentialTriggers[0] != "dehydration" {
		t.Fatalf("expected deduped triggers, got %#v", payload.CheckIn.DeliriumPotentialTriggers)
	}
	if len(payload.Reminiscence.RespondedWellTo) != 1 || payload.Reminiscence.RespondedWellTo[0] != "music" {
		t.Fatalf("expected deduped respondedWellTo, got %#v", payload.Reminiscence.RespondedWellTo)
	}
}

func TestValidateAnalysisPayloadRejectsReminiscenceAnchorContradictions(t *testing.T) {
	t.Parallel()

	payload := AnalysisPayload{
		Summary:         "Memory call summary.",
		EscalationLevel: EscalationNone,
		FollowUpIntent: FollowUpIntent{
			TimeframeBucket: TimeframeUnspecified,
			Confidence:      0.3,
		},
		Reminiscence: &ReminiscenceAnalysis{
			AnchorOffered:  false,
			AnchorAccepted: true,
			AnchorType:     AnchorTypeNone,
		},
	}

	if err := validateAnalysisPayload(CallTypeReminiscence, payload); err == nil {
		t.Fatal("expected reminiscence anchor contradiction to fail validation")
	}
}

func TestValidateAnalysisPayloadRequiresSupportForUrgentRisk(t *testing.T) {
	t.Parallel()

	payload := AnalysisPayload{
		Summary:         "Escalation needed.",
		EscalationLevel: EscalationCaregiverNow,
		FollowUpIntent: FollowUpIntent{
			TimeframeBucket: TimeframeSameDay,
			Confidence:      0.8,
		},
		RiskFlags: []AnalysisRiskFlag{
			{
				FlagType:   "acute_confusion",
				Severity:   RiskSeverityUrgent,
				Confidence: 0.8,
				Evidence:   "",
				Reason:     "",
			},
		},
		CheckIn: &CheckInAnalysis{
			OrientationStatus:         OrientationStatusMildlyConfused,
			MealsStatus:               CheckInCaptureUncertain,
			FluidsStatus:              CheckInCaptureUncertain,
			SocialContact:             SocialContactUnknown,
			RemindersNoted:            []ReminderNote{},
			ReminderDeclined:          false,
			Mood:                      CheckInMoodUnknown,
			Sleep:                     SleepStatusUnknown,
			MemoryFlags:               []string{},
			DeliriumWatch:             true,
			DeliriumPotentialTriggers: []string{},
		},
	}

	if err := validateAnalysisPayload(CallTypeCheckIn, payload); err == nil {
		t.Fatal("expected urgent risk without evidence/reason to fail validation")
	}
}
