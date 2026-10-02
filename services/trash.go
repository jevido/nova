package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nova/internal/nova"
)

// The trash follows the freedesktop.org Trash specification
// (https://specifications.freedesktop.org/trash/latest/), with /me/.Trash as
// the account's "home trash": trashed items live in files/, and each has an
// info/<name>.trashinfo recording where it came from and when it was
// trashed. Any client that speaks the spec, such as a file manager on a
// mounted copy of the account, can list and restore what Nova trashed.
const (
	TrashDir      = "/me/.Trash"
	trashFilesDir = TrashDir + "/files"
	trashInfoDir  = TrashDir + "/info"
	trashInfoExt  = ".trashinfo"
	// directorysizes is an optional size cache from the spec. Nova doesn't
	// write it, but leaves one another client wrote alone.
	trashDirSizes = TrashDir + "/directorysizes"
	// Before following the spec, Nova kept trashed items directly in
	// TrashDir and their origins in this JSON index.
	legacyTrashIndex = TrashDir + "/.trashinfo.json"
)

// trashDateLayout is the spec's DeletionDate format, in local time.
const trashDateLayout = "2006-01-02T15:04:05"

// inTrash reports whether p is the trash or anything inside it.
func inTrash(p string) bool { return p == TrashDir || strings.HasPrefix(p, TrashDir+"/") }

func trashInfoPath(name string) string { return nova.Join(trashInfoDir, name+trashInfoExt) }

// escapeTrashPath percent-encodes p for a .trashinfo Path= line, keeping
// "/" and the unreserved characters, as GIO and KIO do.
func escapeTrashPath(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~', c == '/':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// formatTrashInfo returns the .trashinfo contents for an item trashed from
// orig at t. Path= is relative to the folder holding the trash (/me), as the
// spec prefers.
func formatTrashInfo(orig string, t time.Time) []byte {
	rel := strings.TrimPrefix(orig, HomeDir+"/")
	return []byte("[Trash Info]\nPath=" + escapeTrashPath(rel) + "\nDeletionDate=" + t.Local().Format(trashDateLayout) + "\n")
}

// parseTrashInfo reads a .trashinfo file. orig is empty when Path= is
// missing or points outside the home folder; at is zero when DeletionDate=
// is missing or unreadable.
func parseTrashInfo(b []byte) (orig string, at time.Time, err error) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	header := false
	var rawPath, rawDate string
	var havePath, haveDate bool
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !header {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if line != "[Trash Info]" {
				return "", time.Time{}, errors.New("not a trash info file")
			}
			header = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			break // another group; only [Trash Info] counts
		}
		// The first occurrence of each key wins.
		if v, ok := strings.CutPrefix(line, "Path="); ok && !havePath {
			rawPath, havePath = v, true
		} else if v, ok := strings.CutPrefix(line, "DeletionDate="); ok && !haveDate {
			rawDate, haveDate = v, true
		}
	}
	if !header {
		return "", time.Time{}, errors.New("not a trash info file")
	}
	return trashOrigPath(rawPath), parseTrashDate(rawDate), nil
}

// trashOrigPath resolves a Path= value to a path in the account. Relative
// paths are from /me and may not climb out of it; absolute ones must already
// be inside it.
func trashOrigPath(raw string) string {
	if raw == "" {
		return ""
	}
	p, err := url.PathUnescape(raw)
	if err != nil {
		p = raw
	}
	if !strings.HasPrefix(p, "/") {
		for _, seg := range strings.Split(p, "/") {
			if seg == ".." {
				return ""
			}
		}
		p = HomeDir + "/" + p
	}
	p = nova.CleanPath(p)
	if p == HomeDir || !strings.HasPrefix(p, HomeDir+"/") || inTrash(p) {
		return ""
	}
	return p
}

func parseTrashDate(raw string) time.Time {
	for _, layout := range []string{trashDateLayout, "20060102T15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t
		}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	return time.Time{}
}

var errNoTrashInfo = errors.New("no trash info")

// readTrashInfo loads the .trashinfo for the trashed item called name.
func (s *FilesService) readTrashInfo(ctx context.Context, name string) (string, time.Time, error) {
	res, err := s.client.Do(ctx, http.MethodGet, s.client.FileURL(trashInfoPath(name), ""), nil, nil)
	if isNotFound(err) {
		return "", time.Time{}, errNoTrashInfo
	}
	if err != nil {
		return "", time.Time{}, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return "", time.Time{}, err
	}
	return parseTrashInfo(b)
}

