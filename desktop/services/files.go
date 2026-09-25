package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"nova/internal/nova"
)

const (
	HomeDir   = "/me"
	TrashDir  = "/me/.Trash"
	trashInfo = "/me/.Trash/.trashinfo.json"
)

// Entry is a filesystem node as the UI sees it.
type Entry struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
	Mime     string    `json:"mime"`
	Modified time.Time `json:"modified"`
	Created  time.Time `json:"created"`
	Mode     string    `json:"mode"`
	Owner    string    `json:"owner"`
	SHA256   string    `json:"sha256"`
	Shared   bool      `json:"shared"`
	// Trash only: where the item came from.
	OrigPath  string     `json:"origPath,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type Folder struct {
	Path     string  `json:"path"`
	Crumbs   []Entry `json:"crumbs"`
	Children []Entry `json:"children"`
	CanWrite bool    `json:"canWrite"`
}

func toEntry(n nova.Node) Entry {
	return Entry{
		ID: n.ID, Name: n.Name, Path: nova.CleanPath(n.Path), IsDir: n.IsDir(), Size: n.FileSize,
		Mime: n.FileType, Modified: n.Modified, Created: n.Created, Mode: n.ModeString,
		Owner: n.CreatedBy, SHA256: n.SHA256,
		Shared: n.LinkPermissions != nil && n.LinkPermissions.Read,
	}
}

// FilesService exposes filesystem operations.
type FilesService struct {
	client  *nova.Client
	trashMu sync.Mutex
}

func NewFilesService(client *nova.Client) *FilesService { return &FilesService{client: client} }

func isNotFound(err error) bool {
	var apiErr *nova.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

// checkPath rejects paths outside the user's home.
func checkPath(p string) (string, error) {
	c := nova.CleanPath(p)
	if c != HomeDir && !strings.HasPrefix(c, HomeDir+"/") {
		return "", fmt.Errorf("path %q is outside your files", p)
	}
	return c, nil
}

func checkName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "" || name == "." || name == "..":
		return "", errors.New("enter a name")
	case strings.Contains(name, "/"):
		return "", errors.New(`names cannot contain "/"`)
	case len(name) > 255:
		return "", errors.New("name is too long")
	}
	return name, nil
}

func (s *FilesService) List(p string) (*Folder, error) {
	p, err := checkPath(p)
	if err != nil {
		return nil, err
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	l, err := s.client.Stat(ctx, p)
	if err != nil && p == TrashDir && isNotFound(err) {
		// The trash folder is created lazily on first use.
		return &Folder{Path: p, Crumbs: []Entry{{Name: "Trash", Path: p, IsDir: true}}, Children: []Entry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	f := &Folder{Path: p, CanWrite: l.Permissions.Write || l.Permissions.Owner}
	for _, n := range l.Path {
		f.Crumbs = append(f.Crumbs, toEntry(n))
	}
	f.Children = make([]Entry, 0, len(l.Children))
	for _, n := range l.Children {
		f.Children = append(f.Children, toEntry(n))
	}
	if p == TrashDir {
		s.decorateTrash(ctx, f)
	}
	return f, nil
}

func (s *FilesService) Stat(p string) (*Entry, error) {
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
	if l.BaseIndex < 0 || l.BaseIndex >= len(l.Path) {
		return nil, errors.New("unexpected server response")
	}
	e := toEntry(l.Path[l.BaseIndex])
	return &e, nil
}

// StatMany returns entries for the given paths, skipping ones that no longer exist.
func (s *FilesService) StatMany(paths []string) ([]Entry, error) {
	ctx, cancel := ctxTimeout()
	defer cancel()
	out := make([]*Entry, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, raw := range paths {
		p, err := checkPath(raw)
		if err != nil {
			continue
		}
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			l, err := s.client.Stat(ctx, p)
			if err != nil || l.BaseIndex < 0 || l.BaseIndex >= len(l.Path) {
				return
			}
			e := toEntry(l.Path[l.BaseIndex])
			out[i] = &e
		}(i, p)
	}
	wg.Wait()
	res := make([]Entry, 0, len(out))
	for _, e := range out {
		if e != nil {
			res = append(res, *e)
		}
	}
	return res, nil
}

// FolderSize sums file sizes below p. Directories only report their own size.
type FolderSize struct {
	Bytes   int64 `json:"bytes"`
	Files   int   `json:"files"`
	Folders int   `json:"folders"`
}

func (s *FilesService) Measure(p string) (*FolderSize, error) {
	p, err := checkPath(p)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out := &FolderSize{}
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if depth > 32 {
			return nil
		}
		l, err := s.client.Stat(ctx, dir)
		if err != nil {
			return err
		}
		for _, n := range l.Children {
			if n.IsDir() {
				out.Folders++
				if err := walk(nova.CleanPath(n.Path), depth+1); err != nil {
					return err
				}
			} else {
				out.Files++
				out.Bytes += n.FileSize
			}
		}
		return nil
	}
	return out, walk(p, 0)
}

// CreateFolder creates dir/name, returning the new path.
func (s *FilesService) CreateFolder(dir, name string) (string, error) {
	dir, err := checkPath(dir)
	if err != nil {
		return "", err
	}
	if name, err = checkName(name); err != nil {
		return "", err
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	p := nova.Join(dir, name)
	return p, s.client.Mkdir(ctx, p)
}

// Rename renames p within its directory.
func (s *FilesService) Rename(p, newName string) (string, error) {
	p, err := checkPath(p)
	if err != nil {
		return "", err
	}
	if p == HomeDir {
		return "", errors.New("the home folder cannot be renamed")
	}
	if newName, err = checkName(newName); err != nil {
		return "", err
	}
	target := nova.Join(nova.Parent(p), newName)
	if target == p {
		return p, nil
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	return target, s.client.Rename(ctx, p, target)
}

// existingNames lists the child names of dir.
func (s *FilesService) existingNames(ctx context.Context, dir string) (map[string]bool, error) {
	l, err := s.client.Stat(ctx, dir)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(l.Children))
	for _, c := range l.Children {
		names[c.Name] = true
	}
	return names, nil
}

// uniqueName returns name, or "name (copy).ext", "name (copy 2).ext"... like Nautilus.
func uniqueName(name string, taken map[string]bool, isDir bool) string {
	if !taken[name] {
		return name
	}
	stem, ext := name, ""
	if !isDir {
		if e := path.Ext(name); e != "" && e != name {
			stem, ext = strings.TrimSuffix(name, e), e
		}
	}
	for i := 1; ; i++ {
		suffix := " (copy)"
		if i > 1 {
			suffix = fmt.Sprintf(" (copy %d)", i)
		}
		if c := stem + suffix + ext; !taken[c] {
			return c
		}
	}
}

// OpResult reports partial failure of a batch operation.
type OpResult struct {
	Done   []string `json:"done"`
	Errors []string `json:"errors"`
	// From holds the source path of each Done entry, for operations that move.
	From []string `json:"from,omitempty"`
}

func (r *OpResult) fail(p string, err error) {
	r.Errors = append(r.Errors, fmt.Sprintf("%s: %v", nova.Base(p), err))
}

// Move moves paths into destDir. Name clashes get a unique name.
func (s *FilesService) Move(paths []string, destDir string) (*OpResult, error) {
	destDir, err := checkPath(destDir)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	taken, err := s.existingNames(ctx, destDir)
	if err != nil {
		return nil, err
	}
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p, err := checkPath(raw)
		if err != nil || p == HomeDir {
			res.fail(raw, errors.New("cannot be moved"))
			continue
		}
		if destDir == p || strings.HasPrefix(destDir, p+"/") {
			res.fail(p, errors.New("cannot move a folder into itself"))
			continue
		}
		if nova.Parent(p) == destDir {
			continue // moving onto itself is a no-op
		}
		name := uniqueName(nova.Base(p), taken, false)
		target := nova.Join(destDir, name)
		if err := s.client.Rename(ctx, p, target); err != nil {
			res.fail(p, err)
			continue
		}
		taken[name] = true
		res.Done = append(res.Done, target)
		res.From = append(res.From, p)
	}
	return res, nil
}

// Delete permanently deletes paths.
func (s *FilesService) Delete(paths []string) (*OpResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p, err := checkPath(raw)
		if err != nil || p == HomeDir || p == TrashDir {
			res.fail(raw, errors.New("cannot be deleted"))
			continue
		}
		if err := s.client.Delete(ctx, p, true); err != nil {
			res.fail(p, err)
			continue
		}
		res.Done = append(res.Done, p)
	}
	if len(res.Done) > 0 {
		s.dropTrashInfo(ctx, res.Done)
	}
	return res, nil
}

// ---- Trash ----

type trashRecord struct {
	OrigPath  string    `json:"orig"`
	DeletedAt time.Time `json:"deleted"`
}

// readTrashInfo loads the trash index. A missing index is empty; any other
// failure is an error so callers never overwrite the index with a blank one.
func (s *FilesService) readTrashInfo(ctx context.Context) (map[string]trashRecord, error) {
	info := map[string]trashRecord{}
	res, err := s.client.Do(ctx, http.MethodGet, s.client.FileURL(trashInfo, ""), nil, nil)
	if isNotFound(err) {
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read trash index: %w", err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&info); err != nil {
		return nil, fmt.Errorf("trash index is damaged: %w", err)
	}
	return info, nil
}

func (s *FilesService) writeTrashInfo(ctx context.Context, info map[string]trashRecord) error {
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return s.client.Upload(ctx, trashInfo, bytes.NewReader(b), int64(len(b)), "application/json")
}

func (s *FilesService) decorateTrash(ctx context.Context, f *Folder) {
	info, _ := s.readTrashInfo(ctx) // listing still works without origins
	kept := f.Children[:0]
	for _, c := range f.Children {
		if c.Path == trashInfo {
			continue
		}
		if r, ok := info[c.Name]; ok {
			c.OrigPath = r.OrigPath
			t := r.DeletedAt
			c.DeletedAt = &t
		}
		kept = append(kept, c)
	}
	f.Children = kept
}

func (s *FilesService) dropTrashInfo(ctx context.Context, deleted []string) {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	info, err := s.readTrashInfo(ctx)
	if err != nil {
		return
	}
	changed := false
	for _, p := range deleted {
		if nova.Parent(p) == TrashDir {
			if _, ok := info[nova.Base(p)]; ok {
				delete(info, nova.Base(p))
				changed = true
			}
		}
	}
	if changed {
		_ = s.writeTrashInfo(ctx, info)
	}
}

// Trash moves paths into the trash folder and remembers where they came from.
func (s *FilesService) Trash(paths []string) (*OpResult, error) {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := s.client.MkdirAll(ctx, TrashDir); err != nil {
		return nil, err
	}
	taken, err := s.existingNames(ctx, TrashDir)
	if err != nil {
		return nil, err
	}
	info, err := s.readTrashInfo(ctx)
	if err != nil {
		return nil, err
	}
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p, err := checkPath(raw)
		if err != nil || p == HomeDir || p == TrashDir || strings.HasPrefix(p, TrashDir+"/") {
			res.fail(raw, errors.New("cannot be moved to the trash"))
			continue
		}
		name := uniqueName(nova.Base(p), taken, true)
		target := nova.Join(TrashDir, name)
		if err := s.client.Rename(ctx, p, target); err != nil {
			res.fail(p, err)
			continue
		}
		taken[name] = true
		info[name] = trashRecord{OrigPath: p, DeletedAt: time.Now().UTC()}
		res.Done = append(res.Done, p)
	}
	if len(res.Done) > 0 {
		if err := s.writeTrashInfo(ctx, info); err != nil {
			res.Errors = append(res.Errors, "could not save trash info: "+err.Error())
		}
	}
	return res, nil
}

// Restore moves trashed items back to where they came from.
func (s *FilesService) Restore(paths []string) (*OpResult, error) {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	info, err := s.readTrashInfo(ctx)
	if err != nil {
		return nil, err
	}
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p := nova.CleanPath(raw)
		if nova.Parent(p) != TrashDir {
			res.fail(raw, errors.New("not in the trash"))
			continue
		}
		name := nova.Base(p)
		orig, err := checkPath(info[name].OrigPath)
		if err != nil || orig == HomeDir || strings.HasPrefix(orig, TrashDir+"/") {
			orig = nova.Join(HomeDir, name)
		}
		dir := nova.Parent(orig)
		if err := s.client.MkdirAll(ctx, dir); err != nil {
			res.fail(p, err)
			continue
		}
		taken, err := s.existingNames(ctx, dir)
		if err != nil {
			res.fail(p, err)
			continue
		}
		target := nova.Join(dir, uniqueName(nova.Base(orig), taken, true))
		if err := s.client.Rename(ctx, p, target); err != nil {
			res.fail(p, err)
			continue
		}
		delete(info, name)
		res.Done = append(res.Done, target)
	}
	if len(res.Done) > 0 {
		_ = s.writeTrashInfo(ctx, info)
	}
	return res, nil
}

// EmptyTrash permanently deletes everything in the trash.
func (s *FilesService) EmptyTrash() error {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := s.client.Delete(ctx, TrashDir, true)
	if isNotFound(err) {
		return nil
	}
	return err
}

// TrashCount returns the number of items in the trash.
func (s *FilesService) TrashCount() int {
	ctx, cancel := ctxTimeout()
	defer cancel()
	l, err := s.client.Stat(ctx, TrashDir)
	if err != nil {
		return 0
	}
	n := 0
	for _, c := range l.Children {
		if nova.CleanPath(c.Path) != trashInfo {
			n++
		}
	}
	return n
}

// ---- Search & sharing ----

// Search finds entries below dir whose name matches term.
func (s *FilesService) Search(dir, term string) ([]Entry, error) {
	dir, err := checkPath(dir)
	if err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	if term == "" {
		return []Entry{}, nil
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	paths, err := s.client.Search(ctx, dir, term, 100)
	if err != nil {
		return nil, err
	}
	// The search API only returns paths; stat them concurrently.
	out := make([]*Entry, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, p := range paths {
		if p == trashInfo || strings.HasPrefix(p, TrashDir+"/") {
			continue
		}
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			l, err := s.client.Stat(ctx, p)
			if err != nil || l.BaseIndex >= len(l.Path) {
				return
			}
			e := toEntry(l.Path[l.BaseIndex])
			out[i] = &e
		}(i, p)
	}
	wg.Wait()
	res := make([]Entry, 0, len(out))
	for _, e := range out {
		if e != nil {
			res = append(res, *e)
		}
	}
	sort.SliceStable(res, func(i, j int) bool { return len(res[i].Path) < len(res[j].Path) })
	return res, nil
}

// Share enables or disables the public link and returns the link URL.
func (s *FilesService) Share(p string, shared bool) (string, error) {
	p, err := checkPath(p)
	if err != nil {
		return "", err
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	n, err := s.client.SetShared(ctx, p, shared)
	if err != nil {
		return "", err
	}
	if !shared || n.ID == "" {
		return "", nil
	}
	return s.client.BaseURL() + "/view/" + n.ID, nil
}
