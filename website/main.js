// Nova's site. The page works without JavaScript; this adds the device-aware
// install button, the style switcher, the live demo windows and the latest
// release notes.
const REPO = "jevido/nova";
// next.nova.jevido.app is the same site from the next branch. It offers the
// "next" pre-release, the test builds CI makes of that branch, instead of
// the latest release.
const NEXT = location.hostname.startsWith("next.");
const DL = NEXT ? `https://github.com/${REPO}/releases/download/next/` : `https://github.com/${REPO}/releases/latest/download/`;
const INSTALL_CMD = NEXT
  ? "curl -fsSL https://raw.githubusercontent.com/jevido/nova/next/install.sh | NOVA_CHANNEL=dev sh"
  : "curl -fsSL https://raw.githubusercontent.com/jevido/nova/main/install.sh | sh";
const ICONS = "assets/icons.svg";
const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
const icon = (name, cls = "ic") => `<svg class="${cls}" aria-hidden="true"><use href="${ICONS}#i-${name}"/></svg>`;
const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);

/* ---------------------------------------------------------------- next */
if (NEXT) {
  for (const a of $$('a[href*="/releases/latest"]')) {
    a.href = a.href.replace("/releases/latest/download/", "/releases/download/next/").replace(/\/releases\/latest$/, "/releases/tag/next");
  }
  $("#cmd-linux").textContent = INSTALL_CMD;
  for (const t of $$(".tag, .files em")) if (/updates itself/.test(t.textContent)) t.remove();
  const banner = document.createElement("div");
  banner.className = "next-banner";
  banner.innerHTML = `<strong>Nova (dev).</strong> These downloads are test builds of the next branch. They install next to Nova with their own settings, may be broken and don't update themselves. <a href="https://nova.jevido.app">Get Nova</a>`;
  document.body.prepend(banner);
}

/* ---------------------------------------------------------------- toasts */
function toast(host, text, { action, onAction, timeout = 5000 } = {}) {
  host.querySelectorAll(".toast").forEach((t) => t.remove());
  const t = document.createElement("div");
  t.className = "toast";
  t.setAttribute("role", "status");
  t.innerHTML = `<span>${esc(text)}</span>${action ? `<button type="button" class="act">${esc(action)}</button>` : ""}<button type="button" class="x" aria-label="Close">${icon("window-close")}</button>`;
  host.append(t);
  const close = () => {
    t.classList.add("out");
    setTimeout(() => t.remove(), reduced ? 0 : 200);
  };
  t.querySelector(".x").onclick = close;
  if (action) t.querySelector(".act").onclick = () => (onAction?.(), close());
  const timer = setTimeout(close, timeout);
  t.addEventListener("pointerenter", () => clearTimeout(timer), { once: true });
  return close;
}
const pageToasts = $("#page-toasts");

async function copy(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/* ------------------------------------------------- device-aware install */
const ua = navigator.userAgent;
// iPadOS reports itself as a Mac; touch support gives it away.
const isIOS = /iPhone|iPad|iPod/i.test(ua) || (/Macintosh/i.test(ua) && navigator.maxTouchPoints > 1);
const isAndroid = /Android/i.test(ua);
const isWindows = /Windows/i.test(ua);
const isMac = !isIOS && /Macintosh/i.test(ua);
const isLinux = !isAndroid && /Linux|X11|CrOS/i.test(ua);
const isMobile = isIOS || /Android|Mobile/i.test(ua);
const platform = isAndroid ? "android" : isIOS ? "ios" : isWindows ? "windows" : isLinux ? "linux" : null;

const install = $("#install");
const installLabel = $("[data-install-label]");
const installSub = $("[data-install-sub]");

const INSTALL = {
  linux: ["Install for Linux", "Copies the one-line install. No sudo, updates itself.", "#linux"],
  windows: ["Download for Windows", "Installer · Windows 10 and 11", DL + "nova-windows-amd64-setup.exe"],
  android: ["Download for Android", "APK · Android 5.0 or newer", DL + "nova-android-arm64.apk"],
  ios: ["Get Nova for iPhone", "Sideload with AltStore or SideStore", "#ios"],
};
if (platform) {
  const [label, sub, href] = INSTALL[platform];
  installLabel.textContent = label;
  installSub.textContent = sub;
  install.href = href;
  const card = $(`[data-platform="${platform}"]`);
  card.open = true;
  $("[data-yours]", card).hidden = false;
  $(".platforms").prepend(card);
} else {
  installLabel.textContent = "Download Nova";
  installSub.textContent = isMac ? "No macOS build yet. Linux, Windows, Android and iOS below." : "Linux, Windows, Android, iPhone & iPad";
}
if (platform === "linux") {
  install.addEventListener("click", async (e) => {
    e.preventDefault();
    const ok = await copy(INSTALL_CMD);
    if (ok) {
      toast(pageToasts, "Install command copied. Paste it into a terminal.");
    } else {
      // No clipboard access: show the command where it can be copied.
      $("#linux").open = true;
      $("#cmd-linux").scrollIntoView({ block: "center" });
      const r = document.createRange();
      r.selectNodeContents($("#cmd-linux"));
      getSelection().removeAllRanges();
      getSelection().addRange(r);
      toast(pageToasts, "Press Ctrl+C to copy the install command.");
    }
  });
}

// On a computer, offer a QR code to get the APK onto a phone.
if (!isMobile) $("[data-qr]").hidden = false;

// Links to a platform open its row.
function openFromHash() {
  const el = location.hash ? document.getElementById(location.hash.slice(1)) : null;
  if (el?.classList.contains("expander")) el.open = true;
}
addEventListener("hashchange", openFromHash);
openFromHash();
$$('a[href="#ios"], a[href="#linux"]').forEach((a) => a.addEventListener("click", () => setTimeout(openFromHash)));

// Copy buttons.
$$("[data-copy]").forEach((btn) => {
  btn.addEventListener("click", async () => {
    const text = $(btn.dataset.copy).textContent.trim();
    if (await copy(text)) {
      btn.classList.add("done");
      btn.innerHTML = icon("object-select");
      toast(pageToasts, "Copied to clipboard");
      setTimeout(() => {
        btn.classList.remove("done");
        btn.innerHTML = icon("edit-copy");
      }, 2000);
    } else {
      const r = document.createRange();
      r.selectNodeContents($(btn.dataset.copy));
      getSelection().removeAllRanges();
      getSelection().addRange(r);
      toast(pageToasts, "Press Ctrl+C to copy");
    }
  });
});

/* -------------------------------------------------------- style switcher */
const root = document.documentElement;
const menuBtn = $("#menu-btn");
const menu = $("#main-menu");
function currentStyle() {
  return root.dataset.theme || "system";
}
function setStyle(s) {
  if (s === "system") delete root.dataset.theme;
  else root.dataset.theme = s;
  try {
    if (s === "system") localStorage.removeItem("nova-site-style");
    else localStorage.setItem("nova-site-style", s);
  } catch {}
  $$("[data-style]").forEach((b) => b.setAttribute("aria-checked", String(b.dataset.style === s)));
}
setStyle(currentStyle());
$$("[data-style]").forEach((b) => b.addEventListener("click", () => setStyle(b.dataset.style)));

function toggleMenu(open = menu.hidden) {
  menu.hidden = !open;
  menuBtn.setAttribute("aria-expanded", String(open));
  if (open) $("[aria-checked='true']", menu)?.focus();
}
menuBtn.addEventListener("click", (e) => {
  e.stopPropagation();
  toggleMenu();
});
document.addEventListener("click", (e) => {
  if (!menu.hidden && !menu.contains(e.target)) toggleMenu(false);
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && !menu.hidden) {
    toggleMenu(false);
    menuBtn.focus();
  }
});

