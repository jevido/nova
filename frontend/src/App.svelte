<script lang="ts">
  import { Window } from "@wailsio/runtime";
  import { onMount } from "svelte";
  import { app, HOME, ZOOM_SIZES } from "./lib/store.svelte";
  import FileView from "./components/FileView.svelte";
  import HeaderBar from "./components/HeaderBar.svelte";
  import Login from "./components/Login.svelte";
  import Menu from "./components/Menu.svelte";
  import Modals from "./components/Modals.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Toasts from "./components/Toasts.svelte";
  import Settings from "./components/Settings.svelte";
  import SearchEverywhere from "./components/SearchEverywhere.svelte";
  import Icon from "./components/Icon.svelte";
  import ViewControls from "./components/ViewControls.svelte";
  import MobileBar from "./components/MobileBar.svelte";

  let sidebarWidth = $state(240);
  let width = $state(innerWidth);
  // Wide windows dock the sidebar; phones and narrow windows use a drawer.
  const docked = $derived(!app.mobile && width >= 560);
  const showSidebar = $derived(docked && app.prefs.sidebarOpen);

  onMount(() => {
    try {
      sidebarWidth = Number(localStorage.getItem("sidebarWidth")) || 240;
    } catch {
      /* storage may be unavailable */
    }
    app.boot();
    const mq = matchMedia("(prefers-color-scheme: dark)");
    const onScheme = () => app.applyTheme();
    mq.addEventListener("change", onScheme);
    return () => mq.removeEventListener("change", onScheme);
  });

  function startResize(e: MouseEvent) {
    e.preventDefault();
    const x0 = e.clientX;
    const w0 = sidebarWidth;
    const move = (ev: MouseEvent) => (sidebarWidth = Math.max(180, Math.min(400, w0 + ev.clientX - x0)));
    const up = () => {
      removeEventListener("mousemove", move);
      removeEventListener("mouseup", up);
      try {
        localStorage.setItem("sidebarWidth", String(sidebarWidth));
      } catch {
        /* ignore */
      }
    };
    addEventListener("mousemove", move);
    addEventListener("mouseup", up);
  }

  function toggleSidebar() {
    if (docked) {
      app.prefs.sidebarOpen = !app.prefs.sidebarOpen;
      app.savePrefs();
    } else {
      app.drawerOpen = !app.drawerOpen;
    }
  }

  function inText(e: KeyboardEvent) {
    const t = e.target as HTMLElement;
    return t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.isContentEditable;
  }

  function setZoom(z: number) {
    app.prefs.zoom = Math.max(0, Math.min(ZOOM_SIZES.length - 1, z));
    app.savePrefs();
  }

  // Global shortcuts, matching Nautilus where possible.
  function onKey(e: KeyboardEvent) {
    if (!app.session?.signedIn || app.modal || app.menu || app.settingsOpen || app.globalSearchOpen) return;
    if (e.key === "Escape" && app.drawerOpen) {
      app.drawerOpen = false;
      e.preventDefault();
      return;
    }
    const ctrl = e.ctrlKey || e.metaKey;
    const k = e.key.length === 1 ? e.key.toLowerCase() : e.key;
    const typing = inText(e);
    let handled = true;

    if (ctrl && k === "q") Window.Close();
    else if (ctrl && k === "w") Window.Close();
    else if (ctrl && !e.shiftKey && k === "n") app.newWindow();
    else if (ctrl && k === "l") app.editingLocation = true;
    else if (ctrl && e.shiftKey && k === "f") app.searchEverywhere();
    else if (ctrl && k === "f") app.searchOpen ? app.closeSearch() : app.openSearch();
    else if ((ctrl && k === "r") || k === "F5") app.reload();
    else if (ctrl && k === "h") (app.prefs.showHidden = !app.prefs.showHidden), app.savePrefs();
    else if (ctrl && k === "1") (app.prefs.view = "list"), app.savePrefs();
    else if (ctrl && k === "2") (app.prefs.view = "grid"), app.savePrefs();
    else if (ctrl && (k === "+" || k === "=")) setZoom(app.prefs.zoom + 1);
    else if (ctrl && k === "-") setZoom(app.prefs.zoom - 1);
    else if (ctrl && k === "0") setZoom(1);
    else if (k === "F9") (app.prefs.sidebarOpen = !app.prefs.sidebarOpen), app.savePrefs();
    else if (k === "F10") {
      // The main menu lives in the sidebar; open the drawer first when it is hidden.
      if (!docked && !app.drawerOpen) app.drawerOpen = true;
      queueMicrotask(() => (document.querySelector("[data-main-menu]") as HTMLElement)?.click());
    }
    else if (ctrl && (k === "?" || (e.shiftKey && k === "/"))) app.modal = { kind: "shortcuts" };
    else if (ctrl && k === ",") app.settingsOpen = true;
    else if (e.altKey && k === "ArrowLeft") app.back();
    else if (e.altKey && k === "ArrowRight") app.forward();
    else if (e.altKey && k === "ArrowUp") app.up();
    else if (e.altKey && k === "Home") app.navigate(HOME);
    else if (ctrl && k === "d") app.toggleBookmark();
    else if (typing) handled = false;
    else if (k === "Backspace") app.back();
    else if (ctrl && e.shiftKey && k === "n") app.newFolder();
    else if (ctrl && k === "u") app.pickUpload(false);
    else if (ctrl && e.shiftKey && k === "i") app.invertSelection();
    else if (ctrl && k === "i") app.selection[0] && (app.modal = { kind: "properties", entry: app.selection[0] });
    else if (ctrl && k === "a") app.selectAll();
    else if (ctrl && k === "c") app.copy(false);
    else if (ctrl && k === "x") app.copy(true);
    else if (ctrl && k === "v") app.paste();
    else if (ctrl && k === "z") app.undo();
    else if (k === "F2") app.startRename();
    else if (k === "Delete" && e.shiftKey) app.deleteForever();
    else if (k === "Delete") app.trash();
    else handled = false;

    if (handled) e.preventDefault();
  }
