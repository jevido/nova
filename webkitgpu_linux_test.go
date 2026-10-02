//go:build linux && !android

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeDRM builds a sysfs drm folder: cards maps a card to its driver, and
// connectors maps a connector (card1-eDP-1) to its status.
func fakeDRM(t *testing.T, cards, connectors map[string]string) string {
	root := t.TempDir()
	for card, driver := range cards {
		drv := filepath.Join(root, "drivers", driver)
		os.MkdirAll(drv, 0o755)
		os.MkdirAll(filepath.Join(root, card, "device"), 0o755)
		if err := os.Symlink(drv, filepath.Join(root, card, "device", "driver")); err != nil {
			t.Fatal(err)
		}
	}
	for conn, status := range connectors {
		os.MkdirAll(filepath.Join(root, conn), 0o755)
		os.WriteFile(filepath.Join(root, conn, "status"), []byte(status+"\n"), 0o644)
	}
	return root
}

func TestKeepDMABufDisabled(t *testing.T) {
	hybrid := map[string]string{"card1": "nvidia", "card2": "i915"}
	cases := []struct {
		name       string
		environ    string
		cards      map[string]string
		connectors map[string]string
		keep       bool
	}{
		{"screen on Intel, NVIDIA idle", "", hybrid,
			map[string]string{"card1-eDP-1": "disconnected", "card2-eDP-2": "connected"}, false},
		{"screen on NVIDIA", "", hybrid,
			map[string]string{"card1-eDP-1": "connected", "card2-eDP-2": "disconnected"}, true},
		{"external monitor on NVIDIA", "", hybrid,
			map[string]string{"card1-HDMI-A-1": "connected", "card2-eDP-2": "connected"}, true},
		{"set by the user", "HOME=/x\x00WEBKIT_DISABLE_DMABUF_RENDERER=1\x00", hybrid,
			map[string]string{"card2-eDP-2": "connected"}, true},
		{"no display found", "", hybrid, map[string]string{}, true},
	}
	for _, c := range cases {
		drm := fakeDRM(t, c.cards, c.connectors)
		if got := keepDMABufDisabled([]byte(c.environ), drm); got != c.keep {
			t.Errorf("%s: keep = %v, want %v", c.name, got, c.keep)
		}
	}
	// This machine, for the record.
	t.Logf("here: keep = %v", keepDMABufDisabled(nil, "/sys/class/drm"))
}
