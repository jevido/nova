# Nova Desktop

A desktop file manager for [nova.storage](https://nova.storage), built with
[Wails v3](https://v3.wails.io) (Go) and Svelte 5. It is modelled on GNOME Files
(Nautilus) from the GTK 3 era: Adwaita headerbar with a path bar, places
sidebar, icon and list views, GtkMenu-style context menus, in-app "Undo"
notifications and the familiar keyboard shortcuts. Folder and file-type icons
come from your system icon theme, so it matches the rest of your desktop.

## Features

- Grid and list views, zoom levels, sorting, hidden files, rubber-band selection
- Path bar with editable location entry (`Ctrl+L`), back/forward/up history
- Search below the current folder (`Ctrl+F`)
- Upload files and folders: menu, `Ctrl+U`, or drag from your desktop file
  manager onto the window (or onto a folder)
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

Requirements: Go 1.25+, [Bun](https://bun.sh), the `wails3` CLI
(`v3.0.0-beta.18`, matching `go.mod`) and on Linux WebKitGTK/GTK development
packages.

```sh
wails3 dev          # hot-reloading dev build
task build          # production binary in bin/nova
task package        # AppImage / deb / rpm on Linux
```

Sign in with your nova.storage username and password, or with an API key. For
development you can skip the sign-in screen by exporting `NOVA_API_KEY`; that
key is used for the session only and is never written to disk.

### Layout

```
main.go                 app + window setup, OS file-drop wiring
internal/nova           API client for the nova.storage filesystem API (no Wails deps)
internal/config         settings + credentials in ~/.config/nova-desktop (0600)
internal/icons          freedesktop icon-theme lookup with a bundled Adwaita fallback
services/               Wails services: session, files, transfers, media (HTTP route)
frontend/src            Svelte UI; lib/store.svelte.ts holds all app state
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
