<script lang="ts">
  import WhatsNew from "./WhatsNew.svelte";
  import * as Files from "../../bindings/nova/services/filesservice";
  import * as Updates from "../../bindings/nova/services/updateservice";
  import type { Entry, FolderSize } from "../../bindings/nova/services/models";
  import { app, displayPath, parentOf, TRASH, type Modal, isVirtual } from "../lib/store.svelte";
  import { itemMenu } from "../lib/menus";
  import { formatAgo, formatDateLong, formatSize, pluralize, stemLength } from "../lib/format";
  import { fileIconUrl, hasThumbnail, isAudio, isImage, isText, isVideo, rawUrl, thumbUrl } from "../lib/icons";
  import FileIcon from "./FileIcon.svelte";
  import Icon from "./Icon.svelte";
  import ShareDialog from "./ShareDialog.svelte";

  let promptValue = $state("");
  let promptInput = $state<HTMLInputElement>();
  let dialogEl = $state<HTMLDivElement>();

  function close(result?: boolean) {
    const m = app.modal;
    app.modal = null;
    if (m?.kind === "confirm") m.resolve(!!result);
    if (m?.kind === "prompt") m.resolve(result ? promptValue.trim() : null);
    queueMicrotask(() => (document.querySelector(".view") as HTMLElement | null)?.focus());
  }

  $effect(() => {
    const m = app.modal;
    if (!m) return;
    if (m.kind === "prompt") {
      promptValue = m.value;
      queueMicrotask(() => {
        promptInput?.focus();
        promptInput?.setSelectionRange(0, m.select ?? stemLength(m.value, true));
      });
    } else {
      // Destructive dialogs focus Cancel (safe default); others focus the action.
      const safe = m.kind === "confirm" && m.destructive;
      queueMicrotask(() => {
        const target = safe ? null : dialogEl?.querySelector<HTMLElement>(".default");
        (target ?? dialogEl?.querySelector<HTMLElement>("button"))?.focus();
      });
    }
  });

  // ---------- properties ----------
  let measured = $state<FolderSize | null>(null);
  let measuring = $state(false);
  let fresh = $state<Entry | null>(null);

  $effect(() => {
    const m = app.modal;
    measured = null;
    fresh = null;
    if (m?.kind !== "properties") return;
    const path = m.entry.path;
    Files.Stat(path).then((e) => {
      if (app.modal?.kind === "properties" && app.modal.entry.path === path) fresh = e;
    });
    if (m.entry.isDir) {
      measuring = true;
      Files.Measure(path)
        .then((s) => {
          if (app.modal?.kind === "properties" && app.modal.entry.path === path) measured = s;
        })
        .finally(() => (measuring = false));
    }
  });

  // ---------- preview ----------
  let text = $state<string | null>(null);
  $effect(() => {
    const m = app.modal;
    text = null;
    if (m?.kind !== "preview" || !isText(m.entry)) return;
    const ctrl = new AbortController();
    fetch(rawUrl(m.entry.path), { headers: { Range: "bytes=0-262143" }, signal: ctrl.signal })
      .then((r) => r.text())
      .then((t) => (text = t))
      .catch(() => {});
    return () => ctrl.abort();
  });

  function previewStep(d: number) {
    const m = app.modal;
    if (m?.kind !== "preview") return;
    const files = app.entries.filter((e) => !e.isDir);
    const i = files.findIndex((e) => e.path === m.entry.path);
    const next = files[(i + d + files.length) % files.length];
    if (next) {
      app.selectPaths([next.path]);
      app.modal = { kind: "preview", entry: next };
    }
  }

  // Phone viewer: swipe sideways for the next file, tap to hide the bars.
  let bare = $state(false);
  let fullLoaded = $state(false);
  let fullFailed = $state(false);
  let thumbFailed = $state(false);
  // Android: media is fetched into the app cache and served from disk (with
  // seeking) rather than squeezed through the WebView bridge.
  const onAndroid = !!(window as unknown as { NovaAndroid?: unknown }).NovaAndroid;
  let mediaSrc = $state<string | null>(null);
  $effect(() => {
    const m = app.modal;
    fullLoaded = false;
    fullFailed = false;
    thumbFailed = false;
    mediaSrc = null;
    if (m?.kind !== "preview") return;
    const e = m.entry;
    if (!(isImage(e) || isVideo(e) || isAudio(e))) return;
    if (!onAndroid) {
      mediaSrc = rawUrl(e.path, e.mime);
      return;
    }
    let live = true;
    Files.CacheForView(e.path)
      .then((rel) => live && (mediaSrc = `/__capture__/${rel.split("/").map(encodeURIComponent).join("/")}?type=${encodeURIComponent(e.mime)}`))
      .catch(() => live && (fullFailed = true));
    return () => (live = false);
  });
  let showInfo = $state(false);
  let touch: { x: number; y: number } | null = null;
  $effect(() => {
    const m = app.modal;
    if (m?.kind !== "preview") bare = false;
    // Files Nova can't show open with their details instead.
    showInfo = m?.kind === "preview" && !(isImage(m.entry) || isVideo(m.entry) || isAudio(m.entry) || isText(m.entry));
  });
  function touchStart(e: TouchEvent) {
    touch = e.touches.length === 1 ? { x: e.touches[0].clientX, y: e.touches[0].clientY } : null;
  }
  function touchEnd(e: TouchEvent) {
    if (!touch) return;
    const dx = e.changedTouches[0].clientX - touch.x;
    const dy = e.changedTouches[0].clientY - touch.y;
    touch = null;
    if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy) * 1.5) previewStep(dx < 0 ? 1 : -1);
  }
  function previewMore(e: MouseEvent, entry: Entry) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    app.openMenu(r.left, r.top, itemMenu([entry], app.results !== null || isVirtual(app.path)));
  }

  function onKey(e: KeyboardEvent) {
    const m = app.modal;
    if (!m) return;
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close(false);
    } else if (m.kind === "preview") {
      if (e.key === " ") {
        e.preventDefault();
        e.stopPropagation();
        close();
      } else if (e.key === "ArrowRight" || e.key === "ArrowDown") {
        e.preventDefault();
        previewStep(1);
      } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
        e.preventDefault();
        previewStep(-1);
      } else if (e.key === "Enter") {
        close();
        app.open(m.entry);
      }
    } else if (e.key === "Enter" && m.kind === "confirm" && !(document.activeElement instanceof HTMLButtonElement)) {
      // A focused button handles Enter itself, so Enter on Cancel cancels.
      e.preventDefault();
      close(true);
    }
  }

  const shortcuts: [string, [string, string][]][] = [
    [
      "Navigation",
      [
        ["Alt+Left / Backspace", "Go back"],
        ["Alt+Right", "Go forward"],
        ["Alt+Up", "Go to parent folder"],
        ["Alt+Home", "Go to home folder"],
        ["Mouse back / forward", "Go back / forward"],
        ["Middle click a folder", "Open in a new window"],
        ["Ctrl+L", "Enter location"],
        ["Ctrl+F", "Search this folder"],
        ["Shift+Ctrl+F", "Search everywhere"],
        ["F5 / Ctrl+R", "Reload"],
      ],
    ],
    [
      "View",
      [
        ["Ctrl+1 / Ctrl+2", "List / grid view"],
        ["Ctrl+Plus / Ctrl+Minus", "Zoom in / out"],
        ["Ctrl+0", "Reset zoom"],
        ["Ctrl+H", "Show hidden files"],
        ["F9", "Toggle sidebar"],
        ["Space", "Preview"],
      ],
    ],
    [
      "Editing",
      [
        ["Shift+Ctrl+N", "New folder"],
        ["Ctrl+U", "Upload files"],
        ["F2", "Rename"],
        ["Delete", "Move to trash"],
        ["Shift+Delete", "Delete permanently"],
        ["Ctrl+C / Ctrl+X / Ctrl+V", "Copy / cut / paste"],
        ["Ctrl+N", "New window"],
        ["Ctrl+A", "Select all"],
        ["Shift+Ctrl+I", "Invert selection"],
        ["Ctrl+Z", "Undo"],
        ["Ctrl+I / Alt+Return", "Properties"],
        ["Ctrl+D", "Bookmark location"],
        ["Ctrl+,", "Settings"],
      ],
    ],
  ];

  function kindLabel(e: Entry): string {
    if (e.isDir) return "Folder";
    return (e.mime || "unknown").split(";")[0];
  }