/* ----------------------------------------------- header view switcher */
const sections = [
  ["#top", $(".app-header")],
  ["#news", $("#news")],
  ["#download", $("#download")],
];
const navLinks = $$(".switcher a");
const visible = new Map();
const io = new IntersectionObserver(
  (entries) => {
    entries.forEach((en) => visible.set(en.target, en.isIntersecting));
    let cur = "#top";
    for (const [href, el] of sections) if (visible.get(el)) cur = href;
    if (scrollY < 200) cur = "#top";
    navLinks.forEach((a) => a.setAttribute("aria-current", String(a.getAttribute("href") === cur)));
  },
  { rootMargin: "-45% 0px -50% 0px" },
);
sections.forEach(([, el]) => io.observe(el));

/* ===================================================== the live demo */
// Sample files, shared by every window on the page, so a file trashed on
// the desktop is gone on the phone too (as it is in Nova).
const KB = 1024;
const MB = 1024 * KB;
// The wallpapers are real files on this site; their sizes are the real ones.
const WALLPAPERS = [
  ["amber", 860], ["blobs", 878], ["curvy", 3910], ["dithered-sun", 4194], ["fold", 1328],
  ["glass-chip", 3464], ["map", 1748], ["morphogenesis", 4924], ["pills", 4326], ["tubes", 1104],
];
let nextId = 1;
const items = new Map();
function add(parent, name, type, size = 0, extra = {}) {
  const id = nextId++;
  items.set(id, { id, parent, name, type, size, mtime: extra.mtime ?? Date.now() - Math.random() * 40 * 864e5, ...extra });
  return id;
}
const HOME = add(null, "Home", "folder");
const F = {};
for (const n of ["Backups", "Documents", "Downloads", "Music", "Pictures", "Projects", "Videos"]) F[n] = add(HOME, n, "folder");
add(HOME, "notes.md", "text", 4 * KB);
add(HOME, "budget-2026.ods", "sheet", 18 * KB);
const letters = add(F.Documents, "Letters", "folder");
add(letters, "landlord.odt", "doc", 31 * KB);
const report = add(F.Documents, "report.odt", "doc", 92 * KB, { starred: true });
add(F.Documents, "invoices-2026.ods", "sheet", 24 * KB);
add(F.Documents, "readme.txt", "text", 2 * KB);
add(F.Downloads, "podcast-episode-12.mp3", "audio", 54 * MB);
add(F.Downloads, "photos-backup.tar.gz", "archive", 412 * MB);
const rec = add(F.Music, "Field Recordings", "folder");
add(rec, "harbour-morning.ogg", "audio", 8.2 * MB);
add(rec, "rain-on-roof.ogg", "audio", 11.4 * MB);
add(F.Music, "voice-memo.ogg", "audio", 1.1 * MB);
const WP = add(F.Pictures, "Wallpapers", "folder", 0, { starred: true, shared: true });
add(F.Pictures, "Screenshots", "folder");
for (const [n, bytes] of WALLPAPERS) add(WP, `${n}.webp`, "image", bytes, { thumb: `assets/samples/${n}.webp` });
add(F.Projects, "website", "folder");
add(F.Projects, "todo.txt", "text", 1 * KB);
add(F.Videos, "holiday-2026.mp4", "video", 184 * MB);
add(F.Backups, "phone-2026-09.tar", "archive", 2.1 * 1024 * MB);
const trash = [];
trash.push({ id: add(-1, "old-draft.odt", "doc", 40 * KB), from: F.Documents });

