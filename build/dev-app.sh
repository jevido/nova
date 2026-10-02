#!/bin/sh
# Turns this checkout into Nova (dev), the test builds of the next branch.
# CI runs it before building there; never commit what it changes.
#
# Nova (dev) installs next to Nova: it has its own name, app IDs, settings
# (~/.config/nova-desktop-dev, %APPDATA%\nova-desktop-dev), single-instance
# ID, Linux package (nova-dev), Windows install folder and Start menu entry,
# Android package and iOS bundle ID.
set -eu
cd "$(dirname "$0")/.."

# perl rather than sed -i, which differs between GNU and macOS.
edit() { # file perl-expression
  grep -q . "$1" || { echo "dev-app.sh: $1 is missing" >&2; exit 1; }
  perl -0pi -e "$2" "$1"
}

# Go: name, settings and cache folders, single-instance and program ID.
edit internal/version/version.go 's/\tChannel = ""/\tChannel = "dev"/'
grep -q 'Channel = "dev"' internal/version/version.go

# Linux packages: nova-dev, with its own binary, icon and launcher entry.
edit build/linux/nfpm/nfpm.yaml 's/^name: "nova"/name: "nova-dev"/m;
  s|dst: "/usr/bin/nova"|dst: "/usr/bin/nova-dev"|;
  s|apps/nova.png"|apps/nova-dev.png"|;
  s|src: "./build/linux/nova.desktop"\n    dst: "/usr/share/applications/nova.desktop"|src: "./build/linux/nova-dev.desktop"\n    dst: "/usr/share/applications/nova-dev.desktop"|'
grep -q 'nova-dev.desktop' build/linux/nfpm/nfpm.yaml
# The AppImage's launcher entry.
edit build/linux/Taskfile.yml 's/generate .desktop -name "Nova"/generate .desktop -name "Nova (dev)"/'

# Windows: installs into %LOCALAPPDATA%\Programs\Nova (dev).
edit build/windows/nsis/project.nsi 's/!define INFO_PRODUCTNAME "Nova"/!define INFO_PRODUCTNAME "Nova (dev)"/;
  s/!define UNINST_KEY_NAME  "Nova"/!define UNINST_KEY_NAME  "Nova (dev)"/;
  s/nova-desktop\\webview/nova-desktop-dev\\webview/'
edit build/windows/info.json 's/"FileDescription": "Nova"/"FileDescription": "Nova (dev)"/; s/"ProductName": "Nova"/"ProductName": "Nova (dev)"/'

# Android: its own package, so it installs next to Nova.
edit build/android/app/build.gradle 's/applicationId "storage.nova.app"/applicationId "storage.nova.app.dev"/'
edit build/android/Taskfile.yml 's/default "storage.nova.app"/default "storage.nova.app.dev"/'
edit build/android/app/src/main/res/values/strings.xml 's|<string name="app_name">Nova</string>|<string name="app_name">Nova (dev)</string>|'

# iOS: its own bundle ID.
edit build/ios/Info.plist 's|<string>storage.nova.app</string>|<string>storage.nova.app.dev</string>|;
  s|(<key>CFBundleDisplayName</key>\s*)<string>Nova</string>|$1<string>Nova (dev)</string>|'

echo "This checkout now builds Nova (dev)."
