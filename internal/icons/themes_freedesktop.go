//go:build (linux && !android) || freebsd || openbsd || netbsd || dragonfly

package icons

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// iconDirs lists the freedesktop icon theme base directories.
func iconDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".icons"), filepath.Join(home, ".local/share/icons"))
	}
	data := os.Getenv("XDG_DATA_DIRS")
	if data == "" {
		data = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(data, ":") {
		if d != "" {
			dirs = append(dirs, filepath.Join(d, "icons"))
		}
	}
	return dirs
}

// currentTheme is the desktop's icon theme (GNOME and most GTK desktops).
func currentTheme() string {
	if t := os.Getenv("NOVA_ICON_THEME"); t != "" {
		return t
	}
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "icon-theme").Output()
	if err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'")
}
