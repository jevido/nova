//go:build windows

package services

func osSafeName(name string) string { return windowsSafeName(name) }
