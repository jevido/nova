//go:build !linux || android

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Only the GTK build reads files copied in other applications so far.
func enableOSClipboard(*application.WebviewWindow) {}

func readOSClipboardFiles() ([]string, error) { return nil, nil }

func claimOSClipboard(string) {}
