//go:build android || ios

package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"nova/internal/version"
)

// Phones can't use the Wails updater (it swaps the running binary), so Nova
// talks to the GitHub API itself. The platform files implement run: Android
// downloads the new APK (updates_android.go), iOS only announces the release
// (updates_ios.go).

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

// The desktop BrowserManager shells out to xdg-open/open, which phones lack.
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

// checkLatest fetches the latest release and reports it when it is newer
// than this build. It sets the checking/up-to-date/error states itself; on
// a newer release it fills in the version details and leaves the state to
// the caller. ok is false when there is nothing more to do.
func (s *UpdateService) checkLatest(ctx context.Context) (rel ghRelease, latest string, ok bool) {
	s.set(func(st *UpdateStatus) { st.State = "checking"; st.Error = "" })
	now := time.Now().Format(time.RFC3339)
	if err := getJSON(ctx, "https://api.github.com/repos/"+version.Repo+"/releases/latest", &rel); err != nil {
		s.set(func(st *UpdateStatus) { st.State = "error"; st.Error = err.Error(); st.CheckedAt = now })
		return rel, "", false
	}
	latest = strings.TrimPrefix(rel.TagName, "v")
	if rel.Prerelease || !version.Newer(latest, version.Version) {
		s.set(func(st *UpdateStatus) { st.State = "up-to-date"; st.CheckedAt = now })
		return rel, latest, false
	}
	s.set(func(st *UpdateStatus) {
		st.LatestVersion = latest
		st.Notes = rel.Body
		st.ReleaseURL = rel.HTMLURL
		st.CheckedAt = now
	})
	return rel, latest, true
}

func (s *UpdateService) fail(err error) {
	s.set(func(st *UpdateStatus) { st.State = "error"; st.Error = err.Error() })
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

// GuardUpdaterHelper is only needed for the desktop updater.
func GuardUpdaterHelper() {}
