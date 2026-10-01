package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"nova/internal/nova"
)

// memFS is an in-memory stand-in for the filesystem API: stat, download,
// upload, mkdir, rename and delete.
type memFS struct {
	mu    sync.Mutex
	dirs  map[string]bool
	files map[string][]byte
}

func newMemFS(files map[string]string) *memFS {
	m := &memFS{dirs: map[string]bool{"/": true, "/me": true}, files: map[string][]byte{}}
	for p, data := range files {
		m.mkdirAll(nova.Parent(p))
		m.files[p] = []byte(data)
	}
	return m
}

func (m *memFS) mkdirAll(p string) {
	for ; p != "/" && p != ""; p = nova.Parent(p) {
		m.dirs[p] = true
	}
}

func (m *memFS) node(p string) nova.Node {
	if m.dirs[p] {
		return nova.Node{Name: nova.Base(p), Path: p, Type: "dir"}
	}
	return nova.Node{Name: nova.Base(p), Path: p, Type: "file", FileSize: int64(len(m.files[p]))}
}

func (m *memFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := nova.CleanPath(strings.TrimPrefix(r.URL.Path, "/api/filesystem"))
	_, isFile := m.files[p]
	exists := isFile || m.dirs[p]
	notFound := func() { w.WriteHeader(http.StatusNotFound) }
	switch r.Method {
	case http.MethodGet:
		if !exists {
			notFound()
			return
		}
		if r.URL.RawQuery == "stat" {
			l := nova.Listing{Path: []nova.Node{m.node(p)}, Children: []nova.Node{}}
			for q := range m.dirs {
				if q != p && nova.Parent(q) == p {
					l.Children = append(l.Children, m.node(q))
				}
			}
			for q := range m.files {
				if nova.Parent(q) == p {
					l.Children = append(l.Children, m.node(q))
				}
			}
			_ = json.NewEncoder(w).Encode(l)
			return
		}
		_, _ = w.Write(m.files[p])
	case http.MethodPut:
		if !m.dirs[nova.Parent(p)] {
			notFound()
			return
		}
		m.files[p], _ = io.ReadAll(r.Body)
	case http.MethodPost:
		_ = r.ParseForm()
		switch r.Form.Get("action") {
		case "mkdirall":
			m.mkdirAll(p)
		case "rename":
			to := nova.CleanPath(r.Form.Get("target"))
			if !exists || !m.dirs[nova.Parent(to)] {
				notFound()
				return
			}
			if _, ok := m.files[to]; ok || m.dirs[to] {
				w.WriteHeader(http.StatusConflict)
				return
			}
			m.move(p, to)
		}
	case http.MethodDelete:
		if !exists {
			notFound()
			return
		}
		m.remove(p)
	}
}

func (m *memFS) move(from, to string) {
	for q, data := range m.files {
		if q == from || strings.HasPrefix(q, from+"/") {
			delete(m.files, q)
			m.files[to+strings.TrimPrefix(q, from)] = data
		}
	}
	for q := range m.dirs {
		if q == from || strings.HasPrefix(q, from+"/") {
			delete(m.dirs, q)
			m.dirs[to+strings.TrimPrefix(q, from)] = true
		}
	}
}

func (m *memFS) remove(p string) {
	for q := range m.files {
		if q == p || strings.HasPrefix(q, p+"/") {
			delete(m.files, q)
		}
	}
	for q := range m.dirs {
		if q == p || strings.HasPrefix(q, p+"/") {
			delete(m.dirs, q)
		}
	}
}

func (m *memFS) paths(prefix string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for q := range m.files {
		if strings.HasPrefix(q, prefix) {
			out = append(out, q)
		}
	}
	sort.Strings(out)
	return out
}

func (m *memFS) read(p string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.files[p])
}

func newTrashTest(t *testing.T, files map[string]string) (*FilesService, *memFS) {
	t.Helper()
	fs := newMemFS(files)
	srv := httptest.NewServer(fs)
	t.Cleanup(srv.Close)
	return NewFilesService(nova.NewClient(srv.URL, "key")), fs
}

func TestTrashInfoRoundTrip(t *testing.T) {
	at := time.Date(2026, 9, 30, 14, 5, 6, 0, time.Local)
	orig := "/me/Docs/100% ü & more/a b.txt"
	b := formatTrashInfo(orig, at)
	want := "[Trash Info]\nPath=Docs/100%25%20%C3%BC%20%26%20more/a%20b.txt\nDeletionDate=2026-09-30T14:05:06\n"
	if string(b) != want {
		t.Fatalf("trashinfo =\n%s\nwant\n%s", b, want)
	}
	gotPath, gotAt, err := parseTrashInfo(b)
	if err != nil || gotPath != orig || !gotAt.Equal(at) {
		t.Fatalf("parse = %q %v %v, want %q %v", gotPath, gotAt, err, orig, at)
	}
}

