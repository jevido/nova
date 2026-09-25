<script lang="ts">
  import { Window } from "@wailsio/runtime";
  import * as Session from "../../bindings/nova/services/sessionservice";
  import { app } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";

  let mode = $state<"password" | "key">("password");
  let server = $state(app.session?.server || "");
  let username = $state("");
  let password = $state("");
  let key = $state("");
  let busy = $state(false);
  let error = $state("");
  let advanced = $state(false);

  $effect(() => {
    if (!server) Session.DefaultServer().then((s) => (server ||= s));
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      if (mode === "key") await app.signInWithKey(server, key);
      else await app.signIn(server, username, password);
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
    <button class="btn image flat round" title="Close" onclick={() => Window.Close()}><Icon name="window-close" /></button>
  </header>
  <form class="card" onsubmit={submit}>
    <img src="/nova.png" alt="" width="88" height="88" />
    <h1>Sign in to Nova</h1>
    <p class="dim">Your nova.storage files, right on your desktop.</p>

    <div class="linked modes">
      <button type="button" class="btn" class:checked={mode === "password"} onclick={() => (mode = "password")}>Password</button>
      <button type="button" class="btn" class:checked={mode === "key"} onclick={() => (mode = "key")}>API Key</button>
    </div>

    {#if mode === "password"}
      <input class="entry" placeholder="Username or e-mail" autocomplete="username" bind:value={username} disabled={busy} />
      <input class="entry" type="password" placeholder="Password" autocomplete="current-password" bind:value={password} disabled={busy} />
    {:else}
      <input class="entry mono" placeholder="API key" bind:value={key} disabled={busy} spellcheck="false" />
    {/if}

    {#if advanced}
      <input class="entry" placeholder="Server" bind:value={server} disabled={busy} spellcheck="false" />
    {/if}

    {#if error}<div class="error">{error}</div>{/if}

    <button class="btn suggested submit" type="submit" disabled={busy}>
      {#if busy}<span class="spinner"></span>{/if}Sign In
    </button>
    <button type="button" class="link dim" onclick={() => (advanced = !advanced)}>
      {advanced ? "Hide server settings" : "Use a different server…"}
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
    min-height: 46px;
    padding: 6px;
    background: linear-gradient(to top, var(--header-bottom), var(--header-top));
    border-bottom: 1px solid var(--header-border);
    --wails-draggable: drag;
  }
  .title {
    flex: 1;
    text-align: center;
    font-weight: bold;
    padding-left: 30px;
  }
  .round {
    width: 24px;
    height: 24px;
    min-width: 24px;
    min-height: 24px;
    padding: 0;
    border-radius: 50%;
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
    font-size: 22px;
  }
  p {
    margin: 0 0 8px;
  }
  .modes {
    display: flex;
    margin-bottom: 4px;
  }
  .modes .btn {
    flex: 1;
  }
  .mono {
    font-family: monospace;
  }
  .submit {
    margin-top: 6px;
  }
  .error {
    color: var(--destructive);
    font-size: 13px;
  }
  .link {
    border: 0;
    background: none;
    font-size: 13px;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
