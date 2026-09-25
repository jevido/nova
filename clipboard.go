package main

import (
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventOSClipboardFiles tells every window whether the system clipboard
// holds files copied in another application, which Paste then uploads.
const EventOSClipboardFiles = "clipboard:files"

// EventClipboard tells every window that the items on Nova's clipboard
// changed. The data is the new clipboard, or nil when it was cleared.
const EventClipboard = "clipboard"

// ItemClipboard is what Cut or Copy put aside for the Paste command. It
// lives here in Go so that every Nova window pastes the same items.
type ItemClipboard struct {
	Mode  string   `json:"mode"` // "copy" or "cut"
	Paths []string `json:"paths"`
}

var (
	clipMu  sync.Mutex
	clip    *ItemClipboard
	osFiles bool
)

// Clipboard returns the items waiting to be pasted, or nil.
func (s *WindowService) Clipboard() *ItemClipboard {
	clipMu.Lock()
	defer clipMu.Unlock()
	return clip
}

// SetClipboard puts items on the clipboard shared by all windows; nil (or no
// paths) clears it.
func (s *WindowService) SetClipboard(c *ItemClipboard) {
	if c != nil && (len(c.Paths) == 0 || (c.Mode != "copy" && c.Mode != "cut")) {
		c = nil
	}
	clipMu.Lock()
	clip = c
	clipMu.Unlock()
	s.app.Event.Emit(EventClipboard, c)
	if c != nil {
		// Take over the system clipboard, like Nautilus does, so that Paste
		// uses whatever was copied last, here or in another application.
		lines := make([]string, len(c.Paths))
		for i, p := range c.Paths {
			lines[i] = strings.TrimPrefix(p, "/me")
		}
		claimOSClipboard(strings.Join(lines, "\n"))
	}
}

// ClipboardHasFiles reports whether the system clipboard holds files copied
// in another application.
func (s *WindowService) ClipboardHasFiles() bool {
	clipMu.Lock()
	defer clipMu.Unlock()
	return osFiles
}

// ClipboardFiles returns the local paths of the files on the system
// clipboard.
func (s *WindowService) ClipboardFiles() ([]string, error) {
	return readOSClipboardFiles()
}

func setOSClipboardFiles(has bool) {
	clipMu.Lock()
	changed := osFiles != has
	osFiles = has
	clipMu.Unlock()
	if app := application.Get(); changed && app != nil {
		app.Event.Emit(EventOSClipboardFiles, has)
	}
}
