import { Events } from "@wailsio/runtime";
import * as Windows from "../../bindings/nova/windowservice";
import type { CrossDrag } from "../../bindings/nova/models";
import { app, parentOf, TRASH } from "./store.svelte";

// Internal drag and drop, done with plain mouse events instead of the HTML5
// drag API. In the desktop webviews an HTML5 drag goes through the platform's
// drag machinery (GTK on Linux), which Wails also hooks for OS file drops, so
// it isn't reliable there. Mouse events behave the same everywhere.
// OS file drops still come in through Wails (data-file-drop-target).
//
// When an item drag leaves its window, it is handed to the platform's drag
// and drop (see crossdrag_linux.go), so it can be dropped in another Nova
// window. The window under it then gets "xdrag" events and runs the same
// drop target logic as a drag of its own.
//
// Drop targets:
//  - an element with data-drop-path="<folder>": dropping moves (Ctrl: copies)
//    the dragged items into that folder;
//  - an element using the dropZone action: it decides what a drop means at
//    each point (the sidebar uses this to insert bookmarks between rows).

export type DragPayload = { kind: "items"; paths: string[]; allDirs: boolean; label: string } | { kind: "bookmark"; path: string; label: string };

export type DropIntent = { kind: "into"; path: string } | { kind: "insert"; index: number } | null;

export type DropZone = {
  /** Where would the payload land at this point? */
  over(x: number, y: number, target: Element, payload: DragPayload): DropIntent;
  /** Handle an "insert" drop. "into" drops are handled here in dnd. */
  drop(intent: DropIntent, payload: DragPayload): void;
  leave(): void;
};

const THRESHOLD = 5;
const zones = new WeakMap<Element, DropZone>();

let pending: { x: number; y: number; make: () => DragPayload | null } | null = null;
let payload: DragPayload | null = null;
let ghost: HTMLDivElement | null = null;
let activeZone: DropZone | null = null;
let intent: DropIntent = null;
let copy = false;
let pointer = { x: 0, y: 0 };
let scrollFrame = 0;
// Set while the drag came from another window (or left this one and came back).
let remote = false;
let handingOff = false;
let canHandOff = true;

/** Svelte action: make an element a drop zone. */
export function dropZone(node: HTMLElement, zone: DropZone) {
  zones.set(node, zone);
  node.dataset.dropZone = "";
  return {
    update(z: DropZone) {
      zones.set(node, z);
    },
    destroy() {
      zones.delete(node);
    },
  };
}

/** Call on mousedown over a file or folder. The drag starts once the mouse moves. */
export function pressItems(e: MouseEvent) {
  if (app.mobile || e.button !== 0 || app.inTrash) return;
  arm(e, () => {
    const sel = app.selection;
    if (!sel.length) return null;
    return {
      kind: "items",
      paths: sel.map((x) => x.path),
      allDirs: sel.every((x) => x.isDir),
      label: sel.length === 1 ? sel[0].name : `${sel.length} items`,
    };
  });
}

/** Call on mousedown over a sidebar bookmark, to reorder it. */
export function pressBookmark(e: MouseEvent, path: string, label: string) {
  if (app.mobile || e.button !== 0) return;
  arm(e, () => ({ kind: "bookmark", path, label }));
}

function arm(e: MouseEvent, make: () => DragPayload | null) {
  pending = { x: e.clientX, y: e.clientY, make };
  addEventListener("mousemove", onMove, true);
  addEventListener("mouseup", onUp, true);
}

function begin() {
  canHandOff = true;
  payload = pending?.make() ?? null;
  pending = null;
  if (!payload) return disarm();
  ghost = document.createElement("div");
  ghost.className = "drag-ghost";
  document.body.appendChild(ghost);
  addEventListener("keydown", onKey, true);
  show(payload);
}

function show(p: DragPayload) {
  app.dragging = p.kind === "items" ? { paths: p.paths, allDirs: p.allDirs } : null;
  app.draggingBookmark = p.kind === "bookmark" ? p.path : null;
  document.documentElement.classList.add("dragging");
  cancelAnimationFrame(scrollFrame);
  scrollFrame = requestAnimationFrame(autoScroll);
}

function onMove(e: MouseEvent) {
  pointer = { x: e.clientX, y: e.clientY };
  if (pending) {
    if (Math.hypot(e.clientX - pending.x, e.clientY - pending.y) < THRESHOLD) return;
    begin();
    if (!payload) return;
  }
  if (!payload) return;
  e.preventDefault();
  copy = e.ctrlKey;
  if (outside(e.clientX, e.clientY)) handOff();
  track(e.clientX, e.clientY);
}

function outside(x: number, y: number) {
  return x < 0 || y < 0 || x >= innerWidth || y >= innerHeight;
}

/** Give an item drag that left the window to the platform, so other windows can take it. */
async function handOff() {
  if (handingOff || !canHandOff || payload?.kind !== "items") return;
  handingOff = true;
  const p = payload;
  try {
    if (await Windows.StartDrag(p)) {
      // The platform has the pointer now; this window hears of the drag
      // again through "xdrag" if it comes back.
      if (payload === p) finish();
    } else {
      canHandOff = false; // keep dragging inside this window
    }
  } catch {
    canHandOff = false;
  } finally {
    handingOff = false;
  }
}

