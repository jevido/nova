package services

import (
	"testing"

	"nova/internal/nova"
)

func TestSharingLink(t *testing.T) {
	s := NewFilesService(nova.NewClient("https://example.test", ""))
	pub := &nova.Permissions{Read: true}
	l := &nova.Listing{
		Path: []nova.Node{
			{ID: "home", Type: "dir", Path: "/me"},
			{ID: "pics", Type: "dir", Path: "/me/Pictures", Name: "Pictures", LinkPermissions: pub},
			{ID: "cat", Type: "file", Path: "/me/Pictures/My cat.jpg", Name: "My cat.jpg",
				UserPermissions: map[string]nova.Permissions{"bob": {Read: true}, "Alice": {Read: true}}},
		},
		BaseIndex: 2,
	}
	l.Context.CanShare = true
	sh := s.sharing(l)
	if sh.URL != "https://example.test/d/pics/My%20cat.jpg" || sh.Via != "/me/Pictures" {
		t.Fatalf("inherited link: %+v", sh)
	}
	if sh.DirectURL != "https://example.test/api/filesystem/pics/My%20cat.jpg" {
		t.Fatalf("direct link: %q", sh.DirectURL)
	}
	if sh.PeopleURL != "https://example.test/d/cat" {
		t.Fatalf("people link: %q", sh.PeopleURL)
	}
	if len(sh.People) != 2 || sh.People[0].Name != "Alice" || !sh.CanShare {
		t.Fatalf("people: %+v", sh)
	}

	l.Path[2].LinkPermissions = pub
	if sh = s.sharing(l); sh.URL != "https://example.test/d/cat" || sh.Via != "" {
		t.Fatalf("own link: %+v", sh)
	}

	l.Path[1].LinkPermissions, l.Path[2].LinkPermissions = nil, nil
	if sh = s.sharing(l); sh.URL != "" || sh.DirectURL != "" {
		t.Fatalf("not shared: %+v", sh)
	}
}
