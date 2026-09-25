<script lang="ts">
  import { Window } from "@wailsio/runtime";
  import { app, HOME, STARRED, TRASH, ZOOM_SIZES, displayName } from "../lib/store.svelte";
  import { formatDuration, formatRate, formatSize } from "../lib/format";
  import Icon from "./Icon.svelte";
  import Popover from "./Popover.svelte";
  import { dragOver, dropOn, dragLeave } from "../lib/dnd";

  let viewBtn = $state<HTMLButtonElement>();
  let menuBtn = $state<HTMLButtonElement>();
  let opsBtn = $state<HTMLButtonElement>();
  let viewOpen = $state(false);
  let menuOpen = $state(false);
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

  let { showDrawerButton = false }: { showDrawerButton?: boolean } = $props();

  const active = $derived(app.activeTransfers);
  const opsProgress = $derived.by(() => {
    const total = active.reduce((a, t) => a + t.totalBytes, 0);
    const done = active.reduce((a, t) => a + t.doneBytes, 0);
    return total > 0 ? done / total : 0;
  });

  function setZoom(d: number) {
    app.prefs.zoom = Math.max(0, Math.min(ZOOM_SIZES.length - 1, app.prefs.zoom + d));
    app.savePrefs();
  }

  function setSort(by: string, desc: boolean) {
    app.prefs.sortBy = by;
    app.prefs.sortDesc = desc;
    app.savePrefs();
  }

</script>

