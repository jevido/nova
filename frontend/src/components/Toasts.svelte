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
  .toasts {
    position: absolute;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    z-index: 700;
    display: flex;
    flex-direction: column;
    align-items: center;
    pointer-events: none;
    max-width: calc(100% - 24px);
  }
  /* GtkRevealer'd in-app notification, as Nautilus uses for undo. */
  .app-notification {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 8px 8px 16px;
    min-width: 300px;
    max-width: 640px;
    background: var(--notification-bg);
    color: #fff;
    border-radius: 0 0 6px 6px;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
    animation: slide 180ms ease-out;
  }
  .app-notification.error {
    background: color-mix(in srgb, #a51d2d 90%, transparent);
  }
  .text {
    flex: 1;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .app-notification .btn {
    color: #fff;
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(0, 0, 0, 0.4);
    box-shadow: none;
    min-height: 30px;
  }
  .app-notification .btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }
  .app-notification .close {
    background: none;
    border-color: transparent;
  }
  @keyframes slide {
    from {
      transform: translateY(-100%);
    }
  }
</style>