const recentIds = [...items.values()].filter((i) => i.type !== "folder" && i.parent > 0).sort((a, b) => b.mtime - a.mtime).slice(0, 8).map((i) => i.id);
recentIds.unshift(report);

const FOLDER_ICON = { Documents: "folder-documents", Downloads: "folder-download", Music: "folder-music", Pictures: "folder-pictures", Videos: "folder-videos" };
const TYPE_ICON = { text: "text-x-generic", doc: "x-office-document", sheet: "x-office-spreadsheet", audio: "audio-x-generic", video: "video-x-generic", archive: "package-x-generic", image: "image-x-generic" };
const TYPE_NAME = { text: "Plain text", doc: "Document", sheet: "Spreadsheet", audio: "Audio", video: "Video", archive: "Archive", image: "WebP image", folder: "Folder" };
const PLACES = [
  ["home", "Home", "user-home"],
  ["recent", "Recent", "document-open-recent"],
  ["starred", "Starred", "starred"],
  ["shared", "Shared", "folder-publicshare"],
  ["trash", "Trash", "user-trash"],
];
const BOOKMARKS = [F.Documents, F.Music, F.Pictures, F.Videos, F.Downloads];

function itemIcon(it) {
  if (it.type === "folder") {
    if (it.parent === HOME && FOLDER_ICON[it.name]) return `assets/icons/${FOLDER_ICON[it.name]}.svg`;
    return "assets/icons/folder.svg";
  }
  return `assets/icons/${TYPE_ICON[it.type]}.svg`;
}
function imgFor(it) {
  return it.thumb ? `<img class="thumb" src="${it.thumb}" alt="" loading="lazy" decoding="async">` : `<img src="${itemIcon(it)}" alt="">`;
}
const children = (id) => [...items.values()].filter((i) => i.parent === id).sort((a, b) => (a.type === "folder") === (b.type === "folder") ? a.name.localeCompare(b.name) : a.type === "folder" ? -1 : 1);
function fmtSize(it) {
  if (it.type === "folder") {
    const n = children(it.id).length;
    return n === 1 ? "1 item" : `${n} items`;
  }
  const b = it.size;
  if (b < KB) return `${b} bytes`;
  if (b < MB) return `${(b / KB).toFixed(1)} kB`;
  if (b < 1024 * MB) return `${(b / MB).toFixed(1)} MB`;
  return `${(b / 1024 / MB).toFixed(1)} GB`;
}
function fmtDate(t) {
  const d = new Date(t);
  const today = new Date();
  if (d.toDateString() === today.toDateString()) return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  return d.toLocaleDateString(undefined, { day: "numeric", month: "short" });
}
function pathOf(id) {
  const out = [];
  for (let it = items.get(id); it; it = items.get(it.parent)) out.unshift(it);
  return out;
}

// Undo stack, shared like Nova's.
const undo = [];
const windows = new Set();
const redrawAll = () => windows.forEach((w) => w.render());

function trashItems(ids, w) {
  const moved = ids.map((id) => ({ id, from: items.get(id).parent }));
  moved.forEach((m) => {
    items.get(m.id).parent = -1;
    trash.unshift(m);
  });
  undo.push({ kind: "trash", moved });
  const it = items.get(ids[0]);
  w.toast(ids.length === 1 ? `“${it.name}” moved to Trash` : `${ids.length} items moved to Trash`, "Undo", () => doUndo(w));
  redrawAll();
}
function restore(ids, w) {
  ids.forEach((id) => {
    const i = trash.findIndex((t) => t.id === id);
    if (i < 0) return;
    items.get(id).parent = trash[i].from;
    trash.splice(i, 1);
  });
  w.toast(ids.length === 1 ? `“${items.get(ids[0]).name}” restored` : `${ids.length} items restored`);
  redrawAll();
}
function doUndo(w) {
  const op = undo.pop();
  if (!op) return w.toast("Nothing to undo");
  if (op.kind === "trash") {
    op.moved.forEach((m) => {
      items.get(m.id).parent = m.from;
      const i = trash.findIndex((t) => t.id === m.id);
      if (i >= 0) trash.splice(i, 1);
    });
    w.toast(op.moved.length === 1 ? `Restored “${items.get(op.moved[0].id).name}”` : "Restored");
  } else if (op.kind === "star") {
    items.get(op.id).starred = !items.get(op.id).starred;
  }
  redrawAll();
}
function toggleStar(id, w) {
  const it = items.get(id);
  it.starred = !it.starred;
  undo.push({ kind: "star", id });
  w.toast(it.starred ? `“${it.name}” starred` : `“${it.name}” unstarred`);
  redrawAll();
}

class NovaWindow {
  constructor(host, { layout, start, view }) {
    this.host = host;
    this.layout = layout; // "desktop" | "phone"
    this.loc = start;
    this.back = [];
    this.fwd = [];
    this.sel = new Set();
    this.view = view;
    this.query = null;
    this.drawer = false;
    this.el = document.createElement("div");
    this.el.className = `nw ${layout === "phone" ? "phone" : ""}`;
    this.el.setAttribute("aria-label", layout === "phone" ? "Nova on a phone (live demo)" : "Nova on the desktop (live demo)");
    host.append(this.el);
    this.el.addEventListener("click", (e) => this.onClick(e));
    this.el.addEventListener("dblclick", (e) => this.onDbl(e));
    this.el.addEventListener("contextmenu", (e) => this.onMenu(e));
    this.el.addEventListener("keydown", (e) => this.onKey(e));
    this.el.addEventListener("input", (e) => {
      if (e.target.matches(".nw-search")) {
        this.query = e.target.value;
        this.renderView();
      }
    });
    windows.add(this);
    this.render();
  }

