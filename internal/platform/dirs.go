//go:build !android && !ios

// Package platform resolves per-OS locations for settings, caches and downloads.
package platform

import (
	"os"
	"path/filepath"
)

// Mobile reports whether the build targets a phone.
const Mobile = false

// ConfigDir is where settings and credentials are stored.
func ConfigDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "nova-desktop"), nil
}

// CacheDir holds thumbnails and files opened in other applications.
func CacheDir() string {
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "nova-desktop")
}

// DownloadDir is the default download location.
func DownloadDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Downloads")
	}
	return ""
}
