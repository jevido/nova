package services

import (
	"errors"
	"net/url"
	"sort"
	"strings"

	"nova/internal/nova"
)

// Access is what someone may do with a shared file or folder. Write and
// Delete only mean something for folders.
type Access struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}

// Person is a nova.storage user the item is shared with.
type Person struct {
	Name   string `json:"name"`
	Access Access `json:"access"`
}

// Sharing describes who can reach an item, as the Share dialog shows it.
type Sharing struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	// CanShare is false when the account's plan doesn't include sharing.
	CanShare bool `json:"canShare"`
	// Link is what anyone with this item's own link may do.
	Link Access `json:"link"`
	// URL opens the item in the browser, through its own link or that of a
	// shared parent folder. Empty when neither is public.
	URL string `json:"url"`
	// DirectURL downloads a file straight away (a hotlink).
	DirectURL string `json:"directUrl"`
	// PeopleURL is the item's own link, for the people it is shared with. It
	// only works for them once they sign in, unless the item is public.
	PeopleURL string `json:"peopleUrl"`
	// Via is the path of the parent folder whose link URL goes through, when
	// the item isn't public itself.
	Via    string   `json:"via"`
	People []Person `json:"people"`
	// Abuse is set when the item was reported and taken down.
	Abuse string `json:"abuse"`
}

func isPublic(n nova.Node) bool { return n.LinkPermissions != nil && n.LinkPermissions.Read }

func toAccess(p nova.Permissions) Access { return Access{Read: p.Read, Write: p.Write, Delete: p.Delete} }

func fromAccess(a Access) nova.Permissions {
	return nova.Permissions{Read: a.Read, Write: a.Write, Delete: a.Delete}
}

// Sharing returns the sharing settings of p.
func (s *FilesService) Sharing(p string) (*Sharing, error) {
	p, err := checkPath(p)
	if err != nil {
		return nil, err
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	l, err := s.client.Stat(ctx, p)
	if err != nil {
		return nil, err
	}
	return s.sharing(l), nil
}

func (s *FilesService) sharing(l *nova.Listing) *Sharing {
	chain := l.Path[:min(l.BaseIndex+1, len(l.Path))]
	n := chain[len(chain)-1]
	sh := &Sharing{
		Path:     nova.CleanPath(n.Path),
		IsDir:    n.IsDir(),
		CanShare: l.Context.CanShare,
		People:   []Person{},
		Abuse:    n.AbuseType,
	}
	if n.LinkPermissions != nil {
		sh.Link = toAccess(*n.LinkPermissions)
	}
	if n.ID != "" {
		sh.PeopleURL = s.client.BaseURL() + "/d/" + url.PathEscape(n.ID)
	}
	for name, perm := range n.UserPermissions {
		sh.People = append(sh.People, Person{Name: name, Access: toAccess(perm)})
	}
	sort.Slice(sh.People, func(i, j int) bool { return strings.ToLower(sh.People[i].Name) < strings.ToLower(sh.People[j].Name) })

	// Like the website: the link goes through the nearest public folder, with
	// the rest of the path after its id.
	for i := len(chain) - 1; i >= 0; i-- {
		if !isPublic(chain[i]) || chain[i].ID == "" {
			continue
		}
		rel := url.PathEscape(chain[i].ID)
		for _, c := range chain[i+1:] {
			rel += "/" + url.PathEscape(c.Name)
		}
		base := s.client.BaseURL()
		sh.URL = base + "/d/" + rel
		if !sh.IsDir {
			sh.DirectURL = base + "/api/filesystem/" + rel
		}
		if i != len(chain)-1 {
			sh.Via = nova.CleanPath(chain[i].Path)
		}
		break
	}
	return sh
}

// SetLinkAccess changes what anyone with the link may do with p.
func (s *FilesService) SetLinkAccess(p string, a Access) (*Sharing, error) {
	p, err := checkPath(p)
	if err != nil {
		return nil, err
	}
	perm := fromAccess(a)
	return s.update(p, &perm, nil)
}

// SetPeople replaces the list of users p is shared with. Names can be
// usernames or e-mail addresses; the server rejects unknown ones.
func (s *FilesService) SetPeople(p string, people []Person) (*Sharing, error) {
	p, err := checkPath(p)
	if err != nil {
		return nil, err
	}
	users := make(map[string]nova.Permissions, len(people))
	for _, u := range people {
		name := strings.TrimSpace(u.Name)
		if name == "" {
			return nil, errors.New("enter a username or e-mail address")
		}
		users[name] = fromAccess(u.Access)
	}
	return s.update(p, nil, users)
}

// ShareLink returns a link to p, first making p public when neither it nor a
// parent folder is.
func (s *FilesService) ShareLink(p string) (string, error) {
	sh, err := s.Sharing(p)
	if err != nil {
		return "", err
	}
	if sh.URL == "" {
		a := sh.Link
		a.Read = true
		if sh, err = s.SetLinkAccess(p, a); err != nil {
			return "", err
		}
	}
	return sh.URL, nil
}

// StopSharing turns off p's public link. People it is shared with keep access.
func (s *FilesService) StopSharing(p string) error {
	_, err := s.SetLinkAccess(p, Access{})
	return err
}

func (s *FilesService) update(p string, link *nova.Permissions, users map[string]nova.Permissions) (*Sharing, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	if _, err := s.client.SetPermissions(ctx, p, link, users); err != nil {
		return nil, err
	}
	// The update response has the node but not its parents, which the link
	// depends on.
	l, err := s.client.Stat(ctx, p)
	if err != nil {
		return nil, err
	}
	return s.sharing(l), nil
}
