package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sync/singleflight"

	"lotustab/internal/core/connection"
	"lotustab/internal/core/driver"
	"lotustab/internal/secrets"
	"lotustab/internal/storage"
	"lotustab/internal/tunnel"
)

const connectionsChangedEvent = "connections:changed"

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
	ops      *connection.OpRegistry
	flight   singleflight.Group
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
		ops:      connection.NewOpRegistry(),
	}
}

func (s *ConnectionsService) ServiceShutdown() error {
	s.ops.CancelEvery()
	s.manager.CloseAll()
	return nil
}

// Track registers an in-flight operation for a profile so Disconnect can cancel it.
func (s *ConnectionsService) Track(ctx context.Context, profileID string) (context.Context, func()) {
	if s == nil {
		return ctx, func() {}
	}
	return s.ops.Track(ctx, profileID)
}

func (s *ConnectionsService) notifyChanged() {
	if appl := application.Get(); appl != nil {
		appl.Event.Emit(connectionsChangedEvent)
	}
}

func (s *ConnectionsService) afterChange(err error) error {
	if err == nil {
		s.notifyChanged()
	}
	return err
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
	d, profile, secretVals, err := s.prepare(input, true)
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
	for _, kind := range secrets.Kinds {
		if err := s.secrets.Set(profile.ID, kind, secretVals[kind]); err != nil {
			return connection.Profile{}, err
		}
	}
	s.notifyChanged()
	return profile, nil
}

// Delete removes a profile, its secret, and any open session.
func (s *ConnectionsService) Delete(id string) error {
	s.ops.CancelAll(id)
	if err := s.manager.Remove(id); err != nil {
		return err
	}
	if err := s.secrets.Delete(id); err != nil {
		return err
	}
	return s.afterChange(s.store.Delete(id))
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
	ctx, done := s.ops.Track(ctx, id)
	defer done()
	if _, err := s.ensureOpen(ctx, id); err != nil {
		return err
	}
	_ = s.store.TouchRecent(id)
	s.notifyChanged()
	return nil
}

// Disconnect cancels in-flight operations and closes the live session.
func (s *ConnectionsService) Disconnect(id string) error {
	s.ops.CancelAll(id)
	return s.afterChange(s.manager.Remove(id))
}

// ConnectedIDs returns profile IDs with an open session.
func (s *ConnectionsService) ConnectedIDs() []string {
	return s.manager.IDs()
}

// SetPinned pins or unpins a profile in the sidebar rail.
func (s *ConnectionsService) SetPinned(id string, pinned bool) error {
	return s.afterChange(s.store.SetPinned(id, pinned))
}

// SetColor updates a profile's palette color.
func (s *ConnectionsService) SetColor(id, color string) error {
	color, err := resolveColor(color, "")
	if err != nil {
		return err
	}
	return s.afterChange(s.store.SetColor(id, color))
}

// TouchRecent records a profile as recently used.
func (s *ConnectionsService) TouchRecent(id string) error {
	return s.afterChange(s.store.TouchRecent(id))
}

// Databases lists databases on the server for a connected (or just-opened) profile.
func (s *ConnectionsService) Databases(ctx context.Context, id string) ([]string, error) {
	ctx, done := s.ops.Track(ctx, id)
	defer done()
	conn, err := s.ensureOpen(ctx, id)
	if err != nil {
		return nil, err
	}
	return conn.Databases(ctx)
}

func (s *ConnectionsService) ensureOpen(ctx context.Context, id string) (driver.Conn, error) {
	if conn, ok := s.manager.Get(id); ok {
		if err := pingConn(ctx, conn); err == nil {
			return conn, nil
		}
		_ = s.manager.Remove(id)
	}
	return s.openOnce(ctx, id)
}

func (s *ConnectionsService) reconnect(ctx context.Context, id string) (driver.Conn, error) {
	_ = s.manager.Remove(id)
	return s.openOnce(ctx, id)
}

func (s *ConnectionsService) openOnce(ctx context.Context, id string) (driver.Conn, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		v, err, shared := s.flight.Do(id, func() (any, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if conn, ok := s.manager.Get(id); ok {
				if err := pingConn(ctx, conn); err == nil {
					return conn, nil
				}
				_ = s.manager.Remove(id)
			}
			conn, err := s.openSaved(ctx, id)
			if err != nil {
				s.notifyChanged()
				return nil, err
			}
			s.manager.Put(id, conn)
			s.notifyChanged()
			return conn, nil
		})
		if err == nil {
			return v.(driver.Conn), nil
		}
		if shared && ctx.Err() == nil && errors.Is(err, context.Canceled) {
			continue
		}
		return nil, err
	}
}

func pingConn(ctx context.Context, conn driver.Conn) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return conn.Ping(pingCtx)
}