func (s *FilesService) writeTrashInfo(ctx context.Context, name, orig string, t time.Time) error {
	b := formatTrashInfo(orig, t)
	return s.client.Upload(ctx, trashInfoPath(name), bytes.NewReader(b), int64(len(b)), "application/x-trash-info")
}

// trashNames returns the names in use in files/ and info/ together, so a new
// item never reuses the name of an orphaned info file.
func (s *FilesService) trashNames(ctx context.Context) (map[string]bool, error) {
	taken, err := s.existingNames(ctx, trashFilesDir)
	if err != nil {
		return nil, err
	}
	infos, err := s.existingNames(ctx, trashInfoDir)
	if err != nil {
		return nil, err
	}
	for n := range infos {
		taken[strings.TrimSuffix(n, trashInfoExt)] = true
	}
	return taken, nil
}

// moveToTrash trashes p under a free name, writing its info file first as
// the spec requires. The API has no exclusive create, so trashMu, held by
// the caller, is what keeps two trashings from picking the same name.
func (s *FilesService) moveToTrash(ctx context.Context, p string, taken map[string]bool, at time.Time) error {
	name := uniqueName(nova.Base(p), taken, true)
	if err := s.writeTrashInfo(ctx, name, p, at); err != nil {
		return fmt.Errorf("could not save trash info: %w", err)
	}
	if err := s.client.Rename(ctx, p, nova.Join(trashFilesDir, name)); err != nil {
		_ = s.client.Delete(ctx, trashInfoPath(name), false)
		return err
	}
	taken[name] = true
	return nil
}

// migrateLegacyTrash moves items from the old layout (directly in TrashDir,
// origins in a JSON index) into files/ with an info file each. It runs
// before every trash operation and costs one stat once there is nothing
// left to move. The caller holds trashMu.
func (s *FilesService) migrateLegacyTrash(ctx context.Context) error {
	l, err := s.client.Stat(ctx, TrashDir)
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var legacy []nova.Node
	hasIndex := false
	for _, c := range l.Children {
		switch nova.CleanPath(c.Path) {
		case trashFilesDir, trashInfoDir, trashDirSizes:
		case legacyTrashIndex:
			hasIndex = true
		default:
			legacy = append(legacy, c)
		}
	}
	if len(legacy) == 0 && !hasIndex {
		return nil
	}

	type record struct {
		OrigPath  string    `json:"orig"`
		DeletedAt time.Time `json:"deleted"`
	}
	index := map[string]record{}
	if hasIndex {
		res, err := s.client.Do(ctx, http.MethodGet, s.client.FileURL(legacyTrashIndex, ""), nil, nil)
		if err != nil {
			return fmt.Errorf("could not read the old trash index: %w", err)
		}
		err = json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&index)
		res.Body.Close()
		if err != nil {
			return fmt.Errorf("the old trash index is damaged: %w", err)
		}
	}

	for _, dir := range []string{trashFilesDir, trashInfoDir} {
		if err := s.client.MkdirAll(ctx, dir); err != nil {
			return err
		}
	}
	taken, err := s.trashNames(ctx)
	if err != nil {
		return err
	}
	var failed error
	for _, c := range legacy {
		r := index[c.Name]
		orig := trashOrigPath(r.OrigPath)
		if orig == "" {
			orig = nova.Join(HomeDir, c.Name)
		}
		at := r.DeletedAt
		if at.IsZero() {
			at = c.Modified
		}
		// Keep the trashed name when it's free, so paths the UI holds for
		// an item still point at it after the move.
		name := uniqueName(c.Name, taken, true)
		if err := s.writeTrashInfo(ctx, name, orig, at); err != nil {
			failed = err
			continue
		}
		if err := s.client.Rename(ctx, nova.CleanPath(c.Path), nova.Join(trashFilesDir, name)); err != nil {
			_ = s.client.Delete(ctx, trashInfoPath(name), false)
			failed = err
			continue
		}
		taken[name] = true
	}
	if failed != nil {
		return fmt.Errorf("could not move old trash items: %w", failed)
	}
	if hasIndex {
		_ = s.client.Delete(ctx, legacyTrashIndex, false)
	}
	return nil
}

