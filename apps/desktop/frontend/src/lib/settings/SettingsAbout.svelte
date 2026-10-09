<script lang="ts">
  import { api, type AppInfo } from '../api/backend';
  import { app } from '../app/app.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import mark from '../assets/plugboard-mark.png';

  let { info }: { info: AppInfo | null } = $props();

  let checking = $state(false);
  let checkError = $state('');
  let upToDate = $state(false);

  async function checkForUpdate() {
    checking = true;
    checkError = '';
    const error = await app.checkForUpdate();
    checking = false;
    checkError = error;
    upToDate = !error && !app.update;
  }

  const folderLabel = $derived(
    info?.platform.startsWith('darwin') ? 'Show in Finder' : info?.platform.startsWith('windows') ? 'Show in Explorer' : 'Open folder',
  );

  async function openDataFolder() {
    try {
      await api.openDataFolder();
    } catch (err) {
      app.notify(err);
    }
  }
</script>

<div class="about">
  <img src={mark} alt="" width="80" height="80" draggable="false" />
  <h2>Plugboard</h2>
  <p class="version">Version {info?.version ?? '—'}</p>
  <div class="update-row">
    {#if app.update?.status === 'available'}
      <span>Version {app.update.info.version} is available.</span>
      <button class="btn sm primary" onclick={() => app.installUpdate()}>Install</button>
    {:else if app.update?.status === 'installing'}
      <Spinner size={11} /><span>Installing {app.update.info.version}…</span>
    {:else if app.update?.status === 'installed'}
      <span>Version {app.update.info.version} is installed.</span>
      <button class="btn sm primary" onclick={() => app.restartToUpdate()}>Restart now</button>
    {:else if upToDate}
      <span class="up-to-date"><Icon name="check" size={13} />Plugboard is up to date</span>
    {:else}
      {#if app.update?.status === 'failed'}<span class="update-error" title={app.update.error}>{app.update.error}</span>{:else if checkError}<span class="update-error" title={checkError}>{checkError}</span>{/if}
      <button class="btn sm" onclick={checkForUpdate} disabled={checking}>{#if checking}<Spinner size={11} />Checking…{:else}Check for updates{/if}</button>
    {/if}
  </div>
  <p class="blurb">
    A native database client for PostgreSQL, MySQL, SQLite, ClickHouse and Redis. Connections and credentials stay on this
    computer — passwords are kept in the system keychain, and Plugboard has no accounts, no cloud sync
    and no telemetry.
  </p>
  <dl class="details">
    <dt>Version</dt><dd class="mono">{info?.version ?? '—'}</dd>
    <dt>Platform</dt><dd class="mono">{info?.platform ?? '—'}</dd>
    <dt>Runtime</dt><dd class="mono">{info?.goVersion ?? '—'}</dd>
    <dt>Data folder</dt>
    <dd class="data-dir">
      <span class="mono" title={info?.dataDir}>{info?.dataDir ?? '—'}</span>
      <button class="btn sm" onclick={openDataFolder}>{folderLabel}</button>
    </dd>
    <dt>Logs</dt>
    <dd class="data-dir">
      <span class="mono faint">Crash logs</span>
      <button class="btn sm" onclick={() => api.openLogs().catch(err => app.notify(err))}>Open</button>
    </dd>
  </dl>
  <p class="copyright">{info?.copyright ?? ''} · Part of the Relay family of developer tools.</p>
</div>

<style>
  .about { display: flex; flex-direction: column; align-items: center; text-align: center; padding-top: 8px; }
  .about img { border-radius: 18px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18); }
  .about h2 { margin: 14px 0 2px; font-size: 20px; }
  .update-row { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; margin: 6px 0 2px; font-size: 12px; color: var(--text-2); min-height: 24px; }
  .up-to-date { display: inline-flex; align-items: center; gap: 6px; color: var(--ok); }
  .update-error { max-width: 320px; color: var(--danger); }
  .version { margin: 0; font-size: 12.5px; color: var(--text-3); }
  .blurb { max-width: 430px; margin: 16px 0 18px; color: var(--text-2); line-height: 1.55; }
  .details {
    display: grid;
    grid-template-columns: 96px 1fr;
    gap: 8px 14px;
    width: 100%;
    max-width: 440px;
    margin: 0;
    padding: 14px 16px;
    border-radius: 8px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    text-align: left;
    font-size: 12.5px;
  }
  .details dt { color: var(--text-3); }
  .details dd { margin: 0; min-width: 0; user-select: text; -webkit-user-select: text; }
  .details .mono { font-size: 11.5px; }
  .data-dir { display: flex; align-items: center; gap: 8px; }
  .data-dir .mono { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .copyright { margin: 16px 0 0; font-size: 11.5px; color: var(--text-3); }
</style>
