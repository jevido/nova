//go:build !linux || android

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Only the GTK build hands drags to the platform so far. Elsewhere a drag
// stays in the window it started in.
func enableCrossDrag(*application.WebviewWindow) {}

func startNativeDrag(*application.WebviewWindow, *DragPayload) bool { return false }
