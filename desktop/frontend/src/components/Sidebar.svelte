<script lang="ts">
  import { app, HOME, STARRED, TRASH } from "../lib/store.svelte";
  import { formatSize } from "../lib/format";
  import { dragLeave, dragOver, dropOn } from "../lib/dnd";
  import Icon from "./Icon.svelte";

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
      <Icon name="network-server" />
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
    border-right: 1px solid var(--border);
  }
  .rows {
    flex: 1;
    overflow-y: auto;
    padding: 6px 0;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 36px;
    padding: 0 14px;
    border: 0;
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
  .row :global(.icon) {
    opacity: 0.8;
  }
  .row:hover {
    background: var(--row-hover);
  }
  .row.selected {
    background: var(--accent-dim);
    color: #fff;
  }
  :global(:root[data-theme="light"]) .row.selected {
    background: var(--accent);
  }
  .row.selected :global(.icon) {
    opacity: 1;
  }
  .row.drop,
  .row:global(.file-drop-target-active) {
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .row:focus-visible {
    box-shadow: inset 0 0 0 1px var(--focus);
  }
  .sep {
    height: 1px;
    margin: 6px 0;
    background: var(--border);
    opacity: 0.6;
  }
  .account {
    padding: 10px 14px 12px;
    border-top: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
    font-size: 13px;
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
  .acct-host {
    font-size: 12px;
  }
  .bar {
    height: 4px;
    border-radius: 3px;
    background: color-mix(in srgb, var(--fg) 15%, transparent);
    overflow: hidden;
    margin-bottom: 4px;
  }
  .bar > div {
    height: 100%;
    background: var(--accent);
  }
</style>
