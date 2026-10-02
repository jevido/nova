# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

One libadwaita-style design language rendered in a webview (Wails v3 + Svelte 5)
on Linux, Windows, Android and iPhone/iPad. Per-OS native adaptation is
undecided, not ruled out (see Capabilities and Constraints).

## Users

Primary: nova.storage account holders on Linux desktops (GNOME, Omarchy and
similar), who want their cloud storage to behave like a local disk in a file
manager they already know. They browse, upload, move, preview and share files as
part of everyday desktop work, and expect GNOME Files' shortcuts and behaviour
to carry over.

Secondary: the same people on Windows, Android and iPhone/iPad, reaching the
same files from another device. Phone use is real (bottom bar, viewer, native
downloads) but follows the desktop, not the other way round.

## Product Purpose

Nova is a free, open-source file manager for nova.storage. It puts a
nova.storage account in a window that works like GNOME Files (Nautilus) with
libadwaita: sidebar, path bar, grid and list views, popover menus, Undo toasts,
drag and drop, and the familiar keyboard shortcuts. Success is a nova.storage
user forgetting the files are remote: no browser tab, no extra account, nothing
new to learn.

## Positioning

Nova copies GNOME Files where it can, down to the keyboard shortcuts, and on
Linux uses the system icon theme so it matches the rest of the desktop. State
that lives on the server is shared with the nova.storage website and other
devices: Trash in `/me/.Trash`, bookmarks in `/me/.nova/bookmarks.json`. It is a
desktop-grade client, not a web uploader in a wrapper.

## Operating Context

- Used alongside the user's local file manager: drag files in from it, drag out
  to it (Linux), paste files copied there (`Ctrl+V`, Linux), open files with the
  default local app via a download cache.
- Keyboard-driven desktop workflow: `Ctrl+L`, `Ctrl+F`, `Shift+Ctrl+F` Search
  Everywhere, `F2`, `Space` preview, `Ctrl+Z` undo, `Ctrl+N` windows, `Ctrl+?`
  shortcut list, mouse back/forward, middle-click to new window.
- Narrow windows and phones move navigation and view buttons to a bottom bar.
- Installed via GitHub Releases (install script, tarball, deb, rpm, Arch
  package, AppImage, Windows installer/portable, APK, sideloaded IPA); most
  builds update themselves in the background and ask before restarting.
- Release notes live in `CHANGELOG.md` and appear in the in-app What's New
  dialog; they are written for users.
- A marketing/download site lives in `website/` (nova.jevido.app).

## Capabilities and Constraints

- Sign-in through the browser (OAuth 2.0 authorization code + PKCE, loopback
  redirect on 127.0.0.1, client ID `jevido-nova`), with nova.storage username +
  password, or with an API key. Browser and password sign-in keys are revoked on
  sign-out. A browser sign-in may be limited to some folders (`filesystem_dirs`,
  stored with the key): each is a top folder at `/{id}`, and Home, Trash and
  bookmark sync are unavailable.
- Features: grid/list views, zoom, sort, hidden files, rubber-band selection,
  path bar with editable location, history, search, upload/download (recursive),
  cut/copy/paste across windows, drag-to-move (Ctrl to copy), rename, Trash with
  restore/empty, permanent delete, undo for rename/move/trash, Starred, Recent,
  Shared, sidebar bookmarks, quick preview (image/video/audio/text), Properties
  with folder size, checksums and public-link toggle, background transfers with
  progress/speed/cancel, settings with storage/transfer graphs and a recommended
  folder layout, light/dark following the system or forced.
- The API has no server-side copy; copies stream down and back up through the
  transfer service.
- The API key never reaches JavaScript; the `media` service serves icons,
  thumbnails and file contents to the webview at `/nova/`.
- Non-destructive by default: uploads never overwrite (clashes become "name
  (copy)"); the folder-layout helper only adds folders.
- Design language: GNOME Files / libadwaita is the current, binding reference on
  every platform. Whether Windows, Android and iOS later follow their own native
  conventions is an open decision.
- No macOS build yet. Not on the App Store or TestFlight. Windows builds are not
  code-signed.
- Terminology follows GNOME Files: Trash, Starred, Recent, Shared, Properties,
  Search Everywhere, New Bookmark, Main Menu.

## Brand Commitments

- Name: **Nova**. App icon `build/appicon.png`; website assets
  `website/assets/nova-logo.png`, `nova-header.webp`.
- Relationship to nova.storage: present Nova as an **independent, open-source
  client** for nova.storage on all public surfaces, until the nova.storage owner
  (fornax) acknowledges it. Do not describe it as official, endorsed or made by
  nova.storage.
- Voice (from README, website, changelog): plain, direct, second person, short
  sentences, concrete about what happens ("Existing files are never
  overwritten"), honest about limits (unsigned builds, no macOS, iOS refresh
  every 7 days). No hype.
- Type: Adwaita Sans and Adwaita Mono (SIL OFL), self-hosted on the
  website (`website/assets/fonts/`); the app falls back to Inter
  (`frontend/Inter Font License.txt`).

## Evidence on Hand

- Real feature set and release history: `README.md`, `CHANGELOG.md` (0.6.0 as of
  2026-09-28).
- Website copy and download instructions: `website/index.html`.
- Android download QR: `website/assets/android-qr.svg`.
- No testimonials, user counts, reviews, press or benchmarks exist. Do not
  invent any.

## Product Principles

1. **Feel like GNOME Files.** When in doubt, do what Nautilus does; familiarity
   beats novelty.
2. **Remote files, local manners.** Hide the network: background transfers,
   previews without downloading, undo, server-side Trash and bookmarks shared
   with the website.
3. **Never lose user data.** No silent overwrites, undo where possible, additive
   helpers, checksums verified.
4. **Linux desktop first, every device reachable.** Design for the Linux
   desktop; adapt for narrow windows and phones without breaking the desktop
   model.
5. **Say what's true.** Plain, honest copy, including the rough edges.
