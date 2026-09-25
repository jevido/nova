//go:build windows

package services

import (
	"os"
	"path/filepath"
)

// Windows won't let anyone delete or overwrite an .exe while it runs, and the
// updater's helper runs from the very file it replaces. The Wails helper
// therefore renames the running nova.exe aside to nova.exe.old.<n> (renaming
// a running executable is allowed), moves the verified download into its
// place, relaunches it and exits. The aside can only be deleted once no
// process maps it any more, which cleanupAfterUpdate does on a later start.
//
// A copy in Program Files (the machine-wide installer) isn't writable without
// admin rights; canSelfUpdate notices and the user gets a download link. The
// portable exe and the per-user installer (%LOCALAPPDATA%\Programs\Nova)
// update themselves.

// packageManaged: Windows has no package manager Nova is installed with.
func packageManaged() bool { return false }

// stageNextToApp isn't needed: the helper falls back to copying when the
// temp directory is on another drive than the app.
func stageNextToApp() {}

// cleanupAfterUpdate deletes executables an earlier update renamed aside.
func cleanupAfterUpdate() {
	exe, err := selfPath()
	if err != nil {
		return
	}
	old, _ := filepath.Glob(exe + ".old.*")
	for _, p := range old {
		_ = os.Remove(p) // fails harmlessly while the old helper is still exiting
	}
}
