<script lang="ts">
  import { app, ZOOM_SIZES } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";
  import Popover from "./Popover.svelte";

  // Grid/list toggle plus the view options dropdown.
  let btn = $state<HTMLButtonElement>();
  let open = $state(false);

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

<div class="linked split">
  <button
    class="btn image flat"
    title={app.prefs.view === "grid" ? "Show list (Ctrl+1)" : "Show grid (Ctrl+2)"}
    onclick={() => {
      app.prefs.view = app.prefs.view === "grid" ? "list" : "grid";
      app.savePrefs();
    }}
  >
    <Icon name={app.prefs.view === "grid" ? "view-list" : "view-grid"} />
  </button>
  <span class="split-sep"></span>
  <button class="btn image flat narrow" bind:this={btn} class:checked={open} title="View options" onclick={() => (open = !open)}>
    <Icon name="pan-down" />
  </button>
</div>

<Popover anchor={btn} bind:open align="end">
  <div class="viewopts">
    <div class="zoom">
      <button class="btn image flat round-lg" title="Zoom out (Ctrl+−)" disabled={app.prefs.zoom <= 0} onclick={() => setZoom(-1)}><Icon name="zoom-out" /></button>
      <button class="btn flat" title="Reset zoom (Ctrl+0)" onclick={() => ((app.prefs.zoom = 1), app.savePrefs())}>{Math.round((ZOOM_SIZES[app.prefs.zoom] / 64) * 100)}%</button>
      <button class="btn image flat round-lg" title="Zoom in (Ctrl++)" disabled={app.prefs.zoom >= ZOOM_SIZES.length - 1} onclick={() => setZoom(1)}><Icon name="zoom-in" /></button>
    </div>
    <div class="psep"></div>
    <div class="label">Sort</div>
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
  </div>
</Popover>

<style>
  .split {
    align-items: center;
  }
  .split-sep {
    width: 1px;
    height: 16px;
    background: var(--border);
  }
  .narrow {
    min-width: 24px;
    padding: 5px 4px;
  }
  .viewopts {
    display: flex;
    flex-direction: column;
    min-width: 230px;
  }
  .zoom {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 2px;
  }
  .round-lg {
    border-radius: 50%;
  }
  .label {
    padding: 6px 12px 4px;
    font-size: 0.87em;
    font-weight: bold;
    color: var(--fg-dim);
  }
</style>
