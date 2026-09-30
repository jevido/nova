//go:build linux && !android

package main

import (
	"os"
	"path/filepath"
)

// The AppImage bundles WebKitGTK and GStreamer in a way that needs a few
// environment variables before GTK starts (build/linux/appimage/fix-appimage.sh):
//
//   - WebKit finds its helper processes by a path relative to the AppDir.
//     Its bubblewrap sandbox can't bind a relative path and fails to start,
//     so it is turned off. The webview only loads Nova's own frontend.
//   - GStreamer uses only the plugins bundled in the AppDir, and keeps its
//     registry apart from the system's so the two don't overwrite each other.
func init() {
	appdir := os.Getenv("APPDIR")
	if os.Getenv("APPIMAGE") == "" || appdir == "" {
		return
	}
	os.Setenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS", "1")

	os.Setenv("GST_PLUGIN_SYSTEM_PATH_1_0", filepath.Join(appdir, "usr/lib/gstreamer-1.0"))
	os.Setenv("GST_PLUGIN_SCANNER_1_0", filepath.Join(appdir, "usr/libexec/gstreamer-1.0/gst-plugin-scanner"))
	if cache, err := os.UserCacheDir(); err == nil {
		os.Setenv("GST_REGISTRY_1_0", filepath.Join(cache, "nova", "gstreamer-registry.bin"))
	}
}
