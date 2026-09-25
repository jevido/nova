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
  let pos = $state({ x: 0, y: 0, arrow: 0 });

  $effect(() => {
    if (!open || !anchor) return;
    tick().then(() => {
      if (!el || !anchor) return;
      const a = anchor.getBoundingClientRect();
      const r = el.getBoundingClientRect();
      const cx = a.left + a.width / 2;
      let x = align === "start" ? a.left : align === "end" ? a.right - r.width : cx - r.width / 2;
      x = Math.max(6, Math.min(innerWidth - r.width - 6, x));
      pos = { x, y: a.bottom + 8, arrow: cx - x };
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
    border: 1px solid rgba(0, 0, 0, 0.23);
    border-radius: var(--radius);
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.2);
    color: var(--fg);
    animation: pop 100ms ease-out;
    --wails-draggable: no-drag;
  }
  .popover:focus {
    outline: none;
  }
  .popover::before {
    content: "";
    position: absolute;
    top: -7px;
    left: calc(var(--arrow) - 7px);
    width: 12px;
    height: 12px;
    background: var(--popover-bg);
    border-left: 1px solid rgba(0, 0, 0, 0.23);
    border-top: 1px solid rgba(0, 0, 0, 0.23);
    transform: rotate(45deg);
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
  }
  :global(.popover .modelbutton) {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-height: 30px;
    padding: 4px 10px;
    border: 0;
    border-radius: 4px;
    background: none;
    text-align: left;
    color: var(--fg);
    outline: none;
  }
  :global(.popover .modelbutton:hover),
  :global(.popover .modelbutton:focus-visible) {
    background: var(--row-hover);
    background: color-mix(in srgb, var(--fg) 8%, transparent);
  }
  :global(.popover .modelbutton .accel) {
    margin-left: auto;
    padding-left: 20px;
    color: var(--fg-dim);
    font-size: 13px;
  }
  :global(.popover .modelbutton .radio) {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 1px solid var(--border-dark);
    background: var(--entry-bg);
    flex: none;
  }
  :global(.popover .modelbutton .radio.on) {
    border: 4px solid var(--accent);
  }
  :global(.popover .modelbutton .checkbox) {
    width: 14px;
    height: 14px;
    border-radius: 3px;
    border: 1px solid var(--border-dark);
    background: var(--entry-bg);
    flex: none;
    position: relative;
  }
  :global(.popover .modelbutton .checkbox.on) {
    background: var(--accent);
    border-color: var(--accent-dim);
  }
  :global(.popover .modelbutton .checkbox.on::after) {
    content: "";
    position: absolute;
    left: 4px;
    top: 1px;
    width: 4px;
    height: 8px;
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