  // Location: a folder id or a place name.
  list() {
    const l = this.loc;
    let list;
    if (l === "recent") list = recentIds.map((id) => items.get(id)).filter((i) => i.parent > 0);
    else if (l === "starred") list = [...items.values()].filter((i) => i.starred && i.parent > 0);
    else if (l === "shared") list = [...items.values()].filter((i) => i.shared && i.parent > 0);
    else if (l === "trash") list = trash.map((t) => items.get(t.id));
    else list = children(l);
    if (this.query) list = list.filter((i) => i.name.toLowerCase().includes(this.query.toLowerCase()));
    return list;
  }
  title() {
    const l = this.loc;
    const p = PLACES.find((p) => p[0] === l);
    return p ? p[1] : items.get(l).name;
  }
  go(loc, push = true) {
    if (typeof loc === "number" && items.get(loc)?.parent === -1) return;
    if (push && loc !== this.loc) {
      this.back.push(this.loc);
      this.fwd = [];
    }
    this.loc = loc === HOME ? HOME : loc;
    this.sel.clear();
    this.query = null;
    this.drawer = false;
    this.render();
    this.onNavigate?.();
  }
  goBack() {
    if (!this.back.length) return;
    this.fwd.push(this.loc);
    this.go(this.back.pop(), false);
  }
  goFwd() {
    if (!this.fwd.length) return;
    this.back.push(this.loc);
    this.go(this.fwd.pop(), false);
  }
  open(id) {
    const it = items.get(id);
    if (!it) return;
    if (this.loc === "trash") return this.toast("Restore it first to open it");
    if (it.type === "folder") return this.go(id);
    if (it.type === "image") return this.preview(id);
    this.toast(`Nova would open “${it.name}” in your default app`);
  }
  preview(id) {
    const it = items.get(id);
    if (!it?.thumb) return;
    const p = document.createElement("div");
    p.className = "nw-preview";
    p.innerHTML = `<img src="${it.thumb}" alt="${esc(it.name)}"><p>${esc(it.name)} · ${fmtSize(it)}</p>`;
    p.addEventListener("click", () => p.remove());
    this.el.append(p);
    this.previewEl = p;
  }
  toast(text, action, onAction) {
    toast($(".nw-toasts", this.el), text, { action, onAction });
  }

