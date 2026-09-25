<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import * as Account from "../../bindings/nova/services/accountservice";
  import type { FolderStatus, UsageSeries } from "../../bindings/nova/services/models";
  import { app, HOME } from "../lib/store.svelte";
  import { formatSize, pluralize } from "../lib/format";
  import Icon from "./Icon.svelte";
  import UsageChart from "./UsageChart.svelte";

  // An AdwPreferencesDialog-style page: account, usage graphs, folder setup.
  type Page = "account" | "usage" | "folders";
  let page = $state<Page>("account");
  let dialogEl = $state<HTMLDivElement>();

  const user = $derived(app.session?.user);
  const sub = $derived(user?.subscription);
  const used = $derived(user?.filesystem_storage_used ?? 0);
  const limit = $derived(sub?.storage_limit ?? -1);
  const egressCap = $derived(user?.monthly_transfer_cap || sub?.monthly_transfer_cap || 0);
  const egressUsed = $derived(user?.monthly_transfer_used ?? 0);

  $effect(() => {
    if (app.settingsOpen) {
      queueMicrotask(() => dialogEl?.focus());
      // Show the dialog first; fresh numbers can follow.
      setTimeout(() => app.refreshUser(), 150);
    }
  });

  function close() {
    app.settingsOpen = false;
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape" && !app.modal && !app.menu) {
      e.preventDefault();
      e.stopPropagation();
      close();
    }
  }

  // ---------- usage ----------
  const RANGES = [
    { id: "day", label: "Day", minutes: 1440, interval: 60 },
    { id: "week", label: "Week", minutes: 10080, interval: 180 },
    { id: "month", label: "Month", minutes: 43200, interval: 1440 },
    { id: "quarter", label: "Quarter", minutes: 131040, interval: 1440 },
    { id: "year", label: "Year", minutes: 525600, interval: 1440 },
  ] as const;
  let range = $state<(typeof RANGES)[number]>(RANGES[2]);
  let egress = $state<UsageSeries | null>(null);
  let downloads = $state<UsageSeries | null>(null);
  let usageError = $state("");
  let usageLoading = $state(false);
  let usageSeq = 0;

  async function loadUsage() {
    const seq = ++usageSeq;
    usageLoading = true;
    usageError = "";
    try {
      const [e, d] = await Promise.all([
        Account.Usage("egress", range.minutes, range.interval),
        Account.Usage("downloads", range.minutes, range.interval),
      ]);
      if (seq !== usageSeq) return;
      egress = e;
      downloads = d;
    } catch (err) {
      if (seq === usageSeq) usageError = err instanceof Error ? err.message : String(err);
    } finally {
      if (seq === usageSeq) usageLoading = false;
    }
  }

  $effect(() => {
    if (app.settingsOpen && page === "usage") {
      void range;
      loadUsage();
    }
  });

  const count = (v: number) => (v >= 1000 ? `${(v / 1000).toFixed(v >= 10000 ? 0 : 1)}k` : String(Math.round(v)));

  // ---------- folders ----------
  let folders = $state<FolderStatus[] | null>(null);
  let foldersError = $state("");
  let addToSidebar = $state(true);
  let creating = $state(false);
  const missing = $derived((folders ?? []).filter((f) => !f.exists));

  async function loadFolders() {
    foldersError = "";
    try {
      folders = await Account.RecommendedFolders();
    } catch (err) {
      foldersError = err instanceof Error ? err.message : String(err);
    }
  }

  $effect(() => {
    if (app.settingsOpen && page === "folders") loadFolders();
  });

  async function createFolders() {
    creating = true;
    try {
      const made = (await Account.CreateRecommendedFolders()) ?? [];
      if (addToSidebar && made.length) app.addBookmarks(made);
      app.toast(made.length ? `Created ${pluralize(made.length, "folder", "folders")}` : "All folders already exist");
      if (app.path === HOME) app.reload(true);
    } catch (err) {
      app.toast(err instanceof Error ? err.message : String(err), { error: true });
    } finally {
      creating = false;
      loadFolders();
    }
  }

  // ---------- updates ----------
  const updateText = $derived.by(() => {
    const u = app.update;
    switch (u?.state) {
      case "checking":
        return "Checking for updates…";
      case "downloading":
        return `Downloading Nova ${u.latestVersion}…`;
      case "ready":
        return `Nova ${u.latestVersion} is ready to install`;
      case "manual":
        return u.packageManaged ? `Nova ${u.latestVersion} is available from your package manager` : `Nova ${u.latestVersion} is available`;
      case "up-to-date":
        return "Nova is up to date";
      case "error":
        return `Update check failed: ${u.error}`;
      case "disabled":
        return "Development build, updates are off";
      default:
        return "Nova checks for updates automatically";
    }
  });
