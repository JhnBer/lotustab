package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
)

// SaveProfileInput is the payload used to create, update, or test a profile.
type SaveProfileInput struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	DriverID string            `json:"driverId"`
	Params   map[string]string `json:"params"`
	Color    string            `json:"color"`
}

// ConnectionsState is the saved profile list plus navigation prefs.
type ConnectionsState struct {
	Profiles  []connection.Profile `json:"profiles"`
	PinnedIDs []string             `json:"pinnedIds"`
	RecentIDs []string             `json:"recentIds"`
}

// ConnectionsService exposes connection-profile and session APIs to the frontend.
type ConnectionsService struct {
	registry *driver.Registry
	store    *storage.Store
	secrets  *secrets.Store
	manager  *connection.Manager
}

// NewConnectionsService constructs the Wails connections service.
func NewConnectionsService(
	registry *driver.Registry,
	store *storage.Store,
	secretsStore *secrets.Store,
	manager *connection.Manager,
) *ConnectionsService {
	return &ConnectionsService{
		registry: registry,
		store:    store,
		secrets:  secretsStore,
		manager:  manager,
	}
}

func (s *ConnectionsService) ServiceShutdown() error {
	s.manager.CloseAll()
	return nil
}

// List returns saved profiles and navigation prefs (never includes passwords).
func (s *ConnectionsService) List() (ConnectionsState, error) {
	profiles, pinned, recent, err := s.store.ListState()
	if err != nil {
		return ConnectionsState{}, err
	}
	if profiles == nil {
		profiles = []connection.Profile{}
	}
	if pinned == nil {
		pinned = []string{}
	}
	if recent == nil {
		recent = []string{}
	}
	return ConnectionsState{Profiles: profiles, PinnedIDs: pinned, RecentIDs: recent}, nil
}

// Save creates or updates a profile. An empty password leaves the keychain value unchanged.
func (s *ConnectionsService) Save(input SaveProfileInput) (connection.Profile, error) {
	d, profile, password, err := s.prepare(input, true)
	if err != nil {
		return connection.Profile{}, err
	}
	_ = d

	now := time.Now().UTC()
	if profile.ID == "" {
		profile.ID = uuid.NewString()
		profile.CreatedAt = now
		color, err := resolveColor(input.Color, connection.DefaultColor)
		if err != nil {
			return connection.Profile{}, err
		}
		profile.Color = color
	} else {
		existing, getErr := s.store.Get(profile.ID)
		if getErr != nil {
			profile.CreatedAt = now
			color, err := resolveColor(input.Color, connection.DefaultColor)
			if err != nil {
				return connection.Profile{}, err
			}
			profile.Color = color
		} else {
			profile.CreatedAt = existing.CreatedAt
			fallback := existing.Color
			if fallback == "" {
				fallback = connection.DefaultColor
			}
			if strings.TrimSpace(input.Color) == "" {
				profile.Color = fallback
			} else {
				color, err := resolveColor(input.Color, fallback)
				if err != nil {
					return connection.Profile{}, err
				}
				profile.Color = color
			}
		}
	}
	profile.UpdatedAt = now

	if err := s.store.Upsert(profile); err != nil {
		return connection.Profile{}, err
	}
	if err := s.secrets.Set(profile.ID, password); err != nil {
		return connection.Profile{}, err
	}
	return profile, nil
}

// Delete removes a profile, its secret, and any open session.
func (s *ConnectionsService) Delete(id string) error {
	if err := s.manager.Remove(id); err != nil {
		return err
	}
	if err := s.secrets.Delete(id); err != nil {
		return err
	}
	return s.store.Delete(id)
}

// Test opens a throwaway connection using the form values.
func (s *ConnectionsService) Test(ctx context.Context, input SaveProfileInput) error {
	conn, err := s.openFromInput(ctx, input)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping(ctx)
}

// FetchDatabases opens a throwaway connection from the form values and lists databases.
func (s *ConnectionsService) FetchDatabases(ctx context.Context, input SaveProfileInput) ([]string, error) {
	conn, err := s.openFromInput(ctx, input)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Databases(ctx)
}

