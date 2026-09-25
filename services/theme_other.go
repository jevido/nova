//go:build (!linux && !windows) || android

package services

// macOS takes its accent colour from CSS (AccentColor); phones keep Nova's
// own look.
func readPlatformTheme(*SystemTheme) {}
