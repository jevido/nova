package services

import (
	"context"
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
	// Name and Palette describe a desktop colour theme (Omarchy's
	// colors.toml): background, foreground, accent, red, ...
	Name    string            `json:"name"`
	Palette map[string]string `json:"palette"`
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
	return s.cur
}

// ServiceStartup reads the desktop theme and keeps following it. Desktop
// settings have no single change signal across platforms, so this polls;
// reading them is cheap.
func (s *ThemeService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.cur = readTheme()
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
