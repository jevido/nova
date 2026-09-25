<script lang="ts">
  import { app, baseName, HOME, STARRED, TRASH } from "../lib/store.svelte";
  import type { Entry } from "../../bindings/nova/services/models";
  import { formatSize } from "../lib/format";
  import { dropZone, pressBookmark, type DropZone } from "../lib/dnd";
  import Icon from "./Icon.svelte";
  import MainMenu from "./MainMenu.svelte";

  let { onHide }: { onHide?: () => void } = $props();

  const user = $derived(app.session?.user);
  const used = $derived(user?.filesystem_storage_used ?? 0);
  const limit = $derived(user?.subscription?.storage_limit ?? -1);
  const tier = $derived(user?.subscription?.name || "Free");

  const bookmarks = $derived(app.prefs.bookmarks ?? []);

  /** A folder in the sidebar, as the menus and dialogs expect it. */
  function folderEntry(path: string): Entry {
    return { id: "", name: path === HOME ? "Home" : baseName(path), path, isDir: true, size: 0, mime: "", modified: "", created: "", mode: "", owner: "", shared: false } as Entry;
  }

  /** The same menu a folder gets in the file view, plus the bookmark's own items. */
  function bookmarkMenu(e: MouseEvent, path: string) {
    e.preventDefault();
    const i = bookmarks.findIndex((b) => b.path === path);
    const entry = folderEntry(path);
    app.openMenu(e.clientX, e.clientY, [
      { label: "Open", run: () => app.navigate(path) },
      ...(app.mobile ? [] : [{ label: "Open in New Window", run: () => app.newWindow(path) }]),
      { sep: true },
      { label: "Cut", run: () => app.copy(true, [path]) },
      { label: "Copy", run: () => app.copy(false, [path]) },
      { label: "Paste Into Folder", disabled: !app.clipboard, run: () => app.paste(path) },
      { sep: true },
      { label: app.mobile ? "Download" : "Download…", run: () => app.download([entry]) },
      { label: "Share…", run: () => app.openShare(entry) },
      { label: "Copy Public Link", run: () => app.copyLink(entry) },
      { sep: true },
      { label: app.isStarred(path) ? "Unstar" : "Star", run: () => app.toggleStar([entry]) },
      { label: "Rename Folder…", run: () => renameFolder(path) },
      { label: "Move to Trash", run: () => app.trash([entry]).then(() => app.removeBookmark(path)) },
      { sep: true },
      { label: "Rename Bookmark…", run: () => renameBookmark(path) },
      { label: "Move Up", disabled: i <= 0, run: () => app.moveBookmark(path, i - 1) },
      { label: "Move Down", disabled: i >= bookmarks.length - 1, run: () => app.moveBookmark(path, i + 2) },
      { label: "Remove from Sidebar", run: () => app.removeBookmark(path) },
      { sep: true },
      { label: "Copy Location", run: () => app.copyPath(entry) },
      { label: "Properties", run: () => (app.modal = { kind: "properties", entry }) },
    ]);
  }

  function homeMenu(e: MouseEvent) {
    e.preventDefault();
    const entry = folderEntry(HOME);
    app.openMenu(e.clientX, e.clientY, [
      { label: "Open", run: () => app.navigate(HOME) },
      ...(app.mobile ? [] : [{ label: "Open in New Window", run: () => app.newWindow(HOME) }]),
      { sep: true },
      { label: "Paste Into Folder", disabled: !app.clipboard, run: () => app.paste(HOME) },
      { sep: true },
      { label: "Copy Location", run: () => app.copyPath(entry) },
      { label: "Properties", run: () => (app.modal = { kind: "properties", entry }) },
    ]);
  }

  async function renameFolder(path: string) {
    const name = baseName(path);
    const v = await app.prompt({ title: "Rename Folder", label: "Name", value: name, confirm: "Rename" });
    // app.rename also updates the bookmark (and its label if it was the folder's name).
    if (v?.trim() && v.trim() !== name) await app.rename(path, v.trim());
  }

  async function renameBookmark(path: string) {
    const b = bookmarks.find((x) => x.path === path);
    if (!b) return;
    const v = await app.prompt({ title: "Rename Bookmark", label: "Name", value: b.name, confirm: "Rename" });
    if (v?.trim()) app.renameBookmarkTo(path, v.trim());
  }

  // ---------- editing the bookmarks by drag and drop, like Nautilus ----------
  /** Where a dragged folder or bookmark would be inserted, as a list index. */
  let insertAt = $state<number | null>(null);

  const addingFolders = $derived(!!app.dragging?.allDirs);
  const editing = $derived(addingFolders || app.draggingBookmark !== null);

  // The whole sidebar is one drop zone. Over the middle of a folder row
  // (Home, Trash, a bookmark), dragged items move into that folder; anywhere
  // else a dragged folder becomes a bookmark at that spot, like Nautilus.
  let rowsEl = $state<HTMLElement>();

  function bookmarkIndexAt(y: number): number {
    const rows = [...(rowsEl?.querySelectorAll<HTMLElement>(".bookmarks .row[data-path]") ?? [])];
    return rows.filter((r) => {
      const b = r.getBoundingClientRect();
      return b.top + b.height / 2 < y;
    }).length;
  }

  const sidebarZone: DropZone = {
    over(x, y, target, payload) {
      const row = target.closest<HTMLElement>(".row[data-drop-path]");
      if (payload.kind === "items" && row) {
        const r = row.getBoundingClientRect();
        const t = (y - r.top) / r.height;
        if (!payload.allDirs || (t >= 0.25 && t <= 0.75)) {
          insertAt = null;
          return { kind: "into", path: row.dataset.dropPath! };
        }
      }
      if (payload.kind === "items" && !payload.allDirs) {
        insertAt = null;
        return null;
      }
      insertAt = bookmarkIndexAt(y);
      return { kind: "insert", index: insertAt };
    },
    drop(intent, payload) {
      if (intent?.kind !== "insert") return;
      if (payload.kind === "bookmark") app.moveBookmark(payload.path, intent.index);
      else if (payload.allDirs) app.addBookmarks(payload.paths, intent.index);
    },
    leave() {
      insertAt = null;
    },
  };

  function middleClick(e: MouseEvent, path: string) {
    if (e.button !== 1 || app.mobile) return;
    e.preventDefault();
    app.newWindow(path);
  }
