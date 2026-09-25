package main

import "sync"

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
	clipMu sync.Mutex
	clip   *ItemClipboard
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
}
