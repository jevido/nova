//go:build !android && !ios

package services

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestMatchAsset(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "checksums.txt"},
		{Name: "nova-linux-x86_64.AppImage"},
		{Name: "nova-linux-amd64.deb"},
		{Name: "nova-linux-amd64.tar.gz"},
		{Name: "nova-android-arm64.apk"},
		{Name: "nova-windows-amd64-setup.exe"},
		{Name: "nova-windows-amd64.exe"},
		{Name: "nova-ios.ipa"},
	}
	cases := []struct{ platform, arch, want string }{
		{"linux", "amd64", "nova-linux-amd64.tar.gz"},
		{"linux", "arm64", ""},
		{"windows", "amd64", "nova-windows-amd64.exe"},
		{"windows", "arm64", ""},
		{"darwin", "arm64", ""},
	}
	for _, c := range cases {
		i := matchAsset(updater.CheckRequest{Platform: c.platform, Arch: c.arch}, assets)
		got := ""
		if i >= 0 {
			got = assets[i].Name
		}
		if got != c.want {
			t.Errorf("%s/%s: got %q, want %q", c.platform, c.arch, got, c.want)
		}
	}
}
