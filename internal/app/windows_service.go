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
	valueWindowPrefix      = "value-"
	workspaceHandoffEvent  = "workspace:handoff"
	valueHandoffEvent      = "value:handoff"
	valueDraftEvent        = "value:draft"
)

// WindowsService opens (or focuses) a native window for a connection profile.
type WindowsService struct {
	mu            sync.Mutex
	handoffs      map[string]string
	valueHandoffs map[string]string
	valueReturns  map[string]string
	valueDrafts   map[string]string
}

// NewWindowsService constructs the Wails windows service.
func NewWindowsService() *WindowsService {
	return &WindowsService{
		handoffs:      make(map[string]string),
		valueHandoffs: make(map[string]string),
		valueReturns:  make(map[string]string),
		valueDrafts:   make(map[string]string),
	}
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

// OpenValueWindow shows (or creates) a value inspector window.
// token identifies the parked payload; payload is opaque JSON (value is not put in the URL).
func (s *WindowsService) OpenValueWindow(token, title, payload string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token is required")
	}
	appl := application.Get()
	if appl == nil {
		return fmt.Errorf("application is not running")
	}

	parked := s.parkValue(token, payload)

	name := valueWindowPrefix + token
	if win, ok := appl.Window.GetByName(name); ok {
		win.Show()
		win.Focus()
		if parked {
			appl.Event.Emit(valueHandoffEvent)
		}
		return nil
	}

	if strings.TrimSpace(title) == "" {
		title = "Value"
	}
	opts := WindowOptions(name, title, "/#/value?id="+url.QueryEscape(token))
	opts.Width = 720
	opts.Height = 640
	appl.Window.NewWithOptions(opts)
	return nil
}

// TakeValueHandoff returns and clears the parked payload for a value window.
func (s *WindowsService) TakeValueHandoff(token string) string {
	token = strings.TrimSpace(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.valueHandoffs[token]
	delete(s.valueHandoffs, token)
	return snap
}

// ReturnValueToPanel parks the payload for the source window, focuses it, and closes the value window.
// Returns an error when the source window is gone so the value window can stay open.
func (s *WindowsService) ReturnValueToPanel(token, sourceWindow, payload string) error {
	token = strings.TrimSpace(token)
	sourceWindow = strings.TrimSpace(sourceWindow)
	if token == "" {
		return fmt.Errorf("token is required")
	}
	if sourceWindow == "" {
		return fmt.Errorf("source window is required")
	}
	appl := application.Get()
	if appl == nil {
		return fmt.Errorf("application is not running")
	}

	src, ok := appl.Window.GetByName(sourceWindow)
	if !ok {
		return fmt.Errorf("source window is closed")
	}

	s.parkValueReturn(token, payload)
	appl.Event.Emit(valueHandoffEvent)
	src.Show()
	src.Focus()

	if win, ok := appl.Window.GetByName(valueWindowPrefix + token); ok {
		win.Close()
	}
	return nil
}

// TakeValueReturn returns and clears the payload parked for return to a side panel.
func (s *WindowsService) TakeValueReturn(token string) string {
	token = strings.TrimSpace(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.valueReturns[token]
	delete(s.valueReturns, token)
	return snap
}

// PushValueDraft parks a live edit from the value window for the source grid.
// Does not focus or close windows — the source applies the draft in the background.
func (s *WindowsService) PushValueDraft(token, sourceWindow, payload string) error {
	token = strings.TrimSpace(token)
	sourceWindow = strings.TrimSpace(sourceWindow)
	if token == "" {
		return fmt.Errorf("token is required")
	}
	if sourceWindow == "" {
		return fmt.Errorf("source window is required")
	}
	appl := application.Get()
	if appl == nil {
		return fmt.Errorf("application is not running")
	}
	if _, ok := appl.Window.GetByName(sourceWindow); !ok {
		return fmt.Errorf("source window is closed")
	}
	if !s.parkValueDraft(token, payload) {
		return fmt.Errorf("payload is required")
	}
	appl.Event.Emit(valueDraftEvent)
	return nil
}

// TakeValueDraft returns and clears the parked live draft for a token.
func (s *WindowsService) TakeValueDraft(token string) string {
	token = strings.TrimSpace(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.valueDrafts[token]
	delete(s.valueDrafts, token)
	return snap
}

func (s *WindowsService) parkValue(token, payload string) bool {
	payload = strings.TrimSpace(payload)
	if token == "" || payload == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.valueHandoffs[token] = payload
	return true
}

func (s *WindowsService) parkValueReturn(token, payload string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == "" {
		return
	}
	s.valueReturns[token] = payload
}

func (s *WindowsService) parkValueDraft(token, payload string) bool {
	payload = strings.TrimSpace(payload)
	if token == "" || payload == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.valueDrafts[token] = payload
	return true
}
