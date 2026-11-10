package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// AboutInfo is the static identity shown on Settings → About.
type AboutInfo struct {
	Version        string `json:"version"`
	Goos           string `json:"goos"`
	Goarch         string `json:"goarch"`
	UpdatesEnabled bool   `json:"updatesEnabled"`
	State          string `json:"state"`
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

	mu             sync.Mutex
	checkedVersion string
}

// NewAboutService constructs the Wails about/updater service.
func NewAboutService(version string, updatesEnabled bool) *AboutService {
	return &AboutService{version: version, updatesEnabled: updatesEnabled}
}

// Info returns the running version, architecture, and updater state.
func (s *AboutService) Info() AboutInfo {
	info := AboutInfo{
		Version:        s.version,
		Goos:           runtime.GOOS,
		Goarch:         runtime.GOARCH,
		UpdatesEnabled: s.updatesEnabled,
		State:          string(updater.StateUnconfigured),
	}
	if u, err := s.updater(); err == nil && u.State() != updater.StateUnconfigured {
		info.State = string(u.State())
	}
	return info
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
	s.mu.Lock()
	if rel == nil || !releaseIsNewer(rel.Version, s.version) {
		s.checkedVersion = ""
		s.mu.Unlock()
		return UpdateCheckResult{}, nil
	}
	s.checkedVersion = rel.Version
	s.mu.Unlock()
	return checkResult(rel), nil
}

// Install downloads the pending release and stores it for the next launch.
func (s *AboutService) Install() error {
	u, err := s.ensureUpdater()
	if err != nil {
		return err
	}
	if err := u.DownloadAndInstall(context.Background()); err != nil {
		return err
	}
	return s.persistDownloaded(u)
}

// InstallAndRestart stores the pending release if needed and restarts the app.
// The swap happens on the next launch, the same way as quitting and opening again.
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
	if u.DownloadedPath() != "" {
		if err := s.persistDownloaded(u); err != nil {
			return err
		}
	}
	return Relaunch()
}

// Restart starts the app again. A stored update is applied on that launch.
func (s *AboutService) Restart() error {
	return Relaunch()
}

func (s *AboutService) persistDownloaded(u *updater.Updater) error {
	path := u.DownloadedPath()
	if path == "" {
		return fmt.Errorf("downloaded update has no path")
	}
	s.mu.Lock()
	version := s.checkedVersion
	s.mu.Unlock()
	if version == "" {
		return fmt.Errorf("update version is unknown")
	}
	root, err := pendingRoot()
	if err != nil {
		return err
	}
	return stagePending(root, version, path)
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
	if err := InitUpdater(appl, s.version); err != nil {
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
