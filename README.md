# Nova

A file manager for [nova.storage](https://nova.storage) for your desktop and
your Android phone, built with [Wails v3](https://v3.wails.io) (Go) and
Svelte 5. It is modelled on GNOME Files (Nautilus) from the GTK 3 era: Adwaita
headerbar with a path bar, places sidebar, icon and list views, GtkMenu-style
context menus, in-app "Undo" notifications and the familiar keyboard shortcuts.
On the desktop, folder and file-type icons come from your system icon theme, so
it matches the rest of your desktop.

## Install

Every push to `main` builds all packages in
[GitHub Actions](https://github.com/jevido/nova/actions/workflows/ci.yml)
(open a run and download the `linux` or `android` artifact). Tagged versions
are published on the [Releases](https://github.com/jevido/nova/releases) page.

### Linux

Nova needs GTK 4 and WebKitGTK 6.0, which recent distributions ship
(Ubuntu 24.04+, Debian 13+, Fedora 40+, Arch).

| Distribution    | File                     | Install                                              |
| --------------- | ------------------------ | ---------------------------------------------------- |
| Arch / Omarchy  | `nova-*.pkg.tar.zst`     | `sudo pacman -U nova-*.pkg.tar.zst`                  |
| Debian / Ubuntu | `nova_*_amd64.deb`       | `sudo apt install ./nova_*_amd64.deb`                |
| Fedora / RHEL   | `nova-*.x86_64.rpm`      | `sudo dnf install ./nova-*.x86_64.rpm`               |
| Anything else   | `nova-*.AppImage`        | `chmod +x nova-*.AppImage && ./nova-*.AppImage`      |

The packages install `nova` to `/usr/bin` and add Nova to your application
launcher. There is also a bare `nova-*-linux-amd64` binary if you want to put
it somewhere yourself.

### Android

1. Download `nova-*-android-arm64.apk` on your phone (from Releases, or from
   the `android` artifact of a CI run, which is a zip you need to extract).
2. Open it. Android asks you to allow installing apps from your browser or
   file manager the first time; allow it and tap **Install**.
3. Updating works the same way: install a newer APK over the old one. All
   builds are signed with the same key, so your sign-in is kept.

Requires Android 5.0 or newer on a 64-bit ARM phone (practically every phone
from the last years). On the phone Nova works with touch: tap to open,
long-press for the actions menu (tapping more items then adds them to the
selection), and the back gesture goes up through your folder history. Downloads
are saved to `Download/Nova`. Opening files in other apps and uploading whole
folders are desktop-only for now.

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
internal/icons             freedesktop icon-theme lookup with a bundled Adwaita fallback
services/                  Wails services: session, files, transfers, media (HTTP route)
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