  sidebar() {
    const row = (loc, label, ic) => `<button type="button" class="nw-row" data-loc="${loc}" aria-current="${this.loc === loc || (loc === "home" && this.loc === HOME)}">${icon(ic)}<span>${label}</span></button>`;
    return `<div class="nw-side">
      <div class="nw-side-hb"><span>Nova</span><button type="button" class="nw-b" data-act="menu" aria-label="Main Menu">${icon("open-menu")}</button></div>
      <nav class="nw-rows" aria-label="Places">
        ${PLACES.map(([l, n, ic]) => row(l, n, ic === "user-trash" && trash.length ? "user-trash" : ic)).join("")}
        <div class="nw-sep"></div>
        ${BOOKMARKS.map((id) => row(id, items.get(id).name, "folder")).join("")}
      </nav>
      <div class="nw-acct"><span class="nw-av" aria-hidden="true">S</span><span>sample<small>nova.storage</small></span></div>
    </div>`;
  }
  pathbar() {
    if (this.query !== null) {
      return `<div class="nw-path"><input class="nw-search" type="search" aria-label="Search current folder" placeholder="Search ${esc(this.title())}" value="${esc(this.query)}" style="flex:1;border:0;background:none;outline:none;padding:0 8px;font:inherit;color:inherit"></div>`;
    }
    let segs;
    if (typeof this.loc === "number") {
      segs = pathOf(this.loc).map((it, i, a) => `${i ? '<span class="nw-slash">/</span>' : ""}<button type="button" class="nw-seg ${i === a.length - 1 ? "cur" : ""}" data-loc="${it.id}">${i === 0 ? icon("user-home") : ""}${esc(it.name)}</button>`).join("");
    } else {
      const p = PLACES.find((p) => p[0] === this.loc);
      segs = `<button type="button" class="nw-seg cur" data-loc="${p[0]}">${icon(p[2])}${p[1]}</button>`;
    }
    return `<div class="nw-path">${segs}<button type="button" class="nw-b" data-act="more" aria-label="Folder menu">${icon("view-more")}</button></div>`;
  }
  render() {
    const keep = $(".nw-toasts", this.el);
    this.renderShell();
    if (keep) $(".nw-toasts", this.el).replaceWith(keep);
    this.renderView();
    const s = $(".nw-search", this.el);
    if (s && this.focusSearch) {
      s.focus();
      s.setSelectionRange(s.value.length, s.value.length);
      this.focusSearch = false;
    }
  }
  renderShell() {
    const desk = this.layout === "desktop";
    if (desk) {
      this.el.innerHTML = `${this.sidebar()}
        <div class="nw-main">
          <div class="nw-hb">
            <button type="button" class="nw-b" data-act="back" aria-label="Back" ${this.back.length ? "" : "disabled"}>${icon("go-previous")}</button>
            <button type="button" class="nw-b" data-act="fwd" aria-label="Forward" ${this.fwd.length ? "" : "disabled"}>${icon("go-next")}</button>
            ${this.pathbar()}
            <button type="button" class="nw-b" data-act="search" aria-label="Search" aria-pressed="${this.query !== null}">${icon("system-search")}</button>
            <button type="button" class="nw-b split" data-act="view" aria-label="${this.view === "grid" ? "Switch to list view" : "Switch to grid view"}">${icon(this.view === "grid" ? "view-list" : "view-grid")}${icon("pan-down", "ic pan")}</button>
            <button type="button" class="nw-b round" data-act="close" aria-label="Close window">${icon("window-close")}</button>
          </div>
          <div class="nw-view" tabindex="0" aria-label="Files"></div>
          <div class="nw-toasts" aria-live="polite"></div>
        </div>`;
    } else {
      this.el.innerHTML = `
        <div class="nw-hb">
          <button type="button" class="nw-b" data-act="drawer" aria-label="Show sidebar">${icon("sidebar-show")}</button>
          <span class="nw-ptitle">${esc(this.title())}</span>
          <button type="button" class="nw-b" data-act="more" aria-label="Folder menu">${icon("view-more")}</button>
        </div>
        <div class="nw-view" tabindex="0" aria-label="Files"></div>
        <div class="nw-bottom">
          <button type="button" class="nw-b" data-act="back" aria-label="Back" ${this.back.length ? "" : "disabled"}>${icon("go-previous")}</button>
          <button type="button" class="nw-b" data-act="fwd" aria-label="Forward" ${this.fwd.length ? "" : "disabled"}>${icon("go-next")}</button>
          <button type="button" class="nw-b" data-act="up" aria-label="Up" ${typeof this.loc === "number" && this.loc !== HOME ? "" : "disabled"}>${icon("go-up")}</button>
          <button type="button" class="nw-b" data-act="view" aria-label="${this.view === "grid" ? "Switch to list view" : "Switch to grid view"}">${icon(this.view === "grid" ? "view-list" : "view-grid")}</button>
          <button type="button" class="nw-b" data-act="search" aria-label="Search">${icon("system-search")}</button>
        </div>
        <div class="nw-toasts" aria-live="polite"></div>
        ${this.drawer ? `<div class="nw-drawer">${this.sidebar()}<div class="nw-scrim" data-act="drawer"></div></div>` : ""}`;
      if (this.query !== null) {
        $(".nw-ptitle", this.el).outerHTML = `<input class="nw-search" type="search" aria-label="Search current folder" placeholder="Search ${esc(this.title())}" value="${esc(this.query)}" style="flex:1;min-width:0;height:36px;border:0;border-radius:6px;background:var(--btn-bg);outline:none;padding:0 12px;font:inherit;color:inherit">`;
      }
    }
  }
  renderView() {
    const v = $(".nw-view", this.el);
    const list = this.list();
    const sel = (it) => `aria-selected="${this.sel.has(it.id)}"`;
    const star = (it) => (it.starred && this.loc !== "starred" ? icon("starred", "nw-star") : "");
    if (!list.length) {
      const [ic, t, s] =
        this.query ? ["system-search", "No Results Found", "Try a different search"]
        : this.loc === "trash" ? ["user-trash", "Trash is Empty", ""]
        : this.loc === "starred" ? ["starred", "No Starred Files", "Right-click a file and choose Star"]
        : ["folder", "Folder is Empty", ""];
      v.innerHTML = `<div class="nw-empty">${icon(ic)}<strong>${t}</strong><span>${s}</span></div>`;
      return;
    }
    if (this.layout === "phone" && this.view === "list") {
      v.innerHTML = `<div class="nw-mlist" role="listbox" aria-label="Files">${list.map((it) => `<button type="button" class="nw-mrow" role="option" data-id="${it.id}" ${sel(it)}>${imgFor(it)}<span><b>${esc(it.name)}</b><small>${fmtSize(it)} · ${fmtDate(it.mtime)}</small></span></button>`).join("")}</div>`;
    } else if (this.view === "grid") {
      v.innerHTML = `<div class="nw-grid" role="listbox" aria-label="Files" aria-multiselectable="true">${list.map((it) => `<button type="button" class="nw-item" role="option" data-id="${it.id}" ${sel(it)} title="${esc(it.name)}"><span class="nw-ib">${imgFor(it)}${star(it)}</span><span class="nw-label">${esc(it.name)}</span></button>`).join("")}</div>`;
    } else {
      v.innerHTML = `<div class="nw-list"><div class="nw-colhead" aria-hidden="true"><span>Name</span><span style="text-align:right">Size</span><span>Modified</span><span></span></div><div role="listbox" aria-label="Files" aria-multiselectable="true">${list.map((it) => `<button type="button" class="nw-lrow" role="option" data-id="${it.id}" ${sel(it)}><span>${imgFor(it)}${esc(it.name)}</span><span class="size">${fmtSize(it)}</span><span>${fmtDate(it.mtime)}</span><span class="st">${it.starred ? icon("starred") : ""}</span></button>`).join("")}</div></div>`;
    }
  }
  selectOnly(id) {
    this.sel = new Set(id ? [id] : []);
    $$("[data-id]", this.el).forEach((b) => b.setAttribute("aria-selected", String(this.sel.has(+b.dataset.id))));
    if (id) this.onSelect?.(id);
  }

