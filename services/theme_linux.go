//go:build linux && !android

package services

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

// On Linux the desktop portal has the colour scheme, accent colour and
// interface font. Omarchy themes and KDE colour schemes (EndeavourOS's
// default desktop among them) also colour every app, and Nova follows those
// as well.
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
			var c [3]int
			valid := true
			for i, x := range rgb {
				f, ok := x.(float64)
				valid = valid && ok && f >= 0 && f <= 1
				c[i] = int(f*255 + 0.5)
			}
			if valid {
				t.Accent = hexRGB(c[0], c[1], c[2])
			}
		}
	}
	if v, ok := portalSetting("org.gnome.desktop.interface", "font-name"); ok {
		if s, ok := v.(string); ok {
			t.Font, t.FontSize = parseFontName(s)
		}
	}
	home, _ := os.UserHomeDir()
	if !readOmarchyTheme(t, home) && isKDE() {
		readKDETheme(t, home, xdgDirs("XDG_CONFIG_DIRS", "/etc/xdg"), xdgDirs("XDG_DATA_DIRS", "/usr/local/share:/usr/share"))
	}
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

func xdgDirs(env, def string) []string {
	v := os.Getenv(env)
	if v == "" {
		v = def
	}
	return filepath.SplitList(v)
}

func isKDE() bool {
	d := strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP") + ":" + os.Getenv("DESKTOP_SESSION"))
	return strings.Contains(d, "KDE") || strings.Contains(d, "PLASMA")
}

// ---- Omarchy ----

var tomlColor = regexp.MustCompile(`^\s*([a-z_]+)\s*=\s*"([^"]*)"`)

// readOmarchyTheme reads the current Omarchy theme's colors.toml, if any.
func readOmarchyTheme(t *SystemTheme, home string) bool {
	dir := filepath.Join(home, ".local", "state", "omarchy", "current")
	f, err := os.Open(filepath.Join(dir, "theme", "colors.toml"))
	if err != nil {
		dir = filepath.Join(home, ".config", "omarchy", "current") // older Omarchy
		if f, err = os.Open(filepath.Join(dir, "theme", "colors.toml")); err != nil {
			return false
		}
	}
	defer f.Close()
	c := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := tomlColor.FindStringSubmatch(sc.Text()); m != nil {
			c[m[1]] = m[2]
		}
	}
	if c["background"] == "" || c["foreground"] == "" {
		return false
	}
	if name, err := os.ReadFile(filepath.Join(dir, "theme.name")); err == nil {
		t.Name = strings.TrimSpace(string(name))
	}
	dark := c["mode"] != "light"
	if c["mode"] == "" {
		dark = luminance(c["background"]) < 0.5
	}
	t.Mode = map[bool]string{true: "dark", false: "light"}[dark]
	if a := c["accent"]; a != "" {
		t.Accent = a
	}
	// Omarchy names its shades by brightness; lay them out like libadwaita,
	// where the sidebar and popovers stand out from the view.
	raised, side := c["lighter_background"], c["lighter_background"]
	if !dark {
		raised, side = c["background"], c["darker_background"]
	}
	t.Palette = compact(map[string]string{
		PalWindow:      c["background"],
		PalView:        c["background"],
		PalHeader:      c["background"],
		PalSidebar:     side,
		PalPopover:     raised,
		PalFg:          c["foreground"],
		PalDestructive: c["red"],
		PalWarning:     c["yellow"],
		PalSuccess:     c["green"],
	})
	return true
}

// ---- KDE Plasma ----

// ini is a KDE config file: group -> key -> value.
type ini map[string]map[string]string

// readINI merges the KDE config file at p into into.
func readINI(p string, into ini) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	group := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "" || line[0] == '#' || line[0] == ';':
		case line[0] == '[':
			group = strings.Trim(line, "[]")
		default:
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			// Drop KDE's [$e] / [$i] markers.
			if i := strings.IndexByte(k, '['); i > 0 {
				k = k[:i]
			}
			if into[group] == nil {
				into[group] = map[string]string{}
			}
			into[group][strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return true
}

// kdeColour turns "r,g,b" into "#rrggbb".
func kdeColour(v string) string {
	parts := strings.Split(v, ",")
	if len(parts) < 3 {
		return ""
	}
	var c [3]int
	for i := range 3 {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || n < 0 || n > 255 {
			return ""
		}
		c[i] = n
	}
	return hexRGB(c[0], c[1], c[2])
}