</script>

<svelte:window
  bind:innerWidth={width}
  onkeydown={onKey}
  oncontextmenu={(e) => e.preventDefault()}
  onmousedown={(e) => (e.button === 3 || e.button === 4) && e.preventDefault()}
  onmouseup={(e) => {
    // Mouse side buttons navigate history. (On Linux they arrive from Go as
    // the mouse:nav event instead; WebKitGTK doesn't pass them to the page.)
    if (e.button !== 3 && e.button !== 4) return;
    e.preventDefault();
    if (!app.session?.signedIn || app.modal || app.settingsOpen) return;
    if (e.button === 3) app.back();
    else app.forward();
  }}
  onfocus={() => {
    // No file monitors on a remote FS: refresh when the user comes back.
    if (app.session?.signedIn && !app.loading && !app.modal && !app.renaming) {
      app.reload(true);
      app.syncBookmarks(false);
    }
  }}
/>

{#if app.booting}
  <div class="boot"><span class="spinner"></span></div>
{:else if !app.session?.signedIn}
  <Login />
{:else}
  <div class="window">
    {#if showSidebar}
      <div class="side" style:width="{sidebarWidth}px">
        <Sidebar />
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="grip" onmousedown={startResize}></div>
      </div>
    {/if}
    <div class="content">
      <HeaderBar compact={!docked} sidebarShown={showSidebar} onToggleSidebar={toggleSidebar} />
      <div class="body">
        <FileView />
        <Toasts />
      </div>
      {#if app.mobile}
        <MobileBar />
      {:else if !docked}
        <footer class="bottombar">
          <button class="btn image flat" title="Back" disabled={!app.history.length} onclick={() => app.back()}><Icon name="go-previous" /></button>
          <button class="btn image flat" title="Forward" disabled={!app.future.length} onclick={() => app.forward()}><Icon name="go-next" /></button>
          <span class="grow"></span>
          <ViewControls />
        </footer>
      {/if}
    </div>
    {#if !docked && app.drawerOpen}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="scrim" onclick={() => (app.drawerOpen = false)}></div>
      <div class="drawer"><Sidebar onHide={() => (app.drawerOpen = false)} /></div>
    {/if}
  </div>
{/if}

{#if app.session?.signedIn}<Settings /><SearchEverywhere />{/if}
<Menu />
<Modals />

<style>
  .boot {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .window {
    position: relative;
    height: 100%;
    display: flex;
    background: var(--view-bg);
  }
  .content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .bottombar {
    display: flex;
    align-items: center;
    gap: 4px;
    min-height: 46px;
    padding: 6px 7px;
    background: var(--header-bg);
    box-shadow: 0 -1px var(--shade);
  }
  .grow {
    flex: 1;
  }
  .body {
    position: relative;
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .side {
    position: relative;
    flex: none;
  }
  .scrim {
    position: absolute;
    inset: 0;
    z-index: 20;
    background: rgba(0, 0, 0, 0.35);
    animation: fade 150ms ease-out;
  }
  .drawer {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    z-index: 21;
    width: min(280px, 82vw);
    box-shadow: 0 0 0 1px var(--shade), 2px 0 16px rgba(0, 0, 0, 0.3);
    animation: slide-in 180ms ease-out;
  }
  /* Phones keep the bottom bar visible under the drawer. */
  :global(:root[data-mobile="true"]) .scrim,
  :global(:root[data-mobile="true"]) .drawer {
    bottom: calc(64px + env(safe-area-inset-bottom, 0px));
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes slide-in {
    from {
      transform: translateX(-100%);
    }
  }
  .grip {
    position: absolute;
    top: 0;
    right: -3px;
    --wails-draggable: no-drag;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 4;
  }
</style>