// Connect opens (or replaces) a live session for a saved profile.
func (s *ConnectionsService) Connect(ctx context.Context, id string) error {
	conn, err := s.openSaved(ctx, id)
	if err != nil {
		return err
	}
	s.manager.Put(id, conn)
	_ = s.store.TouchRecent(id)
	return nil
}

// Disconnect closes the live session for a profile.
func (s *ConnectionsService) Disconnect(id string) error {
	return s.manager.Remove(id)
}

// ConnectedIDs returns profile IDs with an open session.
func (s *ConnectionsService) ConnectedIDs() []string {
	return s.manager.IDs()
}

// SetPinned pins or unpins a profile in the sidebar rail.
func (s *ConnectionsService) SetPinned(id string, pinned bool) error {
	return s.store.SetPinned(id, pinned)
}

// SetColor updates a profile's palette color.
func (s *ConnectionsService) SetColor(id, color string) error {
	color, err := resolveColor(color, "")
	if err != nil {
		return err
	}
	return s.store.SetColor(id, color)
}

// TouchRecent records a profile as recently used.
func (s *ConnectionsService) TouchRecent(id string) error {
	return s.store.TouchRecent(id)
}

// Databases lists databases on the server for a connected (or just-opened) profile.
func (s *ConnectionsService) Databases(ctx context.Context, id string) ([]string, error) {
	conn, err := s.ensureOpen(ctx, id)
	if err != nil {
		return nil, err
	}
	return conn.Databases(ctx)
}

func (s *ConnectionsService) ensureOpen(ctx context.Context, id string) (driver.Conn, error) {
	if conn, ok := s.manager.Get(id); ok {
		return conn, nil
	}
	conn, err := s.openSaved(ctx, id)
	if err != nil {
		return nil, err
	}
	s.manager.Put(id, conn)
	return conn, nil
}

func (s *ConnectionsService) openFromInput(ctx context.Context, input SaveProfileInput) (driver.Conn, error) {
	d, profile, password, err := s.prepare(input, false)
	if err != nil {
		return nil, err
	}
	if password == "" && profile.ID != "" {
		password, err = s.secrets.Get(profile.ID)
		if err != nil {
			return nil, err
		}
	}
	return d.Open(ctx, driver.ConnConfig{Params: profile.Params, Password: password})
}

func (s *ConnectionsService) openSaved(ctx context.Context, id string) (driver.Conn, error) {
	profile, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	d, err := s.registry.Get(profile.DriverID)
	if err != nil {
		return nil, err
	}
	password, err := s.secrets.Get(id)
	if err != nil {
		return nil, err
	}
	return d.Open(ctx, driver.ConnConfig{Params: profile.Params, Password: password})
}

func (s *ConnectionsService) prepare(input SaveProfileInput, requireName bool) (driver.Driver, connection.Profile, string, error) {
	name := strings.TrimSpace(input.Name)
	if requireName && name == "" {
		return nil, connection.Profile{}, "", fmt.Errorf("name is required")
	}
	if strings.TrimSpace(input.DriverID) == "" {
		return nil, connection.Profile{}, "", fmt.Errorf("driver is required")
	}
	d, err := s.registry.Get(input.DriverID)
	if err != nil {
		return nil, connection.Profile{}, "", err
	}

	params := map[string]string{}
	for k, v := range input.Params {
		params[k] = v
	}
	password := params["password"]
	delete(params, "password")

	spec := d.ConnectionSpec()
	for _, field := range spec {
		if field.Secret {
			delete(params, field.Key)
			continue
		}
		if _, ok := params[field.Key]; !ok && field.Default != "" {
			params[field.Key] = field.Default
		}
		if field.Required && strings.TrimSpace(params[field.Key]) == "" {
			return nil, connection.Profile{}, "", fmt.Errorf("%s is required", field.Label)
		}
	}

	profile := connection.Profile{
		ID:       strings.TrimSpace(input.ID),
		Name:     name,
		DriverID: d.ID(),
		Params:   params,
	}
	return d, profile, password, nil
}

func resolveColor(color, fallback string) (string, error) {
	c := strings.TrimSpace(color)
	if c == "" {
		if connection.ValidColor(fallback) {
			return fallback, nil
		}
		return connection.DefaultColor, nil
	}
	if !connection.ValidColor(c) {
		return "", fmt.Errorf("invalid color")
	}
	return c, nil
}
