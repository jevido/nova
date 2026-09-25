<p align="center"><img src="build/appicon.png" width="128" alt="Nova logo"></p>

# Nova

[![CI](https://github.com/jevido/nova/actions/workflows/ci.yml/badge.svg)](https://github.com/jevido/nova/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/jevido/nova)](https://github.com/jevido/nova/releases/latest)

A file manager for [nova.storage](https://nova.storage) for Linux, Windows,
Android and iPhone/iPad, built with [Wails v3](https://v3.wails.io) (Go) and
Svelte 5. It looks and works like today's GNOME Files (Nautilus) with
libadwaita: a sidebar with the main menu, a path bar with a folder menu, icon
and list views, popover menus, "Undo" toasts and the familiar keyboard
shortcuts. In narrow windows and on phones the navigation and view buttons move
to a bottom bar.
On Linux, folder and file-type icons come from your system icon theme, so it
matches the rest of your desktop; elsewhere Nova uses its bundled Adwaita
icons.

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
and tap it when the download finishes. The first time, Android asks you to allow
installing apps from your browser; allow it and tap **Install**.

Requires Android 5.0 or newer on a 64-bit ARM phone. Nova updates itself from
then on (see [Updates](#updates)).

### Windows

Download
**[nova-windows-amd64-setup.exe](https://github.com/jevido/nova/releases/latest/download/nova-windows-amd64-setup.exe)**
and run it. It installs Nova for your user account only (into
`%LOCALAPPDATA%\Programs\Nova`, no admin rights needed) and adds it to the Start
menu. To run Nova without installing it, download the portable
**[nova-windows-amd64.exe](https://github.com/jevido/nova/releases/latest/download/nova-windows-amd64.exe)**,
put it in a folder you can write to and double-click it. Both update themselves.

Nova isn't code-signed yet, so the first time Windows SmartScreen says
"Windows protected your PC". Click **More info**, then **Run anyway**.

Requires Windows 10 or 11 (64-bit) with the WebView2 runtime, which comes with
current Windows versions; the installer adds it when it's missing. Settings are
kept in `%APPDATA%\nova-desktop`.

### iPhone and iPad (sideloading)

Nova isn't on the App Store or TestFlight: both need a paid Apple Developer
account, which the project doesn't have. The release has an unsigned
**[nova-ios.ipa](https://github.com/jevido/nova/releases/latest/download/nova-ios.ipa)**
that you sign with your own Apple ID when you install it:

1. Install [AltStore](https://altstore.io) or [SideStore](https://sidestore.io)
   on your iPhone, or [Sideloadly](https://sideloadly.io) on your computer.
2. Download `nova-ios.ipa` and open it with that app (in Sideloadly: connect
   your phone, drop the IPA in, enter your Apple ID and click **Start**).
3. The first time, trust your Apple ID under **Settings › General › VPN &
   Device Management**, and turn on **Developer Mode** if iOS asks for it.

With a free Apple ID an installed app expires after 7 days; AltStore and
SideStore refresh it in the background, with Sideloadly you install it again.
Requires iOS 15 or newer. Nova can't update itself on iOS; it tells you when a
new version is out, and you install that the same way. Downloads go to
**Files › On My iPhone › Nova**.

There is no macOS build yet.

### Manual download

Download the latest build from
[Releases](https://github.com/jevido/nova/releases/latest).

| Platform | Download | Updates itself |
| --- | --- | --- |
| Linux | `nova-linux-amd64.tar.gz`: extract anywhere you can write, run `./nova` | yes |
| Arch / Omarchy | `nova-linux-x86_64.pkg.tar.zst`: `sudo pacman -U nova-linux-x86_64.pkg.tar.zst` | no, you're told when a new version is out |
| Debian / Ubuntu | `nova-linux-amd64.deb`: `sudo apt install ./nova-linux-amd64.deb` | no, you're told when a new version is out |
| Fedora / RHEL | `nova-linux-x86_64.rpm`: `sudo dnf install ./nova-linux-x86_64.rpm` | no, you're told when a new version is out |
| Any Linux | `nova-linux-x86_64.AppImage`: `chmod +x` it and run it | no, you get a download link |
| Windows | `nova-windows-amd64-setup.exe`: per-user installer | yes |
| Windows (portable) | `nova-windows-amd64.exe`: run it from a folder you can write to | yes |
| Android | `nova-android-arm64.apk` | yes |
| iPhone / iPad | `nova-ios.ipa`: sideload it (see above) | no, you're told when a new version is out |

Linux needs GTK 4 and WebKitGTK 6.0 (`libgtk-4-1 libwebkitgtk-6.0-4` on
Debian/Ubuntu 24.04+, `gtk4 webkitgtk-6.0` on Arch, `gtk4 webkitgtk6.0` on
Fedora).

Pushes to `main` only run the tests. To build packages without releasing,
start the CI workflow by hand in
[GitHub Actions](https://github.com/jevido/nova/actions/workflows/ci.yml)
(**Run workflow**) and download the `linux`, `windows`, `android` or `ios` artifact from the run.
Those builds don't update themselves.

### Updates

Release builds check GitHub Releases shortly after startup and then every 6
hours; **Check for Updates** in the menu checks right away. A newer version is
downloaded in the background and verified against the release's
`checksums.txt`, then Nova shows a notification:

- **Desktop**: click **Restart** and Nova relaunches into the new version. On
  Windows the running `nova.exe` can't be overwritten, so it is renamed aside
  (`nova.exe.old.*`, deleted on a later start) and the new one takes its place.
  A copy installed where you can't write (like Program Files) gets a download
  link instead.
- **Android**: tap **Install** and Android's installer takes over. The first
  time, Android asks you to allow Nova to install apps. Android only accepts
  the update because it is signed with the same key as the installed app.
- **iOS**: the notification links to the release; install the new IPA the way
  you installed Nova.

Pre-releases are never offered. Set `NOVA_NO_UPDATE=1` to turn checking off on
the desktop.

### Signing in

Sign in with your nova.storage username and password, or paste an API key.
Keys created by a password sign-in are revoked again when you sign out.

## Features

- Grid and list views, zoom levels (`Ctrl+scroll`), sorting, hidden files,
  rubber-band selection
- Path bar with editable location entry (`Ctrl+L`), back/forward/up history
- Search below the current folder (`Ctrl+F`), or everywhere at once with
  Search Everywhere (`Shift+Ctrl+F`, like GNOME Files): Enter opens a match,
  `Alt+Enter` shows it in its folder
- Upload files and folders: menu, `Ctrl+U`, or drag from your desktop file
  manager onto the window (or onto a folder). Existing files are never
  overwritten; clashing names become "name (copy)".
- Download files and folders (recursively) to a local folder, or drag them
  out of Nova onto your file manager or desktop (Linux)
- Paste files copied in your file manager (`Ctrl+V`) to upload them (Linux)
- Open files with the default local application (downloaded to a cache first)
- Quick preview for images, video, audio and text (`Space`)
- Cut/copy/paste, also between Nova windows, drag files and folders onto a folder, a path bar segment or
  a sidebar entry to move them (hold `Ctrl` to copy, `Esc` cancels), rename
  (`F2`)
- Mouse back/forward buttons navigate; middle-click a folder to open it in a
  new window
- Trash with restore and empty, stored on the server in `/me/.Trash`, so it
  works across devices; permanent delete with `Shift+Delete`
- Undo (`Ctrl+Z`) for rename, move and trash
- Starred items, and your own shortcuts in the sidebar: drag a folder between
  the sidebar rows (or onto "New Bookmark") to add it, drag bookmarks to
  reorder them, right-click to rename, move or remove. Bookmarks are stored in
  `/me/.nova/bookmarks.json`, the same file the nova.storage website uses, so
  they are the same everywhere.
- Settings (click your name in the sidebar, or `Ctrl+,`): storage and transfer
  used, your plan, graphs of transfer and downloads over a day up to a year,
  update checks and sign-out. **Folders** creates a recommended layout
  (Documents, Downloads, Music, Pictures, Videos, Projects, Backups); it only
  adds folders you don't have and never removes or renames anything.
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
task package GOOS=windows ARCH=amd64  # bin/nova.exe + bin/nova-amd64-installer.exe (cross-compiles; needs makensis)
task ios:package:ipa IOS_PLATFORM=device # bin/nova.ipa, ad-hoc signed (macOS with Xcode only)
```

For development you can skip the sign-in screen by exporting `NOVA_API_KEY`;
that key is used for the session only and is never written to disk.

### Releasing

Add a section for the version to `CHANGELOG.md` first: the app shows it in
the What's New dialog after updating, and CI uses it as the release notes.
Then push a version tag; CI builds everything and publishes a GitHub release:

```sh
git tag v0.2.0
git push origin v0.2.0
```

Release assets use version-less names (`nova-linux-amd64.tar.gz`,
`nova-windows-amd64.exe`, `nova-android-arm64.apk`, `nova-ios.ipa`, …) so `releases/latest/download/…` links always
point at the newest release. Tags with a `-` (like `v0.3.0-rc.1`) are
published as pre-releases.

The Android APK is signed with the key in the `ANDROID_KEYSTORE_BASE64`,
`ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS` and `ANDROID_KEY_PASSWORD`
repository secrets. Keep a backup of that keystore: Android refuses to update
an installed app with an APK signed by a different key. Without the secrets
(for example in pull requests from forks) CI signs with a throwaway debug key.

The Windows exe and installer are not code-signed. The iOS IPA is ad-hoc
signed, for sideloading. To sign it properly (and to be able to use
TestFlight or the App Store later) add an Apple Developer certificate as the
`IOS_CERTIFICATE_P12_BASE64` / `IOS_CERTIFICATE_PASSWORD` secrets and a
provisioning profile for `storage.nova.app` as `IOS_PROVISIONING_PROFILE_BASE64`;
CI then signs with them.

### Layout

```
main.go                    app + window setup, OS file-drop wiring
internal/nova              API client for the nova.storage filesystem API (no Wails deps)
internal/config            settings + credentials (0600) in the user config dir
internal/platform          per-OS config, cache and download locations (desktop / Android / iOS)
internal/version           build version (stamped by CI) and version comparison
internal/icons             freedesktop icon-theme lookup (Linux) with a bundled Adwaita fallback
services/                  Wails services: session, files, transfers, updates, media (HTTP route)
frontend/src               Svelte UI; lib/store.svelte.ts holds all app state
build/                     packaging: Linux (nfpm, AppImage), Windows (NSIS), Android (Gradle), iOS
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
