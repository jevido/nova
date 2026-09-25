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

  let sidebarWidth = $state(200);
  let width = $state(innerWidth);
  const showSidebar = $derived(app.prefs.sidebarOpen && width >= 560);

  onMount(() => {
    try {
      sidebarWidth = Number(localStorage.getItem("sidebarWidth")) || 200;
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
    const move = (ev: MouseEvent) => (sidebarWidth = Math.max(140, Math.min(400, w0 + ev.clientX - x0)));
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
    if (!app.session?.signedIn || app.modal || app.menu) return;
    const ctrl = e.ctrlKey || e.metaKey;
    const k = e.key.length === 1 ? e.key.toLowerCase() : e.key;
    const typing = inText(e);
    let handled = true;

    if (ctrl && k === "q") Window.Close();
    else if (ctrl && k === "w") Window.Close();
    else if (ctrl && !e.shiftKey && k === "n") app.newWindow();
    else if (ctrl && k === "l") app.editingLocation = true;
    else if (ctrl && k === "f") app.searchOpen ? app.closeSearch() : app.openSearch();
    else if ((ctrl && k === "r") || k === "F5") app.reload();
    else if (ctrl && k === "h") (app.prefs.showHidden = !app.prefs.showHidden), app.savePrefs();
    else if (ctrl && k === "1") (app.prefs.view = "list"), app.savePrefs();
    else if (ctrl && k === "2") (app.prefs.view = "grid"), app.savePrefs();
    else if (ctrl && (k === "+" || k === "=")) setZoom(app.prefs.zoom + 1);
    else if (ctrl && k === "-") setZoom(app.prefs.zoom - 1);
    else if (ctrl && k === "0") setZoom(1);
    else if (k === "F9") (app.prefs.sidebarOpen = !app.prefs.sidebarOpen), app.savePrefs();
    else if (k === "F10") (document.querySelector('.headerbar [title^="Menu"]') as HTMLElement)?.click();
    else if (ctrl && (k === "?" || (e.shiftKey && k === "/"))) app.modal = { kind: "shortcuts" };
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
  onmouseup={(e) => {
    // Mouse side buttons navigate history.
    if (!app.session?.signedIn) return;
    if (e.button === 3) app.back();
    else if (e.button === 4) app.forward();
  }}
  onfocus={() => {
    // No file monitors on a remote FS: refresh when the user comes back.
    if (app.session?.signedIn && !app.loading && !app.modal && !app.renaming) app.reload(true);
  }}
/>

{#if app.booting}
  <div class="boot"><span class="spinner"></span></div>
{:else if !app.session?.signedIn}
  <Login />
{:else}
  <div class="window">
    <HeaderBar />
    <div class="body">
      {#if showSidebar}
        <div class="side" style:width="{sidebarWidth}px">
          <Sidebar />
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div class="grip" onmousedown={startResize}></div>
        </div>
      {/if}
      <FileView />
      <Toasts />
    </div>
  </div>
{/if}

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
    height: 100%;
    display: flex;
    flex-direction: column;
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
  .grip {
    position: absolute;
    top: 0;
    right: -3px;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 4;
  }
</style>
