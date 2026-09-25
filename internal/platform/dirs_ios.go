//go:build ios

package platform

import (
	"os"
	"path/filepath"
)

// Mobile reports whether the build targets a phone.
const Mobile = true

// home is the app's sandbox container; iOS points HOME at it.
func home() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return os.TempDir()
}

// ConfigDir is Library/Application Support inside the sandbox: backed up,
// but not visible in the Files app.
func ConfigDir() (string, error) {
	return filepath.Join(home(), "Library", "Application Support", "nova"), nil
}

// CacheDir is Library/Caches, which iOS may purge when low on space.
func CacheDir() string { return filepath.Join(home(), "Library", "Caches", "nova") }

// DownloadDir is Documents/Nova. Info.plist turns on file sharing, so it
// shows up in the Files app under "On My iPhone > Nova".
func DownloadDir() string { return filepath.Join(home(), "Documents", "Nova") }
