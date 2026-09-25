<script lang="ts">
  // One usage series as an area chart: 2px line, faint fill, recessive grid,
  // and a crosshair tooltip. The heading above names the series, so there is
  // no legend.
  let {
    label,
    timestamps,
    amounts,
    format,
    bucket,
  }: {
    label: string;
    timestamps: string[];
    amounts: number[];
    format: (v: number) => string;
    /** Bucket length in minutes, used to label the tooltip. */
    bucket: number;
  } = $props();

  const H = 168;
  const PAD = { top: 10, right: 10, bottom: 24, left: 56 };
  let width = $state(560);
  let hover = $state<number | null>(null);
  const uid = $props.id();

  const n = $derived(amounts.length);
  const max = $derived(niceMax(Math.max(0, ...amounts)));
  const plotW = $derived(Math.max(10, width - PAD.left - PAD.right));
  const plotH = H - PAD.top - PAD.bottom;
  const x = (i: number) => PAD.left + (n <= 1 ? plotW / 2 : (i / (n - 1)) * plotW);
  const y = (v: number) => PAD.top + plotH - (max ? (v / max) * plotH : 0);

  const line = $derived(amounts.map((v, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(""));
  const area = $derived(n ? `${line}L${x(n - 1).toFixed(1)},${y(0)}L${x(0).toFixed(1)},${y(0)}Z` : "");
  const ticks = $derived([0, max / 2, max]);
  const empty = $derived(!amounts.some((v) => v > 0));

  function niceMax(v: number): number {
    if (v <= 0) return 1;
    const mag = 10 ** Math.floor(Math.log10(v));
    for (const m of [1, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10]) if (v <= m * mag) return m * mag;
    return 10 * mag;
  }

  function when(i: number, long = false): string {
    const d = new Date(timestamps[i]);
    if (isNaN(d.getTime())) return "";
    if (bucket < 1440)
      return long
        ? d.toLocaleString(undefined, { weekday: "short", hour: "2-digit", minute: "2-digit" })
        : d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
    return d.toLocaleDateString(undefined, long ? { weekday: "short", day: "numeric", month: "short", year: "numeric" } : { day: "numeric", month: "short" });
  }

  const xLabels = $derived(n > 1 ? [0, Math.floor((n - 1) / 2), n - 1] : n ? [0] : []);

  function onMove(e: PointerEvent) {
    if (!n) return;
    const r = (e.currentTarget as SVGElement).getBoundingClientRect();
    const px = e.clientX - r.left;
    const i = n <= 1 ? 0 : Math.round(((px - PAD.left) / plotW) * (n - 1));
    hover = Math.max(0, Math.min(n - 1, i));
  }
</script>

<div class="chart" bind:clientWidth={width}>
  <svg
    width={width}
    height={H}
    role="img"
    aria-label="{label} over time"
    onpointermove={onMove}
    onpointerleave={() => (hover = null)}
  >
    <defs>
      <linearGradient id="fill-{uid}" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="var(--accent)" stop-opacity="0.28" />
        <stop offset="1" stop-color="var(--accent)" stop-opacity="0.02" />
      </linearGradient>
    </defs>
    {#each ticks as t, i (i)}
      <line class="grid" x1={PAD.left} x2={PAD.left + plotW} y1={y(t)} y2={y(t)} />
      <text class="tick" x={PAD.left - 8} y={y(t) + 4} text-anchor="end">{t ? format(t) : "0"}</text>
    {/each}
    {#each xLabels as i, k (i)}
      <text class="tick" x={x(i)} y={H - 6} text-anchor={k === 0 && xLabels.length > 1 ? "start" : k === xLabels.length - 1 && xLabels.length > 1 ? "end" : "middle"}>{when(i)}</text>
    {/each}
    {#if n}
      <path d={area} fill="url(#fill-{uid})" />
      <path d={line} class="line" />
    {/if}
    {#if hover !== null}
      <line class="cross" x1={x(hover)} x2={x(hover)} y1={PAD.top} y2={PAD.top + plotH} />
      <circle class="dot" cx={x(hover)} cy={y(amounts[hover])} r="4.5" />
    {/if}
  </svg>
  {#if hover !== null}
    {@const left = Math.min(Math.max(x(hover), 90), width - 90)}
    <div class="tip" style:left="{left}px">
      <div class="tip-when">{when(hover, true)}</div>
      <div class="tip-val">{format(amounts[hover])}</div>
    </div>
  {/if}
  {#if empty}<div class="empty dim">No {label.toLowerCase()} in this period</div>{/if}

  <table class="sr-only">
    <caption>{label}</caption>
    <thead><tr><th>Time</th><th>{label}</th></tr></thead>
    <tbody>
      {#each amounts as v, i (i)}<tr><td>{when(i, true)}</td><td>{format(v)}</td></tr>{/each}
    </tbody>
  </table>
</div>

<style>
  .chart {
    position: relative;
    width: 100%;
  }
  svg {
    display: block;
    touch-action: pan-y;
  }
  .grid {
    stroke: var(--border);
    stroke-width: 1;
    opacity: 0.6;
  }
  .tick {
    fill: var(--fg-dim);
    font-size: 11px;
  }
  .line {
    fill: none;
    stroke: var(--accent);
    stroke-width: 2;
    stroke-linejoin: round;
    stroke-linecap: round;
  }
  .cross {
    stroke: var(--fg-dim);
    stroke-width: 1;
    stroke-dasharray: 3 3;
  }
  .dot {
    fill: var(--accent);
    stroke: var(--dialog-bg);
    stroke-width: 2;
  }
  .tip {
    position: absolute;
    top: -6px;
    transform: translate(-50%, -100%);
    padding: 6px 10px;
    border-radius: 8px;
    background: var(--popover-bg);
    box-shadow: var(--menu-shadow);
    pointer-events: none;
    white-space: nowrap;
    text-align: center;
    z-index: 2;
  }
  .tip-when {
    color: var(--fg-dim);
    font-size: 0.87em;
  }
  .tip-val {
    font-weight: bold;
  }
  .empty {
    position: absolute;
    inset: 0 0 24px 56px;
    display: grid;
    place-items: center;
    pointer-events: none;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
</style>
