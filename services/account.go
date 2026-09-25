package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nova/internal/config"
	"nova/internal/nova"
)

// BookmarksFile is where the nova.storage web interface keeps its bookmarks.
// The sidebar uses the same file, so bookmarks follow you between the web
// and every device.
const BookmarksFile = HomeDir + "/.nova/bookmarks.json"

// SidebarFile keeps what only Nova's sidebar has, so the web interface's
// bookmarks file stays in the format it expects.
const SidebarFile = HomeDir + "/.nova/sidebar.json"

// sidebarLayout is SidebarFile: each divider sits right after the bookmark
// with path After, or at the top when After is empty.
type sidebarLayout struct {
	Dividers []dividerAnchor `json:"dividers"`
}

type dividerAnchor struct {
	After string `json:"after"`
}

// dividerAnchors records where the dividers of list are.
func dividerAnchors(list []config.Bookmark) []dividerAnchor {
	out := []dividerAnchor{}
	after := ""
	for _, b := range list {
		if b.Divider {
			out = append(out, dividerAnchor{After: after})
		} else {
			after = b.Path
		}
	}
	return out
}

// placeDividers puts dividers back between bookmarks. Dividers whose
// bookmark is gone are dropped.
func placeDividers(list []config.Bookmark, anchors []dividerAnchor) []config.Bookmark {
	count := map[string]int{}
	for _, d := range anchors {
		count[d.After]++
	}
	out := make([]config.Bookmark, 0, len(list)+len(anchors))
	n := 0
	add := func(after string) {
		for range count[after] {
			n++
			out = append(out, config.Bookmark{Path: fmt.Sprintf("divider:%d", n), Divider: true})
		}
	}
	add("")
	for _, b := range list {
		out = append(out, b)
		add(b.Path)
	}
	return out
}

// remoteBookmark is one entry of BookmarksFile, in the web interface's format.
type remoteBookmark struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Label string `json:"label"`
}

// RecommendedFolders is the folder layout offered in Settings. Creating them
// never touches anything that already exists.
var RecommendedFolders = []string{"Documents", "Downloads", "Music", "Pictures", "Videos", "Projects", "Backups"}

// AccountService backs the settings page: usage statistics, synced bookmarks
// and the recommended folder layout.
type AccountService struct {
	client *nova.Client
	store  *config.Store
}

func NewAccountService(client *nova.Client, store *config.Store) *AccountService {
	return &AccountService{client: client, store: store}
}

// UsageSeries is one chart's worth of data for the UI.
type UsageSeries struct {
	Kind       string    `json:"kind"`
	Timestamps []string  `json:"timestamps"`
	Amounts    []float64 `json:"amounts"`
	Total      float64   `json:"total"`
}

// Usage returns egress ("egress", bytes) or downloads ("downloads") over the
// last `minutes`, bucketed per `interval` minutes.
func (a *AccountService) Usage(kind string, minutes, interval int) (*UsageSeries, error) {
	if kind != "egress" && kind != "downloads" {
		return nil, fmt.Errorf("unknown statistic %q", kind)
	}
	if minutes <= 0 || interval <= 0 || minutes/interval > 5000 {
		return nil, errors.New("invalid time range")
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	end := time.Now()
	s, err := a.client.TimeSeries(ctx, kind, end.Add(-time.Duration(minutes)*time.Minute), end, time.Duration(interval)*time.Minute)
	if err != nil {
		return nil, err
	}
	out := &UsageSeries{Kind: kind, Timestamps: s.Timestamps, Amounts: s.Amounts}
	if out.Timestamps == nil {
		out.Timestamps = []string{}
	}
	if out.Amounts == nil {
		out.Amounts = []float64{}
	}
	for _, v := range out.Amounts {
		out.Total += v
	}
	return out, nil
}

func (a *AccountService) readRemote(ctx context.Context) ([]remoteBookmark, bool, error) {
	res, err := a.client.Do(ctx, http.MethodGet, a.client.FileURL(BookmarksFile, ""), nil, nil)
	if isNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	var list []remoteBookmark
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&list); err != nil {
		return nil, false, fmt.Errorf("bookmarks file is damaged: %w", err)
	}
	return list, true, nil
}

// readLayout returns the dividers stored on the server, or ok false when
// there is no (readable) sidebar file.
func (a *AccountService) readLayout(ctx context.Context) ([]dividerAnchor, bool) {
	res, err := a.client.Do(ctx, http.MethodGet, a.client.FileURL(SidebarFile, ""), nil, nil)
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	var l sidebarLayout
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&l) != nil {
		return nil, false
	}
	return l.Dividers, true
}

