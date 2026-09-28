package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	// Shared: someone besides the owner can reach it. Public: through its link.
	Shared bool `json:"shared"`
	Public bool `json:"public"`
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
		Public: isPublic(n),
		Shared: isPublic(n) || len(n.UserPermissions) > 0,
	}
}

// FilesService exposes filesystem operations.
type FilesService struct {
	client  *nova.Client
	trashMu sync.Mutex

	// The last full walk of the home folder, reused briefly by Recent and
	// Shared so switching between them doesn't walk everything twice.
	walkMu   sync.Mutex
	walked   []Entry
	walkedAt time.Time
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

// The API has no listing of recent or shared items, so Nova walks the home
// folder (one request per folder, a few at a time) and filters the result.
const (
	walkMaxFolders = 5000
	walkTTL        = 30 * time.Second
	recentLimit    = 50
)

// walkAll returns every entry below home except the trash, reusing a walk
// from the last walkTTL.
func (s *FilesService) walkAll(fresh bool) ([]Entry, error) {
	s.walkMu.Lock()
	defer s.walkMu.Unlock()
	if !fresh && s.walked != nil && time.Since(s.walkedAt) < walkTTL {
		return s.walked, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var (
		mu      sync.Mutex
		out     []Entry
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 8)
		folders int
		first   error
	)
	var visit func(dir string)
	visit = func(dir string) {
		defer wg.Done()
		sem <- struct{}{}
		l, err := s.client.Stat(ctx, dir)
		<-sem
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if first == nil {
				first = err
			}
			return
		}
		for _, n := range l.Children {
			e := toEntry(n)
			if e.Path == TrashDir {
				continue
			}
			out = append(out, e)
			if e.IsDir && folders < walkMaxFolders {
				folders++
				wg.Add(1)
				go visit(e.Path)
			}
		}
	}
	wg.Add(1)
	go visit(HomeDir)
	wg.Wait()
	if out == nil && first != nil {
		return nil, first
	}
	s.walked, s.walkedAt = out, time.Now()
	return out, nil
}

// Recent returns the most recently modified files, newest first.
func (s *FilesService) Recent(fresh bool) ([]Entry, error) {
	all, err := s.walkAll(fresh)
	if err != nil {
		return nil, err
	}
	files := make([]Entry, 0, len(all))
	for _, e := range all {
		// Hidden files and anything in a hidden folder (such as Nova's own
		// .nova settings) aren't something you'd look for here.
		if !e.IsDir && !strings.Contains(e.Path, "/.") {
			files = append(files, e)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Modified.After(files[j].Modified) })
	if len(files) > recentLimit {
		files = files[:recentLimit]
	}
	return files, nil
}

// Shared returns the items that anyone with the link or named people can
// reach. Items inside a shared folder are left out; the folder stands for them.
func (s *FilesService) Shared(fresh bool) ([]Entry, error) {
	all, err := s.walkAll(fresh)
	if err != nil {
		return nil, err
	}
	sharedDirs := map[string]bool{}
	for _, e := range all {
		if e.Shared && e.IsDir {
			sharedDirs[e.Path] = true
		}
	}
	out := []Entry{}
	for _, e := range all {
		if !e.Shared {
			continue
		}
		inside := false
		for d := nova.Parent(e.Path); d != HomeDir && d != "/" && d != "."; d = nova.Parent(d) {
			if sharedDirs[d] {
				inside = true
				break
			}
		}
		if !inside {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

// viewCacheAge is how long files fetched for the phone viewer are kept.
const viewCacheAge = 24 * time.Hour

// CacheForView downloads a file into the app cache for the phone's viewer and
// returns its path relative to the cache ("view/…"). Android serves that
// file straight from disk, with seeking, instead of passing the whole file
// through the WebView bridge. A file already fetched is reused.
func (s *FilesService) CacheForView(p string) (string, error) {
	p, err := checkPath(p)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	l, err := s.client.Stat(ctx, p)
	if err != nil {
		return "", err
	}
	if l.BaseIndex < 0 || l.BaseIndex >= len(l.Path) || l.Path[l.BaseIndex].IsDir() {
		return "", errors.New("only files can be viewed")
	}
	n := l.Path[l.BaseIndex]
	h := sha1.Sum([]byte(p + "\x00" + n.SHA256 + "\x00" + n.Modified.String()))
	rel := path.Join("view", hex.EncodeToString(h[:])+strings.ToLower(path.Ext(n.Name)))
	dir := filepath.Join(cacheDir(), "view")
	local := filepath.Join(cacheDir(), filepath.FromSlash(rel))
	if st, err := os.Stat(local); err == nil && st.Size() == n.FileSize {
		return rel, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	pruneOld(dir, viewCacheAge)
	res, err := s.client.Do(ctx, http.MethodGet, s.client.FileURL(p, ""), nil, nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	tmp, err := os.CreateTemp(dir, ".part-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, res.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if err := os.Rename(tmp.Name(), local); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return rel, nil
}

// pruneOld removes files in dir that weren't touched within age.
func pruneOld(dir string, age time.Duration) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > age {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
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
	emitChanged(res.From, destDir)
	return res, nil
}

// emitChanged tells every window that the parents of paths, and extra, have
// changed, so another window showing one of them refreshes.
func emitChanged(paths []string, extra ...string) {
	if len(paths) == 0 {
		return
	}
	dirs := map[string]bool{}
	for _, p := range paths {
		dirs[nova.Parent(p)] = true
	}
	for _, d := range extra {
		dirs[d] = true
	}
	for d := range dirs {
		emit(EventChanged, d)
	}
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
	emitChanged(res.Done, TrashDir)
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
