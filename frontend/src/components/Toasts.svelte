<script lang="ts">
  import { app } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";
</script>

<div class="toasts" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="app-notification" class:error={t.error} role="status">
      <span class="text">{t.text}</span>
      {#if t.action}
        <button
          class="btn"
          onclick={() => {
            t.action?.run();
            app.dismissToast(t.id);
          }}>{t.action.label}</button
        >
      {/if}
      <button class="btn image flat close" title="Close" onclick={() => app.dismissToast(t.id)}><Icon name="window-close" /></button>
    </div>
  {/each}
</div>

<style>
  /* AdwToastOverlay: toasts stack at the bottom centre */
  .toasts {
    position: absolute;
    bottom: 12px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 700;
    display: flex;
    flex-direction: column-reverse;
    align-items: center;
    gap: 6px;
    pointer-events: none;
    max-width: calc(100% - 24px);
  }
  .app-notification {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 42px;
    padding: 5px 5px 5px 18px;
    max-width: 640px;
    background: var(--toast-bg);
    color: #fff;
    border-radius: 9999px;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1), 0 2px 8px 2px rgba(0, 0, 0, 0.25);
    animation: rise 200ms ease-out;
  }
  .app-notification.error {
    background: color-mix(in srgb, #c01c28 94%, transparent);
  }
  .text {
    flex: 1;
    min-width: 0;
    margin-right: 6px;
    overflow-wrap: anywhere;
  }
  .app-notification .btn {
    color: #fff;
    background: rgba(255, 255, 255, 0.1);
    min-height: 32px;
    border-radius: 9999px;
    padding: 4px 14px;
  }
  .app-notification .btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }
  .app-notification .close {
    background: none;
    min-width: 32px;
    padding: 0;
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(12px) scale(0.96);
    }
  }
</style>
