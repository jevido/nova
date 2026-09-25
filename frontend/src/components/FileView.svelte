<script lang="ts">
  import { untrack } from "svelte";
  import type { Entry } from "../../bindings/nova/services/models";
  import { app, displayName, HOME, STARRED, TRASH, ZOOM_SIZES, LIST_ZOOM_SIZES, parentOf, type MenuItem } from "../lib/store.svelte";
  import { formatDate, formatSize, pluralize, stemLength } from "../lib/format";
  import { pressItems } from "../lib/dnd";
  import { itemMenu } from "../lib/menus";
  import FileIcon from "./FileIcon.svelte";
  import Icon from "./Icon.svelte";

  let view = $state<HTMLDivElement>();
  let band = $state<{ x0: number; y0: number; x1: number; y1: number } | null>(null);
  let bandBase: Set<string> = new Set();
  let bandRects: { path: string; r: DOMRect }[] = [];
  let pressed: { path: string; x: number; y: number; wasSelected: boolean } | null = null;
  let showSpinner = $state(false);

  const grid = $derived(app.prefs.view !== "list");
  const iconSize = $derived(ZOOM_SIZES[app.prefs.zoom] ?? 64);
  const rowIcon = $derived(LIST_ZOOM_SIZES[app.prefs.zoom] ?? 24);
  const entries = $derived(app.entries);
  // Search results and Starred both mix folders, so they show a Location column.
  const isSearch = $derived(app.results !== null || app.path === STARRED);

  $effect(() => {
    if (!app.loading) {
      showSpinner = false;
      return;
    }
    const t = setTimeout(() => (showSpinner = true), 300);
    return () => clearTimeout(t);
  });

  // Keep keyboard navigation working after changing folders, like GtkTreeView.
  $effect(() => {
    void app.path;
    const el = document.activeElement as HTMLElement | null;
    if (!el || el === document.body || el.closest(".sidebar, .pathbar")) view?.focus();
  });

  // ---------- selection with the mouse ----------

  function itemDown(e: MouseEvent, entry: Entry) {
    // Touch: tap activates, long-press selects (handled in click/contextmenu).
    if (app.mobile) return;
    if (e.button === 2) {
      if (!app.selected.has(entry.path)) app.selectPaths([entry.path]);
      return;
    }
    if (e.button === 1) {
      // Middle click opens in a new window (on auxclick); no autoscroll or paste.
      e.preventDefault();
      return;
    }
    if (e.button !== 0) return;
    view?.focus();
    const wasSelected = app.selected.has(entry.path);
    pressed = { path: entry.path, x: e.clientX, y: e.clientY, wasSelected };
    if (e.shiftKey || e.ctrlKey || e.metaKey) {
      app.clickSelect(entry.path, e);
    } else if (!wasSelected) {
      app.selectPaths([entry.path]);
    }
    app.cursor = entry.path;
    // Keep the webview from starting a text selection; a drag starts on move.
    e.preventDefault();
    if (app.selected.has(entry.path)) pressItems(e);
  }

  function itemClick(e: MouseEvent, entry: Entry) {
    if (app.mobile) {
      // Like Android's Files: once something is selected, taps toggle selection.
      if (app.selected.size) app.clickSelect(entry.path, { ctrlKey: true, shiftKey: false });
      else app.open(entry);
      return;
    }
    // Clicking one item of a multi-selection selects just that item (GTK).
    if (pressed?.path === entry.path && pressed.wasSelected && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
      if (app.selected.size > 1) app.selectPaths([entry.path]);
    }
    pressed = null;
  }

  function bgDown(e: MouseEvent) {
    if (app.mobile) return;
    if (e.button !== 0 || (e.target as HTMLElement).closest(".item, .row, .colhead, .rename")) return;
    view?.focus();
    const additive = e.ctrlKey || e.shiftKey || e.metaKey;
    if (!additive) app.clearSelection();
    bandBase = new Set(additive ? app.selected : []);
    bandRects = [...(view?.querySelectorAll<HTMLElement>("[data-path].sel") ?? [])].map((el) => ({
      path: el.dataset.path!,
      r: el.getBoundingClientRect(),
    }));
    band = { x0: e.clientX, y0: e.clientY, x1: e.clientX, y1: e.clientY };
    e.preventDefault();
  }

  function onMove(e: MouseEvent) {
    if (!band) return;
    band = { ...band, x1: e.clientX, y1: e.clientY };
    const l = Math.min(band.x0, band.x1),
      r = Math.max(band.x0, band.x1),
      t = Math.min(band.y0, band.y1),
      b = Math.max(band.y0, band.y1);
    const hit = new Set(bandBase);
    for (const { path, r: rect } of bandRects) {
      if (rect.right >= l && rect.left <= r && rect.bottom >= t && rect.top <= b) hit.add(path);
    }
    app.selected.clear();
    for (const p of hit) app.selected.add(p);
    // Autoscroll near edges.
    if (view) {
      const vr = view.getBoundingClientRect();
      if (e.clientY > vr.bottom - 20) view.scrollTop += 12;
      else if (e.clientY < vr.top + 20) view.scrollTop -= 12;
    }
  }

  function onUp() {
    band = null;
  }

  const bandStyle = $derived.by(() => {
    if (!band || !view) return "";
    const vr = view.getBoundingClientRect();
    const l = Math.max(Math.min(band.x0, band.x1), vr.left);
    const t = Math.max(Math.min(band.y0, band.y1), vr.top);
    const r = Math.min(Math.max(band.x0, band.x1), vr.right);
    const b = Math.min(Math.max(band.y0, band.y1), vr.bottom);
    return `left:${l}px;top:${t}px;width:${Math.max(0, r - l)}px;height:${Math.max(0, b - t)}px`;
  });

  // ---------- keyboard ----------

  function gridColumns(): number {
    const items = view?.querySelectorAll<HTMLElement>(".item");
    if (!items || items.length < 2) return 1;
    const top = items[0].offsetTop;
    let n = 0;
    for (const it of items) {
      if (it.offsetTop !== top) break;
      n++;
    }
    return Math.max(1, n);
  }

  function onKey(e: KeyboardEvent) {
    if (app.renaming || app.modal || app.menu) return;
    const shift = e.shiftKey;
    const cols = grid ? gridColumns() : 1;
    const page = grid ? cols * 4 : Math.max(1, Math.floor((view?.clientHeight ?? 400) / (rowIcon + 12)));
    switch (e.key) {
      case "ArrowRight":
        if (!grid || e.altKey) return;
        app.moveCursor(1, shift);
        break;
      case "ArrowLeft":
        if (!grid || e.altKey) return;
        app.moveCursor(-1, shift);
        break;
      case "ArrowDown":
        if (e.altKey) {
          app.openSelection();
          break;
        }
        app.moveCursor(cols, shift);
        break;
      case "ArrowUp":
        if (e.altKey) return;
        app.moveCursor(-cols, shift);
        break;
      case "PageDown":
        app.moveCursor(page, shift);
        break;
      case "PageUp":
        app.moveCursor(-page, shift);
        break;
      case "Home":
        if (e.altKey) return;
        app.moveCursor(0, shift, "start");
        break;
      case "End":
        app.moveCursor(0, shift, "end");
        break;
      case "Enter":
        if (e.altKey) {
          if (app.selection[0]) app.modal = { kind: "properties", entry: app.selection[0] };
        } else app.openSelection();
        break;
      case " ":
        if (e.ctrlKey && app.cursor) {
          app.clickSelect(app.cursor, { ctrlKey: true, shiftKey: false });
        } else app.preview();
        break;
      case "Escape":
        if (app.searchOpen) app.closeSearch();
        else app.clearSelection();
        break;
      default:
        if (e.key.length === 1 && !e.ctrlKey && !e.altKey && !e.metaKey) {
          // Nautilus starts a search when typing; type-ahead is faster for jumping.
          app.typeAhead(e.key);
          break;
        }
        return;
    }
    e.preventDefault();
  }

  // ---------- context menus ----------

  function openItemMenu(e: MouseEvent, entry: Entry) {
    e.preventDefault();
    e.stopPropagation();
    if (app.mobile) {
      // Long-press selects, like Android's Files; the bottom bar has the actions.
      if (!app.selected.has(entry.path)) {
        app.selected.add(entry.path);
        navigator.vibrate?.(12);
      }
      return;
    }
    if (!app.selected.has(entry.path)) app.selectPaths([entry.path]);
    app.openMenu(e.clientX, e.clientY, itemMenu(app.selection, isSearch));
  }

  /** ⋮ on a phone row: the menu for just that item. */
  function rowMenu(e: MouseEvent, entry: Entry) {
    e.stopPropagation();
    app.selectPaths([entry.path]);
    app.openMenu(e.clientX, e.clientY, itemMenu([entry], isSearch));
  }

  function subtitle(e: Entry): string {
    const when = formatDate(app.inTrash && e.deletedAt ? e.deletedAt : e.modified);
    const where = isSearch ? (parentOf(e.path).replace(/^\/me/, "") || "/") + " · " : "";
    return where + (e.isDir ? when : `${formatSize(e.size)} · ${when}`);
  }

  function bgMenu(e: MouseEvent) {
    e.preventDefault();
    if (app.mobile) return;
    if ((e.target as HTMLElement).closest("[data-path].sel")) return;
    app.clearSelection();
    if (app.inTrash) {
      app.openMenu(e.clientX, e.clientY, [
        { label: "Empty Trash", disabled: !app.entries.length, run: () => app.emptyTrash() },
        { sep: true },
        { label: "Select All", accel: "Ctrl+A", run: () => app.selectAll() },
        { label: "Reload", accel: "F5", run: () => app.reload() },
      ]);
      return;
    }
    app.openMenu(e.clientX, e.clientY, [
      { label: "New Folder…", accel: "Shift+Ctrl+N", disabled: !app.canWrite, run: () => app.newFolder() },
      { label: "Upload Files…", accel: "Ctrl+U", disabled: !app.canWrite, run: () => app.pickUpload(false) },
      ...(app.mobile ? [] : [{ label: "Upload Folder…", disabled: !app.canWrite, run: () => app.pickUpload(true) }]),
      { sep: true },
      { label: "Paste", accel: "Ctrl+V", disabled: !app.clipboard || !app.canWrite, run: () => app.paste() },
      { label: "Select All", accel: "Ctrl+A", run: () => app.selectAll() },
      { sep: true },
      { label: "Show Hidden Files", accel: "Ctrl+H", checked: app.prefs.showHidden, run: () => ((app.prefs.showHidden = !app.prefs.showHidden), app.savePrefs()) },
      { label: "Reload", accel: "F5", run: () => app.reload() },
      { sep: true },
      { label: "Properties", run: () => app.folder?.crumbs?.length && (app.modal = { kind: "properties", entry: app.folder.crumbs[app.folder.crumbs.length - 1] }) },
    ]);
  }

  // ---------- middle click: open folders in a new window ----------

  function itemAux(e: MouseEvent, entry: Entry) {
    if (e.button !== 1 || app.mobile) return;
    e.preventDefault();
    if (entry.isDir && !app.inTrash) app.newWindow(entry.path);
    else if (!entry.isDir) app.open(entry);
  }

  // ---------- rename popover ----------

  let renameInput = $state<HTMLInputElement>();
  let renameValue = $state("");
  let renamePos = $state({ x: 0, y: 0, w: 0 });
  const renameEntry = $derived(app.renaming ? entries.find((x) => x.path === app.renaming) : undefined);
  const renameError = $derived.by(() => {
    const v = renameValue.trim();
    if (!renameEntry) return "";
    if (v.includes("/")) return "File names cannot contain “/”.";
    if (v === "." || v === "..") return `A file cannot be called “${v}”.`;
    if (v !== renameEntry.name && entries.some((x) => x.name === v)) return `A ${renameEntry.isDir ? "folder" : "file"} with that name already exists.`;
    if (v.startsWith(".") && !renameEntry.name.startsWith(".")) return "Files with “.” at the beginning of their name are hidden.";
    return "";
  });

  $effect(() => {
    // Only re-run when a rename starts, not when a background reload
    // replaces the entry objects (that would wipe what the user typed).
    if (!app.renaming) return;
    const e = untrack(() => renameEntry);
    if (!e) return;
    renameValue = e.name;
    queueMicrotask(() => {
      const el = view?.querySelector<HTMLElement>(`[data-path="${CSS.escape(e.path)}"]`);
      if (!el) return;
      el.scrollIntoView({ block: "nearest" });
      // Point at the name, not the whole row, in list view.
      const r = (el.querySelector(".name, .label") ?? el).getBoundingClientRect();
      const w = 300;
      renamePos = { x: Math.max(8, Math.min(innerWidth - w - 8, r.left + r.width / 2 - w / 2)), y: r.bottom + 8, w };
      if (renamePos.y > innerHeight - 110) renamePos.y = r.top - 100;
      renameInput?.focus();
      renameInput?.setSelectionRange(0, stemLength(e.name, e.isDir));
    });
  });

  function commitRename() {
    if (!renameEntry || (renameError && !renameError.startsWith("Files with"))) return;
    app.rename(renameEntry.path, renameValue.trim());
  }

  // ---------- status ----------

  const status = $derived.by(() => {
    const sel = app.selection;
    if (!sel.length) return "";
    if (sel.length === 1) {
      const s = sel[0];
      return s.isDir ? `“${s.name}” selected` : `“${s.name}” selected (${formatSize(s.size)})`;
    }
    const folders = sel.filter((s) => s.isDir).length;
    const files = sel.length - folders;
    const bytes = sel.reduce((a, s) => a + (s.isDir ? 0 : s.size), 0);
    const parts = [];
    if (folders) parts.push(`${pluralize(folders, "folder", "folders")} selected`);
    if (files) parts.push(`${pluralize(files, "item", "items")} selected (${formatSize(bytes)})`);
    return parts.join(", ");
  });

  function typeLabel(e: Entry): string {
    if (e.isDir) return "Folder";
    const m = (e.mime || "").split(";")[0];
    return m || "Unknown";
  }