func TestParseTrashInfo(t *testing.T) {
	cases := []struct {
		name, in, path string
		date           bool
		err            bool
	}{
		{"spec example date", "[Trash Info]\nPath=foo/bar/meow.bow-wow\nDeletionDate=20040831T22:32:08\n", "/me/foo/bar/meow.bow-wow", true, false},
		{"first occurrence wins", "[Trash Info]\nPath=a\nPath=b\nDeletionDate=2004-08-31T22:32:08\n", "/me/a", true, false},
		{"other groups ignored", "[Trash Info]\nDeletionDate=2004-08-31T22:32:08\n[Other]\nPath=x\n", "", true, false},
		{"absolute inside home", "[Trash Info]\nPath=/me/x.txt\n", "/me/x.txt", false, false},
		{"absolute outside home", "[Trash Info]\nPath=/etc/passwd\n", "", false, false},
		{"climbs out", "[Trash Info]\nPath=../x\n", "", false, false},
		{"points into trash", "[Trash Info]\nPath=.Trash/files/x\n", "", false, false},
		{"missing header", "Path=a\n", "", false, true},
	}
	for _, c := range cases {
		p, at, err := parseTrashInfo([]byte(c.in))
		if (err != nil) != c.err || p != c.path || at.IsZero() == c.date {
			t.Errorf("%s: got %q %v %v", c.name, p, at, err)
		}
	}
}

func TestTrashAndRestore(t *testing.T) {
	s, fs := newTrashTest(t, map[string]string{
		"/me/Docs/a.txt": "docs",
		"/me/a.txt":      "home",
		// An info file whose item is gone: its name must not be reused.
		"/me/.Trash/info/b.txt.trashinfo": "[Trash Info]\nPath=b.txt\n",
		"/me/b.txt":                       "b",
	})

	res, err := s.Trash([]string{"/me/Docs/a.txt", "/me/a.txt", "/me/b.txt", "/me/.Trash/files"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Done) != 3 || len(res.Errors) != 1 {
		t.Fatalf("trash result = %+v", res)
	}
	got := strings.Join(fs.paths("/me/.Trash/"), " ")
	want := "/me/.Trash/files/a.txt /me/.Trash/files/a.txt (copy) /me/.Trash/files/b.txt (copy) " +
		"/me/.Trash/info/a.txt (copy).trashinfo /me/.Trash/info/a.txt.trashinfo /me/.Trash/info/b.txt (copy).trashinfo /me/.Trash/info/b.txt.trashinfo"
	if got != want {
		t.Fatalf("trash contents =\n%s\nwant\n%s", got, want)
	}
	if info := fs.read("/me/.Trash/info/a.txt (copy).trashinfo"); !strings.HasPrefix(info, "[Trash Info]\nPath=a.txt\nDeletionDate=") {
		t.Fatalf("info = %q", info)
	}

	f, err := s.List(TrashDir)
	if err != nil {
		t.Fatal(err)
	}
	origins := map[string]string{}
	for _, e := range f.Children {
		if e.DeletedAt == nil {
			t.Errorf("%s has no deletion date", e.Path)
		}
		origins[e.Path] = e.OrigPath
	}
	if origins["/me/.Trash/files/a.txt"] != "/me/Docs/a.txt" || origins["/me/.Trash/files/a.txt (copy)"] != "/me/a.txt" {
		t.Fatalf("origins = %v", origins)
	}
	if n := s.TrashCount(); n != 3 {
		t.Fatalf("count = %d", n)
	}

	res, err = s.Restore([]string{"/me/.Trash/files/a.txt (copy)"})
	if err != nil || len(res.Done) != 1 || res.Done[0] != "/me/a.txt" {
		t.Fatalf("restore = %+v %v", res, err)
	}
	if fs.read("/me/a.txt") != "home" || fs.read("/me/.Trash/info/a.txt (copy).trashinfo") != "" {
		t.Fatal("restore did not move the file back and drop its info")
	}

	if _, err := s.Delete([]string{"/me/.Trash/files/a.txt"}); err != nil {
		t.Fatal(err)
	}
	if p := fs.paths("/me/.Trash/info/a.txt."); len(p) != 0 {
		t.Fatalf("info left after delete: %v", p)
	}
}

func TestLegacyTrashMigrates(t *testing.T) {
	s, fs := newTrashTest(t, map[string]string{
		"/me/.Trash/old.txt":         "old",
		"/me/.Trash/stray.txt":       "stray",
		"/me/.Trash/.trashinfo.json": `{"old.txt":{"orig":"/me/Docs/old.txt","deleted":"2026-09-01T10:00:00Z"}}`,
	})
	if n := s.TrashCount(); n != 2 {
		t.Fatalf("count = %d", n)
	}
	got := strings.Join(fs.paths("/me/.Trash/"), " ")
	want := "/me/.Trash/files/old.txt /me/.Trash/files/stray.txt /me/.Trash/info/old.txt.trashinfo /me/.Trash/info/stray.txt.trashinfo"
	if got != want {
		t.Fatalf("migrated =\n%s\nwant\n%s", got, want)
	}
	orig, at, err := parseTrashInfo([]byte(fs.read("/me/.Trash/info/old.txt.trashinfo")))
	if err != nil || orig != "/me/Docs/old.txt" || !at.Equal(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("migrated info = %q %v %v", orig, at, err)
	}
	if orig, _, _ := parseTrashInfo([]byte(fs.read("/me/.Trash/info/stray.txt.trashinfo"))); orig != "/me/stray.txt" {
		t.Fatalf("stray origin = %q", orig)
	}
}
