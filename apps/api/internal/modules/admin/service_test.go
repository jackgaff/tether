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
	callRun            CallRun
	callRunFound       bool
	analysisJob        AnalysisJob
	analysisJobFound   bool
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

func (f *fakeServiceStore) GetCallRun(_ context.Context, _ string) (CallRun, bool, error) {
	return f.callRun, f.callRunFound, nil
}

func (f *fakeServiceStore) GetAnalysisJob(_ context.Context, _ string) (AnalysisJob, bool, error) {
	return f.analysisJob, f.analysisJobFound, nil
}

func (f *fakeServiceStore) UpsertAnalysisJob(_ context.Context, input UpsertAnalysisJobParams) (AnalysisJob, error) {
	return AnalysisJob{
		ID:                    "job-1",
		CallRunID:             input.CallRunID,
		Status:                AnalysisJobStatusPending,
		AnalysisPromptVersion: input.AnalysisPromptVersion,
		SchemaVersion:         input.AnalysisSchemaVersion,
		ModelProvider:         input.ModelProvider,
		ModelName:             input.ModelName,
	}, nil
}

func (f *fakeServiceStore) UpdateNextCallPlan(_ context.Context, _ string, _ UpdateNextCallPlanStoreInput) (NextCallPlan, error) {
	return NextCallPlan{}, nil
}

type failingVoiceCreator struct{}

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
		template:      CallTemplate{ID: "tmpl-check-in-v1", CallType: CallTypeCheckIn, SystemPromptTemplate: "prompt", IsActive: true},
		templateFound: true,
		createdCallRun: CallRun{
			ID: "call-run-1",
		},
	}

	service := NewService(store, failingVoiceCreator{}, "analysis-model")
	service.now = func() time.Time { return now }

	_, err := service.CreateCall(context.Background(), "patient-1", CreateCallRequest{
		CallTemplateID: "tmpl-check-in-v1",
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
		template:      CallTemplate{ID: "tmpl-check-in-v1", CallType: CallTypeCheckIn, SystemPromptTemplate: "prompt", IsActive: true},
		templateFound: true,
	}

	service := NewService(store, failingVoiceCreator{}, "analysis-model")
	_, err := service.CreateCall(context.Background(), "patient-1", CreateCallRequest{
		CallTemplateID: "tmpl-check-in-v1",
		Channel:        "satellite",
	})
	if err == nil {
		t.Fatal("expected unsupported channel to fail")
	}
	if !isValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestServiceEnqueueAnalysisRejectsIncompleteCall(t *testing.T) {
	t.Parallel()

	store := &fakeServiceStore{
		callRun: CallRun{
			ID:     "call-run-1",
			Status: CallRunStatusRequested,
		},
		callRunFound:  true,
		template:      CallTemplate{ID: "tmpl-check-in-v1", AnalysisPromptVersion: "v1"},
		templateFound: true,
	}

	service := NewService(store, failingVoiceCreator{}, "analysis-model")
	_, err := service.EnqueueAnalysis(context.Background(), "call-run-1", false)
	if !errors.Is(err, ErrCallRunNotCompleted) {
		t.Fatalf("expected ErrCallRunNotCompleted, got %v", err)
	}
}
