//go:build linux && !android

package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

// On Linux the desktop portal has the colour scheme, accent colour and
// interface font. Omarchy themes also colour every app from one
// colors.toml, which Nova follows as well.
func readPlatformTheme(t *SystemTheme) {
	if v, ok := portalSetting("org.freedesktop.appearance", "color-scheme"); ok {
		switch n, _ := v.(uint32); n {
		case 1:
			t.Mode = "dark"
		case 2:
			t.Mode = "light"
		}
	}
	if v, ok := portalSetting("org.freedesktop.appearance", "accent-color"); ok {
		// (ddd), each 0..1; out of range means "no accent colour".
		if rgb, ok := v.([]any); ok && len(rgb) == 3 {
			var c [3]float64
			valid := true
			for i, x := range rgb {
				f, ok := x.(float64)
				valid = valid && ok && f >= 0 && f <= 1
				c[i] = f
			}
			if valid {
				t.Accent = fmt.Sprintf("#%02x%02x%02x", int(c[0]*255+0.5), int(c[1]*255+0.5), int(c[2]*255+0.5))
			}
		}
	}
	if v, ok := portalSetting("org.gnome.desktop.interface", "font-name"); ok {
		if s, ok := v.(string); ok {
			t.Font, t.FontSize = parseFontName(s)
		}
	}
	readOmarchyTheme(t)
}

var (
	busOnce sync.Once
	bus     *dbus.Conn
)

func portalSetting(ns, key string) (any, bool) {
	busOnce.Do(func() { bus, _ = dbus.SessionBus() })
	if bus == nil {
		return nil, false
	}
	obj := bus.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	var v dbus.Variant
	if err := obj.Call("org.freedesktop.portal.Settings.ReadOne", 0, ns, key).Store(&v); err != nil {
		// Older portals only have Read, which wraps the value once more.
		if err := obj.Call("org.freedesktop.portal.Settings.Read", 0, ns, key).Store(&v); err != nil {
			return nil, false
		}
		if inner, ok := v.Value().(dbus.Variant); ok {
			v = inner
		}
	}
	return v.Value(), true
}

// parseFontName splits a GSettings font name like "Adwaita Sans 11".
func parseFontName(s string) (family string, size int) {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, ' '); i > 0 {
		if n, err := strconv.ParseFloat(s[i+1:], 64); err == nil {
			return strings.TrimSpace(s[:i]), int(n + 0.5)
		}
	}
	return s, 0
}

var tomlColor = regexp.MustCompile(`^\s*([a-z_]+)\s*=\s*"([^"]*)"`)

// readOmarchyTheme reads the current Omarchy theme's colors.toml, if any.
func readOmarchyTheme(t *SystemTheme) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".local", "state", "omarchy", "current")
	f, err := os.Open(filepath.Join(dir, "theme", "colors.toml"))
	if err != nil {
		dir = filepath.Join(home, ".config", "omarchy", "current") // older Omarchy
		if f, err = os.Open(filepath.Join(dir, "theme", "colors.toml")); err != nil {
			return
		}
	}
	defer f.Close()
	pal := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := tomlColor.FindStringSubmatch(sc.Text()); m != nil {
			pal[m[1]] = m[2]
		}
	}
	if pal["background"] == "" || pal["foreground"] == "" {
		return
	}
	if name, err := os.ReadFile(filepath.Join(dir, "theme.name")); err == nil {
		t.Name = strings.TrimSpace(string(name))
	}
	if m := pal["mode"]; m == "dark" || m == "light" {
		t.Mode = m
	}
	delete(pal, "mode")
	if a := pal["accent"]; a != "" {
		t.Accent = a
	}
	t.Palette = pal
}
