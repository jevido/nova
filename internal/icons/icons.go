// Package icons resolves freedesktop icon names against the user's icon theme,
// so the app shows the same folder and mimetype icons as the native file
// manager. A small bundled Adwaita set is used when no theme is available.
package icons

import (
	"bufio"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

//go:embed fallback
var fallback embed.FS

var validName = regexp.MustCompile(`^[A-Za-z0-9._+-]{1,128}$`)

type file struct {
	path  string
	size  int // 0 for scalable
	embed bool
}

// Resolver finds icon files by name. It indexes theme directories lazily.
type Resolver struct {
	once   sync.Once
	themes []string
	mu     sync.Mutex
	index  map[string]file
}

func NewResolver() *Resolver { return &Resolver{} }

// iconDirs and currentTheme are per OS: freedesktop icon themes exist on
// Linux and the BSDs (themes_freedesktop.go); everywhere else (Windows,
// macOS, phones) Nova uses the bundled fallback set only.

func findTheme(name string) string {
	for _, d := range iconDirs() {
		p := filepath.Join(d, name)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

func inherits(themeDir string) []string {
	f, err := os.Open(filepath.Join(themeDir, "index.theme"))
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if v, ok := strings.CutPrefix(line, "Inherits="); ok {
			var out []string
			for _, s := range strings.Split(v, ",") {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func (r *Resolver) init() {
	r.index = map[string]file{}
	seen := map[string]bool{}
	var walk func(name string)
	walk = func(name string) {
		if name == "" || seen[name] || len(seen) > 12 {
			return
		}
		seen[name] = true
		dir := findTheme(name)
		if dir == "" {
			return
		}
		r.themes = append(r.themes, dir)
		for _, parent := range inherits(dir) {
			walk(parent)
		}
	}
	walk(currentTheme())
	walk("Adwaita")
	walk("AdwaitaLegacy")
	walk("hicolor")
	// Index themes in priority order: the first theme that has a name wins,
	// within a theme scalable beats the largest raster.
	for _, dir := range r.themes {
		found := map[string]file{}
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if strings.Contains(d.Name(), "@") { // hi-dpi duplicates
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(p)
			if ext != ".svg" && ext != ".png" {
				return nil
			}
			name := strings.TrimSuffix(d.Name(), ext)
			size := dirSize(p)
			cur, ok := found[name]
			if !ok || better(file{path: p, size: size}, cur) {
				found[name] = file{path: p, size: size}
			}
			return nil
		})
		for k, v := range found {
			if _, ok := r.index[k]; !ok {
				r.index[k] = v
			}
		}
	}
	_ = fs.WalkDir(fallback, "fallback", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".svg") {
			return nil
		}
		name := strings.TrimSuffix(d.Name(), ".svg")
		if _, ok := r.index[name]; !ok {
			r.index[name] = file{path: p, embed: true}
		}
		return nil
	})
}

// dirSize extracts N from a ".../NxN/..." path, 0 means scalable.
func dirSize(p string) int {
	for _, seg := range strings.Split(filepath.ToSlash(p), "/") {
		if a, _, ok := strings.Cut(seg, "x"); ok {
			if n, err := strconv.Atoi(a); err == nil {
				return n
			}
		}
	}
	return 0
}

func better(a, b file) bool {
	aSVG, bSVG := strings.HasSuffix(a.path, ".svg"), strings.HasSuffix(b.path, ".svg")
	if aSVG != bSVG {
		return aSVG
	}
	if a.size == 0 || b.size == 0 {
		return a.size == 0 && b.size != 0
	}
	return a.size > b.size
}

// Lookup returns the file contents and content type for the first name found.
func (r *Resolver) Lookup(names ...string) ([]byte, string, bool) {
	r.once.Do(r.init)
	for _, n := range names {
		if !validName.MatchString(n) {
			continue
		}
		f, ok := r.index[n]
		if !ok {
			continue
		}
		var b []byte
		var err error
		if f.embed {
			b, err = fallback.ReadFile(f.path)
		} else {
			b, err = os.ReadFile(f.path)
		}
		if err != nil {
			continue
		}
		ct := "image/png"
		if strings.HasSuffix(f.path, ".svg") {
			ct = "image/svg+xml"
		}
		return b, ct, true
	}
	return nil, "", false
}
