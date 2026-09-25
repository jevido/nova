//go:build !linux || android

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// Other webviews (WebView2, WKWebView) pass the back/forward buttons to the
// page as mouse buttons 3 and 4, which the UI handles itself.
func enableMouseNav(*application.WebviewWindow) {}
