package admin

import (
	"context"
	"fmt"
	"log"
	"time"
)

type AnalysisRunner interface {
	Analyze(ctx context.Context, promptContext AnalysisPromptContext) (AnalysisPayload, error)
}

type analysisWorkerStore interface {
	ClaimNextAnalysisJob(ctx context.Context, now time.Time) (AnalysisJob, bool, error)
	GetAnalysisPromptContext(ctx context.Context, callRunID string) (AnalysisPromptContext, error)
	SaveAnalysisResult(ctx context.Context, input SaveAnalysisResultInput) (AnalysisRecord, error)
	MarkAnalysisJobFailed(ctx context.Context, jobID, lastError string, now time.Time) error
}

type AnalysisWorker struct {
	store    analysisWorkerStore
	analyzer AnalysisRunner
	now      func() time.Time
}

func NewAnalysisWorker(store analysisWorkerStore, analyzer AnalysisRunner) *AnalysisWorker {
	return &AnalysisWorker{
		store:    store,
		analyzer: analyzer,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (w *AnalysisWorker) Run(ctx context.Context, pollInterval time.Duration) {
	if w == nil || w.analyzer == nil {
		return
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := w.drain(ctx); err != nil && ctx.Err() == nil {
			log.Printf("analysis worker iteration failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *AnalysisWorker) drain(ctx context.Context) error {
	for {
		processed, err := w.ProcessOnce(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
}

func (w *AnalysisWorker) ProcessOnce(ctx context.Context) (bool, error) {
	job, ok, err := w.store.ClaimNextAnalysisJob(ctx, w.now())
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	promptContext, err := w.store.GetAnalysisPromptContext(ctx, job.CallRunID)
	if err != nil {
		_ = w.store.MarkAnalysisJobFailed(ctx, job.ID, err.Error(), w.now())
		return true, nil
	}

	payload, err := w.analyzer.Analyze(ctx, promptContext)
	if err != nil {
		_ = w.store.MarkAnalysisJobFailed(ctx, job.ID, err.Error(), w.now())
		return true, nil
	}

	if err := validateAnalysisPayload(promptContext.CallRun.CallType, payload); err != nil {
		_ = w.store.MarkAnalysisJobFailed(ctx, job.ID, err.Error(), w.now())
		return true, nil
	}

	if _, err := w.store.SaveAnalysisResult(ctx, SaveAnalysisResultInput{
		CallRunID:             promptContext.CallRun.ID,
		PatientID:             promptContext.Patient.ID,
		PatientTimezone:       promptContext.Patient.Timezone,
		CallTemplateID:        promptContext.CallTemplate.ID,
		CallType:              promptContext.CallRun.CallType,
		CallPromptVersion:     promptContext.CallTemplate.CallPromptVersion,
		AnalysisPromptVersion: job.AnalysisPromptVersion,
		SchemaVersion:         job.SchemaVersion,
		ModelProvider:         job.ModelProvider,
		ModelName:             job.ModelName,
		Result:                payload,
		GeneratedAt:           w.now(),
	}); err != nil {
		_ = w.store.MarkAnalysisJobFailed(ctx, job.ID, err.Error(), w.now())
		return true, nil
	}

	return true, nil
}

type screeningSchedulerStore interface {
	ListDueScreeningSchedules(ctx context.Context, now time.Time, limit int) ([]ScreeningSchedule, error)
	CreateScheduledScreeningCallRun(ctx context.Context, schedule ScreeningSchedule, now time.Time) (CallRun, bool, error)
}

type ScreeningScheduler struct {
	store screeningSchedulerStore
	now   func() time.Time
}

func NewScreeningScheduler(store screeningSchedulerStore) *ScreeningScheduler {
	return &ScreeningScheduler{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (s *ScreeningScheduler) Run(ctx context.Context, pollInterval time.Duration) {
	if s == nil {
		return
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := s.ProcessOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("screening scheduler iteration failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *ScreeningScheduler) ProcessOnce(ctx context.Context) error {
	schedules, err := s.store.ListDueScreeningSchedules(ctx, s.now(), 25)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if _, _, err := s.store.CreateScheduledScreeningCallRun(ctx, schedule, s.now()); err != nil {
			return fmt.Errorf("create scheduled screening call for patient %s: %w", schedule.PatientID, err)
		}
	}

	return nil
}