</script>

{#if app.settingsOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="backdrop" onmousedown={(e) => e.target === e.currentTarget && close()} onkeydown={onKey}>
    <div class="dialog settings" bind:this={dialogEl} role="dialog" aria-modal="true" aria-label="Settings" tabindex="-1">
      <header class="head">
        <span class="spacer"></span>
        <div class="switcher" role="tablist">
          {#each [["account", "Account", "avatar-default"], ["usage", "Usage", "power-profile-performance"], ["folders", "Folders", "folder"]] as [id, label, icon] (id)}
            <button role="tab" aria-selected={page === id} class:on={page === id} onclick={() => (page = id as Page)}>
              <Icon name={icon} /><span>{label}</span>
            </button>
          {/each}
        </div>
        <span class="spacer end">
          <button class="btn image round" title="Close" onclick={close}><Icon name="window-close" /></button>
        </span>
      </header>

      <div class="body">
        {#if page === "account"}
          <div class="profile">
            <span class="avatar">{(user?.username ?? "N").slice(0, 1).toUpperCase()}</span>
            <h2>{user?.username ?? "Nova"}</h2>
            {#if user?.email}<p class="dim selectable">{user.email}</p>{/if}
          </div>

          <h3>Storage</h3>
          <div class="boxed">
            <div class="row">
              <div class="row-text">
                <div>Space used</div>
                <div class="dim sub">{limit > 0 ? `${formatSize(used)} of ${formatSize(limit)}` : formatSize(used)}</div>
              </div>
              {#if limit > 0}<div class="meter" title="{Math.round((used / limit) * 100)}%"><div style:width="{Math.min(100, (used / limit) * 100)}%"></div></div>{/if}
            </div>
            <div class="row">
              <div class="row-text">
                <div>Transfer this month</div>
                <div class="dim sub">{egressCap > 0 ? `${formatSize(egressUsed)} of ${formatSize(egressCap)}` : formatSize(egressUsed)}</div>
              </div>
              {#if egressCap > 0}<div class="meter" title="{Math.round((egressUsed / egressCap) * 100)}%"><div style:width="{Math.min(100, (egressUsed / egressCap) * 100)}%"></div></div>{/if}
            </div>
            <div class="row">
              <div class="row-text"><div>Files and folders</div></div>
              <span class="dim">{(user?.filesystem_node_count ?? 0).toLocaleString()}</span>
            </div>
          </div>

          <h3>Subscription</h3>
          <div class="boxed">
            <div class="row">
              <div class="row-text"><div>Plan</div></div>
              <span class="dim">{sub?.name || "Free"}</span>
            </div>
            {#if user && user.balance_micro_eur !== 0}
              <div class="row">
                <div class="row-text"><div>Balance</div></div>
                <span class="dim">€ {(user.balance_micro_eur / 1e6).toFixed(2)}</span>
              </div>
            {/if}
            <button class="row link-row" onclick={() => Browser.OpenURL("https://nova.storage/user")}>
              <div class="row-text"><div>Manage on nova.storage</div></div>
              <Icon name="adw-external-link" />
            </button>
          </div>

          <h3>Updates</h3>
          <div class="boxed">
            <div class="row">
              <div class="row-text">
                <div>Nova {app.update?.currentVersion ?? ""}</div>
                <div class="dim sub">{updateText}</div>
              </div>
              {#if app.update?.state === "ready"}
                <button class="btn suggested" onclick={() => app.applyUpdate()}>{app.mobile ? "Install" : "Restart"}</button>
              {:else}
                <button class="btn" disabled={app.update?.state === "checking" || app.update?.state === "downloading"} onclick={() => app.checkForUpdates()}>
                  {#if app.update?.state === "checking"}<span class="spinner"></span>{/if}Check for Updates
                </button>
              {/if}
            </div>
          </div>

          <div class="boxed signout">
            <button class="row link-row" onclick={() => app.signOut().then(() => !app.session?.signedIn && close())}>
              <div class="row-text"><div class="danger">Sign Out</div></div>
            </button>
          </div>
        {:else if page === "usage"}
          <div class="ranges" role="group" aria-label="Time range">
            {#each RANGES as r (r.id)}
              <button class:on={range.id === r.id} onclick={() => (range = r)}>{r.label}</button>
            {/each}
          </div>
          {#if usageError}
            <div class="status dim">Could not load statistics: {usageError}</div>
          {:else}
            <section class="card" class:loading={usageLoading}>
              <div class="card-head">
                <div>
                  <div class="card-title">Transfer</div>
                  <div class="dim sub">Data downloaded from your files</div>
                </div>
                <div class="hero">{egress ? formatSize(egress.total) : "–"}</div>
              </div>
              {#if egress}<UsageChart label="Transfer" timestamps={egress.timestamps ?? []} amounts={egress.amounts ?? []} format={formatSize} bucket={range.interval} />{/if}
            </section>
            <section class="card" class:loading={usageLoading}>
              <div class="card-head">
                <div>
                  <div class="card-title">Downloads</div>
                  <div class="dim sub">Times your files were downloaded</div>
                </div>
                <div class="hero">{downloads ? Math.round(downloads.total).toLocaleString() : "–"}</div>
              </div>
              {#if downloads}<UsageChart label="Downloads" timestamps={downloads.timestamps ?? []} amounts={downloads.amounts ?? []} format={count} bucket={range.interval} />{/if}
            </section>
          {/if}
        {:else}
          <p class="intro">
            Start with a tidy home folder. Nova creates the folders below that you don't have yet. It never removes, renames or
            moves anything.
          </p>
          {#if foldersError}
            <div class="status dim">Could not read your home folder: {foldersError}</div>
          {:else if !folders}
            <div class="status"><span class="spinner"></span></div>
          {:else}
            <div class="boxed">
              {#each folders as f (f.name)}
                <div class="row">
                  <Icon name={f.exists ? "folder" : "folder-new"} size={20} />
                  <div class="row-text"><div>{f.name}</div></div>
                  {#if f.exists}
                    <span class="state ok"><Icon name="object-select" />Already there</span>
                  {:else}
                    <span class="state dim">Will be created</span>
                  {/if}
                </div>
              {/each}
            </div>
            <div class="boxed">
              <label class="row">
                <div class="row-text">
                  <div>Add to sidebar</div>
                  <div class="dim sub">Bookmark the new folders</div>
                </div>
                <button class="switch" class:on={addToSidebar} role="switch" aria-checked={addToSidebar} aria-label="Add to sidebar" onclick={() => (addToSidebar = !addToSidebar)}><span></span></button>
              </label>
            </div>
            <div class="actions">
              <button class="btn suggested pill" disabled={!missing.length || creating} onclick={createFolders}>
                {#if creating}<span class="spinner"></span>{/if}
                {missing.length ? `Create ${pluralize(missing.length, "Folder", "Folders")}` : "All Set"}
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 940;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.25);
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  .dialog {
    width: 640px;
    max-width: calc(100vw - 32px);
    height: min(720px, calc(100vh - 32px));
    display: flex;
    flex-direction: column;
    background: var(--dialog-bg);
    border-radius: 15px;
    box-shadow: var(--dialog-shadow);
    outline: none;
    overflow: hidden;
    animation: dlg-in 100ms ease-out;
  }
  @keyframes dlg-in {
    from {
      opacity: 0;
    }
  }
  :global(:root[data-mobile="true"]) .dialog {
    width: 100vw;
    max-width: none;
    height: 100%;
    border-radius: 0;
  }
  :global(:root[data-mobile="true"]) .backdrop {
    align-items: stretch;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 47px;
    padding: 6px 7px;
  }
  .spacer {
    flex: 1;
    display: flex;
  }
  .spacer.end {
    justify-content: flex-end;
  }
  /* AdwViewSwitcher */
  .switcher {
    display: flex;
    gap: 3px;
  }
  .switcher button {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 34px;
    padding: 0 12px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg);
    font-weight: bold;
    outline: none;
  }
  .switcher button:hover {
    background: var(--hover);
  }
  .switcher button.on {
    background: var(--btn-active);
  }
  .switcher button:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  /* Narrow: icon above a small label, like AdwViewSwitcher's narrow policy */
  :global(:root[data-mobile="true"]) .switcher button {
    flex-direction: column;
    gap: 2px;
    min-height: 44px;
    padding: 4px 14px;
    font-size: 0.8em;
  }
  .body {
    flex: 1;
    overflow-y: auto;
    padding: 12px 24px 32px;
  }
  .profile {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    margin: 8px 0 12px;
  }
  .profile .avatar {
    display: grid;
    place-items: center;
    width: 80px;
    height: 80px;
    margin-bottom: 8px;
    border-radius: 50%;
    background: color-mix(in srgb, var(--accent) 35%, var(--dialog-bg));
    font-size: 2.2em;
    font-weight: bold;
  }
  .profile h2 {
    margin: 0;
    font-size: 1.5em;
    font-weight: 800;
  }
  .profile p {
    margin: 0;
  }
  .selectable {
    user-select: text;
    -webkit-user-select: text;
  }
  h3 {
    margin: 22px 2px 8px;
    font-size: 1em;
    font-weight: bold;
  }
  /* .boxed-list with AdwActionRows */
  .boxed {
    border-radius: 12px;
    background: var(--card-bg);
    box-shadow: 0 0 0 1px var(--shade), 0 1px 3px var(--shade);
    overflow: hidden;
  }
  .boxed + .boxed,
  .signout {
    margin-top: 18px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 54px;
    padding: 8px 12px;
    border: 0;
    background: none;
    color: var(--fg);
    text-align: left;
    font: inherit;
  }
  .row + .row {
    box-shadow: inset 0 1px var(--shade);
  }
  .row-text {
    flex: 1;
    min-width: 0;
  }
  .sub {
    font-size: 0.87em;
    margin-top: 2px;
  }
  .link-row {
    outline: none;
  }
  .link-row:hover {
    background: var(--hover);
  }
  .link-row:active {
    background: var(--active);
  }
  .link-row:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .danger {
    color: var(--destructive-text);
    font-weight: bold;
    text-align: center;
  }
  .meter {
    width: 140px;
    height: 8px;
    border-radius: 4px;
    background: var(--btn-bg);
    overflow: hidden;
  }
  .meter > div {
    height: 100%;
    border-radius: 4px;
    background: var(--accent);
  }
  /* Time range: a linked toggle group */
  .ranges {
    display: flex;
    gap: 3px;
    padding: 3px;
    margin: 4px 0 18px;
    border-radius: 9px;
    background: var(--btn-bg);
  }
  .ranges button {
    flex: 1;
    min-height: 30px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg);
    font-weight: bold;
    outline: none;
  }
  .ranges button:hover {
    background: var(--hover);
  }
  .ranges button.on {
    background: var(--view-bg);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }
  :global(:root[data-theme="dark"]) .ranges button.on {
    background: color-mix(in srgb, #fff 15%, var(--dialog-bg));
  }
  .ranges button:focus-visible {
    outline: 2px solid var(--focus);
    outline-offset: -2px;
  }
  .card {
    padding: 14px 16px 10px;
    margin-bottom: 14px;
    border-radius: 12px;
    background: var(--card-bg);
    box-shadow: 0 0 0 1px var(--shade), 0 1px 3px var(--shade);
    transition: opacity 150ms;
  }
  .card.loading {
    opacity: 0.6;
  }
  .card-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
  }
  .card-title {
    font-weight: bold;
  }
  .hero {
    font-size: 1.7em;
    font-weight: 800;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
  }
  .status {
    display: flex;
    justify-content: center;
    padding: 32px 0;
  }
  .intro {
    margin: 6px 2px 16px;
    color: var(--fg-dim);
  }
  .state {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 0.87em;
  }
  .state.ok {
    color: var(--success);
  }
  .actions {
    display: flex;
    justify-content: center;
    margin-top: 24px;
  }
  /* GtkSwitch */
  .switch {
    position: relative;
    flex: none;
    width: 48px;
    height: 26px;
    border-radius: 14px;
    border: 0;
    padding: 0;
    background: color-mix(in srgb, var(--fg) 20%, transparent);
    outline: none;
    transition: background 150ms;
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
