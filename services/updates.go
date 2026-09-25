package services

import (
	"sync"
	"time"

	"nova/internal/version"
)

// EventUpdate carries an UpdateStatus whenever the update state changes.
const EventUpdate = "update"

const (
	firstCheckDelay     = 5 * time.Second
	updateCheckInterval = 6 * time.Hour
)

// UpdateStatus is what the UI shows about updates.
type UpdateStatus struct {
	CurrentVersion string `json:"currentVersion"`
	// State is one of: disabled, idle, checking, up-to-date, downloading,
	// ready, manual, error.
	State string `json:"state"`
	// LatestVersion is set when a newer release was found.
	LatestVersion string `json:"latestVersion"`
	Notes         string `json:"notes"`
	ReleaseURL    string `json:"releaseUrl"`
	Error         string `json:"error"`
	CheckedAt     string `json:"checkedAt"`
	// PackageManaged is set when Nova was installed by a package manager
	// (pacman, apt, dnf) and should be updated with it.
	PackageManaged bool `json:"packageManaged"`
	// ApkPath is the downloaded update on Android, handed to the installer.
	ApkPath string `json:"apkPath"`
}

// UpdateService checks GitHub Releases in the background and prepares updates.
// The platform-specific parts live in updates_desktop.go and updates_android.go.
type UpdateService struct {
	mu      sync.Mutex
	status  UpdateStatus
	enabled bool
	running bool
}

func NewUpdateService() *UpdateService {
	return &UpdateService{status: UpdateStatus{CurrentVersion: version.Version, State: "disabled"}}
}

func (s *UpdateService) set(fn func(st *UpdateStatus)) {
	s.mu.Lock()
	fn(&s.status)
	st := s.status
	s.mu.Unlock()
	emit(EventUpdate, st)
}

// Status returns the current update state.
func (s *UpdateService) Status() UpdateStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// beginRun marks a check as running; false means one is already in progress
// or an update is already waiting.
func (s *UpdateService) beginRun() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled || s.running || s.status.State == "ready" {
		return false
	}
	s.running = true
	return true
}

func (s *UpdateService) endRun() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// OpenReleasePage opens the release notes of the available version in the browser.
func (s *UpdateService) OpenReleasePage() error {
	url := s.Status().ReleaseURL
	if url == "" {
		url = "https://github.com/" + version.Repo + "/releases/latest"
	}
	return openURL(url)
}
