package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"nova/internal/nova"
	"nova/internal/platform"
)

// EventTransfer carries a Transfer snapshot whenever a job changes.
const EventTransfer = "transfer"

// EventChanged tells the UI a remote folder changed.
const EventChanged = "fs:changed"

type TransferKind string

const (
	KindUpload   TransferKind = "upload"
	KindDownload TransferKind = "download"
	KindOpen     TransferKind = "open"
	KindCopy     TransferKind = "copy"
)

type TransferState string

const (
	StateQueued    TransferState = "queued"
	StateRunning   TransferState = "running"
	StateDone      TransferState = "done"
	StateFailed    TransferState = "failed"
	StateCancelled TransferState = "cancelled"
)

// Transfer is one batch job (upload of several files, download of a folder...).
type Transfer struct {
	ID         int64         `json:"id"`
	Kind       TransferKind  `json:"kind"`
	Title      string        `json:"title"`
	Dest       string        `json:"dest"`
	Current    string        `json:"current"`
	TotalBytes int64         `json:"totalBytes"`
	DoneBytes  int64         `json:"doneBytes"`
	Files      int           `json:"files"`
	DoneFiles  int           `json:"doneFiles"`
	State      TransferState `json:"state"`
	Error      string        `json:"error"`
	Started    time.Time     `json:"started"`
	RateBps    float64       `json:"rateBps"`
}

type job struct {
	mu       sync.Mutex
	t        Transfer
	cancel   context.CancelFunc
	ctx      context.Context
	run      func(*job) error
	lastEmit time.Time
	moved    atomic.Int64 // bytes moved, updated from io hot path
}

// TransferService runs uploads and downloads in the background.
type TransferService struct {
	client *nova.Client
	mu     sync.Mutex
	jobs   map[int64]*job
	order  []int64
	nextID int64
	sem    chan struct{}
}

func NewTransferService(client *nova.Client) *TransferService {
	return &TransferService{client: client, jobs: map[int64]*job{}, sem: make(chan struct{}, 2)}
}

func emit(name string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

func (j *job) snapshot() Transfer {
	j.mu.Lock()
	defer j.mu.Unlock()
	t := j.t
	t.DoneBytes = j.moved.Load()
	if el := time.Since(t.Started).Seconds(); el > 0.5 && t.State == StateRunning {
		t.RateBps = float64(t.DoneBytes) / el
	}
	return t
}

func (j *job) emit(force bool) {
	j.mu.Lock()
	if !force && time.Since(j.lastEmit) < 150*time.Millisecond {
		j.mu.Unlock()
		return
	}
	j.lastEmit = time.Now()
	j.mu.Unlock()
	emit(EventTransfer, j.snapshot())
}

func (j *job) update(fn func(t *Transfer)) {
	j.mu.Lock()
	fn(&j.t)
	j.mu.Unlock()
	j.emit(true)
}

// progressReader counts bytes flowing through it.
type progressReader struct {
	r io.Reader
	j *job
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.j.moved.Add(int64(n))
		p.j.emit(false)
	}
	return n, err
}

func (s *TransferService) enqueue(kind TransferKind, title, dest string, run func(*job) error) Transfer {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.nextID++
	j := &job{ctx: ctx, cancel: cancel, run: run,
		t: Transfer{ID: s.nextID, Kind: kind, Title: title, Dest: dest, State: StateQueued}}
	s.jobs[j.t.ID] = j
	s.order = append(s.order, j.t.ID)
	s.mu.Unlock()
	j.emit(true)
	go s.execute(j)
	return j.snapshot()
}

func (s *TransferService) execute(j *job) {
	select {
	case s.sem <- struct{}{}:
	case <-j.ctx.Done():
		j.update(func(t *Transfer) { t.State = StateCancelled })
		return
	}
	defer func() { <-s.sem }()
	j.update(func(t *Transfer) { t.State = StateRunning; t.Started = time.Now() })
	err := j.run(j)
	j.update(func(t *Transfer) {
		t.Current = ""
		switch {
		case err == nil:
			t.State = StateDone
		case errors.Is(err, context.Canceled) || j.ctx.Err() != nil:
			t.State = StateCancelled
		default:
			t.State, t.Error = StateFailed, err.Error()
		}
	})
}

// List returns all known transfers, oldest first.
func (s *TransferService) List() []Transfer {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Transfer, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.jobs[id].snapshot())
	}
	return out
}

func (s *TransferService) Cancel(id int64) {
	s.mu.Lock()
	j := s.jobs[id]
	s.mu.Unlock()
	if j != nil {
		j.cancel()
	}
}

