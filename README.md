<p align="center"><img src="build/appicon.png" width="128" alt="Nova logo"></p>

# Nova

[![CI](https://github.com/jevido/nova/actions/workflows/ci.yml/badge.svg)](https://github.com/jevido/nova/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/jevido/nova)](https://github.com/jevido/nova/releases/latest)

A file manager for [nova.storage](https://nova.storage) for your desktop and
your Android phone, built with [Wails v3](https://v3.wails.io) (Go) and
Svelte 5. It is modelled on GNOME Files (Nautilus) from the GTK 3 era: Adwaita
headerbar with a path bar, places sidebar, icon and list views, GtkMenu-style
context menus, in-app "Undo" notifications and the familiar keyboard shortcuts.
On the desktop, folder and file-type icons come from your system icon theme, so
it matches the rest of your desktop.

## Install

### Quick install (Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/jevido/nova/main/install.sh | sh
```

This installs the latest release into your home directory (no sudo), where
Nova can update itself: the binary goes to `~/.local/share/nova`, linked as
`~/.local/bin/nova`, and Nova is added to your app launcher. The download is
checked against the release's `checksums.txt`. If GTK 4 or WebKitGTK 6.0 is
missing, the script prints the command to install them.

To uninstall:

```sh
curl -fsSL https://raw.githubusercontent.com/jevido/nova/main/install.sh | sh -s -- --uninstall
```

Your settings stay in `~/.config/nova-desktop`.

### Quick install (Android)

On your phone, open
**[nova-android-arm64.apk](https://github.com/jevido/nova/releases/latest/download/nova-android-arm64.apk)**
and tap it when the download finishes. The first time, Android asks you to
allow installing apps from your browser; allow it and tap **Install**.

Requires Android 5.0 or newer on a 64-bit ARM phone. Nova updates itself from
then on (see [Updates](#updates)).

### Manual download

Download the latest build from [Releases](https://github.com/jevido/nova/releases/latest).

| Platform | Download | Updates itself |
| --- | --- | --- |
| Linux | `nova-linux-amd64.tar.gz`: extract anywhere you can write, run `./nova` | yes |
| Arch / Omarchy | `nova-linux-x86_64.pkg.tar.zst`: `sudo pacman -U nova-linux-x86_64.pkg.tar.zst` | no, you're told when a new version is out |
| Debian / Ubuntu | `nova-linux-amd64.deb`: `sudo apt install ./nova-linux-amd64.deb` | no, you're told when a new version is out |
| Fedora / RHEL | `nova-linux-x86_64.rpm`: `sudo dnf install ./nova-linux-x86_64.rpm` | no, you're told when a new version is out |
| Any Linux | `nova-linux-x86_64.AppImage`: `chmod +x` it and run it | no, you get a download link |
| Android | `nova-android-arm64.apk` | yes |

Linux needs GTK 4 and WebKitGTK 6.0 (`libgtk-4-1 libwebkitgtk-6.0-4` on
Debian/Ubuntu 24.04+, `gtk4 webkitgtk-6.0` on Arch, `gtk4 webkitgtk6.0` on
Fedora).

Every push to `main` also builds everything in
[GitHub Actions](https://github.com/jevido/nova/actions/workflows/ci.yml)
(open a run and download the `linux` or `android` artifact). Those builds don't
update themselves.

### Updates

Release builds check GitHub Releases shortly after startup and then every 6
hours; **Check for Updates** in the menu checks right away. A newer version is
downloaded in the background and verified against the release's
`checksums.txt`, then Nova shows a notification:

- **Desktop**: click **Restart** and Nova relaunches into the new version.
- **Android**: tap **Install** and Android's installer takes over. The first
  time, Android asks you to allow Nova to install apps. Android only accepts
  the update because it is signed with the same key as the installed app.

Pre-releases are never offered. Set `NOVA_NO_UPDATE=1` to turn checking off on
the desktop.

### Signing in

Sign in with your nova.storage username and password, or paste an API key.
Keys created by a password sign-in are revoked again when you sign out.

## Features

- Grid and list views, zoom levels (`Ctrl+scroll`), sorting, hidden files,
  rubber-band selection
- Path bar with editable location entry (`Ctrl+L`), back/forward/up history
- Search below the current folder (`Ctrl+F`)
- Upload files and folders: menu, `Ctrl+U`, or drag from your desktop file
  manager onto the window (or onto a folder). Existing files are never
  overwritten; clashing names become "name (copy)".
- Download files and folders (recursively) to a local folder
- Open files with the default local application (downloaded to a cache first)
- Quick preview for images, video, audio and text (`Space`)
- Cut/copy/paste, drag-and-drop move (hold `Ctrl` to copy), rename (`F2`)
- Trash with restore and empty, stored on the server in `/me/.Trash`, so it
  works across devices; permanent delete with `Shift+Delete`
- Undo (`Ctrl+Z`) for rename, move and trash
- Starred items and folder bookmarks in the sidebar
- Properties dialog with folder size, checksums and a public-link toggle
- Background transfers with progress, speed and cancel (the operations button)
- Multiple windows (`Ctrl+N`)
- Light/dark style following the system, or forced from the menu

Press `Ctrl+?` in the app for the full list of shortcuts.

## Development

Requirements: Go 1.25+,
[Bun](https://bun.sh), [Task](https://taskfile.dev), the `wails3` CLI
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.18`, matching
`go.mod`) and on Linux the GTK 4 / WebKitGTK 6.0 development packages.

```sh
wails3 dev                    # hot-reloading dev build
task build                    # production binary in bin/nova
task package                  # AppImage, deb, rpm and Arch package in bin/
task android:package ARCH=arm64   # bin/nova.apk (needs the Android SDK + NDK and Java 21)
```

For development you can skip the sign-in screen by exporting `NOVA_API_KEY`;
that key is used for the session only and is never written to disk.

### Releasing

Push a version tag; CI builds everything and publishes a GitHub release:

```sh
git tag v0.2.0
git push origin v0.2.0
```

Release assets use version-less names (`nova-linux-amd64.tar.gz`,
`nova-android-arm64.apk`, …) so `releases/latest/download/…` links always
point at the newest release. Tags with a `-` (like `v0.3.0-rc.1`) are
published as pre-releases.

The Android APK is signed with the key in the `ANDROID_KEYSTORE_BASE64`,
`ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS` and `ANDROID_KEY_PASSWORD`
repository secrets. Keep a backup of that keystore: Android refuses to update
an installed app with an APK signed by a different key. Without the secrets
(for example in pull requests from forks) CI signs with a throwaway debug key.

### Layout

```
main.go                    app + window setup, OS file-drop wiring
internal/nova              API client for the nova.storage filesystem API (no Wails deps)
internal/config            settings + credentials (0600) in the user config dir
internal/platform          per-OS config, cache and download locations (desktop / Android)
internal/version           build version (stamped by CI) and version comparison
internal/icons             freedesktop icon-theme lookup with a bundled Adwaita fallback
services/                  Wails services: session, files, transfers, updates, media (HTTP route)
frontend/src               Svelte UI; lib/store.svelte.ts holds all app state
build/                     packaging: Linux (nfpm, AppImage), Android (Gradle project)
```

The `media` service is mounted at `/nova/` in the asset server and serves icons,
thumbnails and file contents to the webview, so `<img>`/`<video>` tags work
without the API key ever reaching JavaScript.

### Tests

```sh
go test ./internal/... ./services/
NOVA_API_KEY=... go test ./services -run Live -v   # exercises the real API in a temp folder
```

### Notes

- The API has no server-side copy; copies are streamed down and back up by the
  transfer service, with progress.
- Bundled fallback icons are from the Adwaita icon theme (LGPL-3.0 /
  CC-BY-SA-3.0).
