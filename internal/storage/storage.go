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
	recentCap    = 8
)

type filePayload struct {
	Profiles  []connection.Profile `json:"profiles"`
	PinnedIDs []string             `json:"pinnedIds"`
	RecentIDs []string             `json:"recentIds"`
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
	return NewAt(filepath.Join(appDir, profilesFile)), nil
}

// NewAt creates a store that reads and writes the given JSON file.
func NewAt(path string) *Store {
	return &Store{path: path}
}

// List returns all saved profiles.
func (s *Store) List() ([]connection.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return nil, err
	}
	out := make([]connection.Profile, 0, len(payload.Profiles))
	for _, p := range payload.Profiles {
		out = append(out, p.Clone())
	}
	return out, nil
}

// ListState returns profiles together with pin and recent order.
func (s *Store) ListState() ([]connection.Profile, []string, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return nil, nil, nil, err
	}
	profiles := make([]connection.Profile, 0, len(payload.Profiles))
	for _, p := range payload.Profiles {
		profiles = append(profiles, p.Clone())
	}
	return profiles, cloneIDs(payload.PinnedIDs), cloneIDs(payload.RecentIDs), nil
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

// Delete removes a profile by ID and drops it from pin/recent lists.
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
	payload.PinnedIDs = removeID(payload.PinnedIDs, id)
	payload.RecentIDs = removeID(payload.RecentIDs, id)
	return s.write(payload)
}

// SetPinned adds or removes a profile from the pin list.
func (s *Store) SetPinned(id string, pinned bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	if !hasProfile(payload.Profiles, id) {
		return fmt.Errorf("profile not found: %s", id)
	}
	payload.PinnedIDs = removeID(payload.PinnedIDs, id)
	if pinned {
		payload.PinnedIDs = append(payload.PinnedIDs, id)
	}
	return s.write(payload)
}

// SetColor updates a profile's palette color.
func (s *Store) SetColor(id, color string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	for i, p := range payload.Profiles {
		if p.ID == id {
			payload.Profiles[i].Color = color
			return s.write(payload)
		}
	}
	return fmt.Errorf("profile not found: %s", id)
}

// TouchRecent records a profile as most-recently used.
func (s *Store) TouchRecent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.read()
	if err != nil {
		return err
	}
	if !hasProfile(payload.Profiles, id) {
		return fmt.Errorf("profile not found: %s", id)
	}
	next := []string{id}
	for _, existing := range payload.RecentIDs {
		if existing == id {
			continue
		}
		next = append(next, existing)
		if len(next) >= recentCap {
			break
		}
	}
	payload.RecentIDs = next
	return s.write(payload)
}

func hasProfile(profiles []connection.Profile, id string) bool {
	for _, p := range profiles {
		if p.ID == id {
			return true
		}
	}
	return false
}

func removeID(ids []string, id string) []string {
	out := make([]string, 0, len(ids))
	for _, existing := range ids {
		if existing != id {
			out = append(out, existing)
		}
	}
	return out
}

func cloneIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

func (s *Store) read() (filePayload, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return filePayload{
				Profiles:  []connection.Profile{},
				PinnedIDs: []string{},
				RecentIDs: []string{},
			}, nil
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
	if payload.PinnedIDs == nil {
		payload.PinnedIDs = []string{}
	}
	if payload.RecentIDs == nil {
		payload.RecentIDs = []string{}
	}
	return payload, nil
}

func (s *Store) write(payload filePayload) error {
	if payload.Profiles == nil {
		payload.Profiles = []connection.Profile{}
	}
	if payload.PinnedIDs == nil {
		payload.PinnedIDs = []string{}
	}
	if payload.RecentIDs == nil {
		payload.RecentIDs = []string{}
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
