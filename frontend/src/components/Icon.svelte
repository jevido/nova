<script lang="ts">
  import { iconUrl } from "../lib/icons";

  // Symbolic icons are tinted with currentColor through a CSS mask, like GTK does.
  let { name, size = 16 }: { name: string; size?: number } = $props();
</script>

<span
  class="icon"
  aria-hidden="true"
  style:width="{size}px"
  style:height="{size}px"
  style:--src="url({iconUrl(name + '-symbolic', name)})"
></span>

<style>
  .icon {
    display: inline-block;
    flex: none;
    background: currentColor;
    -webkit-mask: var(--src) center / contain no-repeat;
    mask: var(--src) center / contain no-repeat;
  }
</style>