// listTrash lists files/ as the trash root, with each item's origin and
// deletion time from its info file.
func (s *FilesService) listTrash(ctx context.Context) (*Folder, error) {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	if err := s.migrateLegacyTrash(ctx); err != nil {
		return nil, err
	}
	f := &Folder{
		Path:     TrashDir,
		Crumbs:   []Entry{{Name: "Trash", Path: TrashDir, IsDir: true}},
		Children: []Entry{},
		CanWrite: true,
	}
	l, err := s.client.Stat(ctx, trashFilesDir)
	if isNotFound(err) {
		return f, nil // created on first use
	}
	if err != nil {
		return nil, err
	}
	f.Children = make([]Entry, len(l.Children))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, n := range l.Children {
		f.Children[i] = toEntry(n)
		wg.Add(1)
		go func(e *Entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Items without a readable info file still list, without an origin.
			orig, at, err := s.readTrashInfo(ctx, e.Name)
			if err != nil {
				return
			}
			e.OrigPath = orig
			if !at.IsZero() {
				e.DeletedAt = &at
			}
		}(&f.Children[i])
	}
	wg.Wait()
	return f, nil
}

// trashCrumbs hides files/ from the crumbs of a folder inside the trash, so
// the trash reads as its own root.
func trashCrumbs(f *Folder) {
	kept := f.Crumbs[:0]
	for _, c := range f.Crumbs {
		if c.Path != trashFilesDir {
			kept = append(kept, c)
		}
	}
	f.Crumbs = kept
}

// dropTrashInfo removes the info files of trashed items that were deleted
// for good.
func (s *FilesService) dropTrashInfo(ctx context.Context, deleted []string) {
	for _, p := range deleted {
		if nova.Parent(p) == trashFilesDir {
			_ = s.client.Delete(ctx, trashInfoPath(nova.Base(p)), false)
		}
	}
}

// Trash moves paths into the trash and records where they came from.
func (s *FilesService) Trash(paths []string) (*OpResult, error) {
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	if limited() {
		return nil, errLimited
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := s.migrateLegacyTrash(ctx); err != nil {
		return nil, err
	}
	for _, dir := range []string{trashFilesDir, trashInfoDir} {
		if err := s.client.MkdirAll(ctx, dir); err != nil {
			return nil, err
		}
	}
	taken, err := s.trashNames(ctx)
	if err != nil {
		return nil, err
	}
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p, err := checkPath(raw)
		if err != nil || p == HomeDir || inTrash(p) {
			res.fail(raw, errors.New("cannot be moved to the trash"))
			continue
		}
		if err := s.moveToTrash(ctx, p, taken, time.Now()); err != nil {
			res.fail(p, err)
			continue
		}
		res.Done = append(res.Done, p)
	}
	emitChanged(res.Done, TrashDir)
	return res, nil
}

// Restore moves trashed items back to where they came from. An item without
// an origin goes to the home folder under its trashed name.
func (s *FilesService) Restore(paths []string) (*OpResult, error) {
	if limited() {
		return nil, errLimited
	}
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res := &OpResult{Done: []string{}, Errors: []string{}}
	for _, raw := range paths {
		p := nova.CleanPath(raw)
		if nova.Parent(p) != trashFilesDir {
			res.fail(raw, errors.New("not in the trash"))
			continue
		}
		name := nova.Base(p)
		orig, _, err := s.readTrashInfo(ctx, name)
		if err != nil && !errors.Is(err, errNoTrashInfo) {
			res.fail(p, fmt.Errorf("could not read trash info: %w", err))
			continue
		}
		if orig == "" {
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
		_ = s.client.Delete(ctx, trashInfoPath(name), false)
		res.Done = append(res.Done, target)
	}
	return res, nil
}

// EmptyTrash permanently deletes everything in the trash.
func (s *FilesService) EmptyTrash() error {
	if limited() {
		return errLimited
	}
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
	if limited() {
		return 0
	}
	s.trashMu.Lock()
	defer s.trashMu.Unlock()
	ctx, cancel := ctxTimeout()
	defer cancel()
	if err := s.migrateLegacyTrash(ctx); err != nil {
		return 0
	}
	l, err := s.client.Stat(ctx, trashFilesDir)
	if err != nil {
		return 0
	}
	return len(l.Children)
}
