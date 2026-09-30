#!/usr/bin/env bash
# Repacks the AppImage so the bundled WebKitGTK runs, and plays media, on any
# distro. Runs after `wails3 generate appimage`, on the AppDir it leaves.
#
# WebKit: libwebkitgtk has the build host's helper directory compiled in (on
# Ubuntu /usr/lib/x86_64-linux-gnu/webkitgtk-6.0), so on other distros it
# can't start WebKitNetworkProcess and aborts. The AppImage's AppRun always
# starts the app from $APPDIR/usr, so rewriting "/usr" to the same-length
# "././" points WebKit at the helpers bundled in the AppDir. The app turns
# off WebKit's bubblewrap sandbox when run from an AppImage
# (appimage_linux.go), because bwrap can't bind a relative path.
#
# GStreamer: WebKit plays audio and video through GStreamer. linuxdeploy
# bundles its libraries but none of its plugins, and the bundled library only
# looks for plugins next to itself, so media never played. The plugins below
# are copied in with their dependencies; appimage_linux.go points GStreamer
# at them.
#
# Usage: fix-appimage.sh <AppDir> <output AppImage>
set -euo pipefail

appdir="$(realpath "$1")"
out="$2"
arch="$(uname -m)"
builddir="$(dirname "$appdir")"

# Plugins WebKit can't play anything without.
required_plugins=(
    coreelements app playback typefindfunctions autodetect
    audioconvert audioresample volume videoconvertscale
)
# Containers, codecs and outputs. Missing ones are skipped with a warning.
optional_plugins=(
    videofilter opengl pulseaudio alsa
    isomp4 matroska ogg audioparsers videoparsersbad wavparse id3demux icydemux
    vpx opus vorbis theora flac mpg123 libav
)

plugindir="$(dirname "$(find /usr/lib /usr/lib64 -name libgstcoreelements.so -path '*/gstreamer-1.0/*' 2>/dev/null | head -n1 || true)")"
if [[ "$plugindir" == "." ]]; then
    echo "fix-appimage: GStreamer plugins not installed on the build host" >&2
    exit 1
fi
gstdir="$appdir/usr/lib/gstreamer-1.0"
mkdir -p "$gstdir"
for p in "${required_plugins[@]}"; do
    if [[ ! -f "$plugindir/libgst$p.so" ]]; then
        echo "fix-appimage: required GStreamer plugin $p missing from $plugindir" >&2
        exit 1
    fi
    cp "$plugindir/libgst$p.so" "$gstdir/"
done
for p in "${optional_plugins[@]}"; do
    if [[ -f "$plugindir/libgst$p.so" ]]; then
        cp "$plugindir/libgst$p.so" "$gstdir/"
    else
        echo "fix-appimage: skipping GStreamer plugin $p (not installed)" >&2
    fi
done

scanner="$(find /usr/lib /usr/lib64 /usr/libexec -type f -name gst-plugin-scanner 2>/dev/null | head -n1 || true)"
if [[ -z "$scanner" ]]; then
    echo "fix-appimage: gst-plugin-scanner not found on the build host" >&2
    exit 1
fi
mkdir -p "$appdir/usr/libexec/gstreamer-1.0"
cp "$scanner" "$appdir/usr/libexec/gstreamer-1.0/"

# System libraries are already stripped, and linuxdeploy's strip can't handle
# newer toolchains' .relr.dyn sections.
NO_STRIP=1 "$builddir/linuxdeploy-${arch}.AppImage" --appimage-extract-and-run \
    --appdir "$appdir" \
    --deploy-deps-only "$gstdir" \
    --deploy-deps-only "$appdir/usr/libexec/gstreamer-1.0"

# Patch WebKit last, so nothing linuxdeploy copies can undo it.
lib="$(find "$appdir/usr/lib" -maxdepth 1 -type f -name 'libwebkitgtk-6.0.so.*' | head -n1)"
if [[ -z "$lib" ]]; then
    echo "fix-appimage: no libwebkitgtk-6.0 in $appdir/usr/lib" >&2
    exit 1
fi

execdir="$(grep -a -o -m1 -E '/usr/lib[A-Za-z0-9_/-]*/webkitgtk-6\.0' "$lib" | head -n1)"
if [[ -z "$execdir" ]]; then
    echo "fix-appimage: no helper directory found in $lib" >&2
    exit 1
fi
reldir="././${execdir#/usr}"

sed -i "s|${execdir}|${reldir}|g" "$lib"
if grep -a -q "$execdir" "$lib"; then
    echo "fix-appimage: $execdir still present in $lib" >&2
    exit 1
fi

# wails3 copies the helpers without their executable bit.
chmod +x "$appdir$execdir"/WebKit*Process

tool="appimagetool-${arch}.AppImage"
if [[ ! -x "$tool" ]]; then
    wget -q -4 -O "$tool" "https://github.com/AppImage/appimagetool/releases/download/continuous/${tool}"
    chmod +x "$tool"
fi
rm -f "$out"
ARCH="$arch" "./$tool" --appimage-extract-and-run "$appdir" "$out"
