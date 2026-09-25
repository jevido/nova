//go:build android

package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"nova/internal/platform"
)

// On Android the Wails updater can't replace the app, so Nova downloads the
// new APK itself, verifies it against the release's checksums.txt and hands
// it to the system package installer (see MainActivity.installApk). Android
// only installs it over the current app when it is signed with the same key.

const apkAsset = "nova-android-arm64.apk"

func updatesDir() string { return filepath.Join(platform.CacheDir(), "updates") }

func (s *UpdateService) run(ctx context.Context) {
	if !s.beginRun() {
		return
	}
	defer s.endRun()
	rel, latest, ok := s.checkLatest(ctx)
	if !ok {
		return
	}
	var apkURL, sumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case apkAsset:
			apkURL = a.URL
		case "checksums.txt":
			sumsURL = a.URL
		}
	}
	if apkURL == "" || sumsURL == "" {
		s.set(func(st *UpdateStatus) { st.State = "manual" })
		return
	}
	s.set(func(st *UpdateStatus) { st.State = "downloading" })
	path, err := download(ctx, apkURL, sumsURL, latest)
	if err != nil {
		s.fail(err)
		return
	}
	s.set(func(st *UpdateStatus) { st.State = "ready"; st.ApkPath = path })
}

// download fetches the APK into the cache and checks its SHA-256.
func download(ctx context.Context, apkURL, sumsURL, ver string) (string, error) {
	want, err := expectedSum(ctx, sumsURL, apkAsset)
	if err != nil {
		return "", err
	}
	dir := updatesDir()
	_ = os.RemoveAll(dir) // drop older downloads
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apkURL, nil)
	if err != nil {
		return "", err
	}
	res, err := updateHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", res.Status)
	}
	path := filepath.Join(dir, "nova-"+ver+".apk")
	f, err := os.Create(path + ".part")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), res.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path + ".part")
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		os.Remove(path + ".part")
		return "", errors.New("downloaded update failed its checksum")
	}
	return path, os.Rename(path+".part", path)
}

// Restart is a no-op on Android: the UI passes ApkPath to the system
// installer, which replaces and restarts the app.
func (s *UpdateService) Restart(ctx context.Context) error {
	if s.Status().State != "ready" {
		return errors.New("no update has been downloaded yet")
	}
	return nil
}
