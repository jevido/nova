<script lang="ts">
  import { Window } from "@wailsio/runtime";
  import { app, HOME, STARRED, TRASH, displayName } from "../lib/store.svelte";
  import { formatDuration, formatRate, formatSize } from "../lib/format";
  import Icon from "./Icon.svelte";
  import MainMenu from "./MainMenu.svelte";
  import Popover from "./Popover.svelte";
  import ViewControls from "./ViewControls.svelte";
  import { dragOver, dropOn, dragLeave } from "../lib/dnd";

  let opsBtn = $state<HTMLButtonElement>();
  let opsOpen = $state(false);
  let locInput = $state<HTMLInputElement>();
  let searchInput = $state<HTMLInputElement>();
  let locValue = $state("");

  const crumbs = $derived.by(() => {
    if (app.path === STARRED) return [STARRED];
    let list = (app.folder?.crumbs ?? []).map((c) => c.path);
    if (!list.length || list[list.length - 1] !== app.path) {
      // Build from the path while the listing loads.
      const parts = app.path.split("/").filter(Boolean);
      list = parts.map((_, i) => "/" + parts.slice(0, i + 1).join("/"));
    }
    // The trash is its own root, like trash:/// in Nautilus.
    if (app.path.startsWith(TRASH)) list = list.filter((c) => c.startsWith(TRASH));
    return list;
  });

  function relPath(p: string) {
    return p.replace(/^\/me/, "") || "/";
  }

  $effect(() => {
    if (app.editingLocation) {
      locValue = relPath(app.path) + (app.path === HOME ? "" : "/");
      queueMicrotask(() => {
        locInput?.focus();
        locInput?.select();
      });
    }
  });

  $effect(() => {
    if (app.searchOpen) queueMicrotask(() => searchInput?.focus());
  });

  function submitLocation() {
    let v = locValue.trim();
    if (v.startsWith("~")) v = v.slice(1);
    if (v.startsWith("/me/") || v === "/me") v = v.slice(3);
    const target = ("/me/" + v).replace(/\/+/g, "/").replace(/\/$/, "") || HOME;
    app.editingLocation = false;
    app.navigate(target);
  }

  // compact: narrow window or phone. Navigation and view buttons then live in
  // the bottom bar and the sidebar opens as an overlay, like Nautilus.
  let {
    compact = false,
    sidebarShown = true,
    onToggleSidebar,
  }: { compact?: boolean; sidebarShown?: boolean; onToggleSidebar: () => void } = $props();

  const active = $derived(app.activeTransfers);
  const opsProgress = $derived.by(() => {
    const total = active.reduce((a, t) => a + t.totalBytes, 0);
    const done = active.reduce((a, t) => a + t.doneBytes, 0);
    return total > 0 ? done / total : 0;
  });

  function folderMenu(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const here = app.folder?.crumbs?.[app.folder.crumbs.length - 1];
    app.openMenu(r.right, r.bottom + 6, [
      { label: "New Folder…", accel: "Shift+Ctrl+N", disabled: !app.canWrite, run: () => app.newFolder() },
      { label: "Upload Files…", accel: "Ctrl+U", disabled: !app.canWrite, run: () => app.pickUpload(false) },
      ...(app.mobile ? [] : [{ label: "Upload Folder…", disabled: !app.canWrite, run: () => app.pickUpload(true) }]),
      { sep: true },
      {
        label: app.isBookmarked() ? "Remove from Bookmarks" : "Add to Bookmarks",
        accel: "Ctrl+D",
        disabled: app.path === HOME || app.path === STARRED || app.inTrash,
        run: () => app.toggleBookmark(),
      },
      { label: "Copy Location", run: () => app.copyPath({ path: app.path } as never) },
      { label: "Reload", accel: "F5", run: () => app.reload() },
      { sep: true },
      { label: "Properties", disabled: !here || here.path !== app.path, run: () => here && (app.modal = { kind: "properties", entry: here }) },
    ]);
  }

</script>

