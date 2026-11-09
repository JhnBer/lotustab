package app

const fallbackTitleBarHeight = 38.4

// TitleBarHeight is the CSS-pixel height of the overlay strip above the UI.
// macOS uses it so traffic lights sit in empty space; fullscreen and other
// platforms return 0 because they keep a native caption or hide the lights.
func (s *WindowsService) TitleBarHeight() float64 {
	return measureTitleBarHeight()
}

// interpretTitleBarHeight maps the native probe: 0 is fullscreen (no overlay
// chrome), negative is "could not measure" (use the Sequoia-era fallback),
// positive is 2× the offset from the content top to the close-button center.
func interpretTitleBarHeight(measured float64) float64 {
	if measured == 0 {
		return 0
	}
	if measured < 0 {
		return fallbackTitleBarHeight
	}
	return measured
}
