<script lang="ts">
  import { tick, type Snippet } from "svelte";

  // A GtkPopover: anchored below an element with a pointing arrow.
  let {
    anchor,
    open = $bindable(false),
    align = "center",
    children,
  }: { anchor: HTMLElement | undefined; open: boolean; align?: "start" | "center" | "end"; children: Snippet } = $props();

  let el = $state<HTMLDivElement>();
  let pos = $state({ x: 0, y: 0, arrow: 0, up: false });

  $effect(() => {
    if (!open || !anchor) return;
    tick().then(() => {
      if (!el || !anchor) return;
      const a = anchor.getBoundingClientRect();
      const r = el.getBoundingClientRect();
      const cx = a.left + a.width / 2;
      let x = align === "start" ? a.left : align === "end" ? a.right - r.width : cx - r.width / 2;
      x = Math.max(6, Math.min(innerWidth - r.width - 6, x));
      // Open upwards from anchors near the bottom, like the bottom toolbar.
      const up = a.bottom + 8 + r.height > innerHeight - 6 && a.top - 8 - r.height >= 6;
      pos = { x, y: up ? a.top - 8 - r.height : a.bottom + 8, arrow: cx - x, up };
      el.querySelector<HTMLElement>("[autofocus], input, button")?.focus();
    });
  });

  function onDown(e: MouseEvent) {
    if (!open) return;
    const t = e.target as Node;
    if (el?.contains(t) || anchor?.contains(t)) return;
    open = false;
  }
</script>

<svelte:window onmousedown={onDown} onblur={() => (open = false)} />

{#if open}
  <div
    class="popover"
    role="dialog"
    tabindex="-1"
    bind:this={el}
    style:left="{pos.x}px"
    style:top="{pos.y}px"
    style:--arrow="{pos.arrow}px"
    class:up={pos.up}
    onkeydown={(e) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        open = false;
        anchor?.focus();
      }
    }}
  >
    {@render children()}
  </div>
{/if}

<style>
  .popover {
    position: fixed;
    z-index: 900;
    padding: 6px;
    background: var(--popover-bg);
    border-radius: 12px;
    box-shadow: var(--menu-shadow);
    color: var(--fg);
    animation: pop 120ms ease-out;
    --wails-draggable: no-drag;
  }
  .popover:focus {
    outline: none;
  }
  .popover::before {
    content: "";
    position: absolute;
    top: -6px;
    left: calc(var(--arrow) - 7px);
    width: 14px;
    height: 14px;
    background: var(--popover-bg);
    border-radius: 3px 0 0 0;
    transform: rotate(45deg);
  }
  .popover.up::before {
    top: auto;
    bottom: -6px;
    border-radius: 0 0 3px 0;
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: scale(0.97);
    }
  }
  :global(.popover .modelbutton) {
    position: relative;
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-height: 32px;
    padding: 0 12px;
    border: 0;
    border-radius: 6px;
    background: none;
    text-align: left;
    color: var(--fg);
    outline: none;
  }
  :global(.popover .modelbutton:hover),
  :global(.popover .modelbutton:focus-visible) {
    background: var(--hover);
  }
  :global(.popover .modelbutton:active) {
    background: var(--active);
  }
  :global(.popover .modelbutton:disabled) {
    opacity: 0.5;
    background: none;
  }
  :global(.popover .modelbutton .accel) {
    margin-left: auto;
    padding-left: 24px;
    color: var(--fg-dim);
  }
  :global(.popover .modelbutton .radio),
  :global(.popover .modelbutton .checkbox) {
    width: 14px;
    height: 14px;
    flex: none;
    position: relative;
    border-radius: 50%;
    box-shadow: inset 0 0 0 2px color-mix(in srgb, var(--fg) 30%, transparent);
  }
  :global(.popover .modelbutton .checkbox) {
    border-radius: 4px;
  }
  :global(.popover .modelbutton .radio.on) {
    box-shadow: inset 0 0 0 4px var(--accent);
  }
  :global(.popover .modelbutton .checkbox.on) {
    background: var(--accent);
    box-shadow: none;
  }
  :global(.popover .modelbutton .checkbox.on::after) {
    content: "";
    position: absolute;
    left: 4.5px;
    top: 1.5px;
    width: 3.5px;
    height: 7.5px;
    border: solid #fff;
    border-width: 0 2px 2px 0;
    transform: rotate(45deg);
  }
  :global(.popover .psep) {
    height: 1px;
    margin: 6px 0;
    background: var(--border);
    opacity: 0.6;
  }
</style>
