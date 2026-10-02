//go:build linux && !android

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

const dmabufEnv = "WEBKIT_DISABLE_DMABUF_RENDERER"

// fixWebKitDMABuf undoes a Wails workaround where it does more harm than
// good. Whenever the NVIDIA kernel module is loaded, Wails turns off WebKit's
// DMA-BUF renderer, because NVIDIA-driven displays can show blank windows
// with it. On a laptop whose screen runs on the Intel or AMD GPU, with the
// NVIDIA one only loaded for offloading, that leaves WebKit copying every
// frame through shared memory, and the window gets slower the bigger it is
// (hovering in a full-screen window ran at about 30 fps instead of 60).
//
// So the renderer is turned back on unless a display is actually connected
// to an NVIDIA GPU, or the user set the variable themselves.
func fixWebKitDMABuf() {
	environ, _ := os.ReadFile("/proc/self/environ")
	if keepDMABufDisabled(environ, "/sys/class/drm") {
		return
	}
	os.Unsetenv(dmabufEnv)
}

// keepDMABufDisabled reports whether WebKit's DMA-BUF renderer should stay
// off. environ is the environment Nova was started with (Wails changes the
// live one before main runs), drm the sysfs folder of the graphics cards.
func keepDMABufDisabled(environ []byte, drm string) bool {
	for _, kv := range bytes.Split(environ, []byte{0}) {
		if bytes.HasPrefix(kv, []byte(dmabufEnv+"=")) {
			return true // the user's choice
		}
	}
	connectors, _ := filepath.Glob(filepath.Join(drm, "card*-*", "status"))
	connected := false
	for _, status := range connectors {
		b, err := os.ReadFile(status)
		if err != nil || strings.TrimSpace(string(b)) != "connected" {
			continue
		}
		connected = true
		// card1-eDP-1/status belongs to card1.
		card, _, _ := strings.Cut(filepath.Base(filepath.Dir(status)), "-")
		driver, err := filepath.EvalSymlinks(filepath.Join(drm, card, "device", "driver"))
		if err != nil || strings.Contains(filepath.Base(driver), "nvidia") {
			return true
		}
	}
	// Without any display found, leave Wails' choice alone.
	return !connected
}
