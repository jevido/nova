<script lang="ts">
  import * as Updates from "../../bindings/nova/services/updateservice";
  import { releases, formatDate, type Release } from "../lib/changelog";
  import Icon from "./Icon.svelte";

  // The releases to feature (new since the last run); the rest are listed
  // under Earlier Releases. Empty features the newest.
  let { versions, onClose }: { versions: string[]; onClose: () => void } = $props();

  const featured: Release[] = $derived.by(() => {
    const f = releases.filter((r) => versions.includes(r.version));
    return f.length ? f : releases.slice(0, 1);
  });
  const earlier = $derived(releases.filter((r) => !featured.includes(r)));
  const top = $derived(featured[0]);

  let continueBtn = $state<HTMLButtonElement>();
  $effect(() => continueBtn?.focus());

  const tagClass = (kind: string) => kind.toLowerCase().replace(/[^a-z]/g, "");
</script>

<div class="dialog whatsnew" role="dialog" aria-modal="true" aria-label="What's New">
  <button class="btn image flat round close" title="Close" onclick={onClose}><Icon name="window-close" /></button>
  <div class="scroll">
    <header class="hero">
      <img src="/nova.png" alt="" width="80" height="80" />
      <h2>What's New</h2>
      {#if top}
        <div class="ver">
          <span class="pill">Nova {top.version}</span>
          {#if top.date}<span class="dim">{formatDate(top.date)}</span>{/if}
        </div>
        {#if top.summary}<p class="summary">{top.summary}</p>{/if}
      {/if}
    </header>

    {#each featured as rel, r (rel.version)}
      <section class="release">
        {#if r > 0}
          <h3 class="relhead">Nova {rel.version}{#if rel.date}<span class="dim"> · {formatDate(rel.date)}</span>{/if}</h3>
          {#if rel.summary}<p class="summary small">{rel.summary}</p>{/if}
        {/if}
        {#each rel.sections as sec (sec.kind)}
          {#if tagClass(sec.kind) === "new"}
            <div class="cards">
              {#each sec.items as it, i (i)}
                <div class="card" style:animation-delay="{60 + i * 45}ms">
                  <span class="ico"><Icon name={it.icon || "starred"} size={20} /></span>
                  <div class="card-text">
                    {#if it.title}<div class="title">{it.title}</div>{/if}
                    <div class="text">{it.text}</div>
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <div class="minor">
              <span class="tag {tagClass(sec.kind)}">{sec.kind}</span>
              <ul>
                {#each sec.items as it, i (i)}
                  <li>{#if it.title}<b>{it.title}</b>{/if}<span class="dim">{it.text}</span></li>
                {/each}
              </ul>
            </div>
          {/if}
        {/each}
      </section>
    {/each}

    {#if earlier.length}
      <details class="earlier">
        <summary><span>Earlier Releases</span><Icon name="pan-down" /></summary>
        {#each earlier as rel (rel.version)}
          <div class="old">
            <div class="old-head"><b>Nova {rel.version}</b>{#if rel.date}<span class="dim">{formatDate(rel.date)}</span>{/if}</div>
            <ul>
              {#each rel.sections as sec (sec.kind)}
                {#each sec.items as it, i (i)}
                  <li><span class="tag mini {tagClass(sec.kind)}">{sec.kind}</span>{#if it.title}<b>{it.title}</b>{/if}<span class="dim">{it.text}</span></li>
                {/each}
              {/each}
            </ul>
          </div>
        {/each}
      </details>
    {/if}
  </div>
  <footer>
    <button class="link" onclick={() => Updates.OpenReleasePage()}>Release notes on GitHub</button>
    <button class="btn suggested pill-btn" bind:this={continueBtn} onclick={onClose}>Continue</button>
  </footer>
</div>

<style>
  .whatsnew {
    position: relative;
    display: flex;
    flex-direction: column;
    width: 560px;
    max-width: calc(100vw - 32px);
    height: min(720px, calc(100vh - 32px));
    overflow: hidden;
    background: var(--dialog-bg);
    border-radius: 15px;
    box-shadow: var(--dialog-shadow);
    animation: pop 200ms ease-out;
  }
  .close {
    position: absolute;
    top: 8px;
    right: 8px;
    z-index: 2;
    background: var(--btn-bg);
  }
  .close:hover {
    background: var(--btn-hover);
  }
  .scroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0 24px 16px;
  }

  /* ---- header ---- */
  .hero {
    display: flex;
    flex-direction: column;
    align-items: center;
    margin: 0 -24px;
    padding: 36px 32px 20px;
    text-align: center;
    background:
      radial-gradient(120% 90% at 50% -10%, color-mix(in srgb, var(--accent) 28%, transparent), transparent 70%),
      linear-gradient(to bottom, color-mix(in srgb, var(--accent) 6%, transparent), transparent);
  }
  .hero img {
    filter: drop-shadow(0 6px 18px color-mix(in srgb, var(--accent) 45%, transparent));
    animation: pop 420ms cubic-bezier(0.2, 0.9, 0.3, 1.3);
  }
  .hero h2 {
    margin: 14px 0 8px;
    font-size: 1.75em;
    font-weight: 800;
    letter-spacing: -0.01em;
  }
  .ver {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 0.93em;
  }
  .pill {
    padding: 2px 10px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    color: var(--accent-text);
    font-weight: 700;
  }
  .summary {
    max-width: 400px;
    margin: 14px 0 0;
    line-height: 1.45;
  }
  .summary.small {
    margin: 0 0 12px;
    color: var(--fg-dim);
  }

  /* ---- releases ---- */
  .release {
    margin-top: 12px;
  }
  .relhead {
    margin: 22px 0 8px;
    font-size: 1.05em;
  }
  .cards {
    display: grid;
    gap: 8px;
  }
  .card {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 12px 14px;
    border-radius: 12px;
    background: var(--card-bg);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--fg) 5%, transparent);
    animation: rise 320ms ease-out both;
  }
  .ico {
    flex: none;
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    border-radius: 11px;
    background: color-mix(in srgb, var(--accent) 16%, transparent);
    color: var(--accent-text);
  }
  .card-text {
    min-width: 0;
    padding-top: 1px;
  }
  .title {
    font-weight: 700;
  }
  .text {
    margin-top: 2px;
    color: var(--fg-dim);
    line-height: 1.4;
  }
  .minor {
    margin-top: 18px;
  }
  .minor ul,
  .old ul {
    margin: 8px 0 0;
    padding: 0;
    list-style: none;
  }
  .minor li {
    position: relative;
    padding: 5px 0 5px 18px;
    line-height: 1.4;
  }
  .minor li::before {
    content: "";
    position: absolute;
    left: 4px;
    top: 0.95em;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent);
    opacity: 0.7;
  }
  .tag {
    display: inline-block;
    padding: 2px 9px;
    border-radius: 999px;
    font-size: 0.8em;
    font-weight: 700;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    background: color-mix(in srgb, var(--accent) 16%, transparent);
    color: var(--accent-text);
  }
  .tag.improved {
    background: color-mix(in srgb, var(--success) 18%, transparent);
    color: color-mix(in srgb, var(--success) 70%, var(--fg));
  }
  .tag.fixed {
    background: color-mix(in srgb, var(--warning) 20%, transparent);
    color: color-mix(in srgb, var(--warning) 65%, var(--fg));
  }
  .tag.mini {
    margin-right: 8px;
    padding: 0 7px;
    font-size: 0.7em;
    vertical-align: 1px;
  }

  /* ---- earlier releases ---- */
  .earlier {
    margin-top: 24px;
    border-radius: 12px;
    background: var(--card-bg);
  }
  .earlier summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    font-weight: 700;
    cursor: pointer;
    list-style: none;
    border-radius: 12px;
  }
  .earlier summary::-webkit-details-marker {
    display: none;
  }
  .earlier summary:hover {
    background: var(--hover);
  }
  .earlier summary :global(.icon) {
    transition: transform 150ms ease-out;
  }
  .earlier[open] summary :global(.icon) {
    transform: rotate(180deg);
  }
  .old {
    padding: 10px 14px 12px;
    border-top: 1px solid var(--border);
  }
  .old-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }
  .old li {
    padding: 3px 0;
    line-height: 1.4;
  }
  .minor li b,
  .old li b {
    margin-right: 6px;
  }

  /* ---- footer ---- */
  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 12px 16px;
    box-shadow: 0 -1px var(--border);
  }
  .link {
    padding: 4px 6px;
    border: 0;
    background: none;
    color: var(--accent-text);
    cursor: pointer;
  }
  .link:hover {
    text-decoration: underline;
  }
  .pill-btn {
    min-width: 120px;
    border-radius: 999px;
    font-weight: 700;
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(6px);
    }
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: scale(0.8);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .card,
    .hero img {
      animation: none;
    }
  }
  :global(:root[data-mobile="true"]) .whatsnew {
    width: 100vw;
    height: 100vh;
    max-width: none;
    max-height: none;
    border-radius: 0;
  }
</style>
