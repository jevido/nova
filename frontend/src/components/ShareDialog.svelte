<script lang="ts">
  import { Browser, Clipboard } from "@wailsio/runtime";
  import * as Files from "../../bindings/nova/services/filesservice";
  import type { Access, Entry, Person, Sharing } from "../../bindings/nova/services/models";
  import { app } from "../lib/store.svelte";
  import Icon from "./Icon.svelte";

  /** The body of the Share dialog; Modals.svelte draws the dialog around it. */
  let { entry }: { entry: Entry } = $props();

  let sh = $state<Sharing | null>(null);
  let error = $state("");
  let busy = $state(false);
  let newPerson = $state("");
  let changed = false;

  const people = $derived(sh?.people ?? []);
  const locked = $derived(!sh || !sh.canShare || busy);
  const viaName = $derived(!sh?.via ? "" : sh.via === "/me" ? "Home" : sh.via.split("/").pop());

  $effect(() => {
    const path = entry.path;
    sh = null;
    error = "";
    Files.Sharing(path)
      .then((s) => (sh = s))
      .catch((e) => (error = String(e?.message ?? e)));
  });

  // The file list only needs to refresh when something changed.
  $effect(() => () => {
    if (changed) app.reload(true);
  });

  async function apply(fn: () => Promise<Sharing | null>) {
    busy = true;
    try {
      const s = await fn();
      if (s) sh = s;
      changed = true;
    } catch (e) {
      app.toast(String((e as Error)?.message ?? e), { error: true });
      // Show what the server has now, not what we tried.
      sh = await Files.Sharing(entry.path).catch(() => sh);
    } finally {
      busy = false;
    }
  }

  function setLink(a: Access) {
    return apply(() => Files.SetLinkAccess(entry.path, a));
  }

  function setPeople(list: Person[]) {
    return apply(() => Files.SetPeople(entry.path, list));
  }

  function setPersonAccess(i: number, a: Access) {
    return setPeople(people.map((p, j) => (j === i ? { ...p, access: a } : p)));
  }

  async function addPerson() {
    const name = newPerson.trim();
    if (!name || !sh) return;
    if (people.some((p) => p.name.toLowerCase() === name.toLowerCase())) {
      newPerson = "";
      return;
    }
    await setPeople([...people, { name, access: { read: true, write: false, delete: false } }]);
    if (people.some((p) => p.name.toLowerCase() === name.toLowerCase())) newPerson = "";
  }

  async function copy(text: string) {
    await Clipboard.SetText(text);
    app.toast("Link copied to clipboard");
  }
</script>

