package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nova/internal/nova"
)

// treeFS serves a fixed tree for the stat calls the walk makes.
type treeFS map[string][]nova.Node

func (t treeFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := nova.CleanPath(strings.TrimPrefix(r.URL.Path, "/api/filesystem"))
	kids, ok := t[p]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(nova.Listing{Path: []nova.Node{{Path: p, Type: "dir"}}, Children: kids})
}

func TestRecentAndShared(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 9, n, 0, 0, 0, 0, time.UTC) }
	link := &nova.Permissions{Read: true}
	tree := treeFS{
		"/me": {
			{Name: "Docs", Path: "/me/Docs", Type: "dir", Modified: day(1)},
			{Name: "Team", Path: "/me/Team", Type: "dir", Modified: day(2), LinkPermissions: link},
			{Name: "Team b", Path: "/me/Team b", Type: "dir", Modified: day(3)},
			{Name: ".Trash", Path: "/me/.Trash", Type: "dir"},
			{Name: ".nova", Path: "/me/.nova", Type: "dir"},
			{Name: "old.txt", Path: "/me/old.txt", Type: "file", Modified: day(1)},
		},
		"/me/Docs":   {{Name: "new.pdf", Path: "/me/Docs/new.pdf", Type: "file", Modified: day(20)}},
		"/me/Team":   {{Name: "plan.md", Path: "/me/Team/plan.md", Type: "file", Modified: day(10), LinkPermissions: link}},
		"/me/Team b": {{Name: "x.md", Path: "/me/Team b/x.md", Type: "file", Modified: day(5), UserPermissions: map[string]nova.Permissions{"bob": {Read: true}}}},
		"/me/.nova":  {{Name: "sidebar.json", Path: "/me/.nova/sidebar.json", Type: "file", Modified: day(29)}},
		"/me/.Trash": {{Name: "gone.txt", Path: "/me/.Trash/gone.txt", Type: "file", Modified: day(28)}},
	}
	srv := httptest.NewServer(tree)
	t.Cleanup(srv.Close)
	s := NewFilesService(nova.NewClient(srv.URL, "key"))

	recent, err := s.Recent(true)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range recent {
		got = append(got, e.Path)
	}
	if want := "/me/Docs/new.pdf /me/Team/plan.md /me/Team b/x.md /me/old.txt"; strings.Join(got, " ") != want {
		t.Fatalf("recent = %v, want %s", got, want)
	}

	shared, err := s.Shared(false)
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, e := range shared {
		got = append(got, e.Path)
	}
	// plan.md sits inside the shared Team folder, so only the folder is listed.
	if want := "/me/Team b/x.md /me/Team"; strings.Join(got, " ") != want {
		t.Fatalf("shared = %v, want %s", got, want)
	}
}