func (s *ConnectionsService) openFromInput(ctx context.Context, input SaveProfileInput) (driver.Conn, error) {
	d, profile, secretVals, err := s.prepare(input, false)
	if err != nil {
		return nil, err
	}
	secretVals, err = s.fillSecrets(profile.ID, secretVals)
	if err != nil {
		return nil, err
	}
	return s.open(ctx, d, profile, secretVals)
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
	secretVals, err := s.fillSecrets(id, nil)
	if err != nil {
		return nil, err
	}
	return s.open(ctx, d, profile, secretVals)
}

func (s *ConnectionsService) fillSecrets(profileID string, got map[string]string) (map[string]string, error) {
	if got == nil {
		got = map[string]string{}
	}
	if profileID == "" || s.secrets == nil {
		return got, nil
	}
	for _, kind := range secrets.Kinds {
		if strings.TrimSpace(got[kind]) != "" {
			continue
		}
		v, err := s.secrets.Get(profileID, kind)
		if err != nil {
			return nil, err
		}
		got[kind] = v
	}
	return got, nil
}

func (s *ConnectionsService) open(ctx context.Context, d driver.Driver, profile connection.Profile, secretVals map[string]string) (driver.Conn, error) {
	cfg := driver.ConnConfig{
		Params:   profile.Params,
		Password: secretVals[secrets.KindPassword],
	}
	sshCfg, enabled, err := tunnel.ConfigFromParams(profile.Params, secretVals[secrets.KindSSHPassword], secretVals[secrets.KindSSHPassphrase])
	if err != nil {
		return nil, err
	}
	if !enabled {
		return d.Open(ctx, cfg)
	}
	jumpCfg, jumpOn, err := tunnel.JumpConfigFromParams(profile.Params, secretVals[secrets.KindSSHJumpPassword], secretVals[secrets.KindSSHJumpPassphrase])
	if err != nil {
		return nil, err
	}
	var client *tunnel.Client
	if jumpOn {
		jump, err := tunnel.Open(ctx, jumpCfg)
		if err != nil {
			return nil, err
		}
		client, err = tunnel.OpenVia(ctx, jump, sshCfg)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = tunnel.Open(ctx, sshCfg)
		if err != nil {
			return nil, err
		}
	}
	cfg.Dial = client.Dial
	conn, err := d.Open(ctx, cfg)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return &chainedConn{Conn: conn, extra: client}, nil
}

type chainedConn struct {
	driver.Conn
	extra io.Closer
}

func (c *chainedConn) Close() error {
	err := c.Conn.Close()
	if c.extra != nil {
		if e := c.extra.Close(); err == nil {
			err = e
		}
	}
	return err
}

func (s *ConnectionsService) prepare(input SaveProfileInput, requireName bool) (driver.Driver, connection.Profile, map[string]string, error) {
	name := strings.TrimSpace(input.Name)
	if requireName && name == "" {
		return nil, connection.Profile{}, nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(input.DriverID) == "" {
		return nil, connection.Profile{}, nil, fmt.Errorf("driver is required")
	}
	d, err := s.registry.Get(input.DriverID)
	if err != nil {
		return nil, connection.Profile{}, nil, err
	}

	params := map[string]string{}
	for k, v := range input.Params {
		params[k] = v
	}
	secretVals := map[string]string{}
	for _, kind := range secrets.Kinds {
		secretVals[kind] = params[kind]
		delete(params, kind)
	}

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
			return nil, connection.Profile{}, nil, fmt.Errorf("%s is required", field.Label)
		}
	}

	sshCfg, sshOn, err := tunnel.ConfigFromParams(params, secretVals[secrets.KindSSHPassword], secretVals[secrets.KindSSHPassphrase])
	if err != nil {
		return nil, connection.Profile{}, nil, err
	}
	if sshOn && sshCfg.Auth == tunnel.AuthPassword && strings.TrimSpace(input.ID) == "" && strings.TrimSpace(secretVals[secrets.KindSSHPassword]) == "" {
		return nil, connection.Profile{}, nil, fmt.Errorf("SSH password is required")
	}
	jumpCfg, jumpOn, err := tunnel.JumpConfigFromParams(params, secretVals[secrets.KindSSHJumpPassword], secretVals[secrets.KindSSHJumpPassphrase])
	if err != nil {
		return nil, connection.Profile{}, nil, err
	}
	if jumpOn && jumpCfg.Auth == tunnel.AuthPassword && strings.TrimSpace(input.ID) == "" && strings.TrimSpace(secretVals[secrets.KindSSHJumpPassword]) == "" {
		return nil, connection.Profile{}, nil, fmt.Errorf("SSH jump password is required")
	}

	profile := connection.Profile{
		ID:       strings.TrimSpace(input.ID),
		Name:     name,
		DriverID: d.ID(),
		Params:   params,
	}
	return d, profile, secretVals, nil
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