</script>

<svelte:window onmousemove={onMove} onmouseup={onUp} />

<div class="viewport">
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div
    class="view"
    class:grid
    class:list={!grid}
    tabindex="0"
    role="listbox"
    aria-multiselectable="true"
    bind:this={view}
    data-file-drop-target={app.canWrite && !isSearch ? "" : undefined}
    data-path={app.path}
    onmousedown={bgDown}
    oncontextmenu={bgMenu}
    onkeydown={onKey}
    onwheel={(e) => {
      if (!e.ctrlKey) return;
      e.preventDefault();
      app.prefs.zoom = Math.max(0, Math.min(ZOOM_SIZES.length - 1, app.prefs.zoom + (e.deltaY < 0 ? 1 : -1)));
      app.savePrefs();
    }}
    data-drop-path={app.canWrite && !isSearch ? app.path : undefined}
  >
    {#if !grid && entries.length && !app.mobile}
      <div class="colhead" role="row">
        {#each [["name", "Name"], ["size", "Size"], ...(isSearch ? [["location", "Location"]] : [["type", "Type"]]), ["modified", app.inTrash ? "Trashed On" : "Modified"]] as [key, label] (key)}
          <button
            class="col col-{key}"
            class:sorted={app.prefs.sortBy === key}
            disabled={key === "location"}
            onclick={() => {
              if (app.prefs.sortBy === key) app.prefs.sortDesc = !app.prefs.sortDesc;
              else {
                app.prefs.sortBy = key;
                app.prefs.sortDesc = key === "modified" || key === "size";
              }
              app.savePrefs();
            }}
          >
            {label}
            {#if app.prefs.sortBy === key}<Icon name={app.prefs.sortDesc ? "pan-down" : "pan-up"} size={12} />{/if}
          </button>
        {/each}
      </div>
    {/if}

    {#if app.error && !entries.length}
      <div class="placeholder">
        <Icon name="dialog-warning" size={96} />
        <h2>Unable to load this location</h2>
        <p class="dim">{app.error}</p>
        <button class="btn" onclick={() => app.reload()}>Try Again</button>
      </div>
    {:else if !entries.length && !app.loading && !app.searching}
      <div class="placeholder">
        {#if isSearch}
          <Icon name="system-search" size={96} />
          <h2>No Results Found</h2>
          {#if app.results !== null && app.path !== HOME}
            <p class="dim">Only this folder and the folders in it were searched.</p>
            <button class="btn suggested pill" onclick={() => app.searchEverywhere()}>Search Everywhere</button>
          {:else}
            <p class="dim">Try a different search.</p>
          {/if}
        {:else if app.path === STARRED}
          <Icon name="starred" size={96} />
          <h2>No Starred Files</h2>
          <p class="dim">Use the Star menu item to keep track of files you want to find again.</p>
        {:else if app.path === TRASH}
          <Icon name="user-trash" size={96} />
          <h2>Trash is Empty</h2>
        {:else}
          <Icon name="folder" size={96} />
          <h2>Folder is Empty</h2>
          {#if app.canWrite}<p class="dim">{app.mobile ? "Tap + to add files or folders." : "Drop files here to upload them."}</p>{/if}
        {/if}
      </div>
    {:else if grid}
      <div class="items" style:--cell="{app.mobile ? 100 : Math.max(iconSize + 40, 96)}px">
        {#each entries as e (e.path)}
          <!-- svelte-ignore a11y_click_events_have_key_events (keyboard is handled by the listbox) -->
          <div
            class="item sel"
            class:selected={app.selected.has(e.path)}
            class:cursor={app.cursor === e.path}
            class:cut={app.clipboard?.mode === "cut" && app.clipboard.paths.includes(e.path)}
            class:drop={app.dropTarget === e.path}
            class:hidden-file={e.name.startsWith(".")}
            role="option"
            aria-selected={app.selected.has(e.path)}
            tabindex="-1"
            data-path={e.path}
            data-file-drop-target={e.isDir && !app.inTrash ? "" : undefined}
            title={isSearch ? e.path.replace(/^\/me/, "") : undefined}
            data-drop-path={e.isDir && !app.inTrash ? e.path : undefined}
            onmousedown={(ev) => itemDown(ev, e)}
            onauxclick={(ev) => itemAux(ev, e)}
            onclick={(ev) => itemClick(ev, e)}
            ondblclick={() => !app.mobile && app.open(e)}
            oncontextmenu={(ev) => openItemMenu(ev, e)}
          >
            <div class="icon-box" style:height="{app.mobile ? 80 : iconSize}px">
              <FileIcon entry={e} size={app.mobile ? 80 : iconSize} />
              {#if app.mobile && app.selected.has(e.path)}<span class="check"><Icon name="object-select" size={14} /></span>{/if}
            </div>
            <div class="label">{e.name}</div>
          </div>
        {/each}
      </div>
    {:else if app.mobile}
      <div class="mrows">
        {#each entries as e (e.path)}
          {@const on = app.selected.has(e.path)}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <div
            class="mrow sel"
            class:selected={on}
            class:cut={app.clipboard?.mode === "cut" && app.clipboard.paths.includes(e.path)}
            class:hidden-file={e.name.startsWith(".")}
            role="option"
            aria-selected={on}
            tabindex="-1"
            data-path={e.path}
            onclick={(ev) => itemClick(ev, e)}
            oncontextmenu={(ev) => openItemMenu(ev, e)}
          >
            <span class="micon">
              <FileIcon entry={e} size={40} />
              {#if on}<span class="check"><Icon name="object-select" size={14} /></span>{/if}
            </span>
            <span class="mtext">
              <span class="name">{e.name}</span>
              <span class="sub">{subtitle(e)}</span>
            </span>
            {#if !app.selected.size}
              <button class="btn image flat more" title="More" onclick={(ev) => rowMenu(ev, e)}><Icon name="view-more" /></button>
            {/if}
          </div>
        {/each}
      </div>
    {:else}
      <div class="rows">
        {#each entries as e, i (e.path)}
          <!-- svelte-ignore a11y_click_events_have_key_events (keyboard is handled by the listbox) -->
          <div
            class="row sel"
            class:selected={app.selected.has(e.path)}
            class:cursor={app.cursor === e.path}
            class:cut={app.clipboard?.mode === "cut" && app.clipboard.paths.includes(e.path)}
            class:drop={app.dropTarget === e.path}
            class:hidden-file={e.name.startsWith(".")}
            role="option"
            aria-selected={app.selected.has(e.path)}
            tabindex="-1"
            data-path={e.path}
            data-file-drop-target={e.isDir && !app.inTrash ? "" : undefined}
            data-drop-path={e.isDir && !app.inTrash ? e.path : undefined}
            style:min-height="{Math.max(rowIcon + 10, 32)}px"
            onmousedown={(ev) => itemDown(ev, e)}
            onauxclick={(ev) => itemAux(ev, e)}
            onclick={(ev) => itemClick(ev, e)}
            ondblclick={() => !app.mobile && app.open(e)}
            oncontextmenu={(ev) => openItemMenu(ev, e)}
          >
            <div class="col col-name"><FileIcon entry={e} size={rowIcon} /><span class="name">{e.name}</span></div>
            <div class="col col-size">{e.isDir ? "" : formatSize(e.size)}</div>
            {#if isSearch}
              <div class="col col-location">{parentOf(e.path).replace(/^\/me/, "") || "/"}</div>
            {:else}
              <div class="col col-type">{typeLabel(e)}</div>
            {/if}
            <div class="col col-modified">{formatDate(app.inTrash && e.deletedAt ? e.deletedAt : e.modified)}</div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#if band}
    <div class="rubberband" style={bandStyle}></div>
  {/if}

  {#if showSpinner || app.searching}
    <div class="floating left"><span class="spinner"></span>{app.searching ? "Searching…" : "Loading…"}</div>
  {/if}
  {#if status && !app.mobile}
    <div class="floating">{status}</div>
  {/if}

  {#if app.results !== null && app.path !== HOME && entries.length}
    <div class="trashbar">
      <span class="dim">Searching in “{displayName(app.path)}” and the folders in it.</span>
      <button class="btn" onclick={() => app.searchEverywhere()}>Search Everywhere</button>
    </div>
  {/if}
  {#if app.inTrash && entries.length && !isSearch}
    <div class="trashbar">
      <span class="dim">Items in the trash are kept until you empty it.</span>
      <button class="btn" disabled={!app.selection.length} onclick={() => app.restore()}>Restore</button>
      <button class="btn destructive" onclick={() => app.emptyTrash()}>Empty…</button>
    </div>
  {/if}
</div>

{#if renameEntry}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="rename-backdrop" onmousedown={() => (app.renaming = null)}></div>
  <div class="rename" style:left="{renamePos.x}px" style:top="{renamePos.y}px" style:width="{renamePos.w}px">
    <div class="rename-title">{renameEntry.isDir ? "Folder name" : "File name"}</div>
    <div class="rename-row">
      <input
        class="entry"
        class:error={renameError && !renameError.startsWith("Files with")}
        bind:this={renameInput}
        bind:value={renameValue}
        spellcheck="false"
        onkeydown={(ev) => {
          ev.stopPropagation();
          if (ev.key === "Enter") commitRename();
          else if (ev.key === "Escape") {
            app.renaming = null;
            view?.focus();
          }
        }}
      />
      <button class="btn suggested" disabled={!!renameError && !renameError.startsWith("Files with")} onclick={commitRename}>Rename</button>
    </div>
    {#if renameError}<div class="rename-error">{renameError}</div>{/if}
  </div>
{/if}

<style>
  .viewport {
    position: relative;
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--view-bg);
  }
  .view {
    flex: 1;
    overflow: auto;
    outline: none;
    position: relative;
  }
  .view:global(.file-drop-target-active) {
    box-shadow: inset 0 0 0 2px var(--accent);
  }

  /* ---------- grid ---------- */
  .items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(var(--cell), 1fr));
    gap: 6px;
    padding: 12px 18px 48px;
    align-items: start;
  }
  /* Nautilus grid tiles: the whole tile is rounded and tinted */
  .item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 8px 6px 10px;
    border-radius: 12px;
    min-width: 0;
    outline: none;
    transition: background 100ms ease-out;
  }
  .item:hover {
    background: var(--hover);
  }
  .icon-box {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding: 2px;
    box-sizing: content-box;
  }
  .label {
    margin-top: 6px;
    padding: 0 2px;
    max-width: 100%;
    text-align: center;
    overflow-wrap: anywhere;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .item.selected {
    background: var(--selected);
  }
  .item.selected:hover {
    background: var(--selected-hover);
  }
  .item.selected .label {
    -webkit-line-clamp: unset;
    line-clamp: unset;
  }
  .view:focus-visible .item.cursor,
  .view:focus-visible .row.cursor {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .item.drop,
  .item:global(.file-drop-target-active) {
    background: var(--selected);
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .item.cut,
  .row.cut {
    opacity: 0.5;
  }
  .hidden-file {
    opacity: 0.75;
  }

  /* ---------- list ---------- */
  .colhead {
    position: sticky;
    top: 0;
    z-index: 2;
    display: grid;
    grid-template-columns: var(--list-cols);
    padding: 0 12px;
    background: var(--view-bg);
  }
  .list {
    --list-cols: minmax(200px, 1fr) 110px 170px 130px;
  }
  .colhead .col {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 34px;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg-dim);
    font-weight: bold;
    font-size: 0.87em;
    text-align: left;
  }
  .colhead .col:hover:not(:disabled) {
    background: var(--row-hover);
    color: var(--fg);
  }
  .colhead .col.sorted {
    color: var(--fg);
  }
  .rows {
    padding: 0 12px 48px;
  }
  .row {
    display: grid;
    grid-template-columns: var(--list-cols);
    align-items: center;
    min-height: 40px;
    margin-bottom: 2px;
    border-radius: 6px;
    outline: none;
  }
  .row:hover {
    background: var(--hover);
  }
  .row .col {
    padding: 2px 10px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row .col-name {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .row .name {
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .row .col:not(.col-name) {
    color: var(--fg-dim);
  }
  .row .col-size {
    text-align: right;
  }
  .colhead .col-size {
    justify-content: flex-end;
  }
  .row.selected {
    background: var(--selected);
  }
  .row.selected:hover {
    background: var(--selected-hover);
  }
  .row.drop,
  .row:global(.file-drop-target-active) {
    box-shadow: inset 0 0 0 2px var(--accent);
  }

  /* ---------- phone list: two-line rows ---------- */
  .mrows {
    padding: 4px 0 96px;
  }
  .mrow {
    display: flex;
    align-items: center;
    gap: 14px;
    min-height: 64px;
    padding: 6px 4px 6px 16px;
    outline: none;
  }
  .mrow:active {
    background: var(--hover);
  }
  .mrow.selected {
    background: var(--selected);
  }
  .mrow.cut {
    opacity: 0.5;
  }
  .micon {
    position: relative;
    display: flex;
    width: 44px;
    height: 44px;
    align-items: center;
    justify-content: center;
    flex: none;
    border-radius: 8px;
    overflow: hidden;
  }
  .micon :global(.thumb) {
    border-radius: 6px;
  }
  .check {
    position: absolute;
    right: -2px;
    bottom: -2px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-fg);
    box-shadow: 0 0 0 2px var(--view-bg);
  }
  .icon-box {
    position: relative;
  }
  .mrow .more {
    color: var(--fg-dim);
  }
  .mtext {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .mtext .name {
    font-size: 16px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mtext .sub {
    font-size: 13px;
    color: var(--fg-dim);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  :global(:root[data-mobile="true"]) .items {
    gap: 4px;
    padding: 8px 8px 96px;
  }
  :global(:root[data-mobile="true"]) .item:hover {
    background: none;
  }
  :global(:root[data-mobile="true"]) .item.selected {
    background: var(--selected);
  }

  /* ---------- misc ---------- */
  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 100%;
    min-height: 260px;
    color: var(--fg-dim);
    text-align: center;
    padding: 24px;
  }
  .placeholder :global(.icon) {
    opacity: 0.5;
    margin-bottom: 12px;
  }
  .placeholder h2 {
    margin: 0;
    font-size: 20px;
    font-weight: bold;
  }
  .placeholder p {
    margin: 0 0 8px;
  }
  .rubberband {
    position: fixed;
    z-index: 5;
    pointer-events: none;
    border: 1px solid var(--accent);
    border-radius: 4px;
    background: color-mix(in srgb, var(--accent) 20%, transparent);
  }
  .floating {
    position: absolute;
    right: 6px;
    bottom: 6px;
    z-index: 3;
    max-width: 60%;
    padding: 6px 12px;
    background: var(--popover-bg);
    border-radius: 8px;
    box-shadow: var(--menu-shadow);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    pointer-events: none;
  }
  .floating.left {
    right: auto;
    left: 6px;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  /* AdwBanner */
  .trashbar {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 46px;
    padding: 6px 8px 6px 16px;
    background: color-mix(in srgb, var(--accent) 15%, var(--view-bg));
  }
  .trashbar span {
    flex: 1;
  }

  .rename-backdrop {
    position: fixed;
    inset: 0;
    z-index: 800;
  }
  .rename {
    position: fixed;
    z-index: 801;
    padding: 12px;
    background: var(--popover-bg);
    border-radius: 12px;
    box-shadow: var(--menu-shadow);
  }
  .rename-title {
    font-weight: bold;
    margin-bottom: 8px;
  }
  .rename-row {
    display: flex;
    gap: 6px;
  }
  .rename-row input {
    flex: 1;
    min-width: 0;
  }
  .rename-error {
    margin-top: 6px;
    font-size: 0.87em;
    color: var(--fg-dim);
  }
</style>
