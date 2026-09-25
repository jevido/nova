<script lang="ts">
  import { Window } from "@wailsio/runtime";
  import { app } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";

  let mode = $state<"password" | "key">("password");
  let username = $state("");
  let password = $state("");
  let key = $state("");
  let busy = $state(false);
  let error = $state("");

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      if (mode === "key") await app.signInWithKey(key);
      else await app.signIn(username, password);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
  }
</script>

<div class="login">
  <header class="bar">
    <span class="title">Nova</span>
    {#if !app.mobile}
      <button class="btn image round" title="Close" onclick={() => Window.Close()}><Icon name="window-close" /></button>
    {/if}
  </header>
  <form class="card" onsubmit={submit}>
    <img src="/nova.png" alt="" width="88" height="88" />
    <h1>Sign in to Nova</h1>
    <p class="dim">Your nova.storage files, right on your desktop.</p>

    <div class="modes">
      <button type="button" class="btn" class:checked={mode === "password"} onclick={() => (mode = "password")}>Password</button>
      <button type="button" class="btn" class:checked={mode === "key"} onclick={() => (mode = "key")}>API Key</button>
    </div>

    {#if mode === "password"}
      <input class="entry" placeholder="Username or e-mail" autocomplete="username" bind:value={username} disabled={busy} />
      <input class="entry" type="password" placeholder="Password" autocomplete="current-password" bind:value={password} disabled={busy} />
    {:else}
      <input class="entry mono" placeholder="API key" bind:value={key} disabled={busy} spellcheck="false" />
    {/if}

    {#if error}<div class="error">{error}</div>{/if}

    <button class="btn suggested submit" type="submit" disabled={busy}>
      {#if busy}<span class="spinner"></span>{/if}Sign In
    </button>
  </form>
</div>

<style>
  .login {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--bg);
  }
  .bar {
    display: flex;
    align-items: center;
    min-height: 47px;
    padding: 6px 7px;
    --wails-draggable: drag;
  }
  .title {
    flex: 1;
    text-align: center;
    font-weight: bold;
    padding-left: 30px;
  }
  .card {
    margin: auto;
    width: 340px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    text-align: center;
  }
  .card img {
    align-self: center;
  }
  h1 {
    margin: 6px 0 0;
    font-size: 2em;
    font-weight: 800;
  }
  p {
    margin: 0 0 8px;
  }
  /* AdwToggleGroup */
  .modes {
    display: flex;
    gap: 3px;
    padding: 3px;
    margin-bottom: 6px;
    border-radius: 9px;
    background: var(--btn-bg);
  }
  .modes .btn {
    flex: 1;
    min-height: 30px;
    background: none;
    box-shadow: none;
  }
  .modes .btn:hover {
    background: var(--hover);
  }
  .modes .btn.checked {
    background: var(--view-bg);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }
  :global(:root[data-theme="dark"]) .modes .btn.checked {
    background: color-mix(in srgb, #fff 15%, var(--bg));
  }
  .mono {
    font-family: var(--mono);
  }
  .submit {
    margin-top: 12px;
    align-self: center;
    border-radius: 9999px;
    padding: 10px 32px;
    min-width: 200px;
  }
  .error {
    color: var(--destructive);
    font-size: 13px;
  }
</style>
