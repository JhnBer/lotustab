package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"

	"lotustab/internal/secrets"
)

// AboutInfo is the static identity shown on Settings → About.
type AboutInfo struct {
	Version        string `json:"version"`
	Goos           string `json:"goos"`
	Goarch         string `json:"goarch"`
	UpdatesEnabled bool   `json:"updatesEnabled"`
	State          string `json:"state"`
	HasGitHubToken bool   `json:"hasGitHubToken"`
}

// UpdateCheckResult is the outcome of a GitHub Releases check.
type UpdateCheckResult struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Name      string `json:"name"`
	Notes     string `json:"notes"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
}

// AboutService exposes version info and the Wails updater to the frontend.
type AboutService struct {
	version        string
	updatesEnabled bool
	secrets        *secrets.Store
}

// NewAboutService constructs the Wails about/updater service.
func NewAboutService(version string, updatesEnabled bool, secretStore *secrets.Store) *AboutService {
	return &AboutService{version: version, updatesEnabled: updatesEnabled, secrets: secretStore}
}

// Info returns the running version, architecture, and updater state.
func (s *AboutService) Info() AboutInfo {
	info := AboutInfo{
		Version:        s.version,
		Goos:           runtime.GOOS,
		Goarch:         runtime.GOARCH,
		UpdatesEnabled: s.updatesEnabled,
		State:          string(updater.StateUnconfigured),
		HasGitHubToken: s.hasGitHubToken(),
	}
	if u, err := s.updater(); err == nil && u.State() != updater.StateUnconfigured {
		info.State = string(u.State())
	}
	return info
}

// SetGitHubToken stores a PAT in the keychain. Empty value clears it.
func (s *AboutService) SetGitHubToken(token string) error {
	if s.secrets == nil {
		return fmt.Errorf("secrets are not available")
	}
	return s.secrets.SetGitHubToken(strings.TrimSpace(token))
}

// ClearGitHubToken removes the PAT from the keychain.
func (s *AboutService) ClearGitHubToken() error {
	if s.secrets == nil {
		return fmt.Errorf("secrets are not available")
	}
	return s.secrets.DeleteGitHubToken()
}

// Check asks GitHub Releases whether a newer build exists.
func (s *AboutService) Check() (UpdateCheckResult, error) {
	u, err := s.ensureUpdater()
	if err != nil {
		return UpdateCheckResult{}, err
	}
	rel, err := u.Check(context.Background())
	if err != nil {
		return UpdateCheckResult{}, err
	}
	if rel == nil || !releaseIsNewer(rel.Version, s.version) {
		return UpdateCheckResult{}, nil
	}
	return checkResult(rel), nil
}

// Install downloads and stages the pending release without restarting.
func (s *AboutService) Install() error {
	u, err := s.ensureUpdater()
	if err != nil {
		return err
	}
	return u.DownloadAndInstall(context.Background())
}

// InstallAndRestart stages the pending release and swaps the binary.
func (s *AboutService) InstallAndRestart() error {
	u, err := s.ensureUpdater()
	if err != nil {
		return err
	}
	if u.State() != updater.StateReady {
		if err := u.DownloadAndInstall(context.Background()); err != nil {
			return err
		}
	}
	return u.Restart(context.Background())
}

// Restart swaps in an already staged update.
func (s *AboutService) Restart() error {
	u, err := s.updater()
	if err != nil {
		return err
	}
	return u.Restart(context.Background())
}

func (s *AboutService) ensureUpdater() (*updater.Updater, error) {
	u, err := s.updater()
	if err != nil {
		return nil, err
	}
	if u.State() != updater.StateUnconfigured {
		return u, nil
	}
	appl := application.Get()
	if appl == nil {
		return nil, fmt.Errorf("application is not running")
	}
	if err := InitUpdater(appl, s.version, s.gitHubToken); err != nil {
		if errors.Is(err, updater.ErrAlreadyConfigured) {
			return u, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *AboutService) updater() (*updater.Updater, error) {
	if !s.updatesEnabled {
		return nil, fmt.Errorf("updates are only available in release builds")
	}
	appl := application.Get()
	if appl == nil || appl.Updater == nil {
		return nil, fmt.Errorf("updater is not available")
	}
	return appl.Updater, nil
}

func (s *AboutService) gitHubToken() string {
	if s.secrets == nil {
		return ""
	}
	token, err := s.secrets.GetGitHubToken()
	if err != nil {
		return ""
	}
	return token
}

func (s *AboutService) hasGitHubToken() bool {
	return s.gitHubToken() != ""
}

func checkResult(rel *updater.Release) UpdateCheckResult {
	if rel == nil {
		return UpdateCheckResult{}
	}
	return UpdateCheckResult{
		Available: true,
		Version:   rel.Version,
		Name:      rel.Name,
		Notes:     rel.Notes,
		Filename:  rel.Artifact.Filename,
		Size:      rel.Artifact.Size,
	}
}
