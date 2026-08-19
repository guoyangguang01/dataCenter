package mqtt

import (
	"fmt"
	"sync"
	"time"
)

type Session struct {
	DeviceID      string
	ClientID      string
	ConnectedAt   time.Time
	LastSeen      time.Time
	Subscriptions []string
	CleanSession  bool
	KeepAlive     uint16
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (m *SessionManager) Get(clientID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[clientID]
	return s, ok
}

func (m *SessionManager) Set(clientID string, session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[clientID] = session
	fmt.Println("[MQTT] session created for ", clientID)
}

func (m *SessionManager) Remove(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, clientID)
	fmt.Println("[MQTT] session removed for ", clientID)
}

func (m *SessionManager) UpdateLastSeen(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[clientID]; ok {
		s.LastSeen = time.Now()
	}
}

func (m *SessionManager) IsOnline(clientID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[clientID]
	if !ok {
		return false
	}
	return time.Since(s.LastSeen) < 3*time.Minute
}
