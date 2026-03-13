package admin

import "testing"

func TestValidateAnalysisPayloadReturnsValidationError(t *testing.T) {
	t.Parallel()

	err := validateAnalysisPayload(AnalysisPayload{
		CallTypeCompleted: "invalid",
		PatientState: AnalysisPatientState{
			Orientation: AnalysisOrientationGood,
			Mood:        AnalysisMoodNeutral,
			Engagement:  AnalysisEngagementMedium,
			Confidence:  0.8,
		},
		RecommendedNextCall: RecommendedNextCall{
			Type:            CallTypeReminder,
			Timing:          "Tonight",
			DurationMinutes: 4,
			Goal:            "Repeat one routine cue.",
		},
		EscalationLevel: EscalationNone,
	})
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if !isValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}
