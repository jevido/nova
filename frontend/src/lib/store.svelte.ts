import { SvelteSet } from "svelte/reactivity";
import { Clipboard, Events, System } from "@wailsio/runtime";
import * as Files from "../../bindings/nova/services/filesservice";
import * as Session from "../../bindings/nova/services/sessionservice";
import * as Transfers from "../../bindings/nova/services/transferservice";
import * as Windows from "../../bindings/nova/windowservice";
import * as Updates from "../../bindings/nova/services/updateservice";
import type {
  Entry,
  Folder,
  OpResult,
  Session as SessionT,
  Transfer,
  UpdateStatus,
} from "../../bindings/nova/services/models";
import type { Prefs } from "../../bindings/nova/internal/config/models";
import { pluralize } from "./format";

export const HOME = "/me";
export const TRASH = "/me/.Trash";
/** Virtual location listing starred items, like starred:/// in Nautilus. */
export const STARRED = "starred:";

export type SortKey = "name" | "size" | "modified" | "type";
export type Toast = { id: number; text: string; action?: { label: string; run: () => void }; error?: boolean };
export type Modal =
  | { kind: "confirm"; title: string; body: string; confirm: string; destructive?: boolean; resolve: (ok: boolean) => void }
  | { kind: "prompt"; title: string; label: string; value: string; confirm: string; select?: number; resolve: (v: string | null) => void }
  | { kind: "properties"; entry: Entry }
  | { kind: "preview"; entry: Entry }
  | { kind: "shortcuts" }
  | { kind: "about" };
export type MenuItem =
  | { sep: true }
  | { label: string; accel?: string; disabled?: boolean; checked?: boolean; run: () => void; sep?: false };
export type MenuState = { x: number; y: number; items: MenuItem[]; minWidth?: number } | null;

type Undo = { label: string; run: () => Promise<void> };

export const ZOOM_SIZES = [48, 64, 96, 128, 192];
export const LIST_ZOOM_SIZES = [16, 24, 32, 48, 64];

function errText(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  if (e && typeof e === "object" && "message" in e) return String((e as { message: unknown }).message);
  return "Something went wrong";
}

export function baseName(p: string): string {
  return p.slice(p.lastIndexOf("/") + 1);
}
export function parentOf(p: string): string {
  const i = p.lastIndexOf("/");
  return i <= 0 ? "/" : p.slice(0, i);
}
export function displayName(p: string, username?: string): string {
  if (p === HOME) return "Home";
  if (p === TRASH) return "Trash";
  if (p === STARRED) return "Starred";
  return baseName(p) || username || "Home";
}

const defaultPrefs: Prefs = {
  view: "grid",
  showHidden: false,
  sortBy: "name",
  sortDesc: false,
  zoom: 1,
  sidebarOpen: true,
  theme: "system",
  bookmarks: [],
  starred: [],
  foldersFirst: true,
};

class AppState {
  // ---- session ----
  booting = $state(true);
  session = $state<SessionT | null>(null);
  prefs = $state<Prefs>({ ...defaultPrefs });
  /** Phone build: touch interaction, drawer sidebar, no window controls. */
  mobile = $state(false);
  drawerOpen = $state(false);

  // ---- navigation ----
  path = $state(HOME);
  history = $state<string[]>([]);
  future = $state<string[]>([]);
  folder = $state<Folder | null>(null);
  loading = $state(false);
  error = $state<string | null>(null);
  editingLocation = $state(false);
  private loadSeq = 0;

  // ---- search ----
  searchOpen = $state(false);
  query = $state("");
  results = $state<Entry[] | null>(null);
  searching = $state(false);
  private searchTimer: ReturnType<typeof setTimeout> | undefined;
  private searchSeq = 0;

  // ---- selection ----
  selected = new SvelteSet<string>();
  anchor = $state<string | null>(null);
  cursor = $state<string | null>(null);
  renaming = $state<string | null>(null);
  dropTarget = $state<string | null>(null);