// ClearFinished forgets transfers that are no longer running.
func (s *TransferService) ClearFinished() {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.order[:0]
	for _, id := range s.order {
		st := s.jobs[id].snapshot().State
		if st == StateQueued || st == StateRunning {
			kept = append(kept, id)
		} else {
			delete(s.jobs, id)
		}
	}
	s.order = kept
}

// ---- Uploads ----

type localFile struct {
	src    string
	remote string
	size   int64
}

// PickAndUpload asks for local files (or a folder) and uploads them into dir.
func (s *TransferService) PickAndUpload(dir string, folders bool) (*Transfer, error) {
	app := application.Get()
	d := app.Dialog.OpenFile().
		CanChooseFiles(!folders).
		CanChooseDirectories(folders).
		SetTitle("Upload to Nova")
	if w := app.Window.Current(); w != nil {
		d = d.AttachToWindow(w)
	}
	var paths []string
	var err error
	if folders {
		var p string
		p, err = d.SetButtonText("Upload").PromptForSingleSelection()
		if p != "" {
			paths = []string{p}
		}
	} else {
		paths, err = d.SetButtonText("Upload").PromptForMultipleSelection()
	}
	if err != nil || len(paths) == 0 {
		return nil, nil // cancelled
	}
	t, err := s.Upload(dir, paths)
	return &t, err
}

// Upload uploads local files and folders into the remote dir.
func (s *TransferService) Upload(dir string, localPaths []string) (Transfer, error) {
	dir, err := checkPath(dir)
	if err != nil {
		return Transfer{}, err
	}
	if len(localPaths) == 0 {
		return Transfer{}, errors.New("nothing to upload")
	}
	title := filepath.Base(localPaths[0])
	if len(localPaths) > 1 {
		title = fmt.Sprintf("%d items", len(localPaths))
	}
	return s.enqueue(KindUpload, title, dir, func(j *job) error {
		// Never overwrite: clashing top-level names get "name (copy)" like Copy does.
		taken := map[string]bool{}
		if l, err := s.client.Stat(j.ctx, dir); err == nil {
			for _, c := range l.Children {
				taken[c.Name] = true
			}
		}
		var files []localFile
		dirs := map[string]bool{}
		for _, lp := range localPaths {
			lp = filepath.Clean(lp)
			st, err := os.Stat(lp)
			if err != nil {
				return err
			}
			top := uniqueName(filepath.Base(lp), taken, st.IsDir())
			taken[top] = true
			err = filepath.Walk(lp, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(lp, p)
				remote := nova.Join(nova.Join(dir, top), filepath.ToSlash(rel))
				if info.IsDir() {
					dirs[remote] = true
					return nil
				}
				if !info.Mode().IsRegular() {
					return nil // skip sockets, devices, symlinks to nowhere
				}
				files = append(files, localFile{src: p, remote: remote, size: info.Size()})
				return nil
			})
			if err != nil {
				return err
			}
		}
		var total int64
		for _, f := range files {
			total += f.size
		}
		j.update(func(t *Transfer) { t.Files, t.TotalBytes = len(files), total })
		for d := range dirs {
			if err := s.client.MkdirAll(j.ctx, d); err != nil {
				return err
			}
		}
		for _, f := range files {
			if err := j.ctx.Err(); err != nil {
				return err
			}
			j.update(func(t *Transfer) { t.Current = filepath.Base(f.src) })
			if err := s.uploadOne(j, f); err != nil {
				return fmt.Errorf("%s: %w", filepath.Base(f.src), err)
			}
			j.update(func(t *Transfer) { t.DoneFiles++ })
		}
		emit(EventChanged, dir)
		return nil
	}), nil
}

func (s *TransferService) uploadOne(j *job, f localFile) error {
	fh, err := os.Open(f.src)
	if err != nil {
		return err
	}
	defer fh.Close()
	ct := mime.TypeByExtension(filepath.Ext(f.src))
	before := j.moved.Load()
	err = s.client.Upload(j.ctx, f.remote, &progressReader{r: fh, j: j}, f.size, ct)
	if err != nil {
		j.moved.Store(before) // don't count a failed partial upload
	}
	return err
}

// ---- Downloads ----

type remoteFile struct {
	remote string
	local  string
	size   int64
}

// PickAndDownload asks for a local folder and downloads paths into it.
// Phones have no folder picker, so there files go to the Download folder.
func (s *TransferService) PickAndDownload(paths []string) (*Transfer, error) {
	if platform.Mobile {
		t, err := s.Download(paths, platform.DownloadDir())
		return &t, err
	}
	app := application.Get()
	d := app.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		SetTitle("Download to…").
		SetButtonText("Download")
	if dir := platform.DownloadDir(); dir != "" {
		d = d.SetDirectory(dir)
	}
	if w := app.Window.Current(); w != nil {
		d = d.AttachToWindow(w)
	}
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return nil, nil
	}
	t, err := s.Download(paths, dest)
	return &t, err
}

