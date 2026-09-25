<script lang="ts">
  import type { Entry } from "../../bindings/nova/services/models";
  import { fileIconUrl, hasThumbnail, thumbUrl } from "../lib/icons";

  let { entry, size }: { entry: Entry; size: number } = $props();

  let failed = $state(false);
  let loaded = $state(false);
  const thumb = $derived(size >= 32 && hasThumbnail(entry) && !failed);
</script>

<span class="wrap" style:width="{size}px" style:height="{size}px">
  {#if thumb}
    <img
      class="thumb"
      class:loaded
      src={thumbUrl(entry, size * (window.devicePixelRatio || 1))}
      alt=""
      loading="lazy"
      draggable="false"
      onload={() => (loaded = true)}
      onerror={() => (failed = true)}
    />
  {/if}
  {#if !thumb || !loaded}
    <img class="icon" src={fileIconUrl(entry)} alt="" draggable="false" />
  {/if}
  {#if entry.shared}
    <span class="emblem" title="Shared publicly"></span>
  {/if}
</span>

<style>
  .wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
  }
  img {
    max-width: 100%;
    max-height: 100%;
    pointer-events: none;
  }
  .icon {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
  .thumb {
    position: absolute;
    object-fit: contain;
    border-radius: 2px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.35);
    opacity: 0;
  }
  .thumb.loaded {
    position: static;
    opacity: 1;
  }
  .emblem {
    position: absolute;
    right: 0;
    bottom: 0;
    width: max(12px, 30%);
    height: max(12px, 30%);
    border-radius: 50%;
    background: var(--view-bg);
  }
  .emblem::after {
    content: "";
    position: absolute;
    inset: 12%;
    background: var(--accent);
    -webkit-mask: url(/nova/icon/emblem-shared-symbolic,emblem-shared,insert-link-symbolic) center / contain no-repeat;
    mask: url(/nova/icon/emblem-shared-symbolic,emblem-shared,insert-link-symbolic) center / contain no-repeat;
  }
</style>
