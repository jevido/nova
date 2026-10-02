package services

import (
	"errors"
	"strings"
	"sync"

	"nova/internal/nova"
)

// A key from a browser sign-in can be limited to some folders of the
// account. Each of them is then a top folder of its own, at /{id}, and
// nothing else can be reached, not even HomeDir. Without a limit there is one
// top folder, HomeDir.
var (
	rootsMu sync.RWMutex
	roots   []string // "/{id}" of each folder the key is limited to
)

// errLimited is returned for what needs the whole account, like the trash.
var errLimited = errors.New("Nova can only reach some of your folders; sign in again to give it all of them")

func setRoots(ids []string) {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.Trim(id, "/"); id != "" && !strings.Contains(id, "/") {
			out = append(out, "/"+id)
		}
	}
	rootsMu.Lock()
	roots = out
	rootsMu.Unlock()
}

// topFolders returns the folders everything reachable is in.
func topFolders() []string {
	rootsMu.RLock()
	defer rootsMu.RUnlock()
	if len(roots) == 0 {
		return []string{HomeDir}
	}
	return append([]string(nil), roots...)
}

// limited reports whether the key only reaches some folders.
func limited() bool {
	rootsMu.RLock()
	defer rootsMu.RUnlock()
	return len(roots) > 0
}

// rootOf returns the top folder holding the clean path p, or "" when p is
// outside all of them.
func rootOf(p string) string {
	for _, r := range topFolders() {
		if p == r || strings.HasPrefix(p, r+"/") {
			return r
		}
	}
	return ""
}

// isRoot reports whether p is a top folder, which can't be renamed, moved
// or deleted.
func isRoot(p string) bool { return p != "" && rootOf(p) == p }

// sameRoot reports whether a and b are in the same top folder. The server
// can't rename across them.
func sameRoot(a, b string) bool { return rootOf(nova.CleanPath(a)) == rootOf(nova.CleanPath(b)) }