// Download copies remote paths (files or folders) into a local directory.
func (s *TransferService) Download(paths []string, localDir string) (Transfer, error) {
	if len(paths) == 0 {
		return Transfer{}, errors.New("nothing to download")
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return Transfer{}, fmt.Errorf("cannot write to %s: %w", localDir, err)
	}
	for i, p := range paths {
		c, err := checkPath(p)
		if err != nil {
			return Transfer{}, err
		}
		paths[i] = c
	}
	title := nova.Base(paths[0])
	if len(paths) > 1 {
		title = fmt.Sprintf("%d items", len(paths))
	}
	return s.enqueue(KindDownload, title, localDir, func(j *job) error {
		files, err := s.collect(j.ctx, paths, localDir)
		if err != nil {
			return err
		}
		var total int64
		for _, f := range files {
			total += f.size
		}
		j.update(func(t *Transfer) { t.Files, t.TotalBytes = len(files), total })
		for _, f := range files {
			if err := j.ctx.Err(); err != nil {
				return err
			}
			j.update(func(t *Transfer) { t.Current = nova.Base(f.remote) })
			if err := s.downloadOne(j, f); err != nil {
				return fmt.Errorf("%s: %w", nova.Base(f.remote), err)
			}
			j.update(func(t *Transfer) { t.DoneFiles++ })
		}
		return nil
	}), nil
}

