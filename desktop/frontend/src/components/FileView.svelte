<script lang="ts">
  import { untrack } from "svelte";
  import type { Entry } from "../../bindings/nova/services/models";
  import { app, STARRED, TRASH, ZOOM_SIZES, LIST_ZOOM_SIZES, parentOf, type MenuItem } from "../lib/store.svelte";
  import { formatDate, formatSize, pluralize, stemLength } from "../lib/format";
  import { dragLeave, dragOver, dropOn, endDrag, startDrag } from "../lib/dnd";
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
    if (e.button === 2) {
      if (!app.selected.has(entry.path)) app.selectPaths([entry.path]);
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
  }

  function itemClick(e: MouseEvent, entry: Entry) {
    // Clicking one item of a multi-selection selects just that item (GTK).
    if (pressed?.path === entry.path && pressed.wasSelected && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
      if (app.selected.size > 1) app.selectPaths([entry.path]);
    }
    pressed = null;
  }

  function bgDown(e: MouseEvent) {
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

  function itemMenu(e: MouseEvent, entry: Entry) {
    e.preventDefault();
    e.stopPropagation();
    if (!app.selected.has(entry.path)) app.selectPaths([entry.path]);
    const sel = app.selection;
    const one = sel.length === 1 ? sel[0] : null;
    let items: MenuItem[];
    if (app.inTrash) {
      items = [
        { label: "Restore From Trash", run: () => app.restore(sel) },
        { sep: true },
        { label: "Delete Permanently", accel: "Delete", run: () => app.deleteForever(sel) },
        { sep: true },
        { label: "Properties", accel: "Ctrl+I", disabled: !one, run: () => one && (app.modal = { kind: "properties", entry: one }) },
      ];
    } else {
      items = [
        ...(one?.isDir
          ? [
              { label: "Open", accel: "Return", run: () => app.open(one) },
              { label: "Open in New Window", run: () => app.newWindow(one.path) },
            ]
          : [{ label: "Open With Default Application", accel: "Return", run: () => app.openSelection() }]),
        ...(one && !one.isDir ? [{ label: "Preview", accel: "Space", run: () => app.preview(one) }] : []),
        ...(isSearch && one ? [{ label: "Open Item Location", run: () => app.navigate(parentOf(one.path), true, [one.path]) }] : []),
        { sep: true },
        { label: "Cut", accel: "Ctrl+X", run: () => app.copy(true) },
        { label: "Copy", accel: "Ctrl+C", run: () => app.copy(false) },
        ...(one?.isDir ? [{ label: "Paste Into Folder", disabled: !app.clipboard, run: () => app.paste(one.path) }] : []),
        { sep: true },
        { label: "Download…", run: () => app.download(sel) },
        ...(one
          ? [
              one.shared
                ? { label: "Stop Sharing", run: () => app.share(one, false) }
                : { label: "Copy Public Link", run: () => app.share(one, true) },
            ]
          : []),
        { sep: true },
        { label: sel.every((x) => app.isStarred(x.path)) ? "Unstar" : "Star", run: () => app.toggleStar(sel) },
        { label: "Rename…", accel: "F2", disabled: !one, run: () => app.startRename(one!) },
        { label: "Move to Trash", accel: "Delete", run: () => app.trash(sel) },
        { sep: true },
        ...(one?.isDir
          ? [{ label: app.isBookmarked(one.path) ? "Remove from Bookmarks" : "Add to Bookmarks", run: () => app.toggleBookmark(one.path) }]
          : []),
        { label: "Copy Location", disabled: !one, run: () => one && app.copyPath(one) },
        { label: "Properties", accel: "Ctrl+I", disabled: !one, run: () => one && (app.modal = { kind: "properties", entry: one }) },
      ];
    }
    app.openMenu(e.clientX, e.clientY, items);
  }

  function bgMenu(e: MouseEvent) {
    e.preventDefault();
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
      { label: "Upload Folder…", disabled: !app.canWrite, run: () => app.pickUpload(true) },
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

  // ---------- drag and drop ----------

  function onDragStart(e: DragEvent, entry: Entry) {
    if (app.inTrash) {
      e.preventDefault();
      return;
    }
    if (!app.selected.has(entry.path)) app.selectPaths([entry.path]);
    pressed = null;
    startDrag(e, app.selection.map((x) => x.path));
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
    ondragover={(e) => !isSearch && dragOver(e, app.path)}
    ondragleave={(e) => !isSearch && dragLeave(e, app.path)}
    ondrop={(e) => !isSearch && dropOn(e, app.path)}
  >
    {#if !grid && entries.length}
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
          <p class="dim">Try a different search.</p>
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
          {#if app.canWrite}<p class="dim">Drop files here to upload them.</p>{/if}
        {/if}
      </div>
    {:else if grid}
      <div class="items" style:--cell="{Math.max(iconSize + 40, 96)}px">
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
            draggable="true"
            onmousedown={(ev) => itemDown(ev, e)}
            onclick={(ev) => itemClick(ev, e)}
            ondblclick={() => app.open(e)}
            oncontextmenu={(ev) => itemMenu(ev, e)}
            ondragstart={(ev) => onDragStart(ev, e)}
            ondragend={endDrag}
            ondragover={e.isDir ? (ev) => dragOver(ev, e.path) : undefined}
            ondragleave={e.isDir ? (ev) => dragLeave(ev, e.path) : undefined}
            ondrop={e.isDir ? (ev) => dropOn(ev, e.path) : undefined}
          >
            <div class="icon-box" style:height="{iconSize}px"><FileIcon entry={e} size={iconSize} /></div>
            <div class="label">{e.name}</div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="rows">
        {#each entries as e, i (e.path)}
          <!-- svelte-ignore a11y_click_events_have_key_events (keyboard is handled by the listbox) -->
          <div
            class="row sel"
            class:odd={i % 2 === 1}
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
            draggable="true"
            style:min-height="{Math.max(rowIcon + 10, 32)}px"
            onmousedown={(ev) => itemDown(ev, e)}
            onclick={(ev) => itemClick(ev, e)}
            ondblclick={() => app.open(e)}
            oncontextmenu={(ev) => itemMenu(ev, e)}
            ondragstart={(ev) => onDragStart(ev, e)}
            ondragend={endDrag}
            ondragover={e.isDir ? (ev) => dragOver(ev, e.path) : undefined}
            ondragleave={e.isDir ? (ev) => dragLeave(ev, e.path) : undefined}
            ondrop={e.isDir ? (ev) => dropOn(ev, e.path) : undefined}
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
  {#if status}
    <div class="floating">{status}</div>
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
    gap: 6px 4px;
    padding: 12px 12px 48px;
    align-items: start;
  }
  .item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 6px 4px;
    border-radius: 6px;
    min-width: 0;
    outline: none;
  }
  .icon-box {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding: 4px;
    border-radius: 6px;
    box-sizing: content-box;
  }
  .label {
    margin-top: 4px;
    padding: 1px 5px;
    border-radius: 4px;
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
  .item.selected .icon-box {
    background: color-mix(in srgb, var(--accent) 22%, transparent);
  }
  .item.selected .label {
    background: var(--accent);
    color: #fff;
    -webkit-line-clamp: unset;
    line-clamp: unset;
  }
  .item.cursor:not(.selected) .label {
    box-shadow: inset 0 0 0 1px var(--focus);
  }
  .item.drop .icon-box,
  .item:global(.file-drop-target-active) .icon-box {
    background: color-mix(in srgb, var(--accent) 35%, transparent);
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
    background: var(--view-bg);
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
  }
  .list {
    --list-cols: minmax(200px, 1fr) 110px 170px 130px;
  }
  .colhead .col {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-right: 1px solid color-mix(in srgb, var(--border) 50%, transparent);
    background: none;
    color: var(--fg-dim);
    font-weight: bold;
    font-size: 13px;
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
    padding-bottom: 48px;
  }
  .row {
    display: grid;
    grid-template-columns: var(--list-cols);
    align-items: center;
    outline: none;
  }
  .row.odd {
    background: var(--list-alt);
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
    font-size: 13px;
  }
  .row .col-size {
    text-align: right;
  }
  .colhead .col-size {
    justify-content: flex-end;
  }
  .row.selected {
    background: var(--accent);
    color: #fff;
  }
  :global(:root[data-theme="dark"]) .row.selected {
    background: var(--accent-dim);
  }
  .row.selected .col {
    color: #fff;
  }
  .row.cursor:not(.selected) {
    box-shadow: inset 0 0 0 1px var(--focus);
  }
  .row.drop,
  .row:global(.file-drop-target-active) {
    box-shadow: inset 0 0 0 2px var(--accent);
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
    background: color-mix(in srgb, var(--accent) 20%, transparent);
  }
  .floating {
    position: absolute;
    right: 0;
    bottom: 0;
    z-index: 3;
    max-width: 60%;
    padding: 4px 10px;
    background: var(--bg);
    border: 1px solid var(--border);
    border-right: 0;
    border-bottom: 0;
    border-top-left-radius: var(--radius);
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    pointer-events: none;
  }
  .floating.left {
    right: auto;
    left: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    border-left: 0;
    border-right: 1px solid var(--border);
    border-top-left-radius: 0;
    border-top-right-radius: var(--radius);
  }
  .trashbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    background: var(--bg);
    border-top: 1px solid var(--border);
    font-size: 13px;
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
    padding: 10px;
    background: var(--popover-bg);
    border: 1px solid rgba(0, 0, 0, 0.23);
    border-radius: var(--radius);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
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
    font-size: 12px;
    color: var(--fg-dim);
  }
</style>