{#snippet toggles(a: Access, set: (a: Access) => void, label: string)}
  <div class="linked" role="group" aria-label={label}>
    <button class="btn" class:checked={a.read} aria-pressed={a.read} disabled={locked} onclick={() => set({ ...a, read: !a.read })}>View</button>
    <button class="btn" class:checked={a.write} aria-pressed={a.write} disabled={locked} onclick={() => set({ ...a, write: !a.write })}>Add</button>
    <button class="btn" class:checked={a.delete} aria-pressed={a.delete} disabled={locked} onclick={() => set({ ...a, delete: !a.delete })}>Delete</button>
  </div>
{/snippet}

{#snippet linkRow(title: string, url: string)}
  <div class="row">
    <div class="row-text">
      <div>{title}</div>
      <button class="url" title="Open in browser" onclick={() => Browser.OpenURL(url)}>{url}</button>
    </div>
    <button class="btn image" title="Copy" onclick={() => copy(url)}><Icon name="edit-copy" /></button>
  </div>
{/snippet}

<div class="body">
  {#if error}
    <p class="notice error">{error}</p>
  {:else if !sh}
    <div class="loading"><span class="spinner"></span></div>
  {:else}
    {#if !sh.canShare}
      <p class="notice">
        Sharing isn't included in your plan.
        <button class="inline-link" onclick={() => Browser.OpenURL("https://nova.storage/user")}>Upgrade on nova.storage</button>
      </p>
    {/if}
    {#if sh.abuse}
      <p class="notice error">This item was taken down after an abuse report ({sh.abuse.replaceAll("_", " ")}).</p>
    {/if}

    <h3>Public Link</h3>
    <div class="boxed">
      <div class="row">
        <div class="row-text">
          <div>Anyone with the link</div>
          <div class="dim sub">
            {#if sh.link.read}Can open {sh.isDir ? "this folder" : "this file"} without signing in{:else if sh.via}Can already open it through “{viaName}”{:else}Off{/if}
          </div>
        </div>
        {#if busy}<span class="spinner"></span>{/if}
        <button
          class="switch"
          class:on={sh.link.read}
          role="switch"
          aria-checked={sh.link.read}
          aria-label="Public link"
          disabled={locked}
          onclick={() => setLink(sh!.link.read ? { read: false, write: false, delete: false } : { ...sh!.link, read: true })}
        ><span></span></button>
      </div>
      {#if sh.url}
        {@render linkRow(sh.via && !sh.link.read ? `Link through “${viaName}”` : "Link", sh.url)}
        {#if sh.directUrl}
          {@render linkRow("Direct download link", sh.directUrl)}
        {/if}
      {/if}
      {#if sh.isDir && sh.link.read}
        <div class="row">
          <div class="row-text">
            <div>Visitors can also</div>
            <div class="dim sub">They need a nova.storage account to add or delete</div>
          </div>
          <div class="linked" role="group" aria-label="Link access">
            <button class="btn" class:checked={sh.link.write} aria-pressed={sh.link.write} disabled={locked} onclick={() => setLink({ ...sh!.link, write: !sh!.link.write })}>Add Files</button>
            <button class="btn" class:checked={sh.link.delete} aria-pressed={sh.link.delete} disabled={locked} onclick={() => setLink({ ...sh!.link, delete: !sh!.link.delete })}>Delete</button>
          </div>
        </div>
      {/if}
    </div>
    {#if sh.via && !sh.link.read}
      <p class="dim hint">Everything in “{viaName}” is public. Turning on this item's own link gives it a shorter link that keeps working if you stop sharing “{viaName}”.</p>
    {/if}

    <h3>People</h3>
    <div class="boxed">
      {#if people.length && sh.peopleUrl}
        {@render linkRow("Link for these people", sh.peopleUrl)}
      {/if}
      {#each people as p, i (p.name)}
        <div class="row">
          <span class="avatar" aria-hidden="true">{p.name.slice(0, 1).toUpperCase()}</span>
          <div class="row-text name">{p.name}</div>
          {#if sh.isDir}
            {@render toggles(p.access, (a) => setPersonAccess(i, a), `Access for ${p.name}`)}
          {:else}
            <span class="dim">Can view</span>
          {/if}
          <button class="btn image flat" title="Stop sharing with {p.name}" disabled={locked} onclick={() => setPeople(people.filter((_, j) => j !== i))}>
            <Icon name="window-close" />
          </button>
        </div>
      {/each}
      <form
        class="row"
        onsubmit={(e) => {
          e.preventDefault();
          addPerson();
        }}
      >
        <input class="entry grow" placeholder="Username or e-mail address" spellcheck="false" autocomplete="off" bind:value={newPerson} disabled={locked} />
        <button class="btn suggested" type="submit" disabled={locked || !newPerson.trim()}>Add</button>
      </form>
    </div>
    <p class="dim hint">
      People get access right away, after signing in to nova.storage. They aren't sent an e-mail, so send them the link yourself.
    </p>
  {/if}
</div>

<style>
  .body {
    padding: 4px 20px 22px;
    overflow: auto;
  }
  .loading {
    display: flex;
    justify-content: center;
    padding: 30px;
  }
  h3 {
    margin: 16px 2px 8px;
    font-size: 1em;
  }
  .notice {
    margin: 8px 0 4px;
    padding: 10px 12px;
    border-radius: 12px;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .notice.error {
    background: color-mix(in srgb, var(--destructive) 16%, transparent);
  }
  .inline-link {
    border: 0;
    padding: 0;
    background: none;
    color: var(--accent-text);
    text-decoration: underline;
    cursor: pointer;
  }
  .hint {
    margin: 8px 2px 0;
    font-size: 0.9em;
  }

  /* .boxed-list with AdwActionRows, as in Settings */
  .boxed {
    border-radius: 12px;
    background: var(--card-bg);
    box-shadow: 0 0 0 1px var(--shade), 0 1px 3px var(--shade);
    overflow: hidden;
  }
  .row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px 12px;
    min-height: 54px;
    margin: 0;
    padding: 8px 12px;
  }
  .row + .row {
    box-shadow: 0 -1px var(--shade);
  }
  .row-text {
    flex: 1;
    min-width: 0;
  }
  .sub {
    font-size: 0.9em;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .grow {
    flex: 1;
    min-width: 0;
  }
  .url {
    display: block;
    max-width: 100%;
    border: 0;
    padding: 0;
    background: none;
    color: var(--accent-text);
    font-size: 0.9em;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
  }
  .url:hover {
    text-decoration: underline;
  }
  .avatar {
    flex: none;
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--accent) 30%, var(--card-bg));
    font-weight: bold;
  }

  /* GtkSwitch, libadwaita style */
  .switch {
    position: relative;
    flex: none;
    width: 48px;
    height: 26px;
    border-radius: 14px;
    border: 0;
    background: color-mix(in srgb, var(--fg) 20%, transparent);
    padding: 0;
    outline: none;
    transition: background 150ms;
  }
  .switch:disabled {
    opacity: 0.5;
  }
  .switch:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: 2px;
  }
  .switch span {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
    transition: left 150ms;
  }
  .switch.on {
    background: var(--accent);
  }
  .switch.on span {
    left: 25px;
  }
</style>
