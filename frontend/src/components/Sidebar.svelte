<script lang="ts">
  import { app, HOME, STARRED, TRASH } from "../lib/store.svelte";
  import { formatSize } from "../lib/format";
  import { dragLeave, dragOver, dropOn } from "../lib/dnd";
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

  function bookmarkMenu(e: MouseEvent, path: string) {
    e.preventDefault();
    app.openMenu(e.clientX, e.clientY, [
      { label: "Open", run: () => app.navigate(path) },
      { sep: true },
      { label: "Rename…", run: () => renameBookmark(path) },
      { label: "Remove", run: () => app.toggleBookmark(path) },
    ]);
  }

  async function renameBookmark(path: string) {
    const b = app.prefs.bookmarks?.find((x) => x.path === path);
    if (!b) return;
    const v = await app.prompt({ title: "Rename Bookmark", label: "Name", value: b.name, confirm: "Rename" });
    if (v) {
      b.name = v;
      app.savePrefs();
    }
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

    {#if app.prefs.bookmarks?.length}
      <div class="sep"></div>
      {#each app.prefs.bookmarks as b (b.path)}
        <button
          class="row"
          class:selected={app.path === b.path && !app.results}
          class:drop={app.dropTarget === b.path}
          title={b.path.replace(/^\/me/, "")}
          data-path={b.path}
          data-file-drop-target
          onclick={() => app.navigate(b.path)}
          oncontextmenu={(e) => bookmarkMenu(e, b.path)}
          ondragover={(e) => dragOver(e, b.path)}
          ondragleave={(e) => dragLeave(e, b.path)}
          ondrop={(e) => dropOn(e, b.path)}
        >
          <Icon name="folder" /><span>{b.name}</span>
        </button>
      {/each}
    {/if}
  </div>

  <div class="account">
    <div class="acct-row">
      <Icon name="network-server" size={16} />
      <div class="acct-text">
        <div class="acct-name">{user?.username ?? "Nova"}</div>
        <div class="dim acct-host">{host}</div>
      </div>
    </div>
    <div class="usage" title="{formatSize(used)} used">
      {#if limit > 0}
        <div class="bar"><div style:width="{Math.min(100, (used / limit) * 100)}%"></div></div>
        <div class="dim">{formatSize(used)} of {formatSize(limit)} used</div>
      {:else}
        <div class="dim">{formatSize(used)} used</div>
      {/if}
    </div>
  </div>
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
  .sep {
    height: 1px;
    margin: 6px 4px;
    background: var(--border);
    opacity: 0.6;
  }
  .account {
    padding: 12px 16px 14px;
    font-size: 0.87em;
  }
  .acct-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }
  .acct-text {
    min-width: 0;
  }
  .acct-name {
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--btn-bg);
    overflow: hidden;
    margin-bottom: 6px;
  }
  .bar > div {
    height: 100%;
    border-radius: 3px;
    background: var(--accent);
  }
</style>