</script>

<svelte:window onkeydowncapture={(e) => app.modal && onKey(e)} />

{#snippet props(m: Extract<Modal, { kind: "properties" }>)}
  {@const e = fresh ?? m.entry}
  <div class="dialog props" bind:this={dialogEl} role="dialog" aria-modal="true" aria-label="Properties">
    <div class="dlg-head">
      <span class="dlg-title">{e.name || "Home"} Properties</span>
      <button class="btn image flat round" title="Close" onclick={() => close()}><Icon name="window-close" /></button>
    </div>
    <div class="props-body">
      <div class="props-icon"><FileIcon entry={e} size={96} /></div>
      <dl>
        <dt>Name</dt>
        <dd class="selectable">{e.name}</dd>
        <dt>Type</dt>
        <dd>{kindLabel(e)}</dd>
        <dt>{e.isDir ? "Contents" : "Size"}</dt>
        <dd>
          {#if e.isDir}
            {#if measured}{pluralize(measured.files + measured.folders, "item", "items")}, totalling {formatSize(measured.bytes)}{:else if measuring}<span class="spinner"></span>{:else}—{/if}
          {:else}
            {formatSize(e.size)} ({e.size.toLocaleString()} bytes)
          {/if}
        </dd>
        <dt>Parent Folder</dt>
        <dd class="selectable">{displayPath(parentOf(e.path))}</dd>
        {#if e.origPath}
          <dt>Original Location</dt>
          <dd class="selectable">{displayPath(parentOf(e.origPath))}</dd>
        {/if}
        <dt>Modified</dt>
        <dd>{formatDateLong(e.modified)}</dd>
        <dt>Created</dt>
        <dd>{formatDateLong(e.created)}</dd>
        {#if e.owner}
          <dt>Owner</dt>
          <dd>{e.owner}</dd>
        {/if}
        {#if e.mode}
          <dt>Permissions</dt>
          <dd class="mono">{e.mode}</dd>
        {/if}
        {#if e.sha256}
          <dt>SHA-256</dt>
          <dd class="mono selectable small">{e.sha256}</dd>
        {/if}
        {#if !e.path.startsWith(TRASH) && !app.isRoot(e.path)}
          <dt>Sharing</dt>
          <dd class="sharing">
            <span>{e.public ? "Anyone with the link" : e.shared ? "Specific people" : "Not shared"}</span>
            <button class="btn" onclick={() => app.openShare(e)}>Share…</button>
          </dd>
        {/if}
      </dl>
    </div>
  </div>
{/snippet}

{#snippet previewBody(e: Entry)}
  {#if isImage(e)}
    {#key e.path}
      <!-- The thumbnail shows at once; the full image replaces it once loaded. -->
      <span class="pv-img">
        {#if hasThumbnail(e) && !fullLoaded && !thumbFailed}<img class="pv-thumb" src={thumbUrl(e, 128)} alt="" onerror={() => (thumbFailed = true)} />{/if}
        {#if fullFailed}
          <span class="pv-failed">Couldn't load the image.</span>
        {:else if !mediaSrc || !fullLoaded}
          <span class="spinner pv-spin"></span>
        {/if}
        {#if mediaSrc && !fullFailed}
          <img
            class="pv-full"
            class:ready={fullLoaded}
            src={mediaSrc}
            alt={e.name}
            onload={() => (fullLoaded = true)}
            onerror={() => (fullFailed = true)}
          />
        {/if}
      </span>
    {/key}
  {:else if isVideo(e)}
    <!-- svelte-ignore a11y_media_has_caption -->
    {#if mediaSrc}<video src={mediaSrc} controls autoplay></video>{:else if fullFailed}<span class="pv-failed">Couldn't load the video.</span>{:else}<span class="spinner"></span>{/if}
  {:else if isAudio(e)}
    <div class="pv-audio">
      <img src={fileIconUrl(e)} alt="" width="128" height="128" />
      {#if mediaSrc}<audio src={mediaSrc} controls autoplay></audio>{:else if !fullFailed}<span class="spinner"></span>{/if}
    </div>
  {:else if isText(e)}
    <pre class="pv-text">{text ?? "Loading…"}</pre>
  {:else}
    <div class="pv-none">
      <img src={fileIconUrl(e)} alt="" width="128" height="128" />
      <div>{kindLabel(e)}</div>
      {#if app.mobile}
        <button class="btn" onclick={() => app.download([e])}>Download</button>
      {:else}
        <button class="btn" onclick={() => (close(), app.open(e))}>Open With Default Application</button>
      {/if}
    </div>
  {/if}
{/snippet}

{#if app.modal}
  {@const m = app.modal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="backdrop" class:dark={m.kind === "preview"} class:full={m.kind === "preview" && app.mobile} onmousedown={(e) => e.target === e.currentTarget && close(false)}>
    {#if m.kind === "confirm"}
      <div class="dialog message" bind:this={dialogEl} role="alertdialog" aria-modal="true">
        <div class="msg-body">
          <div class="msg-title">{m.title}</div>
          <div class="msg-text">{m.body}</div>
        </div>
        <div class="msg-buttons">
          <button class="msg-btn" onclick={() => close(false)}>Cancel</button>
          <button class="msg-btn default" class:destructive={m.destructive} class:suggested={!m.destructive} onclick={() => close(true)}>{m.confirm}</button>
        </div>
      </div>
    {:else if m.kind === "prompt"}
      <div class="dialog prompt" role="dialog" aria-modal="true" aria-label={m.title}>
        <div class="dlg-head">
          <button class="btn" onclick={() => close(false)}>Cancel</button>
          <span class="dlg-title">{m.title}</span>
          <button class="btn suggested" disabled={!promptValue.trim() || promptValue.includes("/")} onclick={() => close(true)}>{m.confirm}</button>
        </div>
        <div class="prompt-body">
          <label>
            <span>{m.label}</span>
            <input
              class="entry"
              class:error={promptValue.includes("/")}
              bind:this={promptInput}
              bind:value={promptValue}
              spellcheck="false"
              onkeydown={(e) => {
                if (e.key === "Enter" && promptValue.trim() && !promptValue.includes("/")) close(true);
              }}
            />
          </label>
          {#if promptValue.includes("/")}<div class="dim small">Names cannot contain “/”.</div>{/if}
        </div>
      </div>
    {:else if m.kind === "properties"}
      {@render props(m)}
    {:else if m.kind === "share"}
      <div class="dialog share" bind:this={dialogEl} role="dialog" aria-modal="true" aria-label="Share {m.entry.name}">
        <div class="dlg-head">
          <span class="dlg-title">Share “{m.entry.name}”</span>
          <button class="btn image flat round default" title="Close" onclick={() => close()}><Icon name="window-close" /></button>
        </div>
        <ShareDialog entry={m.entry} />
      </div>
    {:else if m.kind === "preview" && app.mobile}
      {@const e = m.entry}
      {@const starred = app.isStarred(e.path)}
      <div class="mpv" class:bare role="dialog" aria-modal="true" aria-label="Preview {e.name}">
        <header class="mpv-top">
          <button class="mpv-btn" title="Back" onclick={() => close()}><Icon name="go-previous" size={22} /></button>
          <div class="mpv-title">
            <div class="mpv-name">{e.name}</div>
            <div class="mpv-sub">{formatSize(e.size)} • {kindLabel(e)} • {formatAgo(e.modified)}</div>
          </div>
          <button class="mpv-btn" class:on={showInfo} title="Details" onclick={() => (showInfo = !showInfo)}><Icon name="dialog-information" size={22} /></button>
        </header>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="mpv-content"
          ontouchstart={touchStart}
          ontouchend={touchEnd}
          onclick={(ev) => (ev.target as HTMLElement).closest?.(".pv-img") && (bare = !bare)}
        >
          {@render previewBody(e)}
          {#if showInfo}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="mpv-info" ontouchstart={(ev) => ev.stopPropagation()} ontouchend={(ev) => ev.stopPropagation()} onclick={(ev) => ev.stopPropagation()}>
              <dl>
                <dt>Type</dt>
                <dd>{kindLabel(e)}{e.mime && e.mime !== kindLabel(e) ? ` (${e.mime})` : ""}</dd>
                <dt>Size</dt>
                <dd>{formatSize(e.size)} ({e.size.toLocaleString()} bytes)</dd>
                <dt>Location</dt>
                <dd class="selectable">{displayPath(parentOf(e.path))}</dd>
                {#if e.origPath}
                  <dt>Original Location</dt>
                  <dd class="selectable">{displayPath(parentOf(e.origPath))}</dd>
                {/if}
                <dt>Modified</dt>
                <dd>{formatDateLong(e.modified)}</dd>
                <dt>Created</dt>
                <dd>{formatDateLong(e.created)}</dd>
                {#if e.owner}
                  <dt>Owner</dt>
                  <dd>{e.owner}</dd>
                {/if}
                {#if e.mode}
                  <dt>Permissions</dt>
                  <dd class="mono">{e.mode}</dd>
                {/if}
                {#if !e.path.startsWith(TRASH)}
                  <dt>Sharing</dt>
                  <dd>{e.public ? "Anyone with the link" : e.shared ? "Specific people" : "Not shared"}</dd>
                {/if}
                {#if e.sha256}
                  <dt>SHA-256</dt>
                  <dd class="mono selectable small">{e.sha256}</dd>
                {/if}
              </dl>
            </div>
          {/if}
        </div>
        <footer class="mpv-actions">
          {#if app.inTrash}
            <button class="mpv-act" onclick={() => (close(), app.restore([e]))}><Icon name="edit-undo" size={22} /><span>Restore</span></button>
            <button class="mpv-act" onclick={() => (close(), app.deleteForever([e]))}><Icon name="edit-delete" size={22} /><span>Delete</span></button>
          {:else}
            <button class="mpv-act" class:on={starred} onclick={() => app.toggleStar([e])}>
              <Icon name={starred ? "starred" : "non-starred"} size={22} /><span>{starred ? "Starred" : "Star"}</span>
            </button>
            <button class="mpv-act" onclick={() => app.openShare(e)}><Icon name="send-to" size={22} /><span>Share</span></button>
            <button class="mpv-act" onclick={() => app.download([e])}><Icon name="folder-download" size={22} /><span>Download</span></button>
            <button class="mpv-act" onclick={() => (close(), app.trash([e]))}><Icon name="user-trash" size={22} /><span>Trash</span></button>
          {/if}
          <button class="mpv-act" onclick={(ev) => previewMore(ev, e)}><Icon name="view-more" size={22} /><span>More</span></button>
        </footer>
      </div>
    {:else if m.kind === "preview"}
      {@const e = m.entry}
      <div class="preview" role="dialog" aria-modal="true" aria-label="Preview {e.name}">
        <div class="pv-bar">
          <span class="pv-name">{e.name}</span>
          <span class="pv-size">{formatSize(e.size)}</span>
          {#if app.mobile}
            <button class="pv-btn" title="Download" onclick={() => app.download([e])}><Icon name="folder-download" /></button>
          {:else}
            <button class="pv-btn" title="Open with default application" onclick={() => (close(), app.open(e))}><Icon name="document-open" /></button>
          {/if}
          <button class="pv-btn" title="Close" onclick={() => close()}><Icon name="window-close" /></button>
        </div>
        <div class="pv-content">
          {@render previewBody(e)}
        </div>
      </div>
    {:else if m.kind === "shortcuts"}
      <div class="dialog shortcuts" bind:this={dialogEl} role="dialog" aria-modal="true" aria-label="Shortcuts">
        <div class="dlg-head">
          <span class="dlg-title">Shortcuts</span>
          <button class="btn image flat round" title="Close" onclick={() => close()}><Icon name="window-close" /></button>
        </div>
        <div class="sc-body">
          {#each shortcuts as [group, list] (group)}
            <section>
              <h3>{group}</h3>
              {#each list as [keys, what] (keys)}
                <div class="sc-row">
                  <span class="keys">{#each keys.split(" / ") as k, i (k)}{#if i}<span class="dim"> / </span>{/if}{#each k.split("+") as part, j (j)}{#if j}<span class="dim">+</span>{/if}<kbd>{part}</kbd>{/each}{/each}</span>
                  <span>{what}</span>
                </div>
              {/each}
            </section>
          {/each}
        </div>
      </div>
    {:else if m.kind === "whatsnew"}
      <WhatsNew versions={m.versions} onClose={() => close()} />
    {:else if m.kind === "about"}
      <div class="dialog about" bind:this={dialogEl} role="dialog" aria-modal="true" aria-label="About">
        <div class="dlg-head">
          <span class="dlg-title">About Nova</span>
          <button class="btn image flat round" title="Close" onclick={() => close()}><Icon name="window-close" /></button>
        </div>
        <div class="about-body">
          <img src="/nova.png" alt="" width="96" height="96" />
          <h2>Nova</h2>
          <div class="dim">{app.update?.currentVersion === "dev" ? "Development build" : `Version ${app.update?.currentVersion ?? ""}`}</div>
          <p>A desktop file manager for nova.storage.</p>
          <p class="dim small">Built with Wails, Go and Svelte. Icons follow your system theme.</p>
          {#if app.update?.state === "ready"}
            <button class="btn suggested" onclick={() => app.applyUpdate()}>{app.mobile ? "Install" : "Restart to Install"} {app.update.latestVersion}</button>
          {:else if app.update?.state !== "disabled"}
            <button class="btn" disabled={app.update?.state === "checking" || app.update?.state === "downloading"} onclick={() => app.checkForUpdates()}>
              {app.update?.state === "downloading" ? "Downloading…" : "Check for Updates"}
            </button>
          {/if}
          <button class="link" onclick={() => (app.modal = { kind: "whatsnew", versions: [] })}>What's New</button>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 950;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.25);
    animation: fade 120ms ease-out;
  }
  .backdrop.dark {
    background: rgba(0, 0, 0, 0.75);
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes dlg-in {
    from {
      opacity: 0;
      transform: scale(0.96);
    }
  }
  /* AdwDialog */
  .dialog {
    background: var(--dialog-bg);
    border-radius: 15px;
    box-shadow: var(--dialog-shadow);
    animation: dlg-in 180ms ease-out;
    overflow: hidden;
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 32px);
    display: flex;
    flex-direction: column;
  }
  .dlg-head {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 47px;
    padding: 6px 7px;
  }
  .dlg-title {
    flex: 1;
    text-align: center;
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding: 0 8px;
  }
  .dlg-head .round {
    background: var(--btn-bg);
  }
  .dlg-head .round:hover {
    background: var(--btn-hover);
  }
  .small {
    font-size: 12px;
  }

  /* AdwAlertDialog */
  .message {
    width: 372px;
  }
  .msg-body {
    padding: 32px 30px 24px;
    text-align: center;
  }
  .msg-title {
    font-weight: 800;
    font-size: 1.36em;
    margin-bottom: 12px;
  }
  .msg-text {
    color: var(--fg);
  }
  .msg-buttons {
    display: flex;
    gap: 12px;
    padding: 0 24px 24px;
  }
  .msg-btn {
    flex: 1;
    min-height: 42px;
    border: 0;
    border-radius: 12px;
    background: var(--btn-bg);
    font-weight: bold;
    outline: none;
  }
  .msg-btn:hover {
    background: var(--btn-hover);
  }
  .msg-btn:active {
    background: var(--btn-active);
  }
  .msg-btn:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .msg-btn.suggested {
    background: var(--accent);
    color: var(--accent-fg);
  }
  .msg-btn.suggested:hover {
    background: color-mix(in srgb, var(--accent) 90%, #fff);
  }
  .msg-btn.destructive {
    background: var(--destructive);
    color: var(--destructive-fg);
  }
  .msg-btn.destructive:hover {
    background: color-mix(in srgb, var(--destructive) 90%, #fff);
  }

  .prompt {
    width: 400px;
  }
  .prompt-body {
    padding: 18px;
  }
  .prompt-body label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .prompt-body input {
    width: 100%;
  }

  /* Properties */
  .props {
    width: 520px;
  }
  .props-body {
    display: flex;
    gap: 20px;
    padding: 20px;
    overflow: auto;
  }
  .props-icon {
    flex: none;
  }
  dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 8px 14px;
    margin: 0;
    min-width: 0;
    align-items: center;
  }
  dt {
    text-align: right;
    color: var(--fg-dim);
  }
  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .selectable {
    user-select: text;
    -webkit-user-select: text;
    cursor: text;
  }
  .mono {
    font-family: var(--mono);
  }
  .sharing {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px 12px;
  }
  .share {
    width: 520px;
  }

  /* Preview (sushi-like) */
  .preview {
    display: flex;
    flex-direction: column;
    width: min(90vw, 1200px);
    height: min(88vh, 900px);
    color: #fff;
  }
  .pv-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 8px;
    background: rgba(30, 30, 30, 0.9);
    border-radius: 8px 8px 0 0;
  }
  .pv-name {
    flex: 1;
    font-weight: bold;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pv-size {
    opacity: 0.7;
    font-size: 13px;
  }
  .pv-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 0;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
  }
  .pv-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }
  .pv-content {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(20, 20, 20, 0.6);
    border-radius: 0 0 8px 8px;
    overflow: hidden;
  }
  .pv-img {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
  }
  .pv-img img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
  .pv-thumb {
    width: 100%;
    height: 100%;
  }
  .pv-spin {
    position: absolute;
  }
  .pv-failed {
    position: absolute;
    bottom: 16px;
    padding: 6px 12px;
    border-radius: 9999px;
    background: rgba(0, 0, 0, 0.7);
    color: #fff;
    font-size: 13px;
  }
  .pv-full:not(.ready) {
    position: absolute;
    opacity: 0;
  }
  .pv-content video {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
  .pv-text {
    align-self: stretch;
    width: 100%;
    margin: 0;
    padding: 16px;
    overflow: auto;
    font: 13px/1.45 var(--mono);
    background: #1e1e1e;
    color: #ddd;
    white-space: pre-wrap;
    user-select: text;
    -webkit-user-select: text;
  }
  .pv-none,
  .pv-audio {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
  }

  /* Phone viewer: full screen, black, bars that get out of the way. */
  .backdrop.full {
    background: #000;
  }
  .mpv {
    position: relative;
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    padding: env(safe-area-inset-top, 0) 0 env(safe-area-inset-bottom, 0);
    background: #000;
    color: #fff;
  }
  .mpv-top,
  .mpv-actions {
    position: relative;
    z-index: 1;
    transition: opacity 150ms ease-out;
  }
  .mpv.bare .mpv-top,
  .mpv.bare .mpv-actions {
    opacity: 0;
    pointer-events: none;
  }
  .mpv-top {
    display: flex;
    align-items: center;
    gap: 4px;
    min-height: 64px;
    padding: 8px 12px 8px 4px;
  }
  .mpv-btn {
    display: grid;
    place-items: center;
    flex: none;
    width: 48px;
    height: 48px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: inherit;
  }
  .mpv-btn:active,
  .mpv-act:active {
    background: rgba(255, 255, 255, 0.12);
  }
  .mpv-title {
    flex: 1;
    min-width: 0;
  }
  .mpv-name {
    font-size: 17px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mpv-sub {
    margin-top: 2px;
    font-size: 13px;
    opacity: 0.65;
  }
  .mpv-btn.on {
    background: rgba(255, 255, 255, 0.16);
  }
  .mpv-info {
    position: absolute;
    left: 12px;
    right: 12px;
    bottom: 8px;
    max-height: 70%;
    overflow: auto;
    padding: 6px 18px;
    border-radius: 20px;
    background: rgba(32, 32, 34, 0.96);
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.5);
    animation: sheet-up 160ms ease-out;
  }
  @keyframes sheet-up {
    from {
      transform: translateY(12px);
      opacity: 0;
    }
  }
  .mpv-info dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 10px 16px;
    margin: 12px 0;
    font-size: 15px;
  }
  .mpv-info dt {
    color: rgba(255, 255, 255, 0.6);
  }
  .mpv-info dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .mpv-info .mono {
    font-family: var(--mono);
  }
  .mpv-info .small {
    font-size: 12px;
  }
  .mpv-info .selectable {
    user-select: text;
    -webkit-user-select: text;
  }
  .mpv-content {
    position: relative;
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }
  .mpv-content video {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
  .mpv-content .pv-text {
    height: 100%;
    background: #111;
  }
  .mpv-actions {
    display: flex;
    min-height: 72px;
    padding: 4px 4px 8px;
  }
  .mpv-act {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 5px;
    border: 0;
    border-radius: 16px;
    background: none;
    color: rgba(255, 255, 255, 0.85);
    font: inherit;
    font-size: 12px;
    font-weight: 600;
  }
  .mpv-act.on {
    color: #f6d32d;
  }

  .shortcuts {
    width: 760px;
  }
  .sc-body {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 8px 28px;
    padding: 18px 22px 22px;
    overflow: auto;
  }
  .sc-body h3 {
    margin: 6px 0 10px;
    font-size: 1em;
  }
  .sc-row {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-bottom: 10px;
  }
  kbd {
    display: inline-block;
    min-width: 22px;
    padding: 1px 7px;
    border-radius: 6px;
    background: var(--btn-bg);
    box-shadow: inset 0 -2px color-mix(in srgb, var(--fg) 12%, transparent);
    font: bold 0.87em var(--font);
    text-align: center;
  }
  .about {
    width: 360px;
  }
  .about-body {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24px;
    text-align: center;
  }
  .about-body h2 {
    margin: 10px 0 2px;
  }
  .about-body p {
    margin: 12px 0 0;
  }
  .about-body .btn {
    margin-top: 16px;
  }
  .about-body .link {
    margin-top: 10px;
    border: 0;
    background: none;
    color: var(--accent-text);
    text-decoration: underline;
    cursor: pointer;
  }
</style>
