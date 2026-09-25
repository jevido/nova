<script lang="ts">
  import { tick } from "svelte";
  import { app, type MenuItem } from "../lib/store.svelte";

  let el = $state<HTMLDivElement>();
  let pos = $state({ x: 0, y: 0 });
  let focused = $state(-1);

  const items = $derived(app.menu?.items ?? []);

  $effect(() => {
    const m = app.menu;
    if (!m) return;
    focused = -1;
    pos = { x: m.x, y: m.y };
    // Keep the menu on screen, flipping like GTK does near edges.
    tick().then(() => {
      if (!el) return;
      const r = el.getBoundingClientRect();
      let { x, y } = m;
      if (x + r.width > innerWidth - 4) x = Math.max(4, x - r.width);
      if (y + r.height > innerHeight - 4) y = Math.max(4, innerHeight - r.height - 4);
      pos = { x, y };
      el.focus();
    });
  });

  function close() {
    app.menu = null;
  }

  function activate(it: MenuItem) {
    if ("sep" in it && it.sep) return;
    const item = it as Exclude<MenuItem, { sep: true }>;
    if (item.disabled) return;
    close();
    item.run();
  }

  function onKey(e: KeyboardEvent) {
    const actionable = items.map((it, i) => (!("sep" in it && it.sep) && !(it as { disabled?: boolean }).disabled ? i : -1)).filter((i) => i >= 0);
    if (e.key === "Escape") {
      close();
    } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      const cur = actionable.indexOf(focused);
      const next = e.key === "ArrowDown" ? cur + 1 : cur <= 0 ? actionable.length - 1 : cur - 1;
      focused = actionable[next % actionable.length] ?? -1;
    } else if (e.key === "Enter" || e.key === " ") {
      if (focused >= 0) activate(items[focused]);
    } else {
      return;
    }
    e.preventDefault();
    e.stopPropagation();
  }
</script>

<svelte:window
  onmousedown={(e) => {
    if (app.menu && el && !el.contains(e.target as Node)) close();
  }}
  onblur={close}
  onresize={close}
/>

{#if app.menu}
  <div
    class="menu"
    role="menu"
    tabindex="-1"
    bind:this={el}
    style:left="{pos.x}px"
    style:top="{pos.y}px"
    onkeydown={onKey}
    oncontextmenu={(e) => e.preventDefault()}
  >
    {#each items as it, i (i)}
      {#if "sep" in it && it.sep}
        <div class="sep" role="separator"></div>
      {:else}
        {@const item = it as Exclude<MenuItem, { sep: true }>}
        <button
          class="item"
          class:focused={focused === i}
          role={item.checked === undefined ? "menuitem" : "menuitemcheckbox"}
          aria-checked={item.checked}
          disabled={item.disabled}
          onmouseenter={() => (focused = i)}
          onclick={() => activate(item)}
        >
          {#if items.some((x) => !("sep" in x && x.sep) && (x as { checked?: boolean }).checked !== undefined)}
            <span class="check">{item.checked ? "✓" : ""}</span>
          {/if}
          <span>{item.label}</span>
          {#if item.accel}<span class="accel">{item.accel}</span>{/if}
        </button>
      {/if}
    {/each}
  </div>
{/if}

<style>
  .menu:focus {
    outline: none;
  }
</style>
