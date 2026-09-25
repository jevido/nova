<script lang="ts">
  import * as Files from "../../bindings/nova/services/filesservice";
  import * as Updates from "../../bindings/nova/services/updateservice";
  import type { Entry, FolderSize } from "../../bindings/nova/services/models";
  import { app, parentOf, TRASH, type Modal } from "../lib/store.svelte";
  import { formatDateLong, formatSize, pluralize, stemLength } from "../lib/format";
  import { fileIconUrl, isAudio, isImage, isText, isVideo, rawUrl } from "../lib/icons";
  import FileIcon from "./FileIcon.svelte";
  import Icon from "./Icon.svelte";

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
        ["Ctrl+L", "Enter location"],
        ["Ctrl+F", "Search"],
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
        <dd class="selectable">{parentOf(e.path).replace(/^\/me/, "") || "/"}</dd>
        {#if e.origPath}
          <dt>Original Location</dt>
          <dd class="selectable">{parentOf(e.origPath).replace(/^\/me/, "") || "/"}</dd>
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
        {#if !e.path.startsWith(TRASH) && e.path !== "/me"}
          <dt>Public Link</dt>
          <dd>
            <button
              class="switch"
              class:on={e.shared}
              role="switch"
              aria-checked={e.shared}
              aria-label="Public link"
              onclick={async () => {
                await app.share(e, !e.shared);
                fresh = await Files.Stat(e.path);
              }}
            ><span></span></button>
          </dd>
        {/if}
      </dl>
    </div>
  </div>
{/snippet}

{#if app.modal}
  {@const m = app.modal}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="backdrop" class:dark={m.kind === "preview"} onmousedown={(e) => e.target === e.currentTarget && close(false)}>
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
          {#if isImage(e)}
            <img src={rawUrl(e.path)} alt={e.name} />
          {:else if isVideo(e)}
            <!-- svelte-ignore a11y_media_has_caption -->
            <video src={rawUrl(e.path)} controls autoplay></video>
          {:else if isAudio(e)}
            <div class="pv-audio">
              <img src={fileIconUrl(e)} alt="" width="128" height="128" />
              <audio src={rawUrl(e.path)} controls autoplay></audio>
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
          <button class="link" onclick={() => Updates.OpenReleasePage()}>Release notes</button>
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
  .dialog {
    background: var(--bg);
    border: 1px solid rgba(0, 0, 0, 0.3);
    border-radius: 8px;
    box-shadow: 0 3px 12px rgba(0, 0, 0, 0.35);
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
    min-height: 46px;
    padding: 6px;
    background: linear-gradient(to top, var(--header-bottom), var(--header-top));
    border-bottom: 1px solid var(--header-border);
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
    width: 24px;
    height: 24px;
    min-width: 24px;
    min-height: 24px;
    padding: 0;
    border-radius: 50%;
  }
  .small {
    font-size: 12px;
  }

  /* GtkMessageDialog */
  .message {
    width: 420px;
  }
  .msg-body {
    padding: 24px 30px 20px;
    text-align: center;
  }
  .msg-title {
    font-weight: bold;
    font-size: 15px;
    margin-bottom: 10px;
  }
  .msg-text {
    color: var(--fg);
    opacity: 0.9;
  }
  .msg-buttons {
    display: flex;
    border-top: 1px solid var(--border);
  }
  .msg-btn {
    flex: 1;
    min-height: 44px;
    border: 0;
    background: var(--bg);
    outline: none;
  }
  .msg-btn + .msg-btn {
    border-left: 1px solid var(--border);
  }
  .msg-btn:hover {
    background: color-mix(in srgb, var(--fg) 6%, var(--bg));
  }
  .msg-btn:focus-visible {
    box-shadow: inset 0 0 0 2px var(--focus);
  }
  .msg-btn.suggested {
    color: var(--accent);
    font-weight: bold;
  }
  .msg-btn.destructive {
    color: var(--destructive);
    font-weight: bold;
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
    font-weight: bold;
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
    font-family: monospace;
  }
  .switch {
    position: relative;
    width: 48px;
    height: 26px;
    border-radius: 14px;
    border: 1px solid var(--border-dark);
    background: color-mix(in srgb, var(--fg) 12%, var(--bg));
    padding: 0;
    outline: none;
  }
  .switch span {
    position: absolute;
    top: 1px;
    left: 1px;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: linear-gradient(to top, var(--btn-bottom), var(--btn-top));
    border: 1px solid var(--border-dark);
    transition: left 150ms;
  }
  .switch.on {
    background: var(--accent);
    border-color: var(--accent-dim);
  }
  .switch.on span {
    left: 23px;
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
  .pv-content img,
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
    font: 13px/1.45 monospace;
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
    font-size: 14px;
  }
  .sc-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-bottom: 10px;
    font-size: 13px;
  }
  kbd {
    display: inline-block;
    min-width: 20px;
    padding: 1px 6px;
    border: 1px solid var(--border);
    border-bottom-width: 2px;
    border-radius: 4px;
    background: var(--view-bg);
    font: 12px var(--font);
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
    color: var(--accent);
    text-decoration: underline;
    cursor: pointer;
  }
</style>
