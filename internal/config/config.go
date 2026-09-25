// Package config persists desktop client settings in the user's config dir.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"nova/internal/platform"
)

// Bookmark is one sidebar row: a folder, or a divider line between rows.
// A divider's Path is only an id ("divider:…").
type Bookmark struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Divider bool   `json:"divider,omitempty"`
}

// Prefs are UI preferences owned by the frontend.
type Prefs struct {
	View         string     `json:"view"`        // "grid" | "list"
	ShowHidden   bool       `json:"showHidden"`  // show dotfiles
	SortBy       string     `json:"sortBy"`      // "name" | "size" | "modified" | "type"
	SortDesc     bool       `json:"sortDesc"`    //
	Zoom         int        `json:"zoom"`        // icon zoom level 0..4
	SidebarOpen  bool       `json:"sidebarOpen"` //
	Theme        string     `json:"theme"`       // "system" | "light" | "dark"
	Bookmarks    []Bookmark `json:"bookmarks"`   //
	Starred      []string   `json:"starred"`     // starred file and folder paths
	FoldersFirst bool       `json:"foldersFirst"`
	SeenVersion  string     `json:"seenVersion"` // last version whose What's New was shown
}

type Config struct {
	APIKey string `json:"apiKey"`
	// KeyFromLogin is true when the key was created by a password sign-in,
	// so it is revoked again on sign-out.
	KeyFromLogin bool  `json:"keyFromLogin"`
	Prefs        Prefs `json:"prefs"`
}

func DefaultPrefs() Prefs {
	return Prefs{View: "grid", SortBy: "name", Zoom: 1, SidebarOpen: true, Theme: "system", Bookmarks: []Bookmark{}, Starred: []string{}, FoldersFirst: true}
}

// Store loads and saves Config atomically to disk.
type Store struct {
	mu   sync.Mutex
	path string
	cfg  Config
}

func DefaultPath() (string, error) {
	dir, err := platform.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, cfg: Config{Prefs: DefaultPrefs()}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.cfg); err != nil {
		// A corrupt config should not brick the app; start fresh.
		s.cfg = Config{Prefs: DefaultPrefs()}
	}
	if s.cfg.Prefs.Bookmarks == nil {
		s.cfg.Prefs.Bookmarks = []Bookmark{}
	}
	if s.cfg.Prefs.Starred == nil {
		s.cfg.Prefs.Starred = []string{}
	}
	return s, nil
}

func (s *Store) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// Update applies fn to the config and saves it.
func (s *Store) Update(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.cfg)
	return s.save()
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	// The file holds an API key, keep it private to the user.
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