// collect expands remote folders into a flat file list with local targets.
func (s *TransferService) collect(ctx context.Context, paths []string, localDir string) ([]remoteFile, error) {
	var out []remoteFile
	// Names are picked before anything is written, so remember them to keep
	// two items of one batch from landing on the same local path.
	reserved := map[string]bool{}
	var walk func(remote, local string, depth int) error
	walk = func(remote, local string, depth int) error {
		if depth > 64 {
			return errors.New("folder structure is too deep")
		}
		l, err := s.client.Stat(ctx, remote)
		if err != nil {
			return err
		}
		if l.BaseIndex >= len(l.Path) {
			return errors.New("unexpected server response")
		}
		self := l.Path[l.BaseIndex]
		if !self.IsDir() {
			out = append(out, remoteFile{remote: remote, local: local, size: self.FileSize})
			return nil
		}
		if err := os.MkdirAll(local, 0o755); err != nil {
			return err
		}
		for _, c := range l.Children {
			if err := walk(nova.CleanPath(c.Path), uniqueLocal(filepath.Join(local, safeLocalName(c.Name)), reserved), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range paths {
		target := uniqueLocal(filepath.Join(localDir, safeLocalName(nova.Base(p))), reserved)
		if err := walk(p, target, 0); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// safeLocalName prevents remote names from escaping the target directory.
func safeLocalName(name string) string {
	name = strings.NewReplacer("/", "_", "\\", "_", "\x00", "").Replace(name)
	if name == "" || name == "." || name == ".." {
		name = "_"
	}
	return name
}

// uniqueLocal returns p, or "name (N).ext", skipping existing and reserved paths.
func uniqueLocal(p string, reserved map[string]bool) string {
	free := func(c string) bool {
		if reserved[c] {
			return false
		}
		_, err := os.Lstat(c)
		return errors.Is(err, os.ErrNotExist)
	}
	c := p
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	for i := 1; !free(c); i++ {
		c = fmt.Sprintf("%s (%d)%s", stem, i, ext)
	}
	reserved[c] = true
	return c
}

func (s *TransferService) downloadOne(j *job, f remoteFile) error {
	res, err := s.client.Do(j.ctx, http.MethodGet, s.client.FileURL(f.remote, "download"), nil, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := os.MkdirAll(filepath.Dir(f.local), 0o755); err != nil {
		return err
	}
	tmp := f.local + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	before := j.moved.Load()
	_, err = io.Copy(out, &progressReader{r: res.Body, j: j})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		j.moved.Store(before)
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, f.local)
}

// ---- Open with default application ----

func cacheDir() string { return platform.CacheDir() }

// Open downloads a file to the cache and opens it with the system default app.
func (s *TransferService) Open(p string) (Transfer, error) {
	if platform.Mobile {
		return Transfer{}, errors.New("opening files in other apps isn't supported on phones yet; use Preview or Download")
	}
	p, err := checkPath(p)
	if err != nil {
		return Transfer{}, err
	}
	return s.enqueue(KindOpen, nova.Base(p), "", func(j *job) error {
		l, err := s.client.Stat(j.ctx, p)
		if err != nil {
			return err
		}
		if l.BaseIndex >= len(l.Path) {
			return errors.New("unexpected server response")
		}
		n := l.Path[l.BaseIndex]
		if n.IsDir() {
			return errors.New("folders cannot be opened externally")
		}
		key := n.SHA256
		if key == "" {
			key = n.ID
		}
		local := filepath.Join(cacheDir(), "open", safeLocalName(key), safeLocalName(n.Name))
		j.update(func(t *Transfer) { t.Files, t.TotalBytes, t.Current, t.Dest = 1, n.FileSize, n.Name, local })
		if st, err := os.Stat(local); err != nil || st.Size() != n.FileSize {
			if err := s.downloadOne(j, remoteFile{remote: p, local: local, size: n.FileSize}); err != nil {
				return err
			}
		} else {
			j.moved.Store(n.FileSize)
		}
		j.update(func(t *Transfer) { t.DoneFiles = 1 })
		return application.Get().Browser.OpenFile(local)
	}), nil
}

// ---- Copy ----

// Copy copies remote paths into destDir. The API has no server-side copy, so
// every file is streamed down and back up. Clashes get "name (copy)" names.
func (s *TransferService) Copy(paths []string, destDir string) (Transfer, error) {
	destDir, err := checkPath(destDir)
	if err != nil {
		return Transfer{}, err
	}
	if len(paths) == 0 {
		return Transfer{}, errors.New("nothing to copy")
	}
	for i, p := range paths {
		c, err := checkPath(p)
		if err != nil || c == HomeDir {
			return Transfer{}, fmt.Errorf("%s cannot be copied", p)
		}
		if destDir == c || strings.HasPrefix(destDir, c+"/") {
			return Transfer{}, errors.New("cannot copy a folder into itself")
		}
		paths[i] = c
	}
	title := nova.Base(paths[0])
	if len(paths) > 1 {
		title = fmt.Sprintf("%d items", len(paths))
	}
	return s.enqueue(KindCopy, title, destDir, func(j *job) error {
		l, err := s.client.Stat(j.ctx, destDir)
		if err != nil {
			return err
		}
		taken := map[string]bool{}
		for _, c := range l.Children {
			taken[c.Name] = true
		}
		type pair struct {
			from, to string
			size     int64
		}
		var files []pair
		var dirs []string
		var walk func(from, to string, depth int) error
		walk = func(from, to string, depth int) error {
			if depth > 64 {
				return errors.New("folder structure is too deep")
			}
			l, err := s.client.Stat(j.ctx, from)
			if err != nil {
				return err
			}
			if l.BaseIndex >= len(l.Path) {
				return errors.New("unexpected server response")
			}
			if self := l.Path[l.BaseIndex]; !self.IsDir() {
				files = append(files, pair{from, to, self.FileSize})
				return nil
			}
			dirs = append(dirs, to)
			for _, c := range l.Children {
				if err := walk(nova.CleanPath(c.Path), nova.Join(to, c.Name), depth+1); err != nil {
					return err
				}
			}
			return nil
		}
		for _, p := range paths {
			isDir := false
			if l, err := s.client.Stat(j.ctx, p); err == nil && l.BaseIndex < len(l.Path) {
				isDir = l.Path[l.BaseIndex].IsDir()
			}
			name := uniqueName(nova.Base(p), taken, isDir)
			taken[name] = true
			if err := walk(p, nova.Join(destDir, name), 0); err != nil {
				return err
			}
		}
		var total int64
		for _, f := range files {
			total += f.size
		}
		j.update(func(t *Transfer) { t.Files, t.TotalBytes = len(files), total })
		for _, d := range dirs {
			if err := s.client.MkdirAll(j.ctx, d); err != nil {
				return err
			}
		}
		defer emit(EventChanged, destDir)
		for _, f := range files {
			if err := j.ctx.Err(); err != nil {
				return err
			}
			j.update(func(t *Transfer) { t.Current = nova.Base(f.from) })
			before := j.moved.Load()
			err := s.client.CopyFile(j.ctx, f.from, f.to, func(r io.Reader) io.Reader {
				return &progressReader{r: r, j: j}
			})
			if err != nil {
				j.moved.Store(before)
				return fmt.Errorf("%s: %w", nova.Base(f.from), err)
			}
			j.update(func(t *Transfer) { t.DoneFiles++ })
		}
		return nil
	}), nil
}