  onClick(e) {
    this.closeMenu();
    const t = e.target.closest("[data-act],[data-loc],[data-id],[data-menu]");
    if (!t) {
      if (e.target.closest(".nw-view")) this.selectOnly(null);
      return;
    }
    if (t.dataset.menu) return this.menuAction(t.dataset.menu, +t.dataset.target);
    if (t.dataset.loc) {
      const l = t.dataset.loc;
      return this.go(l === "home" ? HOME : /^\d+$/.test(l) ? +l : l);
    }
    if (t.dataset.id) {
      const id = +t.dataset.id;
      // On the phone a tap opens, like the app.
      if (this.layout === "phone" && this.loc !== "trash") return this.open(id);
      if (e.ctrlKey || e.metaKey) {
        this.sel.has(id) ? this.sel.delete(id) : this.sel.add(id);
        t.setAttribute("aria-selected", String(this.sel.has(id)));
      } else this.selectOnly(id);
      return;
    }
    const a = t.dataset.act;
    if (a === "back") this.goBack();
    else if (a === "fwd") this.goFwd();
    else if (a === "up") this.go(items.get(this.loc).parent);
    else if (a === "view") {
      this.view = this.view === "grid" ? "list" : "grid";
      this.render();
    } else if (a === "search") {
      this.query = this.query === null ? "" : null;
      this.focusSearch = true;
      this.render();
    } else if (a === "drawer") {
      this.drawer = !this.drawer;
      this.render();
    } else if (a === "close") this.toast("This window stays open. It's a demo");
    else if (a === "menu" || a === "more") this.openMenu(t, null);
  }
  onDbl(e) {
    if (this.layout === "phone") return;
    const t = e.target.closest("[data-id]");
    if (t) this.open(+t.dataset.id);
  }
  onMenu(e) {
    const t = e.target.closest("[data-id]");
    if (!t || !this.el.contains(t)) return;
    e.preventDefault();
    const id = +t.dataset.id;
    if (!this.sel.has(id)) this.selectOnly(id);
    this.openMenu(t, id, e);
  }
  openMenu(anchor, id, ev) {
    this.closeMenu();
    const it = id && items.get(id);
    let rows;
    if (!it) {
      rows = this.loc === "trash"
        ? [["empty", "Empty Trash"]]
        : [["view", this.view === "grid" ? "List View" : "Grid View"], ["undo", "Undo", "Ctrl+Z"], ["select-all", "Select All", "Ctrl+A"]];
    } else if (this.loc === "trash") {
      rows = [["restore", "Restore From Trash"]];
    } else {
      rows = [
        ["open", it.type === "image" ? "Preview" : "Open", it.type === "image" ? "Space" : "Enter"],
        ["star", it.starred ? "Unstar" : "Star"],
        "-",
        ["trash", "Move to Trash", "Delete"],
        "-",
        ["props", "Properties"],
      ];
    }
    const m = document.createElement("div");
    m.className = "nw-menu";
    m.setAttribute("role", "menu");
    m.innerHTML = rows.map((r) => (r === "-" ? "<hr>" : `<button type="button" role="menuitem" data-menu="${r[0]}" data-target="${id ?? ""}"><span>${r[1]}</span>${r[2] ? `<kbd>${r[2]}</kbd>` : ""}</button>`)).join("");
    this.el.append(m);
    const box = this.el.getBoundingClientRect();
    const s = box.width / this.el.offsetWidth || 1; // the window may be scaled
    const r = ev ? { left: ev.clientX, bottom: ev.clientY } : anchor.getBoundingClientRect();
    let x = (r.left - box.left) / s;
    let y = (r.bottom - box.top) / s + 4;
    x = Math.min(x, this.el.offsetWidth - m.offsetWidth - 8);
    y = Math.min(y, this.el.offsetHeight - m.offsetHeight - 8);
    m.style.left = `${Math.max(8, x)}px`;
    m.style.top = `${Math.max(8, y)}px`;
    $("button", m)?.focus({ preventScroll: true });
    this.menuEl = m;
  }
  closeMenu() {
    this.menuEl?.remove();
    this.menuEl = null;
  }
  menuAction(act, id) {
    this.closeMenu();
    const ids = id ? (this.sel.has(id) ? [...this.sel] : [id]) : [...this.sel];
    if (act === "open") this.open(id);
    else if (act === "star") toggleStar(id, this);
    else if (act === "trash") trashItems(ids, this);
    else if (act === "restore") restore(ids, this);
    else if (act === "empty") this.toast("Emptying the Trash can't be undone, so the demo leaves it alone");
    else if (act === "undo") doUndo(this);
    else if (act === "view") {
      this.view = this.view === "grid" ? "list" : "grid";
      this.render();
    } else if (act === "select-all") {
      this.sel = new Set(this.list().map((i) => i.id));
      this.renderView();
    } else if (act === "props") this.onProps?.(id);
  }
  onKey(e) {
    if (e.target.matches("input")) {
      if (e.key === "Escape") {
        this.query = null;
        this.render();
        $(".nw-view", this.el).focus();
      }
      return;
    }
    const ctrl = e.ctrlKey || e.metaKey;
    const ids = [...this.sel];
    if (e.key === "Escape") {
      if (this.previewEl?.isConnected) this.previewEl.remove();
      else if (this.menuEl) this.closeMenu();
      else if (this.drawer) (this.drawer = false), this.render();
      else this.selectOnly(null);
    } else if ((e.key === "Delete" || e.key === "Backspace") && ids.length && this.loc !== "trash") trashItems(ids, this);
    else if (ctrl && e.key.toLowerCase() === "z") doUndo(this);
    else if (ctrl && e.key.toLowerCase() === "a") {
      this.sel = new Set(this.list().map((i) => i.id));
      this.renderView();
    } else if (ctrl && e.key.toLowerCase() === "f") {
      this.query = "";
      this.focusSearch = true;
      this.render();
    } else if (e.key === "Enter" && ids.length === 1) this.open(ids[0]);
    else if (e.key === " " && ids.length === 1) {
      if (this.previewEl?.isConnected) this.previewEl.remove();
      else this.preview(ids[0]);
    } else if (e.altKey && e.key === "ArrowLeft") this.goBack();
    else if (e.altKey && e.key === "ArrowRight") this.goFwd();
    else if (e.altKey && e.key === "ArrowUp" && typeof this.loc === "number" && this.loc !== HOME) this.go(items.get(this.loc).parent);
    else if (["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(e.key) && !e.altKey) this.moveCursor(e.key);
    else return;
    e.preventDefault();
    e.stopPropagation();
  }
  moveCursor(key) {
    const btns = $$("[data-id]", this.el);
    if (!btns.length) return;
    const cur = btns.findIndex((b) => this.sel.has(+b.dataset.id));
    let cols = 1;
    if (this.view === "grid" && btns.length > 1) {
      const y0 = btns[0].offsetTop;
      cols = btns.findIndex((b) => b.offsetTop !== y0);
      if (cols < 1) cols = btns.length;
    }
    const step = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -cols, ArrowDown: cols }[key];
    const next = cur < 0 ? 0 : Math.max(0, Math.min(btns.length - 1, cur + step));
    this.selectOnly(+btns[next].dataset.id);
    btns[next].scrollIntoView({ block: "nearest" });
  }
}

/* Properties dialog: real SHA-256 of the real sample file. */
class PropsDialog {
  constructor(host) {
    this.el = document.createElement("div");
    this.el.className = "nw props";
    this.el.setAttribute("aria-label", "Properties dialog (live demo)");
    host.append(this.el);
    this.el.addEventListener("click", (e) => {
      const t = e.target.closest("[data-act]");
      if (!t) return;
      const it = items.get(this.id);
      if (t.dataset.act === "share") {
        it.shared = !it.shared;
        this.render();
        redrawAll();
      } else if (t.dataset.act === "copy") toast($(".nw-toasts", this.el), "In the app this copies the public link");
      else if (t.dataset.act === "close") toast($(".nw-toasts", this.el), "This dialog stays open. It's a demo");
    });
    windows.add(this);
    this.show([...items.values()].find((i) => i.name === "amber.webp").id);
  }
  show(id) {
    this.id = id;
    this.render();
    const it = items.get(id);
    if (it.thumb && !it.sha && crypto.subtle) {
      fetch(it.thumb)
        .then((r) => r.arrayBuffer())
        .then((b) => ((it.size = b.byteLength), crypto.subtle.digest("SHA-256", b)))
        .then((h) => {
          it.sha = [...new Uint8Array(h)].map((x) => x.toString(16).padStart(2, "0")).join("");
          if (this.id === id) this.render();
        })
        .catch(() => {});
    }
  }
  render() {
    const keep = $(".nw-toasts", this.el);
    this.renderShell();
    if (keep) $(".nw-toasts", this.el).replaceWith(keep);
  }
  renderShell() {
    const it = items.get(this.id);
    if (!it || it.parent === -1) return this.show(WP);
    const folder = it.type === "folder";
    const loc = it.parent === -1 ? "Trash" : pathOf(it.parent).map((p) => p.name).join("/").replace(/^Home/, "~");
    const size = folder ? children(it.id).reduce((s, c) => s + (c.size || 0), 0) : it.size;
    const human = fmtSize({ size: size || 1 });
    this.el.innerHTML = `
      <div class="nw-hb"><strong>Properties</strong><button type="button" class="nw-b round" data-act="close" aria-label="Close dialog">${icon("window-close")}</button></div>
      <div class="props-body">
        <div class="props-top">${it.thumb ? `<img src="${it.thumb}" alt="" style="object-fit:cover;height:72px;width:96px;border-radius:6px">` : `<img src="${itemIcon(it)}" alt="">`}<strong>${esc(it.name)}</strong><span>${folder ? `${fmtSize(it)}, ${human}` : human}</span></div>
        <dl class="props-list" style="margin:0">
          <div><dt>Type</dt><dd>${TYPE_NAME[it.type]}</dd></div>
          <div><dt>Location</dt><dd>${esc(loc)}</dd></div>
          <div><dt>Modified</dt><dd>${new Date(it.mtime).toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" })}</dd></div>
          ${folder ? "" : `<div class="wide"><dt>SHA-256</dt><dd><code>${it.sha ?? (it.thumb ? "Calculating…" : "Shown for files in the app")}</code></dd></div>`}
        </dl>
        <dl class="props-list" style="margin:0">
          <div><dt style="color:var(--fg)">Public link</dt><dd><button type="button" class="sw" role="switch" aria-checked="${!!it.shared}" aria-label="Public link" data-act="share"></button></dd></div>
          ${it.shared ? `<div><dt>Anyone with the link can view</dt><dd class="props-link"><button type="button" class="btn" data-act="copy">Copy Link</button></dd></div>` : ""}
        </dl>
      </div>
      <div class="nw-toasts" aria-live="polite"></div>`;
  }
}

/* ------------------------------------------------------------ carousel */
const track = $("#shots");
const slides = $$(".slide", track);
const dots = $$(".dots [data-to]");
const caption = $("[data-caption]");
const CAPTIONS = [
  'This is the real layout with sample files. Click around: open a folder, switch to list view, select a file and press <kbd>Delete</kbd>.',
  "On a phone the buttons move to a bottom bar and the sidebar becomes a drawer. It shares the same sample account as the desktop window: trash a file there and it's gone here too.",
  "Properties shows sizes, the SHA-256 checksum (this one is calculated from the real sample file) and the public-link switch.",
];
let index = 0;
function goTo(i, smooth = true) {
  index = Math.max(0, Math.min(slides.length - 1, i));
  track.scrollTo({ left: slides[index].offsetLeft - track.offsetLeft, behavior: smooth && !reduced ? "smooth" : "auto" });
  markSlide();
}
function markSlide() {
  dots.forEach((d, i) => d.setAttribute("aria-selected", String(i === index)));
  $(".osd.prev").disabled = index === 0;
  $(".osd.next").disabled = index === slides.length - 1;
  caption.innerHTML = CAPTIONS[+slides[index].dataset.order];
  slides.forEach((s, i) => (s.inert = i !== index));
}
$$("[data-go]").forEach((b) => b.addEventListener("click", () => goTo(index + +b.dataset.go)));
dots.forEach((d) => d.addEventListener("click", () => goTo(+d.dataset.to)));
let scrollTimer;
track.addEventListener("scroll", () => {
  clearTimeout(scrollTimer);
  scrollTimer = setTimeout(() => {
    const i = Math.round(track.scrollLeft / track.clientWidth);
    if (i !== index) {
      index = i;
      markSlide();
    }
  }, 80);
});
$(".shots").addEventListener("keydown", (e) => {
  if (e.target.closest(".nw")) return;
  if (e.key === "ArrowLeft") goTo(index - 1);
  else if (e.key === "ArrowRight") goTo(index + 1);
});

// Narrow screens see the phone first.
slides.forEach((s, i) => (s.dataset.order = i));
if (innerWidth < 700) {
  track.prepend(slides[1]);
  slides.splice(0, 2, slides[1], slides[0]);
  dots[0].setAttribute("aria-label", "Phone");
  dots[1].setAttribute("aria-label", "Desktop");
}

/* Fit each window into its stage, like a screenshot. */
function fit() {
  const avail = track.clientWidth;
  const maxH = Math.min(640, innerHeight * 0.72);
  slides.forEach((s) => {
    const stage = $(".stage", s);
    const win = $(".phone-frame", s) || $(".nw", s);
    if (!win) return;
    const pad = parseFloat(getComputedStyle(s).paddingLeft) * 2;
    const w = win.offsetWidth;
    const h = win.offsetHeight;
    const k = Math.min(1, (avail - pad) / w, maxH / h);
    win.style.transform = k < 1 ? `scale(${k})` : "";
    stage.style.height = `${Math.ceil(h * k)}px`;
  });
}

const desktopHost = $('[data-demo="desktop"]');
const phoneHost = $('[data-demo="phone"]');
const propsHost = $('[data-demo="props"]');
if (desktopHost) {
  const desk = new NovaWindow(desktopHost, { layout: "desktop", start: WP, view: "grid" });
  desk.back = [HOME, F.Pictures];
  desk.selectOnly(null);
  desk.render();
  const phone = new NovaWindow(phoneHost, { layout: "phone", start: HOME, view: "list" });
  const props = new PropsDialog(propsHost);
  const toProps = (id) => {
    props.show(id);
    goTo(slides.findIndex((s) => s.dataset.slide === "props"));
  };
  desk.onProps = toProps;
  phone.onProps = toProps;
  // Properties follows the desktop selection.
  desk.onSelect = (id) => props.show(id);
  const ro = new ResizeObserver(fit);
  ro.observe(track);
  addEventListener("resize", fit);
  fit();
  markSlide();
  goTo(0, false);
}

/* --------------------------------------------------- latest release */
// What's New comes from CHANGELOG.md, the same notes the app shows:
// "## 0.6.0 — 2026-09-28", "### New" sections and
// "- [icon] **Title** — text" items.
function renderChangelog(md) {
  const sec = md.split(/^## /m).slice(1).find((s) => /^\d+\.\d+/.test(s));
  if (!sec) return;
  const [head, ...lines] = sec.split(/\r?\n/);
  const hm = head.match(/^([\d.]+)\s*[—–-]\s*(\d{4}-\d{2}-\d{2})/);
  if (!hm) return;
  let kind = "";
  const out = [];
  for (const line of lines) {
    const h = line.match(/^###\s+(\w+)/);
    if (h) {
      kind = ["New", "Improved", "Fixed"].includes(h[1]) ? h[1] : "";
      continue;
    }
    const m = line.match(/^\s*[-*]\s+(?:\[[\w-]+\]\s+)?\*\*(.+?)\*\*\s*[—–-]?\s*(.*)$/);
    if (m) out.push({ kind, title: m[1], text: m[2].replace(/\*\*(.+?)\*\*/g, "$1") });
  }
  if (!out.length) return;
  $("[data-release-version]").textContent = `Version ${hm[1]}`;
  $("[data-release-date]").textContent = new Date(hm[2] + "T12:00:00").toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
  $("[data-release-items]").innerHTML = out
    .slice(0, 5)
    .map((it) => `<li><strong>${esc(it.title)}${it.kind ? `<span class="kind">${it.kind}</span>` : ""}</strong>${esc(it.text)}</li>`)
    .join("");
}
fetch(`https://raw.githubusercontent.com/${REPO}/main/CHANGELOG.md`)
  .then((r) => (r.ok ? r.text() : Promise.reject()))
  .then(renderChangelog)
  .catch(() => {});

// Say so when nothing is released yet.
fetch(`https://api.github.com/repos/${REPO}/releases/${NEXT ? "tags/next" : "latest"}`, { headers: { Accept: "application/vnd.github+json" } })
  .then((r) => (r.ok ? r.json() : r.status === 404 ? null : Promise.reject()))
  .then((rel) => {
    if (!rel?.tag_name) {
      $("[data-release-note]").innerHTML = `No release is published yet, so these links won't work just yet. Watch the project on <a href="https://github.com/${REPO}">GitHub</a>.`;
    }
  })
  .catch(() => {});

// The header bar shows its shade once the page scrolls under it.
const headerbar = $(".headerbar");
const onScroll = () => headerbar.classList.toggle("scrolled", scrollY > 4);
addEventListener("scroll", onScroll, { passive: true });
onScroll();
