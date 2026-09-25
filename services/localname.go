package services

import (
	"strings"
	"unicode"
)

// windowsSafeName makes a remote name usable as a Windows file name. Names on
// the server may contain characters Windows forbids (<>:"|?* and control
// characters), end in dots or spaces (which Windows silently drops) or be a
// reserved device name like CON or LPT1. It is applied on Windows only (see
// osSafeName), but lives here so it is tested on every platform.
func windowsSafeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`<>:"|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimRightFunc(name, func(r rune) bool { return r == '.' || unicode.IsSpace(r) })
	if name == "" {
		return "_"
	}
	stem, _, _ := strings.Cut(name, ".")
	switch strings.ToUpper(strings.TrimRightFunc(stem, unicode.IsSpace)) {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}
