//go:build !android && !ios

package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"

	"nova/internal/version"
)

// The desktop updater uses Wails' updater: it downloads the release asset,
// verifies it against the release's checksums.txt and, on Restart, has a
// helper process swap the binary and relaunch it. The OS-specific parts live
// in updates_linux.go and updates_windows.go.

func openURL(url string) error { return application.Get().Browser.OpenURL(url) }

// ServiceStartup starts background checks for release builds.
func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if !version.IsRelease() || os.Getenv("NOVA_NO_UPDATE") != "" {
		return nil // development builds never update themselves
	}
	provider, err := github.New(github.Config{
		Repository:    version.Repo,
		ChecksumAsset: "checksums.txt",
		AssetMatcher:  matchAsset,
		// For testing the update flow against a local fake of the GitHub API.
		BaseURL: os.Getenv("NOVA_UPDATE_API"),
	})
	if err != nil {
		return err
	}
	if err := application.Get().Updater.Init(updater.Config{
		CurrentVersion: strings.TrimPrefix(version.Version, "v"),
		Providers:      []updater.Provider{provider},
		Window:         updater.WindowNone, // Nova shows its own notification
	}); err != nil {
		return err
	}
	s.mu.Lock()
	s.enabled = true
	s.status.State = "idle"
	s.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(firstCheckDelay):
		}
		t := time.NewTicker(updateCheckInterval)
		defer t.Stop()
		for {
			s.run(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
	return nil
}

// matchAsset picks the file the release workflow builds for the in-app
// updater: nova-linux-amd64.tar.gz on Linux and the portable
// nova-windows-amd64.exe on Windows (never the -setup.exe installer).
func matchAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	prefix := "nova-" + req.Platform + "-" + req.Arch + "."
	for i, a := range assets {
		name := strings.ToLower(a.Name)
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") ||
			(req.Platform == "windows" && strings.HasSuffix(name, ".exe")) {
			return i
		}
	}
	return -1
}

// selfPath is the running binary with symlinks resolved.
func selfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return exe, nil
}

// canSelfUpdate reports whether the running binary sits in a directory we can
// write to. AppImages, system packages and machine-wide Windows installs
// (Program Files) can't be swapped in place; those users get a link to the
// release instead.
func canSelfUpdate() bool {
	if os.Getenv("APPIMAGE") != "" {
		return false
	}
	exe, err := selfPath()
	if err != nil {
		return false
	}
	f, err := os.CreateTemp(filepath.Dir(exe), ".nova-update-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// run performs one check and, when a newer release exists, downloads it.
func (s *UpdateService) run(ctx context.Context) {
	if !s.beginRun() {
		return
	}
	defer s.endRun()

	u := application.Get().Updater
	s.set(func(st *UpdateStatus) { st.State = "checking"; st.Error = "" })
	rel, err := u.Check(ctx)
	now := time.Now().Format(time.RFC3339)
	if err != nil {
		s.set(func(st *UpdateStatus) { st.State = "error"; st.Error = err.Error(); st.CheckedAt = now })
		return
	}
	if rel == nil {
		s.set(func(st *UpdateStatus) { st.State = "up-to-date"; st.CheckedAt = now })
		return
	}
	releaseURL, _ := rel.Metadata["github.release.htmlURL"].(string)
	s.set(func(st *UpdateStatus) {
		st.LatestVersion = rel.Version
		st.Notes = rel.Notes
		st.ReleaseURL = releaseURL
		st.CheckedAt = now
		st.State = "downloading"
	})
	if !canSelfUpdate() {
		s.set(func(st *UpdateStatus) { st.State = "manual"; st.PackageManaged = packageManaged() })
		return
	}
	stageNextToApp()
	if err := u.DownloadAndInstall(ctx); err != nil {
		s.set(func(st *UpdateStatus) { st.State = "error"; st.Error = err.Error() })
		return
	}
	s.set(func(st *UpdateStatus) { st.State = "ready" })
}

// CheckNow runs a check immediately and returns the resulting state.
func (s *UpdateService) CheckNow(ctx context.Context) UpdateStatus {
	s.run(ctx)
	return s.Status()
}

// Restart quits and relaunches into the downloaded version.
func (s *UpdateService) Restart(ctx context.Context) error {
	if s.Status().State != "ready" {
		return errors.New("no update has been downloaded yet")
	}
	return application.Get().Updater.Restart(ctx)
}

// Workarounds for the Wails updater (wailsapp/wails#6134). Its helper is the
// app itself, started again with WAILS_UPDATER_HELPER set: it waits for the
// app to quit, moves the downloaded version into place and relaunches it.
// When that move fails (see stageNextToApp in updates_linux.go) the helper
// restores the old binary but relaunches it with its helper environment
// still set, so that binary becomes a helper again: a loop that deletes the
// app every few seconds.

const (
	helperEnv     = "WAILS_UPDATER_HELPER"
	helperRanEnv  = "NOVA_UPDATER_HELPER_RAN"
	origTmpDirEnv = "NOVA_ORIG_TMPDIR"
)

// GuardUpdaterHelper must run before application.New, which enters Wails'
// helper mode. A binary relaunched by a helper that failed its swap still has
// the helper environment; it is recognized by the marker the first helper
// sets, and starts as the normal app instead.
func GuardUpdaterHelper() {
	if os.Getenv(helperEnv) == "1" && os.Getenv(helperRanEnv) == "" {
		os.Setenv(helperRanEnv, "1") // this is the helper
		return
	}
	for _, k := range []string{helperEnv, helperEnv + "_TARGET", helperEnv + "_NEW", helperEnv + "_PID", helperEnv + "_LOG", helperRanEnv} {
		os.Unsetenv(k)
	}
	restoreTempDir()
	cleanupAfterUpdate()
}

// restoreTempDir puts back the temp directory a relaunched app inherited from
// stageNextToApp.
func restoreTempDir() {
	orig, set := os.LookupEnv(origTmpDirEnv)
	if !set {
		return
	}
	if orig == "" {
		os.Unsetenv("TMPDIR")
	} else {
		os.Setenv("TMPDIR", orig)
	}
	os.Unsetenv(origTmpDirEnv)
}
