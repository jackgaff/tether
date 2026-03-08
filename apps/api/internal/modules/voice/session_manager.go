package voice

import (
	"sync"
)

type managedSession interface {
	Close() error
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]managedSession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]managedSession)}
}

func (m *SessionManager) Add(id string, session managedSession) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[id]; exists {
		return false
	}

	m.sessions[id] = session
	return true
}

func (m *SessionManager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, id)
}

func (m *SessionManager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]managedSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.sessions = make(map[string]managedSession)
	m.mu.Unlock()

	var firstErr error
	for _, session := range sessions {
		if err := session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
