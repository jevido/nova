import { app, parentOf, TRASH } from "./store.svelte";

// Internal drag and drop of remote entries. OS file drops are handled by the
// Wails runtime through the data-file-drop-target attribute instead.
const MIME = "application/x-nova-paths";
let dragging: string[] = [];

export function startDrag(e: DragEvent, paths: string[]) {
  dragging = paths;
  if (!e.dataTransfer) return;
  e.dataTransfer.effectAllowed = "copyMove";
  e.dataTransfer.setData(MIME, JSON.stringify(paths));
  e.dataTransfer.setData("text/plain", paths.join("\n"));
  const n = paths.length;
  if (n > 1) {
    // A small badge instead of the default ghost for multi-selections.
    const ghost = document.createElement("div");
    ghost.textContent = `${n} items`;
    ghost.style.cssText =
      "position:fixed;top:-100px;padding:4px 10px;border-radius:12px;background:#3584e4;color:#fff;font:bold 13px sans-serif";
    document.body.appendChild(ghost);
    e.dataTransfer.setDragImage(ghost, 10, 10);
    setTimeout(() => ghost.remove(), 0);
  }
}

export function endDrag() {
  dragging = [];
  app.dropTarget = null;
}

function acceptable(target: string): boolean {
  if (!dragging.length) return false;
  if (target.startsWith(TRASH + "/")) return false;
  return !dragging.some((p) => p === target || target.startsWith(p + "/"));
}

export function dragOver(e: DragEvent, target: string) {
  if (!e.dataTransfer?.types.includes(MIME) || !acceptable(target)) return;
  e.preventDefault();
  e.stopPropagation();
  const copy = e.ctrlKey && target !== TRASH;
  const sameDir = dragging.every((p) => parentOf(p) === target);
  e.dataTransfer.dropEffect = copy ? "copy" : sameDir ? "none" : "move";
  app.dropTarget = target;
}

export function dragLeave(e: DragEvent, target: string) {
  const to = e.relatedTarget as Node | null;
  if (to && (e.currentTarget as Node).contains(to)) return;
  if (app.dropTarget === target) app.dropTarget = null;
}

export function dropOn(e: DragEvent, target: string) {
  if (!e.dataTransfer?.types.includes(MIME) || !acceptable(target)) return;
  e.preventDefault();
  e.stopPropagation();
  const paths = [...dragging];
  const copy = e.ctrlKey;
  endDrag();
  if (target === TRASH) {
    const entries = app.entries.filter((x) => paths.includes(x.path));
    app.trash(entries.length ? entries : paths.map((p) => ({ path: p }) as never));
  } else if (copy) {
    app.copyTo(paths, target);
  } else {
    app.move(paths, target);
  }
}
