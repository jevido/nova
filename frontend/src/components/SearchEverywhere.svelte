<script lang="ts">
  import { untrack } from "svelte";
  import * as Files from "../../bindings/nova/services/filesservice";
  import type { Entry } from "../../bindings/nova/services/models";
  import { app, HOME, parentOf } from "../lib/store.svelte";
  import { formatDate, formatSize } from "../lib/format";
  import FileIcon from "./FileIcon.svelte";
  import Icon from "./Icon.svelte";

  // Nautilus' "Search Everywhere" (Ctrl+Shift+F): one entry, matches from
  // your whole storage, Enter goes there.
  let input = $state<HTMLInputElement>();
  let listEl = $state<HTMLDivElement>();
  let query = $state("");
  let results = $state<Entry[]>([]);
  let searching = $state(false);
  let error = $state("");
  let cursor = $state(0);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let seq = 0;

  // Reset when the dialog opens (only then: the rest is read untracked).
  $effect(() => {
    if (!app.globalSearchOpen) return;
    untrack(() => {
      query = app.globalSearchSeed;
      results = [];
      error = "";
      cursor = 0;
      queueMicrotask(() => {
        input?.focus();
        input?.select();
      });
      if (query.trim()) search();
    });
  });

  function onInput(v: string) {
    query = v;
    clearTimeout(timer);
    if (!v.trim()) {
      seq++;
      results = [];
      searching = false;
      error = "";
      return;
    }
    searching = true;
    timer = setTimeout(search, 250);
  }

  async function search() {
    const my = ++seq;
    const q = query.trim();
    if (!q) return;
    searching = true;
    error = "";
    try {
      const r = (await Files.Search(HOME, q)) ?? [];
      if (my !== seq) return;
      results = r;
      cursor = 0;
    } catch (e) {
      if (my === seq) error = e instanceof Error ? e.message : String(e);
    } finally {
      if (my === seq) searching = false;
    }
  }

  function close() {
    clearTimeout(timer);
    seq++;
    app.globalSearchOpen = false;
  }

  /** Folders open; files open with their app. Alt: show the item in its folder. */
  function activate(e: Entry, reveal = false) {
    close();
    if (reveal) app.navigate(parentOf(e.path), true, [e.path]);
    else app.open(e);
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close();
    } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!results.length) return;
      cursor = (cursor + (e.key === "ArrowDown" ? 1 : results.length - 1)) % results.length;
      queueMicrotask(() => listEl?.querySelector(".hit.on")?.scrollIntoView({ block: "nearest" }));
    } else if (e.key === "Enter" && results[cursor]) {
      e.preventDefault();
      activate(results[cursor], e.altKey);
    }
  }

  const where = (e: Entry) => ["Home", ...parentOf(e.path).replace(/^\/me/, "").split("/").filter(Boolean)].join(" / ");
</script>

{#if app.globalSearchOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="backdrop" onmousedown={(e) => e.target === e.currentTarget && close()} onkeydown={onKey}>
    <div class="dialog" role="dialog" aria-modal="true" aria-label="Search Everywhere">
      <div class="entry-row">
        <Icon name="system-search" size={20} />
        <input
          bind:this={input}
          value={query}
          oninput={(e) => onInput(e.currentTarget.value)}
          placeholder="Search everywhere"
          spellcheck="false"
          aria-controls="everywhere-results"
        />
        {#if searching}<span class="spinner"></span>{/if}
      </div>

      <div class="results" id="everywhere-results" bind:this={listEl} role="listbox">
        {#if error}
          <div class="status dim">Search failed: {error}</div>
        {:else if !query.trim()}
          <div class="status dim">
            <Icon name="system-search" size={64} />
            <p>Search all your files and folders by name.</p>
          </div>
        {:else if !results.length && !searching}
          <div class="status dim">
            <Icon name="system-search" size={64} />
            <p>No results found</p>
          </div>
        {:else}
          {#each results as e, i (e.path)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <div
              class="hit"
              class:on={i === cursor}
              role="option"
              tabindex="-1"
              aria-selected={i === cursor}
              onmousemove={() => (cursor = i)}
              onclick={(ev) => activate(e, ev.altKey)}
              onauxclick={(ev) => ev.button === 1 && e.isDir && !app.mobile && (close(), app.newWindow(e.path))}
            >
              <FileIcon entry={e} size={32} />
              <div class="text">
                <div class="name">{e.name}</div>
                <div class="dim where">{where(e)}</div>
              </div>
              <div class="dim meta">{e.isDir ? "" : formatSize(e.size)}<br />{formatDate(e.modified)}</div>
            </div>
          {/each}
        {/if}
      </div>
      {#if results.length && !app.mobile}
        <div class="foot dim"><kbd>↑</kbd><kbd>↓</kbd> select · <kbd>Enter</kbd> open · <kbd>Alt</kbd>+<kbd>Enter</kbd> show in folder</div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 945;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 10vh;
    background: rgba(0, 0, 0, 0.25);
  }
  .dialog {
    width: 640px;
    max-width: calc(100vw - 32px);
    max-height: 72vh;
    display: flex;
    flex-direction: column;
    background: var(--dialog-bg);
    border-radius: 15px;
    box-shadow: var(--dialog-shadow);
    overflow: hidden;
    animation: dlg-in 100ms ease-out;
  }
  @keyframes dlg-in {
    from {
      opacity: 0;
    }
  }
  .entry-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 18px;
    min-height: 58px;
    box-shadow: inset 0 -1px var(--shade);
  }
  .entry-row :global(.icon) {
    opacity: 0.6;
  }
  .entry-row input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: none;
    outline: none;
    font-size: 1.2em;
    user-select: text;
    -webkit-user-select: text;
  }
  .entry-row input::placeholder {
    color: var(--fg-dim);
  }
  .results {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 6px;
  }
  .hit {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 10px;
    border-radius: 8px;
    cursor: default;
  }
  .hit.on {
    background: var(--selected);
  }
  .text {
    flex: 1;
    min-width: 0;
  }
  .name,
  .where {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .where,
  .meta {
    font-size: 0.87em;
  }
  .meta {
    text-align: right;
    white-space: nowrap;
    line-height: 1.4;
  }
  .status {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 40px 20px;
    text-align: center;
  }
  .status :global(.icon) {
    opacity: 0.4;
  }
  .status p {
    margin: 8px 0 0;
  }
  .foot {
    padding: 8px 16px;
    font-size: 0.87em;
    box-shadow: inset 0 1px var(--shade);
  }
  kbd {
    display: inline-block;
    min-width: 18px;
    padding: 0 5px;
    margin: 0 1px;
    border-radius: 5px;
    background: var(--btn-bg);
    font: bold 0.9em var(--font);
    text-align: center;
  }
  :global(:root[data-mobile="true"]) .backdrop {
    padding-top: 0;
    align-items: stretch;
  }
  :global(:root[data-mobile="true"]) .dialog {
    width: 100vw;
    max-width: none;
    max-height: none;
    border-radius: 0;
  }
</style>
