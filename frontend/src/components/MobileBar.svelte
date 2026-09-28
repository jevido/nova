<script lang="ts">
  import { app, HOME, RECENT, SHARED, STARRED, TRASH, isVirtual } from "../lib/store.svelte";
  import { itemMenu } from "../lib/menus";
  import { pluralize } from "../lib/format";
  import Icon from "./Icon.svelte";

  // The phone's bottom bar: places to switch between, or actions for the
  // selected items while selecting. The + button and paste bar float above it.
  const sel = $derived(app.selection);
  const selecting = $derived(app.selected.size > 0);
  const mixed = $derived(app.results !== null || isVirtual(app.path));
  const tab = $derived(app.inTrash ? TRASH : isVirtual(app.path) ? app.path : HOME);
  const tabs: [string, string, string][] = [
    [HOME, "Home", "user-home"],
    [RECENT, "Recent", "document-open-recent"],
    [STARRED, "Starred", "starred"],
    [SHARED, "Shared", "send-to"],
    [TRASH, "Trash", "user-trash"],
  ];

  function go(path: string) {
    app.drawerOpen = false;
    if (app.searchOpen) app.closeSearch(false);
    app.navigate(path);
  }

  function addMenu(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    app.openMenu(r.left, r.top, [
      { label: "Take Photo", run: () => app.takePhoto() },
      { label: "Upload Files…", run: () => app.pickUpload(false) },
      { label: "New Folder…", run: () => app.newFolder() },
    ]);
  }

  function moreMenu(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    app.openMenu(r.left, r.top, itemMenu(sel, mixed));
  }
</script>

<footer class="mobilebar">
  {#if app.clipboard && app.canWrite && !selecting}
    <div class="pastebar">
      <span class="grow">{pluralize(app.clipboard.paths.length, "item", "items")} to {app.clipboard.mode === "cut" ? "move" : "copy"}</span>
      <button class="btn flat" onclick={() => app.setClipboard(null)}>Cancel</button>
      <button class="btn suggested" onclick={() => app.paste()}>{app.clipboard.mode === "cut" ? "Move Here" : "Paste Here"}</button>
    </div>
  {:else if app.canWrite && !selecting && !app.searchOpen}
    <button class="fab" title="New" onclick={addMenu}><Icon name="list-add" size={28} /></button>
  {/if}

  {#if selecting}
    <nav class="tabs">
      {#if app.inTrash}
        <button class="tab" onclick={() => app.restore(sel)}><span class="pill"><Icon name="edit-undo" size={24} /></span><span>Restore</span></button>
        <button class="tab" onclick={() => app.deleteForever(sel)}><span class="pill"><Icon name="edit-delete" size={24} /></span><span>Delete</span></button>
      {:else}
        <button class="tab" onclick={() => app.download(sel)}><span class="pill"><Icon name="folder-download" size={24} /></span><span>Download</span></button>
        <button class="tab" onclick={() => (app.copy(true), app.clearSelection())}><span class="pill"><Icon name="edit-cut" size={24} /></span><span>Move</span></button>
        <button class="tab" onclick={() => (app.copy(false), app.clearSelection())}><span class="pill"><Icon name="edit-copy" size={24} /></span><span>Copy</span></button>
        <button class="tab" onclick={() => app.trash(sel)}><span class="pill"><Icon name="user-trash" size={24} /></span><span>Trash</span></button>
      {/if}
      <button class="tab" onclick={moreMenu}><span class="pill"><Icon name="view-more" size={24} /></span><span>More</span></button>
    </nav>
  {:else}
    <nav class="tabs">
      {#each tabs as [path, label, icon] (path)}
        <button class="tab" class:on={tab === path} onclick={() => go(path)}>
          <span class="pill"><Icon name={path === STARRED && tab !== STARRED ? "non-starred" : icon} size={22} /></span><span>{label}</span>
        </button>
      {/each}
    </nav>
  {/if}
</footer>

<style>
  .mobilebar {
    position: relative;
    z-index: 22;
    background: color-mix(in srgb, var(--fg) 3%, var(--view-bg));
    box-shadow: 0 -1px var(--border);
    padding-bottom: env(safe-area-inset-bottom, 0);
  }
  .tabs {
    display: flex;
    height: 76px;
  }
  .tab {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 5px;
    border: 0;
    background: none;
    color: var(--fg-dim);
    font: inherit;
    font-size: 12px;
    font-weight: 600;
  }
  .pill {
    display: flex;
    padding: 4px 16px;
    border-radius: 9999px;
    transition: background 120ms ease-out;
  }
  .tab.on {
    color: var(--accent-text);
    font-weight: 600;
  }
  .tab:active .pill {
    background: var(--hover);
  }
  .fab {
    position: absolute;
    right: 20px;
    bottom: calc(100% + 20px);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    border: 0;
    border-radius: 20px;
    background: var(--accent);
    color: var(--accent-fg);
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.25);
  }
  .fab:active {
    filter: brightness(1.1);
  }
  .pastebar {
    position: absolute;
    left: 12px;
    right: 12px;
    bottom: calc(100% + 12px);
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 6px 6px 16px;
    border-radius: 14px;
    background: var(--popover-bg);
    box-shadow: var(--menu-shadow);
  }
  .grow {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
