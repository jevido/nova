package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"nova/internal/config"
	"nova/internal/nova"
)

// fakeFS is a tiny in-memory stand-in for the filesystem API.
type fakeFS struct {
	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]bool
}

func (f *fakeFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := nova.CleanPath(strings.TrimPrefix(r.URL.Path, "/api/filesystem"))
	switch {
	case r.Method == http.MethodGet && r.URL.RawQuery == "stat":
		if !f.dirs[p] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		l := nova.Listing{Path: []nova.Node{{ID: "id:" + p, Path: p, Type: "dir"}}}
		for d := range f.dirs {
			if nova.Parent(d) == p && d != p {
				l.Children = append(l.Children, nova.Node{Name: nova.Base(d), Path: d, Type: "dir"})
			}
		}
		_ = json.NewEncoder(w).Encode(l)
	case r.Method == http.MethodGet:
		b, ok := f.files[p]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	case r.Method == http.MethodPut:
		if !f.dirs[nova.Parent(p)] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		f.files[p] = b
	case r.Method == http.MethodPost:
		_ = r.ParseForm()
		switch r.Form.Get("action") {
		case "mkdirall":
			for q := p; q != "/"; q = nova.Parent(q) {
				f.dirs[q] = true
			}
		case "mkdir":
			if f.dirs[p] {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			f.dirs[p] = true
		}
	}
}

func newTestAccount(t *testing.T) (*AccountService, *fakeFS) {
	t.Helper()
	fs := &fakeFS{files: map[string][]byte{}, dirs: map[string]bool{"/me": true, "/me/Music": true, "/me/Work": true}}
	srv := httptest.NewServer(fs)
	t.Cleanup(srv.Close)
	store, err := config.Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewAccountService(nova.NewClient(srv.URL, "key"), store), fs
}

func TestBookmarksRoundTripKeepsForeignEntries(t *testing.T) {
	a, fs := newTestAccount(t)
	fs.dirs["/me/.nova"] = true
	fs.files[BookmarksFile] = []byte(`[{"id":"x","path":"/me/Work","icon":"bookmark","label":"Work"},{"id":"y","path":"/d/abc123","icon":"bookmark","label":"Shared"}]`)

	got, err := a.SyncBookmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "/me/Work" || got[0].Name != "Work" {
		t.Fatalf("SyncBookmarks = %+v", got)
	}

	if err := a.SaveBookmarks([]config.Bookmark{{Name: "Music", Path: "/me/Music"}, {Name: "Work stuff", Path: "/me/Work"}}); err != nil {
		t.Fatal(err)
	}
	var saved []remoteBookmark
	if err := json.Unmarshal(fs.files[BookmarksFile], &saved); err != nil {
		t.Fatal(err)
	}
	want := []remoteBookmark{
		{ID: "id:/me/Music", Path: "/me/Music", Icon: "bookmark", Label: "Music"},
		{ID: "x", Path: "/me/Work", Icon: "bookmark", Label: "Work stuff"},
		{ID: "y", Path: "/d/abc123", Icon: "bookmark", Label: "Shared"},
	}
	if len(saved) != len(want) {
		t.Fatalf("saved %+v", saved)
	}
	for i := range want {
		if saved[i] != want[i] {
			t.Errorf("saved[%d] = %+v, want %+v", i, saved[i], want[i])
		}
	}
}

func TestDividersSyncBesideBookmarks(t *testing.T) {
	a, fs := newTestAccount(t)
	list := []config.Bookmark{
		{Path: "divider:a", Divider: true},
		{Name: "Music", Path: "/me/Music"},
		{Path: "divider:b", Divider: true},
		{Path: "divider:c", Divider: true},
		{Name: "Work", Path: "/me/Work"},
	}
	if err := a.SaveBookmarks(list); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(fs.files[BookmarksFile]), "divider") {
		t.Fatalf("dividers leaked into the web bookmarks: %s", fs.files[BookmarksFile])
	}
	// Another device syncs: same shape back, dividers after the same bookmarks.
	_ = a.store.Update(func(c *config.Config) { c.Prefs.Bookmarks = nil })
	got, err := a.SyncBookmarks()
	if err != nil {
		t.Fatal(err)
	}
	var shape []string
	for _, b := range got {
		if b.Divider {
			shape = append(shape, "-")
		} else {
			shape = append(shape, b.Path)
		}
	}
	if s := strings.Join(shape, " "); s != "- /me/Music - - /me/Work" {
		t.Fatalf("synced %q", s)
	}
}

func TestSyncBookmarksUploadsLocalWhenServerHasNone(t *testing.T) {
	a, fs := newTestAccount(t)
	_ = a.store.Update(func(c *config.Config) { c.Prefs.Bookmarks = []config.Bookmark{{Name: "Work", Path: "/me/Work"}} })
	if _, err := a.SyncBookmarks(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fs.files[BookmarksFile]), `"/me/Work"`) {
		t.Fatalf("local bookmarks not uploaded: %s", fs.files[BookmarksFile])
	}
}

func TestCreateRecommendedFoldersOnlyAddsMissing(t *testing.T) {
	a, fs := newTestAccount(t)
	created, err := a.CreateRecommendedFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != len(RecommendedFolders)-1 {
		t.Fatalf("created %v", created)
	}
	for _, p := range created {
		if p == "/me/Music" {
			t.Fatal("recreated an existing folder")
		}
	}
	if !fs.dirs["/me/Work"] || !fs.dirs["/me/Pictures"] {
		t.Fatal("folders missing afterwards")
	}
	again, err := a.CreateRecommendedFolders()
	if err != nil || len(again) != 0 {
		t.Fatalf("second run created %v, %v", again, err)
	}
}