<!-- Double-click to maximise is handled by the Wails runtime for drag regions. -->
<header class="headerbar" role="toolbar" tabindex="-1">
  <div class="start">
    {#if showDrawerButton}
      <button class="btn image" title="Show sidebar" class:checked={app.drawerOpen} onclick={() => (app.drawerOpen = !app.drawerOpen)}>
        <Icon name="sidebar-show" />
      </button>
    {/if}
    <div class="linked">
      <button class="btn image" title="Go back (Alt+Left)" disabled={!app.history.length} onclick={() => app.back()}>
        <Icon name="go-previous" />
      </button>
      {#if !app.mobile}
        <button class="btn image" title="Go forward (Alt+Right)" disabled={!app.future.length} onclick={() => app.forward()}>
          <Icon name="go-next" />
        </button>
      {/if}
    </div>
  </div>

  <div class="center">
    {#if app.searchOpen}
      <div class="search entry-wrap">
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
        <div class="linked">
          {#each crumbs as c, i (c)}
            <button
              class="btn crumb"
              class:checked={c === app.path}
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
      </nav>
    {/if}
  </div>

  <div class="end">
    {#if app.transfers.length}
      <button class="btn image ops" bind:this={opsBtn} class:checked={opsOpen} title="Show operations" onclick={() => (opsOpen = !opsOpen)}>
        {#if active.length}
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" stroke-opacity="0.3" stroke-width="2" />
            <circle
              cx="8"
              cy="8"
              r="7"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-dasharray={2 * Math.PI * 7}
              stroke-dashoffset={2 * Math.PI * 7 * (1 - opsProgress)}
              transform="rotate(-90 8 8)"
            />
          </svg>
        {:else}
          <Icon name="object-select" />
        {/if}
      </button>
    {/if}
    <button
      class="btn image"
      class:checked={app.searchOpen}
      title="Search (Ctrl+F)"
      onclick={() => (app.searchOpen ? app.closeSearch() : app.openSearch())}
    >
      <Icon name="edit-find" />
    </button>
    <div class="linked">
      <button
        class="btn image"
        title={app.prefs.view === "grid" ? "Show list (Ctrl+1)" : "Show grid (Ctrl+2)"}
        onclick={() => {
          app.prefs.view = app.prefs.view === "grid" ? "list" : "grid";
          app.savePrefs();
        }}
      >
        <Icon name={app.prefs.view === "grid" ? "view-list" : "view-grid"} />
      </button>
      <button class="btn image narrow" bind:this={viewBtn} class:checked={viewOpen} title="View options" onclick={() => (viewOpen = !viewOpen)}>
        <Icon name="pan-down" />
      </button>
    </div>
    <button class="btn image badge-host" bind:this={menuBtn} class:checked={menuOpen} title="Menu (F10)" onclick={() => (menuOpen = !menuOpen)}>
      <Icon name="open-menu" />
      {#if app.update?.state === "ready" || app.update?.state === "manual"}<span class="badge" title="Update available"></span>{/if}
    </button>
    {#if !app.mobile}
    <div class="titlebuttons">
      <button class="btn image flat round" title="Minimise" onclick={() => Window.Minimise()}><Icon name="window-minimize" /></button>
      <button class="btn image flat round" title="Maximise" onclick={() => Window.ToggleMaximise()}><Icon name="window-maximize" /></button>
      <button class="btn image flat round close" title="Close" onclick={() => Window.Close()}><Icon name="window-close" /></button>
    </div>
    {/if}
  </div>
</header>

<Popover anchor={viewBtn} bind:open={viewOpen} align="end">
  <div class="viewopts">
    <div class="linked zoom">
      <button class="btn image" title="Zoom out (Ctrl+−)" disabled={app.prefs.zoom <= 0} onclick={() => setZoom(-1)}><Icon name="zoom-out" /></button>
      <button class="btn" title="Reset zoom (Ctrl+0)" onclick={() => ((app.prefs.zoom = 1), app.savePrefs())}>{Math.round((ZOOM_SIZES[app.prefs.zoom] / 64) * 100)}%</button>
      <button class="btn image" title="Zoom in (Ctrl++)" disabled={app.prefs.zoom >= ZOOM_SIZES.length - 1} onclick={() => setZoom(1)}><Icon name="zoom-in" /></button>
    </div>
    <div class="psep"></div>
    <div class="dim label">Sort</div>
    {#each [["name", false, "A–Z"], ["name", true, "Z–A"], ["modified", true, "Last Modified"], ["modified", false, "First Modified"], ["size", true, "Size"], ["type", false, "Type"]] as [by, desc, label] (label)}
      <button class="modelbutton" onclick={() => setSort(by as string, desc as boolean)}>
        <span class="radio" class:on={app.prefs.sortBy === by && app.prefs.sortDesc === desc}></span>{label}
      </button>
    {/each}
    <div class="psep"></div>
    <button class="modelbutton" onclick={() => ((app.prefs.foldersFirst = !app.prefs.foldersFirst), app.savePrefs())}>
      <span class="checkbox" class:on={app.prefs.foldersFirst}></span>Sort Folders Before Files
    </button>
    <button class="modelbutton" onclick={() => ((app.prefs.showHidden = !app.prefs.showHidden), app.savePrefs())}>
      <span class="checkbox" class:on={app.prefs.showHidden}></span>Show Hidden Files<span class="accel">Ctrl+H</span>
    </button>
    <div class="psep"></div>
    <button class="modelbutton" onclick={() => ((viewOpen = false), app.reload())}>Reload<span class="accel">F5</span></button>
  </div>
</Popover>

<Popover anchor={menuBtn} bind:open={menuOpen} align="end">
  <div class="appmenu" role="presentation" onclick={() => (menuOpen = false)}>
    {#if !app.mobile}
      <button class="modelbutton" onclick={() => app.newWindow()}>New Window<span class="accel">Ctrl+N</span></button>
      <div class="psep"></div>
    {/if}
    <button class="modelbutton" disabled={!app.canWrite} onclick={() => app.newFolder()}>New Folder…<span class="accel">Shift+Ctrl+N</span></button>
    <button class="modelbutton" disabled={!app.canWrite} onclick={() => app.pickUpload(false)}>Upload Files…<span class="accel">Ctrl+U</span></button>
    {#if !app.mobile}
      <button class="modelbutton" disabled={!app.canWrite} onclick={() => app.pickUpload(true)}>Upload Folder…</button>
    {/if}
    <div class="psep"></div>
    <button class="modelbutton" disabled={app.path === HOME || app.path === STARRED || app.inTrash} onclick={() => app.toggleBookmark()}>
      {app.isBookmarked() ? "Remove Bookmark" : "Bookmark this Location"}<span class="accel">Ctrl+D</span>
    </button>
    {#if !app.mobile}
      <button class="modelbutton" onclick={() => ((app.prefs.sidebarOpen = !app.prefs.sidebarOpen), app.savePrefs())}>
        <span class="checkbox" class:on={app.prefs.sidebarOpen}></span>Show Sidebar<span class="accel">F9</span>
      </button>
    {/if}
    <div class="psep"></div>
    <div class="dim label">Style</div>
    <div class="linked theme" role="presentation" onclick={(e) => e.stopPropagation()}>
      {#each [["system", "System"], ["light", "Light"], ["dark", "Dark"]] as [v, l] (v)}
        <button class="btn" class:checked={app.prefs.theme === v} onclick={() => ((app.prefs.theme = v), app.savePrefs())}>{l}</button>
      {/each}
    </div>
    <div class="psep"></div>
    {#if !app.mobile}
      <button class="modelbutton" onclick={() => (app.modal = { kind: "shortcuts" })}>Keyboard Shortcuts<span class="accel">Ctrl+?</span></button>
    {/if}
    {#if app.update?.state === "ready"}
      <button class="modelbutton" onclick={() => app.applyUpdate()}>
        <span class="dot"></span>{app.mobile ? "Install" : "Restart to Install"} Nova {app.update.latestVersion}
      </button>
    {:else if app.update?.state === "manual"}
      <button class="modelbutton" onclick={() => app.checkForUpdates()}>
        <span class="dot"></span>Nova {app.update.latestVersion} Available…
      </button>
    {:else}
      <button class="modelbutton" disabled={app.update?.state === "checking" || app.update?.state === "downloading"} onclick={() => app.checkForUpdates()}>
        {app.update?.state === "downloading" ? "Downloading Update…" : app.update?.state === "checking" ? "Checking for Updates…" : "Check for Updates"}
      </button>
    {/if}
    <button class="modelbutton" onclick={() => (app.modal = { kind: "about" })}>About Nova</button>
    <button class="modelbutton" onclick={() => app.signOut()}>Sign Out…</button>
  </div>
</Popover>

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
    min-height: 46px;
    padding: 6px;
    background: linear-gradient(to top, var(--header-bottom), var(--header-top));
    border-bottom: 1px solid var(--header-border);
    box-shadow: inset 0 1px rgba(255, 255, 255, 0.1);
    --wails-draggable: drag;
    outline: none;
  }
  :global(:root[data-theme="dark"]) .headerbar {
    box-shadow: inset 0 1px rgba(238, 238, 236, 0.07);
  }
  .start,
  .end {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: none;
  }
  .center {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: center;
  }
  .pathbar {
    flex: 1;
    min-width: 0;
    max-width: 720px;
    display: flex;
    overflow: hidden;
    align-self: stretch;
    align-items: center;
    --wails-draggable: no-drag;
  }
  .pathbar .linked {
    min-width: 0;
    max-width: 100%;
    overflow: hidden;
  }
  .crumb {
    flex: 0 1 auto;
    min-width: 34px;
    max-width: 220px;
    overflow: hidden;
  }
  .crumb span {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .crumb.checked {
    font-weight: bold;
  }
  /* Phones: keep only the parent and current folder in the path bar. */
  :global(:root[data-mobile="true"]) .crumb:not(:nth-last-child(-n + 2)) {
    display: none;
  }
  :global(:root[data-mobile="true"]) .headerbar {
    gap: 4px;
  }
  .crumb.drop {
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .location,
  .search {
    flex: 1;
    max-width: 720px;
  }
  .search {
    position: relative;
    display: flex;
    align-items: center;
  }
  .search > :global(.icon) {
    position: absolute;
    left: 9px;
    opacity: 0.6;
  }
  .search input {
    flex: 1;
    padding-left: 32px;
  }
  .spinner.inset {
    position: absolute;
    right: 9px;
  }
  .narrow {
    min-width: 24px;
    padding: 4px 5px;
  }
  .titlebuttons {
    display: flex;
    gap: 6px;
    margin-left: 6px;
    padding-left: 12px;
    border-left: 1px solid color-mix(in srgb, var(--header-border) 60%, transparent);
  }
  .round {
    min-width: 24px;
    min-height: 24px;
    width: 24px;
    height: 24px;
    padding: 0;
    border-radius: 50%;
  }
  .round :global(.icon) {
    width: 14px !important;
    height: 14px !important;
  }
  .titlebuttons .btn:hover {
    background: color-mix(in srgb, var(--fg) 12%, transparent);
    border-color: transparent;
  }
  .viewopts,
  .appmenu {
    display: flex;
    flex-direction: column;
    min-width: 220px;
  }
  .viewopts .zoom,
  .appmenu .theme {
    display: flex;
    margin: 4px;
  }
  .zoom .btn,
  .theme .btn {
    flex: 1;
  }
  .badge-host {
    position: relative;
  }
  .badge {
    position: absolute;
    top: 4px;
    right: 4px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 2px var(--header-bottom);
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    flex: none;
  }
  .label {
    padding: 2px 10px;
    font-size: 13px;
  }
  .modelbutton:disabled {
    opacity: 0.5;
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
    border-top: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
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
    font-size: 12px;
    margin: 2px 0 6px;
  }
  .progress {
    height: 4px;
    border-radius: 3px;
    background: color-mix(in srgb, var(--fg) 15%, transparent);
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
</style>
