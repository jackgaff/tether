package checkins

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Store interface {
	List(ctx context.Context, patientID string) ([]CheckIn, error)
	Create(ctx context.Context, input CreateCheckInRequest) (CheckIn, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	items   []CheckIn
	counter atomic.Uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) List(_ context.Context, patientID string) ([]CheckIn, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]CheckIn, 0, len(s.items))

	for _, item := range s.items {
		if patientID != "" && item.PatientID != patientID {
			continue
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b CheckIn) int {
		switch {
		case a.RecordedAt.After(b.RecordedAt):
			return -1
		case a.RecordedAt.Before(b.RecordedAt):
			return 1
		default:
			return 0
		}
	})

	return items, nil
}

func (s *MemoryStore) Create(_ context.Context, input CreateCheckInRequest) (CheckIn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextID := s.counter.Add(1)
	checkIn := CheckIn{
		ID:         fmt.Sprintf("chk_%04d", nextID),
		PatientID:  strings.TrimSpace(input.PatientID),
		Summary:    strings.TrimSpace(input.Summary),
		Status:     input.Status,
		Agent:      strings.TrimSpace(input.Agent),
		Reminder:   strings.TrimSpace(input.Reminder),
		RecordedAt: time.Now().UTC(),
	}

	s.items = append(s.items, checkIn)

	return checkIn, nil
}
