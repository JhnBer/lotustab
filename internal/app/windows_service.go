package app

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const connectionWindowPrefix = "conn-"

// WindowsService opens (or focuses) a native window for a connection profile.
type WindowsService struct{}

// NewWindowsService constructs the Wails windows service.
func NewWindowsService() *WindowsService {
	return &WindowsService{}
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
		BackgroundColour: application.NewRGB(10, 10, 10),
		URL:              pageURL,
	}
}

// OpenConnection shows the existing window for a profile, or creates one.
func (s *WindowsService) OpenConnection(profileID, title string) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return fmt.Errorf("profile is required")
	}
	appl := application.Get()
	if appl == nil {
		return fmt.Errorf("application is not running")
	}

	name := connectionWindowPrefix + profileID
	if win, ok := appl.Window.GetByName(name); ok {
		win.Show()
		win.Focus()
		return nil
	}

	if strings.TrimSpace(title) == "" {
		title = "LotusTab"
	}
	pageURL := "/#/query?profile=" + url.QueryEscape(profileID)
	appl.Window.NewWithOptions(WindowOptions(name, title, pageURL))
	return nil
}
