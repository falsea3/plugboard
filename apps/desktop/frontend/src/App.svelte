<script lang="ts">
  import { onMount } from 'svelte';
  import { app } from './lib/app/app.svelte';
  import { hasBridge, onMenuCommand } from './lib/api/backend';
  import { handleShortcut, runCommand } from './lib/app/commands';
  import TitleBar from './lib/app/TitleBar.svelte';
  import ConnectionsHome from './lib/connections/ConnectionsHome.svelte';
  import WorkspaceView from './lib/app/WorkspaceView.svelte';
  import ConnectionForm from './lib/connections/ConnectionForm.svelte';
  import ConnectionSwitcher from './lib/app/ConnectionSwitcher.svelte';
  import SettingsModal from './lib/settings/SettingsModal.svelte';
  import Modal from './lib/ui/Modal.svelte';
  import Toasts from './lib/app/Toasts.svelte';
  import UpdateBanner from './lib/app/UpdateBanner.svelte';

  let secrets = $state({ password: '', sshPassword: '', sshPassphrase: '' });

  if (hasBridge()) app.init();
  else app.connectionsLoaded = true;

  onMount(() => onMenuCommand(runCommand));
</script>

<svelte:window onkeydown={handleShortcut} />

<div class="shell">
  <TitleBar />
  <main>
    {#each app.workspaces as ws (ws.id)}
      <WorkspaceView {ws} visible={app.active === ws} />
    {/each}
    {#if !app.active}
      <div class="home"><ConnectionsHome /></div>
    {/if}
  </main>
</div>

{#if app.editing !== undefined}
  <ConnectionForm
    initial={app.editing}
    onclose={() => (app.editing = undefined)}
    onconnect={(c, secrets) => app.open(c, secrets ?? {})}
  />
{/if}

{#if app.passwordFor}
  {@const target = app.passwordFor}
  {@const sshPw = target.ssh?.enabled && target.ssh.auth === 'password'}
  {@const sshKey = target.ssh?.enabled && target.ssh.auth === 'key'}
  <Modal title="Connect to {target.name}" width={400} onclose={() => (app.passwordFor = null)}>
    <form
      id="pw-form"
      class="secrets"
      onsubmit={e => {
        e.preventDefault();
        const c = target;
        const typed = { ...secrets };
        app.passwordFor = null;
        secrets = { password: '', sshPassword: '', sshPassphrase: '' };
        app.open(c, typed);
      }}
    >
      <label>
        <span>Database password</span>
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input" type="password" bind:value={secrets.password} placeholder="{target.user}@{target.host}" autofocus />
      </label>
      {#if sshPw}
        <label>
          <span>SSH password</span>
          <input class="input" type="password" bind:value={secrets.sshPassword} placeholder="{target.ssh.user}@{target.ssh.host}" />
        </label>
      {:else if sshKey}
        <label>
          <span>Key passphrase</span>
          <input class="input" type="password" bind:value={secrets.sshPassphrase} placeholder="Leave empty if the key isn’t encrypted" />
        </label>
      {/if}
    </form>
    {#snippet footer()}
      <span class="faint" style="font-size:11.5px">Not saved — this profile doesn’t keep secrets.</span>
      <span style="flex:1"></span>
      <button class="btn" onclick={() => (app.passwordFor = null)}>Cancel</button>
      <button class="btn primary" type="submit" form="pw-form">Connect</button>
    {/snippet}
  </Modal>
{/if}

{#if app.hostKeyChange}
  {@const h = app.hostKeyChange}
  <Modal title="SSH host key changed" width={480} onclose={() => (app.hostKeyChange = null)}>
    <p class="hk-text">
      <strong>{h.host}</strong> presented a different key than the one remembered for it. That happens when the server is
      rebuilt or its SSH keys are regenerated — or when someone is intercepting the connection.
    </p>
    <div class="hk-fp"><span class="faint">New key</span><code>{h.fingerprint}</code></div>
    <p class="hk-text faint">Trust it only if you know the server changed. You can compare the fingerprint with <code>ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub</code> on the server.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (app.hostKeyChange = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={() => app.trustHostKeyAndConnect()}>Trust new key and connect</button>
    {/snippet}
  </Modal>
{/if}

{#if app.pendingClose}
  {@const p = app.pendingClose}
  <Modal title="Discard uncommitted changes?" width={420} onclose={() => (app.pendingClose = null)}>
    <p style="margin:0;color:var(--text-2);line-height:1.5">
      {p.tables.length === 1 ? `“${p.tables[0]}” has` : `${p.tables.length} tables have`} edits that haven’t been committed.
      {p.tabId ? 'Closing the tab' : `Closing ${p.ws.connection.name}`} throws them away.
    </p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (app.pendingClose = null)}>Keep editing</button>
      <button class="btn primary danger-fill" onclick={() => app.confirmClose()}>Discard and close</button>
    {/snippet}
  </Modal>
{/if}

{#if app.switcherOpen}
  <ConnectionSwitcher />
{/if}

{#if app.settingsOpen}
  <SettingsModal />
{/if}

{#if !hasBridge()}
  <div class="no-bridge">Browser preview — the Go backend isn't attached. Run <code>make dev</code> and open the Wails window.</div>
{/if}

<UpdateBanner />
<Toasts />

<style>
  .shell { height: 100%; display: flex; flex-direction: column; }
  .hk-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .hk-text strong { color: var(--text); }
  .hk-text code, .hk-fp code { font-family: var(--font-mono); font-size: 11.5px; }
  .hk-fp {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 12px;
    padding: 10px 12px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    font-size: 12px;
    user-select: text;
    -webkit-user-select: text;
  }
  .secrets { display: flex; flex-direction: column; gap: 12px; }
  .secrets label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
  main { flex: 1; min-height: 0; position: relative; }
  .home { position: absolute; inset: 0; }
  .no-bridge {
    position: fixed;
    left: 50%;
    bottom: 12px;
    transform: translateX(-50%);
    padding: 6px 12px;
    border-radius: 6px;
    background: var(--elevated);
    border: 1px solid var(--border);
    color: var(--text-2);
    font-size: 12px;
  }
</style>
