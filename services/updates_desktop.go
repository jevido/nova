//go:build !android

package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"

	"nova/internal/version"
)

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

// matchAsset picks the portable archive the release workflow builds for the
// in-app updater: nova-linux-amd64.tar.gz.
func matchAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	prefix := "nova-" + req.Platform + "-" + req.Arch + "."
	for i, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.HasPrefix(name, prefix) && (strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")) {
			return i
		}
	}
	return -1
}

// canSelfUpdate reports whether the running binary sits in a directory we can
// write to. AppImages and system package installs can't be swapped in place;
// those users get a link to the release instead.
func canSelfUpdate() bool {
	if os.Getenv("APPIMAGE") != "" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe, _ = filepath.EvalSymlinks(exe)
	f, err := os.CreateTemp(filepath.Dir(exe), ".nova-update-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// packageManaged reports whether the binary lives where only a package
// manager writes (Linux /usr, /opt).
func packageManaged() bool {
	exe, err := os.Executable()
	if err != nil || runtime.GOOS != "linux" {
		return false
	}
	exe, _ = filepath.EvalSymlinks(exe)
	return strings.HasPrefix(exe, "/usr/") || strings.HasPrefix(exe, "/opt/")
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

// Workarounds for the Wails updater (wailsapp/wails#6134). Its helper
// downloads the new version to the system temp directory and renames it over
// the app. When the temp directory is on another filesystem (tmpfs /tmp on
// Arch and Fedora, or a separate /home partition) the rename fails; the helper
// then restores the old binary but relaunches it with its helper environment
// still set, so that binary becomes a helper again: a loop that deletes the
// app every few seconds.

const (
	helperEnv     = "WAILS_UPDATER_HELPER"
	helperRanEnv  = "NOVA_UPDATER_HELPER_RAN"
	origTmpDirEnv = "NOVA_ORIG_TMPDIR"
	stagingDir    = ".nova-update"
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
}

// stageNextToApp makes the updater download into a directory next to the app,
// on the same filesystem, so moving the new version into place is a rename
// that works. The helper and the relaunched app inherit the setting;
// restoreTempDir undoes it.
func stageNextToApp() {
	if runtime.GOOS != "linux" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	dir := filepath.Join(filepath.Dir(exe), stagingDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	if _, set := os.LookupEnv(origTmpDirEnv); !set {
		os.Setenv(origTmpDirEnv, os.Getenv("TMPDIR"))
	}
	os.Setenv("TMPDIR", dir)
}

// restoreTempDir puts back the temp directory a relaunched app inherited from
// stageNextToApp, and removes what an earlier update left in the staging dir.
func restoreTempDir() {
	if orig, set := os.LookupEnv(origTmpDirEnv); set {
		if orig == "" {
			os.Unsetenv("TMPDIR")
		} else {
			os.Setenv("TMPDIR", orig)
		}
		os.Unsetenv(origTmpDirEnv)
	}
	if runtime.GOOS != "linux" {
		return
	}
	if exe, err := os.Executable(); err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		os.RemoveAll(filepath.Join(filepath.Dir(exe), stagingDir))
	}
}
