package services

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nova/internal/icons"
	"nova/internal/nova"
)

// MediaRoute is where MediaService is mounted in the asset server.
const MediaRoute = "/nova/"

// MediaService serves icons, thumbnails and file contents to the webview so
// <img> and <video> tags can use them without exposing the API key.
type MediaService struct {
	client *nova.Client
	icons  *icons.Resolver
}

func NewMediaService(client *nova.Client, ic *icons.Resolver) *MediaService {
	return &MediaService{client: client, icons: ic}
}

func (m *MediaService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NOVA_DEBUG") != "" {
		log.Printf("media %s %s", r.Method, r.URL.String())
	}
	// The asset server strips the route prefix; accept either form.
	p := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, MediaRoute), "/")
	switch {
	case strings.HasPrefix(p, "icon/"):
		m.icon(w, strings.TrimPrefix(p, "icon/"), r)
	case p == "thumb":
		m.thumb(w, r)
	case p == "raw":
		m.raw(w, r)
	default:
		http.NotFound(w, r)
	}
}

// icon serves /nova/icon/<name>[,<name>...] from the system icon theme.
func (m *MediaService) icon(w http.ResponseWriter, list string, r *http.Request) {
	names := strings.Split(list, ",")
	b, ct, ok := m.icons.Lookup(names...)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "max-age=86400")
	_, _ = w.Write(b)
}

// thumb serves /nova/thumb?path=...&size=N&v=<sha>, cached on disk.
func (m *MediaService) thumb(w http.ResponseWriter, r *http.Request) {
	p, err := checkPath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	switch {
	case size <= 0:
		size = 128
	case size > 512:
		size = 512
	}
	// Server thumbnails come in fixed steps; ask for the next one up.
	for _, s := range []int{32, 64, 128, 256, 512} {
		if size <= s {
			size = s
			break
		}
	}
	h := sha1.Sum([]byte(p + "\x00" + r.URL.Query().Get("v")))
	cache := filepath.Join(cacheDir(), "thumbs", strconv.Itoa(size), hex.EncodeToString(h[:])+".img")
	if b, err := os.ReadFile(cache); err == nil {
		w.Header().Set("Content-Type", http.DetectContentType(b))
		w.Header().Set("Cache-Control", "max-age=86400")
		_, _ = w.Write(b)
		return
	}
	q := "thumbnail&width=" + strconv.Itoa(size) + "&height=" + strconv.Itoa(size)
	res, err := m.client.Do(r.Context(), http.MethodGet, m.client.FileURL(p, q), nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if os.MkdirAll(filepath.Dir(cache), 0o700) == nil {
		_ = os.WriteFile(cache, b, 0o600)
	}
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "max-age=86400")
	_, _ = w.Write(b)
}

// raw streams file contents with Range support for previews and media.
func (m *MediaService) raw(w http.ResponseWriter, r *http.Request) {
	p, err := checkPath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hdr := http.Header{}
	if rg := r.Header.Get("Range"); rg != "" {
		hdr.Set("Range", rg)
	}
	res, err := m.client.Do(r.Context(), http.MethodGet, m.client.FileURL(p, ""), nil, hdr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	for _, k := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"} {
		if v := res.Header.Get(k); v != "" {
			w.Header().Set(k, v)
		}
	}
	// User content must never run as part of the app (it could reach the bindings).
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self'; media-src 'self'")
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}
