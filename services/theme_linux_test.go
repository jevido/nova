//go:build linux && !android

package services

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A fresh EndeavourOS KDE account: only /etc/xdg/kdeglobals from
// eos-settings-plasma, and the look-and-feel says Breeze Dark.
func TestKDEThemeFreshEndeavourOS(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc", "xdg")
	write(t, filepath.Join(etc, "kdeglobals"), "[KDE]\nLookAndFeelPackage=com.endeavouros.breezedarkeos.desktop\n\n[General]\nAccentColor=146,110,228\nLastUsedCustomAccentColor=146,110,228\n")
	var th SystemTheme
	readKDETheme(&th, filepath.Join(root, "home"), []string{etc}, []string{filepath.Join(root, "share")})
	if th.Name != "EndeavourOS" || th.Mode != "dark" || th.Accent != "#926ee4" {
		t.Fatalf("got name %q mode %q accent %q", th.Name, th.Mode, th.Accent)
	}
	if th.Palette[PalView] != "#141618" || th.Palette[PalWindow] != "#202326" {
		t.Fatalf("palette %v", th.Palette)
	}
}

// After Plasma applied a scheme: colours, accent and font in kdeglobals.
func TestKDEThemeUserScheme(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	write(t, filepath.Join(home, ".config", "kdeglobals"), `[General]
ColorScheme=BreezeLight
font=Noto Sans,10,-1,5,50,0,0,0,0,0

[Colors:Window]
BackgroundNormal=239,240,241

[Colors:View]
BackgroundNormal=255,255,255
ForegroundNormal=35,38,41
ForegroundNegative=218,68,83

[Colors:Selection]
BackgroundNormal=61,174,233
ForegroundNormal=255,255,255
`)
	var th SystemTheme
	readKDETheme(&th, home, nil, nil)
	if th.Mode != "light" || th.Accent != "#3daee9" || th.Font != "Noto Sans" || th.FontSize != 10 || th.Name != "BreezeLight" {
		t.Fatalf("got %+v", th)
	}
	if th.Palette[PalFg] != "#232629" || th.Palette[PalDestructive] != "#da4453" {
		t.Fatalf("palette %v", th.Palette)
	}
}

// A scheme named in kdeglobals whose colours are only in its .colors file.
func TestKDEThemeSchemeFile(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	share := filepath.Join(root, "share")
	write(t, filepath.Join(home, ".config", "kdeglobals"), "[General]\nColorScheme=Nordic\n")
	write(t, filepath.Join(share, "color-schemes", "Nordic.colors"), "[Colors:Window]\nBackgroundNormal=46,52,64\n[Colors:View]\nBackgroundNormal=41,46,57\n[Colors:Selection]\nBackgroundNormal=136,192,208\n")
	var th SystemTheme
	readKDETheme(&th, home, nil, []string{share})
	if th.Mode != "dark" || th.Accent != "#88c0d0" || th.Palette[PalView] != "#292e39" {
		t.Fatalf("got %+v", th)
	}
}

func TestOmarchyTheme(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "state", "omarchy", "current")
	write(t, filepath.Join(dir, "theme.name"), "tokyo-night\n")
	write(t, filepath.Join(dir, "theme", "colors.toml"), "mode = \"dark\"\naccent = \"#7aa2f7\"\nbackground = \"#1a1b26\"\nlighter_background = \"#24283b\"\nforeground = \"#a9b1d6\"\nred = \"#f7768e\"\n")
	var th SystemTheme
	if !readOmarchyTheme(&th, home) {
		t.Fatal("not read")
	}
	if th.Name != "tokyo-night" || th.Mode != "dark" || th.Palette[PalSidebar] != "#24283b" || th.Palette[PalDestructive] != "#f7768e" {
		t.Fatalf("got %+v", th)
	}
}
