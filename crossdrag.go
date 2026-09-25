package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"nova/internal/platform"
	"nova/services"
)

// EventCrossDrag tells a window that an item drag from some Nova window is
// over it, has left it, or was dropped on it.
const EventCrossDrag = "xdrag"

// DragPayload is what a window drags: the paths of the items and a label for
// the drag icon.
type DragPayload struct {
	Kind    string   `json:"kind"`
	Paths   []string `json:"paths"`
	AllDirs bool     `json:"allDirs"`
	Label   string   `json:"label"`
}

// CrossDrag is sent to the window under a drag that left its own window.
// X and Y are relative to the page.
type CrossDrag struct {
	Type    string       `json:"type"` // "over", "leave" or "drop"
	X       int          `json:"x"`
	Y       int          `json:"y"`
	Copy    bool         `json:"copy"`
	Payload *DragPayload `json:"payload"`
}

// A drag that left its window is carried on by the platform's own drag and
// drop, so it can end in any Nova window. The payload stays here in Go; the
// platform drag only carries a marker type.
var (
	dragMu      sync.Mutex
	dragPayload *DragPayload
	dragOut     *dragExport
)

func currentDrag() *DragPayload {
	dragMu.Lock()
	defer dragMu.Unlock()
	return dragPayload
}

func setDrag(p *DragPayload) {
	dragMu.Lock()
	dragPayload = p
	dragOut = nil
	dragMu.Unlock()
}

// dragTransfers downloads items dragged out of Nova; set in main.
var dragTransfers *services.TransferService

// A drag that ends in another application (a file manager, the desktop)
// asks for the items as local files. They are downloaded once per drag into
// a folder of their own, and the drop waits for that.
type dragExport struct {
	done   chan struct{} // closed once the download has finished
	ctx    context.Context
	cancel context.CancelFunc
	dir    string
	uris   string
	err    error
}

// exportDrag returns the export of the current drag, or nil if there is none.
func exportDrag() *dragExport {
	dragMu.Lock()
	defer dragMu.Unlock()
	if dragPayload == nil || dragTransfers == nil {
		return nil
	}
	if dragOut == nil {
		ctx, cancel := context.WithCancel(context.Background())
		ex := &dragExport{done: make(chan struct{}), ctx: ctx, cancel: cancel,
			dir: filepath.Join(dragExportRoot(), strconv.FormatInt(time.Now().UnixNano(), 36))}
		go ex.download(dragPayload.Paths)
		dragOut = ex
	}
	return dragOut
}

func (e *dragExport) download(paths []string) {
	defer close(e.done)
	tops, err := services.DownloadAndWait(e.ctx, dragTransfers, paths, e.dir)
	if err != nil {
		e.err = err
		os.RemoveAll(e.dir)
		return
	}
	// text/uri-list: one URI per line, CRLF.
	var b strings.Builder
	for _, t := range tops {
		b.WriteString((&url.URL{Scheme: "file", Path: t}).String())
		b.WriteString("\r\n")
	}
	e.uris = b.String()
}

// wait blocks until the items are on disk and returns them as a uri list.
func (e *dragExport) wait() (string, error) {
	<-e.done
	return e.uris, e.err
}

// endDrag forgets the current drag. A drag that was not dropped anywhere
// throws away what it may have started downloading.
func endDrag(dropped bool) {
	dragMu.Lock()
	ex := dragOut
	dragPayload, dragOut = nil, nil
	dragMu.Unlock()
	if ex == nil {
		return
	}
	if dropped {
		// The receiving application is reading the items, or already has.
		go func() {
			ex.wait()
			ex.cancel()
		}()
		return
	}
	ex.cancel()
	go func() {
		ex.wait()
		os.RemoveAll(ex.dir)
	}()
}

func dragExportRoot() string { return filepath.Join(platform.CacheDir(), "drag") }

// cleanDragExports removes items dragged out more than a day ago. The
// receiving app has copied or moved them by then.
func cleanDragExports() {
	ents, _ := os.ReadDir(dragExportRoot())
	for _, e := range ents {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > 24*time.Hour {
			os.RemoveAll(filepath.Join(dragExportRoot(), e.Name()))
		}
	}
}

// StartDrag hands the calling window's item drag over to the platform once
// the pointer leaves the window. It returns false when that isn't possible;
// the window then keeps the drag to itself.
func (s *WindowService) StartDrag(ctx context.Context, p DragPayload) bool {
	win, ok := ctx.Value(application.WindowKey).(*application.WebviewWindow)
	if !ok || p.Kind != "items" || len(p.Paths) == 0 {
		return false
	}
	return startNativeDrag(win, &p)
}

func sendCrossDrag(win *application.WebviewWindow, d CrossDrag) {
	win.DispatchWailsEvent(&application.CustomEvent{Name: EventCrossDrag, Data: d})
}
