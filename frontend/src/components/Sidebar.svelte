<script lang="ts">
  import { app, HOME, STARRED, TRASH } from "../lib/store.svelte";
  import { formatSize } from "../lib/format";
  import { dragLeave, dragOver, dropOn, endDrag } from "../lib/dnd";
  import Icon from "./Icon.svelte";
  import MainMenu from "./MainMenu.svelte";

  let { onHide }: { onHide?: () => void } = $props();

  const user = $derived(app.session?.user);
  const used = $derived(user?.filesystem_storage_used ?? 0);
  const limit = $derived(user?.subscription?.storage_limit ?? -1);
  const host = $derived.by(() => {
    try {
      return new URL(app.session?.server || "https://nova.storage").host;
    } catch {
      return "nova.storage";
    }
  });

  const bookmarks = $derived(app.prefs.bookmarks ?? []);

  function bookmarkMenu(e: MouseEvent, path: string) {
    e.preventDefault();
    const i = bookmarks.findIndex((b) => b.path === path);
    app.openMenu(e.clientX, e.clientY, [
      { label: "Open", run: () => app.navigate(path) },
      ...(app.mobile ? [] : [{ label: "Open in New Window", run: () => app.newWindow(path) }]),
      { sep: true },
      { label: "Rename…", run: () => renameBookmark(path) },
      { label: "Move Up", disabled: i <= 0, run: () => app.moveBookmark(path, i - 1) },
      { label: "Move Down", disabled: i >= bookmarks.length - 1, run: () => app.moveBookmark(path, i + 2) },
      { sep: true },
      { label: "Remove", run: () => app.removeBookmark(path) },
    ]);
  }

  async function renameBookmark(path: string) {
    const b = bookmarks.find((x) => x.path === path);
    if (!b) return;
    const v = await app.prompt({ title: "Rename Bookmark", label: "Name", value: b.name, confirm: "Rename" });
    if (v?.trim()) app.renameBookmarkTo(path, v.trim());
  }

  // ---------- editing the bookmarks by drag and drop, like Nautilus ----------
  const BM_MIME = "application/x-nova-bookmark";
  let movingBookmark = $state<string | null>(null);
  /** Where a dragged folder or bookmark would be inserted, as a list index. */
  let insertAt = $state<number | null>(null);

  const addingFolders = $derived(!!app.dragging?.allDirs);
  const editing = $derived(addingFolders || movingBookmark !== null);

  function onBookmarkDragStart(e: DragEvent, path: string) {
    movingBookmark = path;
    e.dataTransfer?.setData(BM_MIME, path);
    if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
  }

  function onBookmarkDragEnd() {
    movingBookmark = null;
    insertAt = null;
  }

  /** Over a bookmark row's middle: move the dragged items into that folder.
   * Everywhere else in the list (row edges, gaps) is handled by the list and
   * inserts a bookmark. */
  function inMiddle(e: DragEvent): boolean {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = (e.clientY - r.top) / r.height;
    return y >= 0.25 && y <= 0.75;
  }

  function onRowDragOver(e: DragEvent, path: string) {
    if (movingBookmark !== null || (addingFolders && !inMiddle(e))) return;
    insertAt = null;
    dragOver(e, path);
  }

  function onRowDrop(e: DragEvent, path: string) {
    if (insertAt !== null) return;
    dropOn(e, path);
  }

  function onListDragOver(e: DragEvent) {

    if (!editing || e.defaultPrevented) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = movingBookmark !== null ? "move" : "link";
    const rows = [...(e.currentTarget as HTMLElement).querySelectorAll<HTMLElement>(".row[data-path]")];
    insertAt = rows.filter((r) => {
      const b = r.getBoundingClientRect();
      return b.top + b.height / 2 < e.clientY;
    }).length;
    app.dropTarget = null;
  }

  function dropInsert(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    const at = insertAt ?? bookmarks.length;
    if (movingBookmark !== null) app.moveBookmark(movingBookmark, at);
    else if (app.dragging?.allDirs) app.addBookmarks(app.dragging.paths, at);
    movingBookmark = null;
    insertAt = null;
    endDrag();
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
  <div class="rows">
    <button
      class="row"
      class:selected={app.path === HOME && !app.results}
      class:drop={app.dropTarget === HOME}
      data-path={HOME}
      data-file-drop-target
      onclick={() => app.navigate(HOME)}
      ondragover={(e) => dragOver(e, HOME)}
      ondragleave={(e) => dragLeave(e, HOME)}
      ondrop={(e) => dropOn(e, HOME)}
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
      onclick={() => app.navigate(TRASH)}
      ondragover={(e) => dragOver(e, TRASH)}
      ondragleave={(e) => dragLeave(e, TRASH)}
      ondrop={(e) => dropOn(e, TRASH)}
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
      <div
        class="bookmarks"
        ondragover={onListDragOver}
        ondrop={(e) => insertAt !== null && dropInsert(e)}
        ondragleave={(e) => !(e.currentTarget as Node).contains(e.relatedTarget as Node) && (insertAt = null)}
      >
        {#each bookmarks as b, i (b.path)}
          {#if insertAt === i}<div class="insert"></div>{/if}
          <button
            class="row"
            class:selected={app.path === b.path && !app.results}
            class:drop={app.dropTarget === b.path}
            class:moving={movingBookmark === b.path}
            title={b.path.replace(/^\/me/, "")}
            data-path={b.path}
            data-file-drop-target
            draggable={!app.mobile}
            onclick={() => app.navigate(b.path)}
            oncontextmenu={(e) => bookmarkMenu(e, b.path)}
            ondragstart={(e) => onBookmarkDragStart(e, b.path)}
            ondragend={onBookmarkDragEnd}
            ondragover={(e) => onRowDragOver(e, b.path)}
            ondragleave={(e) => dragLeave(e, b.path)}
            ondrop={(e) => onRowDrop(e, b.path)}
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
        <span class="dim acct-host">{host}</span>
      </span>
      <Icon name="emblem-system" />
    </span>
    <span class="usage" title="{formatSize(used)} used">
      {#if limit > 0}
        <span class="bar"><span style:width="{Math.min(100, (used / limit) * 100)}%"></span></span>
        <span class="dim">{formatSize(used)} of {formatSize(limit)} used</span>
      {:else}
        <span class="dim">{formatSize(used)} used</span>
      {/if}
    </span>
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
    margin-bottom: 10px;
  }
  .acct-row > :global(.icon) {
    margin-left: auto;
    opacity: 0.6;
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
  .usage > span {
    display: block;
  }
  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--btn-bg);
    overflow: hidden;
    margin-bottom: 6px;
  }
  .bar > span {
    height: 100%;
    border-radius: 3px;
    background: var(--accent);
  }
</style>
