package services

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventTheme carries the new SystemTheme when the desktop's look changes.
const EventTheme = "theme"

// SystemTheme is how the desktop looks, so Nova can look like it belongs.
// Empty fields are unknown; the UI keeps its own defaults for those.
type SystemTheme struct {
	Platform string `json:"platform"` // runtime.GOOS
	Mode     string `json:"mode"`     // "dark", "light" or ""
	Accent   string `json:"accent"`   // "#rrggbb"
	Font     string `json:"font"`     // interface font family
	FontSize int    `json:"fontSize"` // interface font size in points
	// Name and Palette describe a desktop colour theme (an Omarchy theme,
	// a KDE colour scheme). Palette maps the surfaces Nova draws to colours;
	// see the Pal* keys. Missing keys keep Nova's own colours.
	Name    string            `json:"name"`
	Palette map[string]string `json:"palette"`
}

// Palette keys, all "#rrggbb".
const (
	PalWindow      = "window"      // window background
	PalView        = "view"        // file view background
	PalHeader      = "header"      // header bar
	PalSidebar     = "sidebar"     // sidebar
	PalPopover     = "popover"     // menus, popovers and dialogs
	PalFg          = "fg"          // text
	PalFgDim       = "fgDim"       // secondary text
	PalAccentFg    = "accentFg"    // text on the accent colour
	PalDestructive = "destructive" // errors, delete buttons
	PalWarning     = "warning"
	PalSuccess     = "success"
)

// hexRGB formats 0..255 channels as "#rrggbb".
func hexRGB(r, g, b int) string { return fmt.Sprintf("#%02x%02x%02x", r&255, g&255, b&255) }

// luminance of a "#rrggbb" colour, 0 (black) to 1 (white); -1 if invalid.
func luminance(hex string) float64 {
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil || len(hex) != 7 {
		return -1
	}
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

// WindowColour is what a new window shows before its page has painted:
// the page background Nova will use, so opening a window doesn't flash.
// pref is the style preference ("system", "light" or "dark").
func (s *ThemeService) WindowColour(pref string) (r, g, b uint8) {
	t := s.Current()
	dark := pref == "dark" || (pref != "light" && t.Mode != "light")
	bg := "#fafafb" // app.css, libadwaita's window colours
	if dark {
		bg = "#222226"
	}
	if w := t.Palette[PalWindow]; w != "" && t.Mode == map[bool]string{true: "dark", false: "light"}[dark] {
		bg = w
	}
	var ri, gi, bi int
	fmt.Sscanf(bg, "#%02x%02x%02x", &ri, &gi, &bi)
	return uint8(ri), uint8(gi), uint8(bi)
}

// ThemeService follows the desktop's colour scheme, accent colour, font and
// colour theme.
type ThemeService struct {
	mu  sync.Mutex
	cur SystemTheme
}

func NewThemeService() *ThemeService { return &ThemeService{} }

// Current returns the desktop's look as last read.
func (s *ThemeService) Current() SystemTheme {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur.Platform == "" { // asked before startup, for the first window
		s.cur = readTheme()
	}
	return s.cur
}

// ServiceStartup reads the desktop theme and keeps following it. Desktop
// settings have no single change signal across platforms, so this polls;
// reading them is cheap.
func (s *ThemeService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.mu.Lock()
	s.cur = readTheme()
	s.mu.Unlock()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			next := readTheme()
			s.mu.Lock()
			changed := !reflect.DeepEqual(next, s.cur)
			s.cur = next
			s.mu.Unlock()
			if changed {
				emit(EventTheme, next)
			}
		}
	}()
	return nil
}

func readTheme() SystemTheme {
	t := SystemTheme{Platform: runtime.GOOS}
	readPlatformTheme(&t)
	return t
}