<!-- Double-click to maximise is handled by the Wails runtime for drag regions. -->
<header class="headerbar" class:compact role="toolbar" tabindex="-1">
  <div class="start">
    {#if !sidebarShown}
      <button class="btn image flat" title="Show Sidebar" onclick={onToggleSidebar}>
        <Icon name="sidebar-show" />
      </button>
    {/if}
    {#if !compact}
      <button class="btn image flat" title="Back (Alt+Left)" disabled={!app.history.length} onclick={() => app.back()}>
        <Icon name="go-previous" />
      </button>
      <button class="btn image flat" title="Forward (Alt+Right)" disabled={!app.future.length} onclick={() => app.forward()}>
        <Icon name="go-next" />
      </button>
    {/if}
  </div>

  <div class="center">
    {#if app.searchOpen}
      <div class="search">
        <Icon name="system-search" />
        <input
          bind:this={searchInput}
          class="entry"
          placeholder="Search {displayName(app.path)}"
          value={app.query}
          oninput={(e) => app.setQuery(e.currentTarget.value)}
          onkeydown={(e) => {
            if (e.key === "Escape") {
              e.stopPropagation();
              app.closeSearch();
            } else if (e.key === "ArrowDown") {
              e.preventDefault();
              (document.querySelector(".view") as HTMLElement)?.focus();
              app.moveCursor(1, false);
            }
          }}
        />
        {#if app.searching}<span class="spinner inset"></span>{/if}
      </div>
    {:else if app.editingLocation}
      <input
        bind:this={locInput}
        class="entry location"
        bind:value={locValue}
        spellcheck="false"
        onkeydown={(e) => {
          if (e.key === "Enter") submitLocation();
          else if (e.key === "Escape") {
            e.stopPropagation();
            app.editingLocation = false;
          }
        }}
        onblur={() => (app.editingLocation = false)}
      />
    {:else}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <nav class="pathbar" onclick={(e) => e.target === e.currentTarget && (app.editingLocation = true)}>
        <div class="crumbs" role="presentation" onclick={(e) => e.target === e.currentTarget && (app.editingLocation = true)}>
          {#each crumbs as c, i (c)}
            {#if i > 0}<span class="slash">/</span>{/if}
            <button
              class="crumb"
              class:current={c === app.path}
              data-path={c}
              title={relPath(c)}
              onclick={() => (c === app.path ? (app.editingLocation = true) : app.navigate(c))}
              ondragover={(e) => dragOver(e, c)}
              ondragleave={(e) => dragLeave(e, c)}
              ondrop={(e) => dropOn(e, c)}
              oncontextmenu={(e) => {
                e.preventDefault();
                app.openMenu(e.clientX, e.clientY, [
                  { label: "Open", run: () => app.navigate(c) },
                  { label: app.isBookmarked(c) ? "Remove from Bookmarks" : "Add to Bookmarks", disabled: c === HOME || c === TRASH || c === STARRED, run: () => app.toggleBookmark(c) },
                  { label: "Copy Location", run: () => app.copyPath({ path: c } as never) },
                ]);
              }}
              class:drop={app.dropTarget === c}
            >
              {#if i === 0}<Icon name={c === TRASH ? "user-trash" : c === STARRED ? "starred" : "user-home"} />{/if}
              {#if i === 0 && c === HOME}<span>Home</span>{:else if i > 0 || c !== HOME}<span>{displayName(c)}</span>{/if}
            </button>
          {/each}
        </div>
        {#if !app.path.startsWith(TRASH) && app.path !== STARRED}
          <button class="btn image flat more" title="Folder menu" onclick={folderMenu}><Icon name="view-more" /></button>
        {/if}
      </nav>
    {/if}
  </div>

  <div class="end">
    {#if app.transfers.length}
      <button class="btn image flat ops" bind:this={opsBtn} class:checked={opsOpen} title="Show operations" onclick={() => (opsOpen = !opsOpen)}>
        {#if active.length}
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <circle cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" stroke-opacity="0.25" stroke-width="3" />
            <circle
              cx="8"
              cy="8"
              r="6.5"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-dasharray={2 * Math.PI * 6.5}
              stroke-dashoffset={2 * Math.PI * 6.5 * (1 - opsProgress)}
              transform="rotate(-90 8 8)"
            />
          </svg>
        {:else}
          <Icon name="object-select" />
        {/if}
      </button>
    {/if}
    <button
      class="btn image flat"
      class:checked={app.searchOpen}
      title="Search (Ctrl+F)"
      onclick={() => (app.searchOpen ? app.closeSearch() : app.openSearch())}
    >
      <Icon name="edit-find" />
    </button>
    {#if !compact}<ViewControls />{/if}
    {#if !sidebarShown && !compact}<MainMenu />{/if}
    {#if !app.mobile}
      <button class="btn image round close" title="Close" onclick={() => Window.Close()}><Icon name="window-close" /></button>
    {/if}
  </div>
</header>

<Popover anchor={opsBtn} bind:open={opsOpen} align="end">
  <div class="ops-list">
    {#each [...app.transfers].reverse() as t (t.id)}
      {@const frac = t.totalBytes ? t.doneBytes / t.totalBytes : t.state === "done" ? 1 : 0}
      <div class="op">
        <div class="op-text">
          <div class="op-title">
            {#if t.kind === "upload"}Uploading{:else if t.kind === "copy"}Copying{:else if t.kind === "open"}Opening{:else}Downloading{/if}
            “{t.title}”
          </div>
          <div class="dim op-sub">
            {#if t.state === "running"}
              {formatSize(t.doneBytes)} of {formatSize(t.totalBytes)}
              {#if t.rateBps > 0}— {formatDuration((t.totalBytes - t.doneBytes) / t.rateBps)} left ({formatRate(t.rateBps)}){/if}
            {:else if t.state === "queued"}Waiting…
            {:else if t.state === "done"}Finished{t.files > 1 ? ` — ${t.files} files` : ""}
            {:else if t.state === "failed"}Failed: {t.error}
            {:else}Cancelled{/if}
          </div>
          <div class="progress" class:failed={t.state === "failed"}><div style:width="{Math.min(100, frac * 100)}%"></div></div>
        </div>
        {#if t.state === "running" || t.state === "queued"}
          <button class="btn image round flat" title="Cancel" onclick={() => app.cancelTransfer(t.id)}><Icon name="process-stop" /></button>
        {/if}
      </div>
    {/each}
    {#if app.transfers.some((t) => t.state !== "running" && t.state !== "queued")}
      <div class="psep"></div>
      <button class="modelbutton" onclick={() => app.clearTransfers()}>Clear Finished</button>
    {/if}
  </div>
</Popover>

<style>
  .headerbar {
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 47px;
    padding: 6px 7px;
    background: var(--header-bg);
    --wails-draggable: drag;
    outline: none;
  }
  .start,
  .end {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: none;
  }
  .end {
    gap: 4px;
  }
  .center {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: center;
    padding: 0 4px;
  }
  /* Nautilus 45+ path bar: one entry-like box with slash-separated segments */
  .pathbar {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    height: 34px;
    padding-left: 3px;
    border-radius: 6px;
    background: var(--btn-bg);
    overflow: hidden;
    --wails-draggable: no-drag;
  }
  .crumbs {
    flex: 1;
    min-width: 0;
    height: 100%;
    display: flex;
    align-items: center;
    overflow: hidden;
  }
  .crumb {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    flex: 0 1 auto;
    min-width: 0;
    max-width: 220px;
    height: 28px;
    padding: 0 7px;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--fg-dim);
    outline: none;
    white-space: nowrap;
  }
  .crumb span {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .crumb:hover {
    background: var(--hover);
    color: var(--fg);
  }
  .crumb.current {
    color: var(--fg);
    font-weight: bold;
    flex-shrink: 0;
  }
  .crumb:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .crumb.drop {
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .slash {
    flex: none;
    color: var(--fg-dim);
    opacity: 0.6;
    padding: 0 1px;
  }
  .more {
    flex: none;
    min-height: 28px;
    min-width: 28px;
    height: 28px;
    margin: 0 3px;
    padding: 0;
    border-radius: 4px;
  }
  /* Phones: keep only the parent and current folder in the path bar. */
  :global(:root[data-mobile="true"]) .crumb:not(:nth-last-child(-n + 3)),
  :global(:root[data-mobile="true"]) .slash:not(:nth-last-child(-n + 3)) {
    display: none;
  }
  :global(:root[data-mobile="true"]) .headerbar {
    gap: 4px;
  }
  .location,
  .search {
    flex: 1;
  }
  .search {
    position: relative;
    display: flex;
    align-items: center;
  }
  .search > :global(.icon) {
    position: absolute;
    left: 10px;
    opacity: 0.6;
  }
  .search input {
    flex: 1;
    padding-left: 34px;
  }
  .spinner.inset {
    position: absolute;
    right: 9px;
  }
  .close {
    margin-left: 6px;
  }
  .close :global(.icon) {
    width: 16px !important;
    height: 16px !important;
  }
  .ops-list {
    width: 380px;
    max-height: 60vh;
    overflow: auto;
    padding: 4px;
  }
  .op {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 6px;
  }
  .op + .op {
    border-top: 1px solid var(--border);
  }
  .op-text {
    flex: 1;
    min-width: 0;
  }
  .op-title,
  .op-sub {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .op-sub {
    font-size: 0.87em;
    margin: 2px 0 6px;
  }
  .progress {
    height: 4px;
    border-radius: 3px;
    background: var(--btn-bg);
    overflow: hidden;
  }
  .progress > div {
    height: 100%;
    background: var(--accent);
    transition: width 150ms linear;
  }
  .progress.failed > div {
    background: var(--destructive);
  }
  .op :global(.round) {
    background: none;
  }
  .op :global(.round:hover) {
    background: var(--hover);
  }
</style>