  // ---- misc ----
  clipboard = $state<{ mode: "copy" | "cut"; paths: string[] } | null>(null);
  transfers = $state<Transfer[]>([]);
  toasts = $state<Toast[]>([]);
  modal = $state<Modal | null>(null);
  menu = $state<MenuState>(null);
  trashCount = $state(0);
  update = $state<UpdateStatus | null>(null);
  private undoStack: Undo[] = [];
  private toastSeq = 0;

  entries: Entry[] = $derived.by(() => {
    const src = this.results ?? this.folder?.children ?? [];
    const hidden = this.prefs.showHidden;
    const list = hidden ? [...src] : src.filter((e) => !e.name.startsWith("."));
    const dir = this.prefs.sortDesc ? -1 : 1;
    const key = this.prefs.sortBy as SortKey;
    const coll = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });
    list.sort((a, b) => {
      if (this.prefs.foldersFirst && a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      let r = 0;
      if (key === "size") r = (a.isDir ? 0 : a.size) - (b.isDir ? 0 : b.size);
      else if (key === "modified") r = Date.parse(a.modified) - Date.parse(b.modified);
      else if (key === "type") r = coll.compare(a.mime || "", b.mime || "");
      if (r === 0) r = coll.compare(a.name, b.name);
      return r * dir;
    });
    return list;
  });

  selection: Entry[] = $derived(this.entries.filter((e) => this.selected.has(e.path)));

  get inTrash(): boolean {
    return this.path === TRASH || this.path.startsWith(TRASH + "/");
  }

  get canWrite(): boolean {
    return !!this.folder?.canWrite && this.results === null;
  }

  // =============== lifecycle ===============

  async boot() {
    this.mobile =
      System.IsMobile() ||
      new URLSearchParams(location.search).has("mobile") ||
      /Android|iPhone|iPad/i.test(navigator.userAgent);
    document.documentElement.dataset.mobile = String(this.mobile);
    // Android's back gesture asks us first (see MainActivity.java).
    (window as unknown as { __novaBack: () => boolean }).__novaBack = () => this.handleBack();
    try {
      this.prefs = { ...defaultPrefs, ...(await Session.Prefs()) };
      if (!this.prefs.bookmarks) this.prefs.bookmarks = [];
      if (!this.prefs.starred) this.prefs.starred = [];
    } catch {
      /* keep defaults */
    }
    this.applyTheme();
    try {
      const s = await Session.Restore();
      this.session = s;
      if (s.signedIn) await this.afterSignIn();
    } catch (e) {
      this.session = { signedIn: false, server: "", user: null };
      this.toast(`Could not reach the server: ${errText(e)}`, { error: true });
    } finally {
      this.booting = false;
    }

    Events.On("transfer", (ev) => this.onTransfer(ev.data));
    Events.On("fs:changed", (ev) => {
      if (ev.data === this.path) this.reload(true);
    });
    Events.On("files:dropped", (ev) => {
      const dir = ev.data.dir || this.path;
      if (!this.session?.signedIn || dir.startsWith(TRASH) || !ev.data.files?.length) return;
      this.upload(dir, ev.data.files);
    });
    Transfers.List().then((t) => (this.transfers = t ?? []));
    Updates.Status().then((u) => (this.update = u));
    Events.On("update", (ev) => this.onUpdate(ev.data));
  }

  /** Forget everything tied to one account; every account's home is /me. */
  private resetAccountState() {
    this.history = [];
    this.future = [];
    this.undoStack = [];
    this.clipboard = null;
    this.clearSelection();
    this.closeSearch(false);
  }

  async afterSignIn() {
    this.resetAccountState();
    // Extra windows are opened on a folder via ?path=.
    const start = new URLSearchParams(location.search).get("path");
    await this.navigate(start && (start === HOME || start.startsWith(HOME + "/")) ? start : HOME, false);
    this.refreshTrashCount();
  }

  async signInWithKey(server: string, key: string) {
    this.session = await Session.SignInWithKey(server, key);
    await this.afterSignIn();
  }

  async signIn(server: string, user: string, pass: string) {
    this.session = await Session.SignIn(server, user, pass);
    await this.afterSignIn();
  }

  async signOut() {
    const ok = await this.confirm({
      title: "Sign out?",
      body: "Nova will forget the key stored on this computer.",
      confirm: "Sign Out",
    });
    if (!ok) return;
    await Session.SignOut();
    this.session = { signedIn: false, server: this.session?.server ?? "", user: null };
    this.folder = null;
    this.resetAccountState();
  }

  async refreshUser() {
    try {
      const u = await Session.Refresh();
      if (this.session && u) this.session.user = u;
    } catch {
      /* not important */
    }
  }

  savePrefs() {
    this.applyTheme();
    Session.SavePrefs($state.snapshot(this.prefs) as Prefs).catch(() => {});
  }

  applyTheme() {
    const t = this.prefs.theme;
    const dark = t === "dark" || (t !== "light" && matchMedia("(prefers-color-scheme: dark)").matches);
    document.documentElement.dataset.theme = dark ? "dark" : "light";
  }

  /** System back: close the topmost thing, else go back in history. */
  handleBack(): boolean {
    if (this.modal) {
      this.dismissModal();
      return true;
    }
    if (this.menu) {
      this.menu = null;
      return true;
    }
    if (this.renaming) {
      this.renaming = null;
      return true;
    }
    if (this.drawerOpen) {
      this.drawerOpen = false;
      return true;
    }
    if (this.searchOpen) {
      this.closeSearch();
      return true;
    }
    if (this.mobile && this.selected.size) {
      this.clearSelection();
      return true;
    }
    if (this.history.length) {
      this.back();
      return true;
    }
    return false;
  }

  /** Close the current dialog as if cancelled. */
  dismissModal() {
    const m = this.modal;
    this.modal = null;
    if (m?.kind === "confirm") m.resolve(false);
    if (m?.kind === "prompt") m.resolve(null);
  }

  // =============== navigation ===============

  private async list(p: string): Promise<Folder | null> {
    if (p === STARRED) {
      const children = await Files.StatMany([...(this.prefs.starred ?? [])]);
      return { path: p, crumbs: [], children: children ?? [], canWrite: false };
    }
    return Files.List(p);
  }

  async navigate(p: string, push = true, keepSelection?: string[]) {
    if (push && p !== this.path) {
      this.history.push(this.path);
      this.future = [];
    }
    this.closeSearch(false);
    this.editingLocation = false;
    this.drawerOpen = false;
    const prev = this.path;
    this.path = p;
    const seq = ++this.loadSeq;
    this.loading = true;
    this.error = null;
    if (prev !== p) this.clearSelection();
    try {
      const f = await this.list(p);
      if (seq !== this.loadSeq || !f) return;
      this.folder = f;
      if (keepSelection) this.selectPaths(keepSelection);
    } catch (e) {
      if (seq !== this.loadSeq) return;
      this.folder = null;
      this.error = errText(e);
    } finally {
      if (seq === this.loadSeq) this.loading = false;
    }
  }

  back() {
    const p = this.history.pop();
    if (p === undefined) return;
    this.future.push(this.path);
    const came = this.path;
    // Desktop highlights the folder you came from; on a phone a selection
    // would switch taps into selection mode.
    this.navigate(p, false, this.mobile ? undefined : [came]);
  }

  forward() {
    const p = this.future.pop();
    if (p === undefined) return;
    this.history.push(this.path);
    this.navigate(p, false);
  }

  up() {
    if (this.path === HOME || this.path === STARRED) return;
    const came = this.path;
    this.navigate(parentOf(this.path), true, this.mobile ? undefined : [came]);
  }

  async reload(quiet = false) {
    if (this.results !== null) return this.runSearch();
    const seq = ++this.loadSeq;
    if (!quiet) this.loading = true;
    try {
      const f = await this.list(this.path);
      if (seq !== this.loadSeq || !f) return;
      this.folder = f;
      this.error = null;
      // Drop selection of entries that vanished.
      const live = new Set((f.children ?? []).map((c) => c.path));
      for (const p of [...this.selected]) if (!live.has(p)) this.selected.delete(p);
    } catch (e) {
      if (seq === this.loadSeq) this.error = errText(e);
    } finally {
      if (seq === this.loadSeq) this.loading = false;
    }
  }

  refreshTrashCount() {
    Files.TrashCount().then((n) => (this.trashCount = n));
  }

  // =============== search ===============

  openSearch() {
    this.searchOpen = true;
  }

  closeSearch(reload = true) {
    clearTimeout(this.searchTimer);
    const had = this.results !== null;
    this.searchOpen = false;
    this.query = "";
    this.results = null;
    this.searching = false;
    if (had && reload) this.clearSelection();
  }

  setQuery(q: string) {
    this.query = q;
    clearTimeout(this.searchTimer);
    if (!q.trim()) {
      this.results = null;
      this.searching = false;
      return;
    }
    this.searching = true;
    this.searchTimer = setTimeout(() => this.runSearch(), 250);
  }

  async runSearch() {
    const seq = ++this.searchSeq;
    const q = this.query.trim();
    if (!q) return;
    this.searching = true;
    try {
      const r = await Files.Search(this.path === STARRED ? HOME : this.path, q);
      if (seq !== this.searchSeq) return;
      this.results = r ?? [];
      this.clearSelection();
    } catch (e) {
      if (seq === this.searchSeq) this.toast(errText(e), { error: true });
    } finally {
      if (seq === this.searchSeq) this.searching = false;
    }
  }

  // =============== selection ===============

  clearSelection() {
    this.selected.clear();
    this.anchor = null;
    this.cursor = null;
  }

  selectPaths(paths: string[]) {
    this.selected.clear();
    for (const p of paths) this.selected.add(p);
    this.anchor = this.cursor = paths[paths.length - 1] ?? null;
  }

  selectAll() {
    for (const e of this.entries) this.selected.add(e.path);
  }

  invertSelection() {
    for (const e of this.entries) {
      if (this.selected.has(e.path)) this.selected.delete(e.path);
      else this.selected.add(e.path);
    }
  }

  /** Click handling with GTK semantics: plain, ctrl-toggle, shift-range. */
  clickSelect(path: string, ev: { ctrlKey: boolean; shiftKey: boolean; metaKey?: boolean }) {
    const ctrl = ev.ctrlKey || !!ev.metaKey;
    if (ev.shiftKey && this.anchor) {
      const list = this.entries.map((e) => e.path);
      const a = list.indexOf(this.anchor);
      const b = list.indexOf(path);
      if (a >= 0 && b >= 0) {
        if (!ctrl) this.selected.clear();
        for (let i = Math.min(a, b); i <= Math.max(a, b); i++) this.selected.add(list[i]);
        this.cursor = path;
        return;
      }
    }
    if (ctrl) {
      if (this.selected.has(path)) this.selected.delete(path);
      else this.selected.add(path);
      this.anchor = this.cursor = path;
      return;
    }
    this.selectPaths([path]);
  }

  /** Keyboard cursor movement; `delta` in items, `columns` for grid rows. */
  moveCursor(delta: number, extend: boolean, absolute?: "start" | "end") {
    const list = this.entries;
    if (!list.length) return;
    let i = this.cursor ? list.findIndex((e) => e.path === this.cursor) : -1;
    if (absolute === "start") i = 0;
    else if (absolute === "end") i = list.length - 1;
    else if (i < 0) i = delta > 0 ? 0 : list.length - 1;
    else i = Math.max(0, Math.min(list.length - 1, i + delta));
    const p = list[i].path;
    if (extend) this.clickSelect(p, { ctrlKey: false, shiftKey: true });
    else this.selectPaths([p]);
    this.cursor = p;
    queueMicrotask(() => document.querySelector(`[data-path="${CSS.escape(p)}"]`)?.scrollIntoView({ block: "nearest" }));
  }

  /** Type-ahead: jump to the first entry starting with the typed prefix. */
  private typeBuf = "";
  private typeTimer: ReturnType<typeof setTimeout> | undefined;
  typeAhead(ch: string) {
    clearTimeout(this.typeTimer);
    this.typeBuf += ch.toLowerCase();
    this.typeTimer = setTimeout(() => (this.typeBuf = ""), 800);
    const hit = this.entries.find((e) => e.name.toLowerCase().startsWith(this.typeBuf));
    if (hit) {
      this.selectPaths([hit.path]);
      queueMicrotask(() => document.querySelector(`[data-path="${CSS.escape(hit.path)}"]`)?.scrollIntoView({ block: "nearest" }));
    }
  }

  // =============== feedback ===============

  /** Show an in-app notification; timeout 0 keeps it until dismissed. */
  toast(text: string, opts: { action?: Toast["action"]; error?: boolean; timeout?: number } = {}) {
    const id = ++this.toastSeq;
    // GTK shows one in-app notification at a time.
    this.toasts = [{ id, text, action: opts.action, error: opts.error }];
    const ms = opts.timeout ?? (opts.action ? 6000 : 4000);
    if (ms > 0) setTimeout(() => this.dismissToast(id), ms);
  }

  dismissToast(id: number) {
    this.toasts = this.toasts.filter((t) => t.id !== id);
  }

  confirm(o: { title: string; body: string; confirm: string; destructive?: boolean }): Promise<boolean> {
    return new Promise((resolve) => (this.modal = { kind: "confirm", ...o, resolve }));
  }

  prompt(o: { title: string; label: string; value: string; confirm: string; select?: number }): Promise<string | null> {
    return new Promise((resolve) => (this.modal = { kind: "prompt", ...o, resolve }));
  }

  openMenu(x: number, y: number, items: MenuItem[]) {
    this.menu = { x, y, items };
  }

  private report(res: OpResult | null, what: string) {
    const errs = res?.errors ?? [];
    if (errs.length) this.toast(`${what}: ${errs[0]}${errs.length > 1 ? ` (+${errs.length - 1} more)` : ""}`, { error: true });
  }

  private pushUndo(u: Undo) {
    this.undoStack.push(u);
    if (this.undoStack.length > 20) this.undoStack.shift();
  }

  /** Undo a specific entry (from its toast) or the most recent one (Ctrl+Z). */
  async undo(entry?: Undo) {
    let u: Undo | undefined;
    if (entry) {
      const i = this.undoStack.indexOf(entry);
      if (i < 0) return; // already undone
      u = this.undoStack.splice(i, 1)[0];
    } else {
      u = this.undoStack.pop();
    }
    if (!u) {
      this.toast("Nothing to undo");
      return;
    }
    try {
      await u.run();
      this.toast(`Undid ${u.label}`);
    } catch (e) {
      this.toast(errText(e), { error: true });
    }
    this.reload(true);
    this.refreshTrashCount();
  }

  private async guard<T>(fn: () => Promise<T>): Promise<T | undefined> {
    try {
      return await fn();
    } catch (e) {
      const msg = errText(e);
      this.toast(msg, { error: true });
      if (/not authorized/i.test(msg) && this.session) this.session = { ...this.session, signedIn: false };
      return undefined;
    }
  }

  // =============== file operations ===============

  open(e: Entry) {
    if (e.isDir) {
      this.navigate(e.path);
      return;
    }
    // Phones can't hand files to other apps yet; show them in-app instead.
    if (this.mobile) {
      this.preview(e);
      return;
    }
    this.guard(() => Transfers.Open(e.path));
  }

  openSelection() {
    const sel = this.selection;
    if (sel.length === 1) this.open(sel[0]);
    else sel.filter((e) => !e.isDir).forEach((e) => this.open(e));
  }

  newWindow(path = this.path) {
    Windows.NewWindow(path).catch(() => {});
  }

  preview(e?: Entry) {
    const target = e ?? this.selection[0];
    if (target) this.modal = { kind: "preview", entry: target };
  }

  async newFolder() {
    if (!this.canWrite) return;
    const taken = new Set((this.folder?.children ?? []).map((c) => c.name));
    let name = "Untitled Folder";
    for (let i = 2; taken.has(name); i++) name = `Untitled Folder ${i}`;
    const v = await this.prompt({ title: "New Folder", label: "Folder name", value: name, confirm: "Create" });
    if (!v) return;
    const dir = this.path;
    const p = await this.guard(() => Files.CreateFolder(dir, v));
    if (p) {
      await this.reload(true);
      this.selectPaths([p]);
    }
  }

  startRename(e?: Entry) {
    const target = e ?? (this.selection.length === 1 ? this.selection[0] : undefined);
    if (target && !this.inTrash) this.renaming = target.path;
  }

  async rename(path: string, name: string) {
    this.renaming = null;
    if (!name || name === baseName(path)) return;
    const np = await this.guard(() => Files.Rename(path, name));
    if (!np) return;
    const oldName = baseName(path);
    this.retargetStars(path, np);
    this.pushUndo({ label: "rename", run: () => Files.Rename(np, oldName).then(() => {}) });
    await this.reload(true);
    this.selectPaths([np]);
  }

  async trash(entries = this.selection) {
    if (!entries.length) return;
    if (this.inTrash) return this.deleteForever(entries);
    const paths = entries.map((e) => e.path);
    const res = await this.guard(() => Files.Trash(paths));
    if (!res) return;
    this.report(res, "Could not move to trash");
    const done = res.done ?? [];
    if (done.length) {
      const text =
        done.length === 1 ? `“${baseName(done[0])}” moved to the trash` : `${done.length} files moved to the trash`;
      const undo = async () => {
        const trashed = (await Files.List(TRASH))?.children ?? [];
        // Restore the newest trashed copy of each original path.
        const newest = new Map<string, Entry>();
        for (const t of trashed) {
          if (!t.origPath || !done.includes(t.origPath)) continue;
          const cur = newest.get(t.origPath);
          if (!cur || Date.parse(t.deletedAt ?? "") > Date.parse(cur.deletedAt ?? "")) newest.set(t.origPath, t);
        }
        await Files.Restore([...newest.values()].map((t) => t.path));
      };
      const entry: Undo = { label: "trash", run: undo };
      this.pushUndo(entry);
      this.toast(text, { action: { label: "Undo", run: () => this.undo(entry) } });
    }
    this.clearSelection();
    this.reload(true);
    this.refreshTrashCount();
  }

  async deleteForever(entries = this.selection) {
    if (!entries.length) return;
    const ok = await this.confirm({
      title:
        entries.length === 1
          ? `Are you sure you want to permanently delete “${entries[0].name}”?`
          : `Are you sure you want to permanently delete the ${entries.length} selected items?`,
      body: "If you delete an item, it will be permanently lost.",
      confirm: "Delete",
      destructive: true,
    });
    if (!ok) return;
    const res = await this.guard(() => Files.Delete(entries.map((e) => e.path)));
    this.report(res ?? null, "Could not delete");
    this.clearSelection();
    this.reload(true);
    this.refreshTrashCount();
    this.refreshUser();
  }

  async restore(entries = this.selection) {
    const res = await this.guard(() => Files.Restore(entries.map((e) => e.path)));
    if (!res) return;
    this.report(res, "Could not restore");
    const n = res.done?.length ?? 0;
    if (n) this.toast(n === 1 ? `“${baseName(res.done![0])}” restored` : `${n} items restored`);
    this.reload(true);
    this.refreshTrashCount();
  }

  async emptyTrash() {
    const ok = await this.confirm({
      title: "Empty all items from Trash?",
      body: "All items in the Trash will be permanently deleted.",
      confirm: "Empty Trash",
      destructive: true,
    });
    if (!ok) return;
    await this.guard(() => Files.EmptyTrash());
    this.refreshTrashCount();
    this.refreshUser();
    if (this.inTrash) this.reload(true);
  }

  copy(cut = false) {
    const paths = this.selection.map((e) => e.path);
    if (!paths.length) return;
    this.clipboard = { mode: cut ? "cut" : "copy", paths };
    this.toast(
      `${pluralize(paths.length, "item", "items")} ${cut ? "will be moved" : "will be copied"} if you select the Paste command`,
    );
  }

  async paste(dir = this.path) {
    const cb = this.clipboard;
    if (!cb || this.inTrash) return;
    if (cb.mode === "cut") {
      await this.move(cb.paths, dir);
      this.clipboard = null;
    } else {
      await this.copyTo(cb.paths, dir);
    }
  }

  async move(paths: string[], dir: string) {
    const valid = paths.filter((p) => parentOf(p) !== dir && p !== dir);
    if (!valid.length) return;
    const res = await this.guard(() => Files.Move(valid, dir));
    if (!res) return;
    this.report(res, "Could not move");
    const done = res.done ?? [];
    const from = res.from ?? [];
    if (done.length) {
      const pairs = done.map((np, i) => [from[i], np] as const).filter(([op]) => !!op);
      for (const [op, np] of pairs) this.retargetStars(op, np);
      const entry: Undo = {
        label: "move",
        run: async () => {
          // Move back, then restore the original name if a clash renamed it.
          for (const [op, np] of pairs) {
            const back = await Files.Move([np], parentOf(op));
            const landed = back?.done?.[0];
            if (landed && landed !== op) await Files.Rename(landed, baseName(op));
            this.retargetStars(np, op);
          }
        },
      };
      this.pushUndo(entry);
      this.toast(
        done.length === 1 ? `“${baseName(done[0])}” moved to “${displayName(dir)}”` : `${done.length} items moved`,
        { action: { label: "Undo", run: () => this.undo(entry) } },
      );
    }
    this.reload(true);
  }

  async copyTo(paths: string[], dir: string) {
    await this.guard(() => Transfers.Copy(paths, dir));
  }

  upload(dir: string, files: string[]) {
    return this.guard(() => Transfers.Upload(dir, files));
  }

  pickUpload(folders = false) {
    if (!this.canWrite) return;
    return this.guard(() => Transfers.PickAndUpload(this.path, folders));
  }

  download(entries = this.selection) {
    if (!entries.length) return;
    return this.guard(() => Transfers.PickAndDownload(entries.map((e) => e.path)));
  }

  async share(e: Entry, on: boolean) {
    const url = await this.guard(() => Files.Share(e.path, on));
    if (url === undefined) return;
    if (on && url) {
      await Clipboard.SetText(url);
      this.toast("Public link copied to clipboard");
    } else {
      this.toast(`“${e.name}” is no longer shared`);
    }
    this.reload(true);
  }

  async copyPath(e: Entry) {
    await Clipboard.SetText(e.path.replace(/^\/me/, "") || "/");
    this.toast("Location copied to clipboard");
  }

  toggleBookmark(path = this.path) {
    if (path === HOME || path === TRASH) return;
    const bm = this.prefs.bookmarks ?? [];
    const i = bm.findIndex((b) => b.path === path);
    if (i >= 0) bm.splice(i, 1);
    else bm.push({ name: baseName(path), path });
    this.prefs.bookmarks = bm;
    this.savePrefs();
  }

  /** Keep stars pointing at renamed or moved items (and their children). */
  private retargetStars(from: string, to: string) {
    const st = this.prefs.starred ?? [];
    let changed = false;
    const next = st.map((p) => {
      if (p === from || p.startsWith(from + "/")) {
        changed = true;
        return to + p.slice(from.length);
      }
      return p;
    });
    if (changed) {
      this.prefs.starred = next;
      this.savePrefs();
    }
  }

  isStarred(path: string): boolean {
    return (this.prefs.starred ?? []).includes(path);
  }

  toggleStar(entries = this.selection) {
    const st = [...(this.prefs.starred ?? [])];
    const allStarred = entries.every((e) => st.includes(e.path));
    for (const e of entries) {
      const i = st.indexOf(e.path);
      if (allStarred && i >= 0) st.splice(i, 1);
      else if (!allStarred && i < 0) st.push(e.path);
    }
    this.prefs.starred = st;
    this.savePrefs();
    if (this.path === STARRED) this.reload(true);
  }

  isBookmarked(path = this.path): boolean {
    return (this.prefs.bookmarks ?? []).some((b) => b.path === path);
  }

  // =============== updates ===============

  private onUpdate(u: UpdateStatus) {
    const prev = this.update?.state;
    this.update = u;
    if (u.state === prev) return;
    if (u.state === "ready") {
      this.toast(`Nova ${u.latestVersion} is ready to install`, {
        action: { label: this.mobile ? "Install" : "Restart", run: () => this.applyUpdate() },
        timeout: 0,
      });
    } else if (u.state === "manual" && prev !== "manual") {
      this.toast(
        u.packageManaged
          ? `Nova ${u.latestVersion} is available — update it with your package manager`
          : `Nova ${u.latestVersion} is available`,
        { action: { label: "Download", run: () => Updates.OpenReleasePage() }, timeout: 0 },
      );
    }
  }

  async checkForUpdates() {
    if (this.update?.state === "ready") return this.applyUpdate();
    if (this.update?.state === "disabled") {
      this.toast("This is a development build; updates are only available in releases");
      return;
    }
    this.toast("Checking for updates…");
    try {
      const u = await Updates.CheckNow();
      this.update = u;
      if (u.state === "up-to-date") this.toast(`Nova ${u.currentVersion} is up to date`);
      else if (u.state === "error") this.toast(`Could not check for updates: ${u.error}`, { error: true });
      else if (u.state === "manual") this.onUpdate({ ...u, state: "manual" });
      // "ready" is announced by the update event.
    } catch (e) {
      this.toast(errText(e), { error: true });
    }
  }

  async applyUpdate() {
    const u = this.update;
    if (!u || u.state !== "ready") return;
    const android = (window as unknown as { NovaAndroid?: { installApk(p: string): string } }).NovaAndroid;
    if (android) {
      const r = android.installApk(u.apkPath);
      if (r === "permission") this.toast("Allow Nova to install apps, then tap Install again", { timeout: 0, action: { label: "Install", run: () => this.applyUpdate() } });
      else if (r !== "ok") this.toast(`Could not start the installer: ${r}`, { error: true });
      return;
    }
    try {
      await Updates.Restart();
    } catch (e) {
      this.toast(`Restart failed: ${errText(e)}`, { error: true });
    }
  }

  // =============== transfers ===============

  private onTransfer(t: Transfer) {
    const i = this.transfers.findIndex((x) => x.id === t.id);
    const prev = i >= 0 ? this.transfers[i] : null;
    if (i >= 0) this.transfers[i] = t;
    else this.transfers.push(t);
    const finished = prev && prev.state !== t.state && (t.state === "done" || t.state === "failed");
    if (!finished) return;
    if (t.state === "failed") {
      this.toast(`${t.kind === "upload" ? "Upload" : t.kind === "copy" ? "Copy" : "Download"} of “${t.title}” failed: ${t.error}`, {
        error: true,
      });
    } else if (t.kind === "upload" || t.kind === "copy") {
      if (t.dest === this.path) this.reload(true);
      this.refreshUser();
    } else if (t.kind === "download") {
      this.toast(this.mobile ? `“${t.title}” saved to Download/Nova` : `Downloaded “${t.title}”`);
    }
  }

  get activeTransfers(): Transfer[] {
    return this.transfers.filter((t) => t.state === "queued" || t.state === "running");
  }

  cancelTransfer(id: number) {
    Transfers.Cancel(id);
  }

  clearTransfers() {
    Transfers.ClearFinished().then(() => Transfers.List().then((t) => (this.transfers = t ?? [])));
  }
}

export const app = new AppState();
