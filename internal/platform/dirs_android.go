//go:build android

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// Mobile reports whether the build targets a phone.
const Mobile = true

// packageName reads the app's package from /proc; Android names the app
// process after it (optionally with a ":process" suffix).
func packageName() string {
	b, err := os.ReadFile("/proc/self/cmdline")
	if err != nil {
		return ""
	}
	name := strings.TrimRight(string(b), "\x00")
	if i := strings.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	if i := strings.IndexByte(name, ':'); i >= 0 {
		name = name[:i]
	}
	return name
}

func appDir(sub string) string {
	pkg := packageName()
	if pkg == "" {
		return filepath.Join(os.TempDir(), sub)
	}
	return filepath.Join("/data/data", pkg, sub)
}

// ConfigDir is inside the app's private files directory.
func ConfigDir() (string, error) { return appDir("files"), nil }

// CacheDir is the app's private cache, which Android may clear when low on space.
func CacheDir() string { return appDir("cache") }

// DownloadDir is the shared Download/Nova folder, visible in the Files app.
// Since Android 11 apps may create files there without a storage permission.
func DownloadDir() string {
	return "/storage/emulated/0/Download/Nova"
}
