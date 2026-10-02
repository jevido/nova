package version

import (
	"runtime"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.2.0", "0.1.0", true},
		{"v0.2.0", "0.2.0", false},
		{"0.10.0", "0.9.9", true},
		{"1.0.0", "1.0.0-rc.1", true},
		{"1.0.0-rc.1", "1.0.0", false},
		{"1.0.0-rc.2", "1.0.0-rc.1", true},
		{"0.1.0", "0.1.1", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestUserAgent(t *testing.T) {
	defer func(v, b, c string) { Version, Build, Channel = v, b, c }(Version, Build, Channel)
	Version, Build, Channel = "v0.7.0", "77d8cc5", ""
	want := "Nova/0.7.0 (" + runtime.GOOS + "; " + runtime.GOARCH + "; build 77d8cc5)"
	if got := UserAgent(); got != want {
		t.Errorf("UserAgent() = %q, want %q", got, want)
	}
	Version, Build, Channel = "dev", "", "dev"
	want = "Nova-dev/dev (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
	if got := UserAgent(); got != want {
		t.Errorf("UserAgent() = %q, want %q", got, want)
	}
}
