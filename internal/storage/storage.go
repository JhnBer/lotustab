package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"lotustab/internal/core/connection"
)

const (
	appDirName   = "lotustab"
	profilesFile = "connections.json"
)

type filePayload struct {
	Profiles []connection.Profile `json:"profiles"`
}

// Store persists connection profiles as JSON in the user config directory.
type Store struct {
	path string
	mu   sync.Mutex
}

// New creates a store rooted at os.UserConfigDir()/lotustab.
func New() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	appDir := filepath.Join(dir, appDirName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return &Store{path: filepath.Join(appDir, profilesFile)}, nil
}

// List returns all saved profiles.
func (s *Store) List() ([]connection.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return nil, err
	}
	if payload.Profiles == nil {
		return []connection.Profile{}, nil
	}
	out := make([]connection.Profile, 0, len(payload.Profiles))
	for _, p := range payload.Profiles {
		out = append(out, p.Clone())
	}
	return out, nil
}

// Get returns a profile by ID.
func (s *Store) Get(id string) (connection.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return connection.Profile{}, err
	}
	for _, p := range payload.Profiles {
		if p.ID == id {
			return p.Clone(), nil
		}
	}
	return connection.Profile{}, fmt.Errorf("profile not found: %s", id)
}

// Upsert creates or replaces a profile.
func (s *Store) Upsert(profile connection.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	found := false
	for i, p := range payload.Profiles {
		if p.ID == profile.ID {
			payload.Profiles[i] = profile.Clone()
			found = true
			break
		}
	}
	if !found {
		payload.Profiles = append(payload.Profiles, profile.Clone())
	}
	return s.write(payload)
}

// Delete removes a profile by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	next := payload.Profiles[:0]
	for _, p := range payload.Profiles {
		if p.ID != id {
			next = append(next, p)
		}
	}
	payload.Profiles = next
	return s.write(payload)
}

func (s *Store) read() (filePayload, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return filePayload{Profiles: []connection.Profile{}}, nil
		}
		return filePayload{}, fmt.Errorf("read profiles: %w", err)
	}
	var payload filePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return filePayload{}, fmt.Errorf("parse profiles: %w", err)
	}
	if payload.Profiles == nil {
		payload.Profiles = []connection.Profile{}
	}
	return payload, nil
}

func (s *Store) write(payload filePayload) error {
	if payload.Profiles == nil {
		payload.Profiles = []connection.Profile{}
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profiles: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write profiles: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace profiles: %w", err)
	}
	return nil
}