function onCrossDrag(d: CrossDrag) {
  if (app.mobile || !d.payload) return;
  if (d.type === "leave") {
    if (remote) finish();
    return;
  }
  if (!remote) {
    if (payload) return; // a drag of our own is still going
    remote = true;
    payload = { kind: "items", paths: d.payload.paths ?? [], allDirs: d.payload.allDirs, label: d.payload.label };
    show(payload);
  }
  copy = d.copy;
  pointer = { x: d.x, y: d.y };
  track(d.x, d.y);
  if (d.type === "drop") drop();
}

Events.On("xdrag", (ev) => onCrossDrag(ev.data));

function track(x: number, y: number) {
  if (!payload) return;
  const target = document.elementFromPoint(x, y);
  const zoneEl = target?.closest("[data-drop-zone]");
  const zone = zoneEl ? zones.get(zoneEl) ?? null : null;
  if (activeZone && activeZone !== zone) activeZone.leave();
  activeZone = zone;

  let next: DropIntent = null;
  if (zone && target) {
    next = zone.over(x, y, target, payload);
  } else {
    const el = target?.closest<HTMLElement>("[data-drop-path]");
    if (el?.dataset.dropPath) next = { kind: "into", path: el.dataset.dropPath };
  }
  if (next?.kind === "into" && !canDropInto(next.path)) next = null;
  intent = next;
  app.dropTarget = intent?.kind === "into" ? intent.path : null;

  if (!ghost) return;
  ghost.style.transform = `translate(${x + 14}px, ${y + 14}px)`;
  ghost.textContent = payload.label;
  ghost.dataset.action = intent === null ? "none" : intent.kind === "insert" ? "bookmark" : intent.path === TRASH ? "trash" : copy ? "copy" : "move";
}

/** Items can't go into themselves, their own children, the trash's insides, or where they already are. */
function canDropInto(target: string): boolean {
  if (payload?.kind !== "items") return false;
  if (target.startsWith(TRASH + "/")) return false;
  if (payload.paths.some((p) => p === target || target.startsWith(p + "/"))) return false;
  if (!copy && target !== TRASH && payload.paths.every((p) => parentOf(p) === target)) return false;
  return true;
}

function onUp(e: MouseEvent) {
  if (pending) return disarm();
  if (!payload) return disarm();
  e.preventDefault();
  copy = e.ctrlKey;
  track(e.clientX, e.clientY);
  // The mouseup would otherwise also count as a click on whatever is below.
  addEventListener("click", swallow, { capture: true, once: true });
  setTimeout(() => removeEventListener("click", swallow, true), 0);
  drop();
}

/** Drop the payload where track() last put it. */
function drop() {
  const p = payload;
  const it = intent;
  const zone = activeZone;
  finish();
  if (!p || !it) return;
  if (it.kind === "insert") zone?.drop(it, p);
  else if (p.kind === "items") dropItems(p.paths, it.path, copy);
}

function swallow(e: Event) {
  e.stopPropagation();
  e.preventDefault();
}

function onKey(e: KeyboardEvent) {
  if (e.key === "Escape") {
    e.preventDefault();
    e.stopPropagation();
    finish();
  } else if (e.key === "Control") {
    copy = true;
    track(pointer.x, pointer.y);
  }
}

function finish() {
  activeZone?.leave();
  activeZone = null;
  intent = null;
  payload = null;
  remote = false;
  ghost?.remove();
  ghost = null;
  cancelAnimationFrame(scrollFrame);
  document.documentElement.classList.remove("dragging");
  app.dropTarget = null;
  app.dragging = null;
  app.draggingBookmark = null;
  removeEventListener("keydown", onKey, true);
  disarm();
}

function disarm() {
  pending = null;
  removeEventListener("mousemove", onMove, true);
  removeEventListener("mouseup", onUp, true);
}

/** Scroll the list under the pointer when dragging near its top or bottom edge. */
function autoScroll() {
  const el = document.elementFromPoint(pointer.x, pointer.y)?.closest<HTMLElement>(".view, .rows");
  if (el) {
    const r = el.getBoundingClientRect();
    const edge = 36;
    const dy = pointer.y < r.top + edge ? -(r.top + edge - pointer.y) / 3 : pointer.y > r.bottom - edge ? (pointer.y - (r.bottom - edge)) / 3 : 0;
    if (dy) {
      el.scrollTop += dy;
      track(pointer.x, pointer.y);
    }
  }
  scrollFrame = requestAnimationFrame(autoScroll);
}

function dropItems(paths: string[], target: string, asCopy: boolean) {
  if (target === TRASH) {
    const entries = app.entries.filter((x) => paths.includes(x.path));
    app.trash(entries.length ? entries : paths.map((p) => ({ path: p }) as never));
  } else if (asCopy) {
    app.copyTo(paths, target);
  } else {
    app.move(paths, target);
  }
}
