<script lang="ts">
  import type { Workspace } from './workspace.svelte';
  import { connectionTarget } from '../ui/format';
  import { engine } from '../engines';
  import ConnAvatar from '../connections/ConnAvatar.svelte';
  import EnvBadge from '../connections/EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import Modal from '../ui/Modal.svelte';

  let { ws }: { ws: Workspace } = $props();

  const conn = $derived(ws.connection);
  let confirmWrites = $state(false);

  function toggleReadOnly() {
    if (ws.readOnly && conn.env === 'prod') {
      confirmWrites = true;
      return;
    }
    ws.setReadOnly(!ws.readOnly);
  }
</script>

  <div class="conn" title="{conn.name} — {connectionTarget(conn)}">
    <ConnAvatar connection={conn} size={32} />
    <div class="conn-meta">
      <div class="conn-name">{conn.name} <EnvBadge env={conn.env} /></div>
      <div class="conn-target">
        {#if conn.ssh?.enabled && !engine(conn.driver).file}<span class="via" class:down={ws.tunnel !== 'ok'} title={ws.tunnel !== 'ok' ? 'SSH connection lost — reconnecting' : `Through SSH ${conn.ssh.user}@${conn.ssh.host || conn.host}`}><Icon name="tunnel" size={11} />{ws.tunnel !== 'ok' ? 'SSH reconnecting…' : 'SSH'}</span>{/if}
        {ws.session.serverVersion}
      </div>
    </div>
  </div>

  <button
    class="ro"
    class:on={ws.readOnly}
    disabled={ws.switchingReadOnly}
    onclick={toggleReadOnly}
    role="switch"
    aria-checked={ws.readOnly}
    title={ws.readOnly ? 'Read-only is on: writes are refused. Click to allow writes for this session.' : 'Read-only is off: writes are allowed. Click to make this session read-only.'}
  >
    {#if ws.switchingReadOnly}<Spinner size={12} />{:else}<Icon name={ws.readOnly ? 'lock' : 'lockOpen'} size={13} />{/if}
    <span>{ws.switchingReadOnly ? 'Reconnecting…' : 'Read-only'}</span>
    <span class="ro-switch" aria-hidden="true"></span>
  </button>

{#if confirmWrites}
  <Modal title="Allow writes on Production?" width={420} onclose={() => (confirmWrites = false)}>
    <p class="confirm-text">
      <strong>{conn.name}</strong> is tagged Production. Turning read-only off lets the SQL editor and the grid write to it for
      this session.
    </p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (confirmWrites = false)}>Keep read-only</button>
      <button class="btn primary danger-fill" onclick={() => { confirmWrites = false; ws.setReadOnly(false); }}>Allow writes</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .confirm-text { margin: 0; color: var(--text-2); line-height: 1.5; }
  .confirm-text strong { color: var(--text); }
  .conn {
    --avatar-ring: var(--surface);
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 12px 4px;
    min-width: 0;
  }
  .conn-meta { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
  .conn-name {
    display: flex;
    align-items: center;
    gap: 7px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
  }
  .conn-target {
    font-size: 11.5px;
    color: var(--text-3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .conn-target { display: flex; align-items: center; gap: 6px; }
  .via {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 0 4px;
    border-radius: 3px;
    background: var(--elevated);
    color: var(--text-2);
    font-size: 10px;
    font-weight: 600;
  }
  .via.down { color: var(--warn); background: color-mix(in srgb, var(--warn) 14%, transparent); }
  .ro {
    display: flex;
    align-items: center;
    gap: 7px;
    margin: 8px 10px 0;
    height: 28px;
    padding: 0 8px 0 9px;
    border: 0;
    border-radius: 6px;
    background: var(--elevated);
    color: var(--text-2);
    font-size: 12px;
  }
  .ro:hover:not(:disabled) { background: var(--hover); color: var(--text); }
  .ro span:nth-child(2) { flex: 1; text-align: left; }
  .ro.on { color: var(--warn); background: color-mix(in srgb, var(--warn) 12%, var(--surface)); }
  .ro.on:hover:not(:disabled) { color: var(--warn); background: color-mix(in srgb, var(--warn) 18%, var(--surface)); }
  .ro-switch {
    position: relative;
    width: 24px;
    height: 14px;
    border-radius: 999px;
    background: var(--active);
    transition: background 0.15s;
  }
  .ro-switch::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.15s;
  }
  .ro.on .ro-switch { background: var(--warn); }
  .ro.on .ro-switch::after { transform: translateX(10px); }
</style>
