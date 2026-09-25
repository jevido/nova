//go:build linux && !android

package services

import (
	"os"
	"path/filepath"
	"strings"
)

const stagingDir = ".nova-update"

// packageManaged reports whether the binary lives where only a package
// manager writes (/usr, /opt).
func packageManaged() bool {
	exe, err := selfPath()
	if err != nil {
		return false
	}
	return strings.HasPrefix(exe, "/usr/") || strings.HasPrefix(exe, "/opt/")
}

// stageNextToApp makes the updater download into a directory next to the app,
// on the same filesystem, so moving the new version into place is a rename
// that works. The helper renames the download over the app, which fails when
// the temp directory is on another filesystem (tmpfs /tmp on Arch and
// Fedora, or a separate /home partition). The helper and the relaunched app
// inherit the setting; restoreTempDir undoes it.
func stageNextToApp() {
	exe, err := selfPath()
	if err != nil {
		return
	}
	dir := filepath.Join(filepath.Dir(exe), stagingDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	if _, set := os.LookupEnv(origTmpDirEnv); !set {
		os.Setenv(origTmpDirEnv, os.Getenv("TMPDIR"))
	}
	os.Setenv("TMPDIR", dir)
}

// cleanupAfterUpdate removes what an earlier update left in the staging dir.
func cleanupAfterUpdate() {
	if exe, err := selfPath(); err == nil {
		os.RemoveAll(filepath.Join(filepath.Dir(exe), stagingDir))
	}
}
