//go:build !windows

package services

// osSafeName: Unix file names can hold anything but "/" and NUL, which
// safeLocalName already replaced.
func osSafeName(name string) string { return name }
