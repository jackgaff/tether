package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"nova-echoes/api/internal/modules/voice"
)

type fakeServiceStore struct {
	patient            Patient
	patientFound       bool
	consent            ConsentState
	consentFound       bool
	template           CallTemplate
	templateFound      bool
	createdCallRun     CallRun
	markFailedCallRun  string
	markFailedReason   string
	markFailedEndedAt  time.Time
	activeNextCallPlan NextCallPlan
	activePlanFound    bool
}

func (f *fakeServiceStore) GetPatient(_ context.Context, _ string) (Patient, bool, error) {
	return f.patient, f.patientFound, nil
}

func (f *fakeServiceStore) GetConsentState(_ context.Context, _ string) (ConsentState, bool, error) {
	return f.consent, f.consentFound, nil
}

func (f *fakeServiceStore) GetCallTemplateByID(_ context.Context, _ string) (CallTemplate, bool, error) {
	return f.template, f.templateFound, nil
}

func (f *fakeServiceStore) ResolveActiveCallTemplateByType(_ context.Context, _ string) (CallTemplate, error) {
	return f.template, nil
}

func (f *fakeServiceStore) CreateCallRun(_ context.Context, _ CreateCallRunParams) (CallRun, error) {
	return f.createdCallRun, nil
}

func (f *fakeServiceStore) MarkCallRunFailed(_ context.Context, callRunID, stopReason string, endedAt time.Time) error {
	f.markFailedCallRun = callRunID
	f.markFailedReason = stopReason
	f.markFailedEndedAt = endedAt
	return nil
}

func (f *fakeServiceStore) GetActiveNextCallPlan(_ context.Context, _ string) (NextCallPlan, bool, error) {
	return f.activeNextCallPlan, f.activePlanFound, nil
}

func (f *fakeServiceStore) GetAnalysisRecord(_ context.Context, _ string) (AnalysisRecord, bool, error) {
	return AnalysisRecord{}, false, nil
}

func (f *fakeServiceStore) GetAnalysisPromptContext(_ context.Context, _ string) (AnalysisPromptContext, error) {
	return AnalysisPromptContext{}, nil
}

func (f *fakeServiceStore) SaveAnalysisResult(_ context.Context, _ SaveAnalysisResultInput) (AnalysisRecord, error) {
	return AnalysisRecord{}, nil
}

func (f *fakeServiceStore) UpdateNextCallPlan(_ context.Context, _ string, _ UpdateNextCallPlanStoreInput) (NextCallPlan, error) {
	return NextCallPlan{}, nil
}

type failingVoiceCreator struct {
	err error
}

func (failingVoiceCreator) CreateSession(_ context.Context, _ voice.CreateSessionRequest) (voice.SessionDescriptor, error) {
	return voice.SessionDescriptor{}, errors.New("bootstrap failed")
}

func TestServiceCreateCallMarksBootstrapFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 12, 15, 4, 5, 0, time.UTC)
	store := &fakeServiceStore{
		patient: Patient{
			ID:                 "patient-1",
			PrimaryCaregiverID: "caregiver-1",
			CallingState:       CallingStateActive,
		},
		patientFound: true,
		consent: ConsentState{
			PatientID:               "patient-1",
			OutboundCallStatus:      ConsentStatusGranted,
			TranscriptStorageStatus: ConsentStatusGranted,
		},
		consentFound:  true,
		template:      CallTemplate{ID: "tmpl-orientation-v1", CallType: CallTypeOrientation, SystemPromptTemplate: "prompt", IsActive: true},
		templateFound: true,
		createdCallRun: CallRun{
			ID: "call-run-1",
		},
	}

	service := NewService(store, failingVoiceCreator{}, nil, "analysis-model")
	service.now = func() time.Time { return now }

	_, err := service.CreateCall(context.Background(), "patient-1", CreateCallRequest{
		CallTemplateID: "tmpl-orientation-v1",
		Channel:        CallChannelBrowser,
	})
	if err == nil {
		t.Fatal("expected browser bootstrap to fail")
	}
	if store.markFailedCallRun != "call-run-1" {
		t.Fatalf("expected failed bootstrap to mark call-run-1 failed, got %q", store.markFailedCallRun)
	}
	if store.markFailedReason != "voice_session_bootstrap_failed" {
		t.Fatalf("expected bootstrap failure reason to be recorded, got %q", store.markFailedReason)
	}
	if !store.markFailedEndedAt.Equal(now) {
		t.Fatalf("expected failure timestamp %s, got %s", now, store.markFailedEndedAt)
	}
}

func TestServiceCreateCallRejectsUnsupportedChannel(t *testing.T) {
	t.Parallel()

	store := &fakeServiceStore{
		patient: Patient{
			ID:                 "patient-1",
			PrimaryCaregiverID: "caregiver-1",
			CallingState:       CallingStateActive,
		},
		patientFound: true,
		consent: ConsentState{
			PatientID:               "patient-1",
			OutboundCallStatus:      ConsentStatusGranted,
			TranscriptStorageStatus: ConsentStatusGranted,
		},
		consentFound:  true,
		template:      CallTemplate{ID: "tmpl-orientation-v1", CallType: CallTypeOrientation, SystemPromptTemplate: "prompt", IsActive: true},
		templateFound: true,
	}

	service := NewService(store, failingVoiceCreator{}, nil, "analysis-model")
	_, err := service.CreateCall(context.Background(), "patient-1", CreateCallRequest{
		CallTemplateID: "tmpl-orientation-v1",
		Channel:        "satellite",
	})
	if err == nil {
		t.Fatal("expected unsupported channel to fail")
	}
	if !isValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

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
			DurationMinutes: 31,
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
