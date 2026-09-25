//go:build !((linux && !android) || freebsd || openbsd || netbsd || dragonfly)

package icons

// Windows, macOS, Android and iOS have no freedesktop icon themes, so every
// lookup is answered from the bundled Adwaita fallback set.

func iconDirs() []string { return nil }

func currentTheme() string { return "" }
