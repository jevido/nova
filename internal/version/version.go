// Package version holds build metadata stamped in by the build:
//
//	-ldflags "-X nova/internal/version.Version=1.2.3 -X nova/internal/version.Repo=owner/repo -X nova/internal/version.Build=abc1234"
package version

import (
	"runtime"
	"strconv"
	"strings"
)

var (
	// Version is "dev" for local builds, which never update themselves.
	Version = "dev"
	// Repo is the GitHub repository releases are fetched from.
	Repo = "jevido/nova"
	// Channel is "dev" in the Nova (dev) builds of the next branch (see
	// build/dev-app.sh). They install next to Nova with their own name,
	// settings and app IDs, so both can be used side by side.
	Channel = ""
	// Build is the commit the build was made from, empty for local builds.
	Build = ""
)

// UserAgent identifies Nova to nova.storage, so its version and build show
// up there: "Nova/0.7.0 (linux; amd64; build abc1234)", or "Nova-dev/..."
// for Nova (dev).
func UserAgent() string {
	product := "Nova"
	if Dev() {
		product = "Nova-dev"
	}
	v := Version
	if v == "" {
		v = "dev"
	}
	ua := product + "/" + strings.TrimPrefix(v, "v") + " (" + runtime.GOOS + "; " + runtime.GOARCH
	if Build != "" {
		ua += "; build " + Build
	}
	return ua + ")"
}

// Dev reports whether this is a Nova (dev) build.
func Dev() bool { return Channel == "dev" }

// AppName is the name people see: "Nova", or "Nova (dev)".
func AppName() string {
	if Dev() {
		return "Nova (dev)"
	}
	return "Nova"
}

// Suffix is appended to file names and IDs that must differ between Nova
// and Nova (dev): "" or "-dev".
func Suffix() string {
	if Dev() {
		return "-dev"
	}
	return ""
}

// IsRelease reports whether this is a stamped release build.
func IsRelease() bool { return Version != "dev" && Version != "" }

// Newer reports whether a is a newer semantic version than b. A leading "v"
// is ignored, and a pre-release (1.2.0-rc.1) sorts before its release.
func Newer(a, b string) bool { return compare(a, b) > 0 }

func compare(a, b string) int {
	a, b = strings.TrimPrefix(a, "v"), strings.TrimPrefix(b, "v")
	ac, apre, _ := strings.Cut(a, "-")
	bc, bpre, _ := strings.Cut(b, "-")
	ap, bp := strings.Split(ac, "."), strings.Split(bc, ".")
	for i := 0; i < 3; i++ {
		x, y := part(ap, i), part(bp, i)
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	switch {
	case apre == bpre:
		return 0
	case apre == "":
		return 1
	case bpre == "":
		return -1
	case apre > bpre:
		return 1
	default:
		return -1
	}
}

func part(p []string, i int) int {
	if i >= len(p) {
		return 0
	}
	n, _ := strconv.Atoi(p[i])
	return n
}
