package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
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
)

func currentDrag() *DragPayload {
	dragMu.Lock()
	defer dragMu.Unlock()
	return dragPayload
}

func setDrag(p *DragPayload) {
	dragMu.Lock()
	dragPayload = p
	dragMu.Unlock()
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