// breezeDark is Plasma's Breeze Dark scheme, which EndeavourOS uses, for
// when its .colors file can't be found.
var breezeDark = ini{
	"Colors:Window":    {"BackgroundNormal": "32,35,38", "ForegroundNormal": "252,252,252", "ForegroundInactive": "161,169,177"},
	"Colors:View":      {"BackgroundNormal": "20,22,24", "ForegroundNormal": "252,252,252", "ForegroundInactive": "161,169,177", "ForegroundNegative": "218,68,83", "ForegroundNeutral": "246,116,0", "ForegroundPositive": "39,174,96"},
	"Colors:Header":    {"BackgroundNormal": "41,44,48"},
	"Colors:Selection": {"BackgroundNormal": "61,174,233", "ForegroundNormal": "252,252,252"},
}

// readKDETheme follows the Plasma colour scheme, accent colour and font in
// kdeglobals. Plasma copies the scheme's colours into kdeglobals when it is
// applied; until then (a fresh EndeavourOS account, say) they come from the
// scheme's .colors file.
func readKDETheme(t *SystemTheme, home string, configDirs, dataDirs []string) {
	g := ini{}
	for i := len(configDirs) - 1; i >= 0; i-- {
		readINI(filepath.Join(configDirs[i], "kdeglobals"), g)
	}
	if !readINI(filepath.Join(home, ".config", "kdeglobals"), g) && len(g) == 0 {
		return
	}
	scheme := g["General"]["ColorScheme"]
	colours := ini{}
	if g["Colors:Window"]["BackgroundNormal"] != "" {
		colours = g
	} else if scheme != "" {
		found := readINI(filepath.Join(home, ".local", "share", "color-schemes", scheme+".colors"), colours)
		for _, d := range dataDirs {
			if found {
				break
			}
			found = readINI(filepath.Join(d, "color-schemes", scheme+".colors"), colours)
		}
		if !found && strings.EqualFold(scheme, "BreezeDark") {
			colours = breezeDark
		}
	}
	lnf := g["KDE"]["LookAndFeelPackage"]
	if len(colours) == 0 && strings.Contains(strings.ToLower(lnf), "endeavouros") {
		colours = breezeDark
	}

	c := func(group, key string) string { return kdeColour(colours["Colors:"+group][key]) }
	window, view := c("Window", "BackgroundNormal"), c("View", "BackgroundNormal")
	if window != "" && view != "" {
		header := c("Header", "BackgroundNormal")
		if header == "" {
			header = window
		}
		t.Mode = map[bool]string{true: "dark", false: "light"}[luminance(window) < 0.5]
		t.Palette = compact(map[string]string{
			PalWindow:      window,
			PalView:        view,
			PalHeader:      header,
			PalSidebar:     window, // Dolphin's Places panel
			PalPopover:     window,
			PalFg:          c("View", "ForegroundNormal"),
			PalFgDim:       c("View", "ForegroundInactive"),
			PalAccentFg:    c("Selection", "ForegroundNormal"),
			PalDestructive: c("View", "ForegroundNegative"),
			PalWarning:     c("View", "ForegroundNeutral"),
			PalSuccess:     c("View", "ForegroundPositive"),
		})
		t.Accent = c("Selection", "BackgroundNormal")
	}
	// An accent picked in System Settings wins over the scheme's selection
	// colour; EndeavourOS sets its purple this way.
	if a := kdeColour(g["General"]["AccentColor"]); a != "" {
		t.Accent = a
	}
	if f := g["General"]["font"]; f != "" {
		// "Noto Sans,10,-1,5,50,0,0,0,0,0"
		parts := strings.Split(f, ",")
		t.Font = parts[0]
		if len(parts) > 1 {
			if n, err := strconv.ParseFloat(parts[1], 64); err == nil && n > 0 {
				t.FontSize = int(n + 0.5)
			}
		}
	}
	switch {
	case strings.Contains(strings.ToLower(lnf), "endeavouros"):
		t.Name = "EndeavourOS"
	case scheme != "":
		t.Name = scheme
	}
}

// compact drops empty entries.
func compact(m map[string]string) map[string]string {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	return m
}
