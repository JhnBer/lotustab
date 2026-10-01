package app

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	connectionWindowPrefix = "conn-"
	workspaceHandoffEvent  = "workspace:handoff"
)

// WindowsService opens (or focuses) a native window for a connection profile.
type WindowsService struct {
	mu       sync.Mutex
	handoffs map[string]string
}

// NewWindowsService constructs the Wails windows service.
func NewWindowsService() *WindowsService {
	return &WindowsService{handoffs: make(map[string]string)}
}

// WindowOptions is the shared chrome for the main window and connection windows.
func WindowOptions(name, title, pageURL string) application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:   name,
		Title:  title,
		Width:  1280,
		Height: 800,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHiddenInset,
		},
		BackgroundColour:   application.NewRGB(10, 10, 10),
		URL:                pageURL,
		UseApplicationMenu: true,
	}
}

// OpenConnection shows the existing window for a profile, or creates one.
// snapshot is opaque JSON from the source window; empty means no tabs to move.
func (s *WindowsService) OpenConnection(profileID, title, snapshot string) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return fmt.Errorf("profile is required")
	}
	appl := application.Get()
	if appl == nil {
		return fmt.Errorf("application is not running")
	}

	parked := s.park(profileID, snapshot)

	name := connectionWindowPrefix + profileID
	if win, ok := appl.Window.GetByName(name); ok {
		win.Show()
		win.Focus()
		if parked {
			appl.Event.Emit(workspaceHandoffEvent)
		}
		return nil
	}

	if strings.TrimSpace(title) == "" {
		title = "LotusTab"
	}
	pageURL := "/#/query?profile=" + url.QueryEscape(profileID)
	appl.Window.NewWithOptions(WindowOptions(name, title, pageURL))
	return nil
}

// TakeHandoff returns and clears the parked workspace snapshot for a profile.
func (s *WindowsService) TakeHandoff(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.handoffs[profileID]
	delete(s.handoffs, profileID)
	return snap
}

func (s *WindowsService) park(profileID, snapshot string) bool {
	snapshot = strings.TrimSpace(snapshot)
	if profileID == "" || snapshot == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handoffs[profileID] = snapshot
	return true
}
