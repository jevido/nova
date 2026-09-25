//go:build android

package services

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"nova/internal/platform"
	"nova/internal/version"
)

// On Android the Wails updater can't replace the app, so Nova downloads the
// new APK itself, verifies it against the release's checksums.txt and hands
// it to the system package installer (see MainActivity.installApk). Android
// only installs it over the current app when it is signed with the same key.

const apkAsset = "nova-android-arm64.apk"

type ghRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

var updateHTTP = &http.Client{Timeout: 10 * time.Minute}

// The desktop BrowserManager shells out to xdg-open, which Android lacks.
func openURL(url string) error {
	application.Mobile.OpenURL(url)
	return nil
}

// ServiceStartup starts background checks for release builds.
func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if !version.IsRelease() {
		return nil
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

func updatesDir() string { return filepath.Join(platform.CacheDir(), "updates") }

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := updateHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (s *UpdateService) run(ctx context.Context) {
	if !s.beginRun() {
		return
	}
	defer s.endRun()
	s.set(func(st *UpdateStatus) { st.State = "checking"; st.Error = "" })
	now := time.Now().Format(time.RFC3339)
	fail := func(err error) {
		s.set(func(st *UpdateStatus) { st.State = "error"; st.Error = err.Error(); st.CheckedAt = now })
	}

	var rel ghRelease
	if err := getJSON(ctx, "https://api.github.com/repos/"+version.Repo+"/releases/latest", &rel); err != nil {
		fail(err)
		return
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	if rel.Prerelease || !version.Newer(latest, version.Version) {
		s.set(func(st *UpdateStatus) { st.State = "up-to-date"; st.CheckedAt = now })
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
	s.set(func(st *UpdateStatus) {
		st.LatestVersion = latest
		st.Notes = rel.Body
		st.ReleaseURL = rel.HTMLURL
		st.CheckedAt = now
		st.State = "downloading"
	})
	if apkURL == "" || sumsURL == "" {
		s.set(func(st *UpdateStatus) { st.State = "manual" })
		return
	}
	path, err := download(ctx, apkURL, sumsURL, latest)
	if err != nil {
		fail(err)
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

func expectedSum(ctx context.Context, url, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := updateHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	sc := bufio.NewScanner(io.LimitReader(res.Body, 1<<20))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("%s is missing from checksums.txt", name)
}

// CheckNow runs a check immediately and returns the resulting state.
func (s *UpdateService) CheckNow(ctx context.Context) UpdateStatus {
	s.run(ctx)
	return s.Status()
}

// Restart is a no-op on Android: the UI passes ApkPath to the system
// installer, which replaces and restarts the app.
func (s *UpdateService) Restart(ctx context.Context) error {
	if s.Status().State != "ready" {
		return errors.New("no update has been downloaded yet")
	}
	return nil
}

// GuardUpdaterHelper is only needed for the desktop updater.
func GuardUpdaterHelper() {}
