<script lang="ts">
  import { app } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";
  import Popover from "./Popover.svelte";

  // The primary menu. Nautilus keeps it in the sidebar's header bar.
  let btn = $state<HTMLButtonElement>();
  let open = $state(false);
</script>

<button class="btn image flat badge-host" data-main-menu bind:this={btn} class:checked={open} title="Main Menu (F10)" onclick={() => (open = !open)}>
  <Icon name="open-menu" />
  {#if app.update?.state === "ready" || app.update?.state === "manual"}<span class="badge" title="Update available"></span>{/if}
</button>

<Popover anchor={btn} bind:open align="end">
  <div class="appmenu" role="presentation" onclick={() => (open = false)}>
    {#if !app.mobile}
      <button class="modelbutton" onclick={() => app.newWindow()}>New Window<span class="accel">Ctrl+N</span></button>
      <button class="modelbutton" onclick={() => ((app.prefs.sidebarOpen = !app.prefs.sidebarOpen), app.savePrefs())}>
        <span class="checkbox" class:on={app.prefs.sidebarOpen}></span>Show Sidebar<span class="accel">F9</span>
      </button>
      <div class="psep"></div>
    {/if}
    <div class="themes" role="presentation" onclick={(e) => e.stopPropagation()}>
      {#each [["system", "Follow System Style"], ["light", "Light Style"], ["dark", "Dark Style"]] as [v, l] (v)}
        <button
          class="swatch {v}"
          class:on={app.prefs.theme === v}
          title={l}
          aria-label={l}
          onclick={() => ((app.prefs.theme = v), app.savePrefs())}
        ></button>
      {/each}
    </div>
    <div class="psep"></div>
    <button class="modelbutton" onclick={() => (app.settingsOpen = true)}>Settings<span class="accel">Ctrl+,</span></button>
    {#if !app.mobile}
      <button class="modelbutton" onclick={() => (app.modal = { kind: "shortcuts" })}>Keyboard Shortcuts<span class="accel">Ctrl+?</span></button>
    {/if}
    {#if app.update?.state === "ready"}
      <button class="modelbutton" onclick={() => app.applyUpdate()}>
        <span class="dot"></span>{app.mobile ? "Install" : "Restart to Install"} Nova {app.update.latestVersion}
      </button>
    {:else if app.update?.state === "manual"}
      <button class="modelbutton" onclick={() => (app.settingsOpen = true)}>
        <span class="dot"></span>Nova {app.update.latestVersion} Available…
      </button>
    {/if}
    <button class="modelbutton" onclick={() => (app.modal = { kind: "about" })}>About Nova</button>
  </div>
</Popover>

<style>
  .appmenu {
    display: flex;
    flex-direction: column;
    min-width: 230px;
  }
  /* The style switcher from the GNOME apps' primary menus */
  .themes {
    display: flex;
    justify-content: center;
    gap: 18px;
    padding: 6px 0;
  }
  .swatch {
    position: relative;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    border: 0;
    padding: 0;
    box-shadow: inset 0 0 0 1px var(--border);
    outline: none;
  }
  .swatch.system {
    background: linear-gradient(-45deg, #1d1d20 49.99%, #ffffff 50.01%);
  }
  .swatch.light {
    background: #ffffff;
  }
  .swatch.dark {
    background: #1d1d20;
  }
  .swatch.on {
    box-shadow: 0 0 0 2px var(--accent);
  }
  .swatch.on::after {
    content: "";
    position: absolute;
    right: -3px;
    bottom: -3px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--accent) no-repeat center / 10px
      url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3E%3Cpath d='M2 8.5l4 4 8-9' fill='none' stroke='white' stroke-width='2.5'/%3E%3C/svg%3E");
  }
  .swatch:focus-visible {
    box-shadow: 0 0 0 2px var(--focus);
  }
  .badge-host {
    position: relative;
  }
  .badge {
    position: absolute;
    top: 5px;
    right: 5px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent);
    flex: none;
  }
</style>