</script>

<aside class="sidebar">
  <header class="side-header">
    {#if onHide}
      <button class="btn image flat" title="Hide Sidebar" onclick={onHide}><Icon name="sidebar-show" /></button>
    {/if}
    <span class="side-title">Nova</span>
    <MainMenu />
  </header>
  <div class="rows" bind:this={rowsEl} use:dropZone={sidebarZone}>
    <button
      class="row"
      class:selected={app.path === HOME && !app.results}
      class:drop={app.dropTarget === HOME}
      data-path={HOME}
      data-drop-path={HOME}
      data-file-drop-target
      onclick={() => app.navigate(HOME)}
      onmousedown={(e) => e.button === 1 && e.preventDefault()}
      onauxclick={(e) => middleClick(e, HOME)}
      oncontextmenu={homeMenu}
    >
      <Icon name="user-home" /><span>Home</span>
    </button>
    <button class="row" class:selected={app.path === STARRED && !app.results} onclick={() => app.navigate(STARRED)}>
      <Icon name="starred" /><span>Starred</span>
    </button>
    <button
      class="row"
      class:selected={app.inTrash}
      class:drop={app.dropTarget === TRASH}
      data-drop-path={TRASH}
      onclick={() => app.navigate(TRASH)}
      oncontextmenu={(e) => {
        e.preventDefault();
        app.openMenu(e.clientX, e.clientY, [
          { label: "Open", run: () => app.navigate(TRASH) },
          { label: "Empty Trash", disabled: !app.trashCount, run: () => app.emptyTrash() },
        ]);
      }}
    >
      <Icon name={app.trashCount ? "user-trash-full" : "user-trash"} /><span>Trash</span>
    </button>

    {#if bookmarks.length || editing}
      <div class="sep"></div>
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="bookmarks">
        {#each bookmarks as b, i (b.path)}
          {#if insertAt === i}<div class="insert"></div>{/if}
          <button
            class="row"
            class:selected={app.path === b.path && !app.results}
            class:drop={app.dropTarget === b.path}
            class:moving={app.draggingBookmark === b.path}
            title={b.path.replace(/^\/me/, "")}
            data-path={b.path}
            data-drop-path={b.path}
            data-file-drop-target
            onclick={() => app.navigate(b.path)}
            oncontextmenu={(e) => bookmarkMenu(e, b.path)}
            onmousedown={(e) => (e.button === 1 ? e.preventDefault() : pressBookmark(e, b.path, b.name))}
            onauxclick={(e) => middleClick(e, b.path)}
          >
            <Icon name="folder" /><span>{b.name}</span>
          </button>
        {/each}
        {#if insertAt === bookmarks.length && bookmarks.length}<div class="insert"></div>{/if}
        {#if addingFolders}
          <div class="row new-bookmark" class:hot={insertAt === bookmarks.length}>
            <Icon name="list-add" /><span>New Bookmark</span>
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <button class="account" title="Account and settings" onclick={() => ((app.settingsOpen = true), (app.drawerOpen = false))}>
    <span class="acct-row">
      <span class="avatar" aria-hidden="true">{(user?.username ?? "N").slice(0, 1).toUpperCase()}</span>
      <span class="acct-text">
        <span class="acct-name">{user?.username ?? "Nova"}</span>
        <span class="dim">{tier}</span>
      </span>
    </span>
    {#if limit > 0}
      <span class="bar" title="{formatSize(used)} of {formatSize(limit)} used"><span style:width="{Math.min(100, (used / limit) * 100)}%"></span></span>
    {/if}
  </button>
</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    background: var(--sidebar-bg);
    box-shadow: inset -1px 0 var(--sidebar-border);
  }
  .side-header {
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 47px;
    padding: 6px 7px;
    --wails-draggable: drag;
  }
  .side-title {
    flex: 1;
    min-width: 0;
    padding-left: 6px;
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .rows {
    flex: 1;
    overflow-y: auto;
    padding: 0 6px 6px;
  }
  /* .navigation-sidebar rows */
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 38px;
    margin: 2px 0;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg);
    text-align: left;
    outline: none;
  }
  .row span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row:hover {
    background: var(--hover);
  }
  .row:active {
    background: var(--active);
  }
  .row.selected {
    background: var(--btn-bg);
  }
  .row.selected:hover {
    background: var(--btn-hover);
  }
  .row.drop,
  .row:global(.file-drop-target-active) {
    background: var(--selected);
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .row:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .insert {
    height: 2px;
    margin: -1px 8px;
    border-radius: 1px;
    background: var(--accent);
    position: relative;
    z-index: 1;
  }
  .row.moving {
    opacity: 0.4;
  }
  .new-bookmark {
    color: var(--fg-dim);
    box-shadow: inset 0 0 0 1px var(--border);
    border-style: dashed;
  }
  .new-bookmark.hot {
    color: var(--fg);
    background: var(--selected);
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .sep {
    height: 1px;
    margin: 6px 4px;
    background: var(--border);
    opacity: 0.6;
  }
  .account {
    display: block;
    margin: 0 6px 6px;
    padding: 10px 10px 12px;
    border: 0;
    border-radius: 9px;
    background: none;
    color: var(--fg);
    text-align: left;
    font-size: 0.87em;
    outline: none;
  }
  .account:hover {
    background: var(--hover);
  }
  .account:active {
    background: var(--active);
  }
  .account:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .account > span {
    display: block;
  }
  .acct-row {
    display: flex !important;
    align-items: center;
    gap: 10px;
  }
  .avatar {
    flex: none;
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--accent) 30%, var(--sidebar-bg));
    color: var(--fg);
    font-weight: bold;
    font-size: 1.1em;
  }
  .acct-text {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .acct-name {
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--btn-bg);
    overflow: hidden;
    margin-top: 10px;
  }
  .bar > span {
    height: 100%;
    border-radius: 3px;
    background: var(--accent);
  }
</style>
