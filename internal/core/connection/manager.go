package connection

import (
	"fmt"
	"sync"

	"lotustab/internal/core/driver"
)

// Manager holds the currently open database sessions, keyed by profile ID.
type Manager struct {
	mu    sync.RWMutex
	conns map[string]driver.Conn
}

// NewManager returns an empty connection manager.
func NewManager() *Manager {
	return &Manager{conns: make(map[string]driver.Conn)}
}

// Get returns the open connection for a profile, if any.
func (m *Manager) Get(profileID string) (driver.Conn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.conns[profileID]
	return c, ok
}

// Put stores a connection, closing any previous session for the same profile.
func (m *Manager) Put(profileID string, conn driver.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.conns[profileID]; ok && old != nil {
		_ = old.Close()
	}
	m.conns[profileID] = conn
}

// Remove closes and drops the session for a profile.
func (m *Manager) Remove(profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conns[profileID]
	if !ok {
		return nil
	}
	delete(m.conns, profileID)
	if c == nil {
		return nil
	}
	if err := c.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}

// IDs returns the profile IDs that currently have an open session.
func (m *Manager) IDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.conns))
	for id := range m.conns {
		out = append(out, id)
	}
	return out
}

// CloseAll closes every open session.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.conns {
		if c != nil {
			_ = c.Close()
		}
		delete(m.conns, id)
	}
}
