<script lang="ts">
  import { tick } from "svelte";
  import { app, type MenuButton, type MenuItem } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";

  let el = $state<HTMLDivElement>();
  let pos = $state({ x: 0, y: 0 });
  /** The focused item, and the button within it when it is a row. */
  let focused = $state<{ i: number; j: number } | null>(null);

  const items = $derived(app.menu?.items ?? []);

  /** Everything the keyboard can land on, in order. */
  const stops = $derived(
    items.flatMap((it, i) => {
      if ("sep" in it && it.sep) return [];
      if (it.row) return it.row.flatMap((b, j) => (b.disabled ? [] : [{ i, j }]));
      return it.disabled ? [] : [{ i, j: -1 }];
    }),
  );

  const isFocused = (i: number, j = -1) => focused?.i === i && focused.j === j;

  function tip(b: MenuButton) {
    return b.accel && !app.mobile ? `${b.label} (${b.accel})` : b.label;
  }

  $effect(() => {
    const m = app.menu;
    if (!m) return;
    focused = null;
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

  function activate(item: { disabled?: boolean; run: () => void }) {
    if (item.disabled) return;
    close();
    item.run();
  }

  function onKey(e: KeyboardEvent) {
    const cur = focused ? stops.findIndex((s) => s.i === focused!.i && s.j === focused!.j) : -1;
    if (e.key === "Escape") {
      close();
    } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      // Up and down move by line; a row counts as one line.
      if (!stops.length) return;
      const down = e.key === "ArrowDown";
      const n = stops.length;
      let k = cur < 0 ? (down ? 0 : n - 1) : cur;
      if (cur >= 0) {
        do k = (k + (down ? 1 : n - 1)) % n;
        while (stops[k].i === stops[cur].i && k !== cur);
      }
      // Land on the first button of a row.
      const first = stops.findIndex((s) => s.i === stops[k].i);
      focused = stops[first];
    } else if ((e.key === "ArrowLeft" || e.key === "ArrowRight") && focused && focused.j >= 0) {
      const next = stops[cur + (e.key === "ArrowRight" ? 1 : -1)];
      if (next?.i === focused.i) focused = next;
    } else if (e.key === "Enter" || e.key === " ") {
      if (!focused) return;
      const it = items[focused.i];
      if ("sep" in it && it.sep) return;
      activate(it.row ? it.row[focused.j] : it);
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
      {:else if it.row}
        <div class="row" role="group">
          {#each it.row as b, j (j)}
            <button
              class="rowbtn"
              class:focused={isFocused(i, j)}
              role="menuitem"
              title={tip(b)}
              aria-label={b.label}
              disabled={b.disabled}
              onmouseenter={() => (focused = { i, j })}
              onclick={() => activate(b)}
            >
              <Icon name={b.icon} size={app.mobile ? 20 : 16} />
            </button>
          {/each}
        </div>
      {:else}
        {@const item = it}
        <button
          class="item"
          class:focused={isFocused(i)}
          role={item.checked === undefined ? "menuitem" : "menuitemcheckbox"}
          aria-checked={item.checked}
          disabled={item.disabled}
          onmouseenter={() => (focused = { i, j: -1 })}
          onclick={() => activate(item)}
        >
          {#if items.some((x) => !("sep" in x && x.sep) && !x.row && x.checked !== undefined)}
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