func (a *AccountService) writeLayout(ctx context.Context, list []config.Bookmark) error {
	body, err := json.Marshal(sidebarLayout{Dividers: dividerAnchors(list)})
	if err != nil {
		return err
	}
	return a.client.Upload(ctx, SidebarFile, bytes.NewReader(body), int64(len(body)), "application/json")
}

// SyncBookmarks returns the bookmarks stored on the server and saves them
// locally. When the server has none yet, the local bookmarks are uploaded.
func (a *AccountService) SyncBookmarks() ([]config.Bookmark, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	remote, found, err := a.readRemote(ctx)
	if err != nil {
		return a.store.Get().Prefs.Bookmarks, err
	}
	if !found {
		local := a.store.Get().Prefs.Bookmarks
		if len(local) > 0 {
			if err := a.writeRemote(ctx, local, nil); err != nil {
				return local, err
			}
			return local, a.writeLayout(ctx, local)
		}
		return local, nil
	}
	list := []config.Bookmark{}
	for _, r := range remote {
		p := nova.CleanPath(r.Path)
		if p != HomeDir && !strings.HasPrefix(p, HomeDir+"/") {
			continue // e.g. shared folders of other users; kept on save
		}
		name := r.Label
		if name == "" {
			name = nova.Base(p)
		}
		list = append(list, config.Bookmark{Name: name, Path: p})
	}
	// Without a sidebar file (yet), keep the dividers this device has.
	anchors, ok := a.readLayout(ctx)
	if !ok {
		anchors = dividerAnchors(a.store.Get().Prefs.Bookmarks)
	}
	list = placeDividers(list, anchors)
	err = a.store.Update(func(c *config.Config) { c.Prefs.Bookmarks = list })
	return list, err
}

// SaveBookmarks stores the sidebar bookmarks locally and on the server.
// Server entries the sidebar can't show are kept as they are.
func (a *AccountService) SaveBookmarks(list []config.Bookmark) error {
	if list == nil {
		list = []config.Bookmark{}
	}
	if err := a.store.Update(func(c *config.Config) { c.Prefs.Bookmarks = list }); err != nil {
		return err
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	remote, _, err := a.readRemote(ctx)
	if err != nil {
		return err
	}
	if err := a.writeRemote(ctx, list, remote); err != nil {
		return err
	}
	return a.writeLayout(ctx, list)
}

func (a *AccountService) writeRemote(ctx context.Context, list []config.Bookmark, prev []remoteBookmark) error {
	ids := map[string]string{}
	var keep []remoteBookmark
	for _, r := range prev {
		p := nova.CleanPath(r.Path)
		if p == HomeDir || strings.HasPrefix(p, HomeDir+"/") {
			ids[p] = r.ID
		} else {
			keep = append(keep, r)
		}
	}
	out := make([]remoteBookmark, 0, len(list)+len(keep))
	for _, b := range list {
		if b.Divider {
			continue
		}
		id := ids[b.Path]
		if id == "" {
			// The web interface removes bookmarks by node ID.
			if l, err := a.client.Stat(ctx, b.Path); err == nil && len(l.Path) > 0 {
				id = l.Path[len(l.Path)-1].ID
			}
		}
		out = append(out, remoteBookmark{ID: id, Path: b.Path, Icon: "bookmark", Label: b.Name})
	}
	out = append(out, keep...)
	body, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if err := a.client.MkdirAll(ctx, nova.Parent(BookmarksFile)); err != nil {
		return err
	}
	return a.client.Upload(ctx, BookmarksFile, bytes.NewReader(body), int64(len(body)), "application/json")
}

// FolderStatus says whether a recommended folder exists already.
type FolderStatus struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// RecommendedFolders lists the recommended folders and which already exist.
func (a *AccountService) RecommendedFolders() ([]FolderStatus, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	l, err := a.client.Stat(ctx, HomeDir)
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	for _, c := range l.Children {
		taken[strings.ToLower(c.Name)] = true
	}
	out := make([]FolderStatus, len(RecommendedFolders))
	for i, n := range RecommendedFolders {
		out[i] = FolderStatus{Name: n, Path: nova.Join(HomeDir, n), Exists: taken[strings.ToLower(n)]}
	}
	return out, nil
}

// CreateRecommendedFolders creates the recommended folders that don't exist
// yet and returns the paths it created. It never deletes or renames anything.
func (a *AccountService) CreateRecommendedFolders() ([]string, error) {
	status, err := a.RecommendedFolders()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	created := []string{}
	for _, f := range status {
		if f.Exists {
			continue
		}
		// Mkdir (not MkdirAll) fails instead of touching a folder that
		// appeared in the meantime.
		if err := a.client.Mkdir(ctx, f.Path); err != nil {
			return created, fmt.Errorf("could not create %s: %w", f.Name, err)
		}
		created = append(created, f.Path)
	}
	return created, nil
}
