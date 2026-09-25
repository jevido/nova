//go:build windows

package services

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func readPlatformTheme(t *SystemTheme) {
	t.Font = "Segoe UI Variable Text"
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\DWM`, registry.QUERY_VALUE); err == nil {
		// AccentColor is 0xAABBGGRR.
		if v, _, err := k.GetIntegerValue("AccentColor"); err == nil {
			t.Accent = fmt.Sprintf("#%02x%02x%02x", v&0xff, (v>>8)&0xff, (v>>16)&0xff)
		}
		k.Close()
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE); err == nil {
		if v, _, err := k.GetIntegerValue("AppsUseLightTheme"); err == nil {
			t.Mode = map[bool]string{true: "light", false: "dark"}[v != 0]
		}
		k.Close()
	}
}
