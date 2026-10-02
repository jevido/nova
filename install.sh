#!/bin/sh
# Nova installer for Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/jevido/nova/main/install.sh | sh
#
# Installs the latest release into your home directory (no sudo), where Nova
# can update itself, and adds it to your app launcher:
#   ~/.local/share/nova/nova, linked as ~/.local/bin/nova
#
# Uninstall: curl -fsSL https://raw.githubusercontent.com/jevido/nova/main/install.sh | sh -s -- --uninstall
#
# NOVA_CHANNEL=dev installs Nova (dev), the test build of the next branch,
# next to Nova instead: ~/.local/share/nova-dev/nova, linked as
# ~/.local/bin/nova-dev. It doesn't update itself.
set -eu

REPO=${NOVA_REPO:-jevido/nova}
if [ "${NOVA_CHANNEL:-}" = dev ]; then
  app=nova-dev name="Nova (dev)" branch=next
  BASE=${NOVA_DOWNLOAD_BASE:-https://github.com/$REPO/releases/download/next}
else
  app=nova name=Nova branch=main
  BASE=${NOVA_DOWNLOAD_BASE:-https://github.com/$REPO/releases/latest/download}
fi
RAW=${NOVA_RAW:-https://raw.githubusercontent.com/$REPO/$branch}

say() { printf '\033[1m%s\033[0m\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required"; }

fetch() { # url dest
  if command -v curl >/dev/null 2>&1; then curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then wget -qO "$2" "$1"
  else fail "curl or wget is required"; fi
}

data=${XDG_DATA_HOME:-$HOME/.local/share}
bin=$HOME/.local/bin
icon=$data/icons/hicolor/256x256/apps/$app.png
desktop=$data/applications/$app.desktop
settings=nova-desktop${app#nova}

uninstall() {
  rm -rf "${data:?}/$app" "$bin/$app" "$desktop" "$icon"
  command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$data/applications" >/dev/null 2>&1 || true
  say "$name removed. Your settings are kept in ~/.config/$settings (delete it to remove them too)."
}

[ "${1:-}" = "--uninstall" ] && { uninstall; exit 0; }

[ "$(uname -s)" = Linux ] || fail "this installer is for Linux; for Android see https://github.com/$REPO#android"
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  *) fail "no Nova build for $(uname -m) yet (only x86_64)" ;;
esac
need tar

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "Downloading $name…"
fetch "$BASE/nova-linux-$arch.tar.gz" "$tmp/nova.tar.gz"
fetch "$BASE/checksums.txt" "$tmp/checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp" && grep " nova-linux-$arch.tar.gz\$" checksums.txt | sed "s| nova-linux-$arch.tar.gz\$| nova.tar.gz|" | sha256sum -c --status) \
    || fail "download failed its checksum; please try again"
fi
tar -xzf "$tmp/nova.tar.gz" -C "$tmp"

mkdir -p "$data/$app" "$bin" "$data/applications" "$(dirname "$icon")"
install -m 755 "$tmp/nova" "$data/$app/nova"
ln -sf "$data/$app/nova" "$bin/$app"
fetch "$RAW/build/appicon.png" "$icon" || true
fetch "$RAW/build/linux/$app.desktop" "$tmp/$app.desktop"
sed "s|^Exec=.*|Exec=$data/$app/nova|" "$tmp/$app.desktop" > "$desktop"
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$data/applications" >/dev/null 2>&1 || true

# Nova needs GTK 4 and WebKitGTK 6.0.
if ! (ldconfig -p 2>/dev/null | grep -q 'libwebkitgtk-6.0.so'); then
  say "$name needs WebKitGTK 6.0, which doesn't seem to be installed:"
  if command -v pacman >/dev/null 2>&1; then echo "  sudo pacman -S --needed gtk4 webkitgtk-6.0"
  elif command -v apt-get >/dev/null 2>&1; then echo "  sudo apt install libgtk-4-1 libwebkitgtk-6.0-4"
  elif command -v dnf >/dev/null 2>&1; then echo "  sudo dnf install gtk4 webkitgtk6.0"
  else echo "  install gtk4 and webkitgtk-6.0 with your package manager"; fi
fi

say "$name is installed. Start it from your app launcher, or run: $app"
case ":$PATH:" in *":$bin:"*) ;; *) echo "(add $bin to your PATH to run it from a terminal)" ;; esac
