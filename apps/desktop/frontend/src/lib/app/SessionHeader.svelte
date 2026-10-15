<script lang="ts">
  import type { Workspace } from './workspace.svelte';
  import { app } from './app.svelte';
  import { layout } from './layout.svelte';
  import { connectionTarget } from '../ui/format';
  import { engine } from '../engines';
  import ConnAvatar from '../connections/ConnAvatar.svelte';
  import EnvBadge from '../connections/EnvBadge.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import Modal from '../ui/Modal.svelte';

  let { ws }: { ws: Workspace } = $props();

  const conn = $derived(ws.connection);
  const via = $derived(conn.ssh?.enabled && !engine(conn.driver).file);
  let confirmWrites = $state(false);

  function toggleReadOnly() {
    if (ws.readOnly && conn.env === 'prod') {
      confirmWrites = true;
      return;
    }
    ws.setReadOnly(!ws.readOnly);
  }
</script>

<div class="top drag">
  <span class="lights" aria-hidden="true"></span>
  <button class="home no-drag" onclick={() => app.goHome()} title="All connections (⇧⌘H)" aria-label="All connections"><Icon name="database" size={16} /></button>
  <button class="home no-drag" onclick={() => layout.toggleSidebar()} title={layout.sidebarHidden ? 'Keep the sidebar open (⌘\\)' : 'Hide the sidebar (⌘\\)'} aria-label={layout.sidebarHidden ? 'Keep the sidebar open' : 'Hide the sidebar'}><Icon name="sidebar" size={16} /></button>
</div>

<button class="conn" data-switch-anchor onclick={() => (app.switcherOpen = true)} title="{conn.name} — {connectionTarget(conn)} · switch (⌘K)" aria-label="Switch connection — {conn.name}" aria-haspopup="dialog" aria-expanded={app.switcherOpen}>
  <ConnAvatar connection={conn} size={38} />
  <span class="meta">
    <span class="name"><span class="text">{conn.name}</span><EnvBadge env={conn.env} /></span>
    <span class="target" title={ws.session.serverVersion}>
      {#if via && ws.tunnel !== 'ok'}<span class="via down" title="SSH connection lost — reconnecting"><Icon name="tunnel" size={10} />reconnecting…</span>{/if}
      <span class="text">{engine(conn.driver).name} · {connectionTarget(conn)}</span>
    </span>
  </span>
  <Icon name="updown" size={14} />
</button>

<button class="ro" class:on={ws.readOnly} disabled={ws.switchingReadOnly} onclick={toggleReadOnly} role="switch" aria-checked={ws.readOnly}
  title={ws.readOnly ? 'Read-only is on: writes are refused. Click to allow writes for this session.' : 'Read-only is off: writes are allowed. Click to make this session read-only.'}>
  {#if ws.switchingReadOnly}<Spinner size={12} />{:else}<Icon name={ws.readOnly ? 'lock' : 'lockOpen'} size={14} />{/if}
  <span>{ws.switchingReadOnly ? 'Reconnecting…' : 'Read only'}</span>
  <span class="track" aria-hidden="true"></span>
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
  .top { display: flex; align-items: center; height: 33px; }
  .lights { width: 64px; flex: none; }
  .home { display: grid; place-items: center; width: 28px; height: 28px; border: 0; border-radius: 8px; background: transparent; color: var(--text-3); }
  .home:hover { background: var(--hover); color: var(--text); }
  .conn {
    --avatar-ring: var(--elevated);
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 6px 8px 0;
    padding: 6px 10px 6px 6px;
    border: 0;
    border-radius: 12px;
    background: transparent;
    color: var(--text-3);
    text-align: left;
    transition: background 0.12s;
  }
  .conn:hover, .conn[aria-expanded='true'] { background: var(--hover); }
  .meta { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .name { display: flex; align-items: center; gap: 6px; min-width: 0; color: var(--text); font-size: 14px; font-weight: 500; }
  .target { display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 12px; color: var(--text-3); }
  .conn > :global(.icon) { color: var(--text-2); margin-right: 4px; }
  .text { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .via { flex: none; display: inline-flex; align-items: center; gap: 3px; padding: 0 5px; border-radius: 4px; background: var(--active); color: var(--text-2); font-size: 10px; font-weight: 600; }
  .via.down { color: var(--warn); background: color-mix(in srgb, var(--warn) 14%, transparent); }
  .ro {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    margin: 14px 12px 0;
    padding: 0 10px;
    border: 0;
    border-radius: var(--radius);
    background: var(--hover);
    color: var(--text-2);
    font-size: 13px;
  }
  .ro:hover:not(:disabled) { color: var(--text); }
  .ro span:nth-child(2) { flex: 1; text-align: left; }
  .ro.on { color: var(--warn); background: color-mix(in srgb, var(--warn) 12%, var(--elevated)); }
  .track { position: relative; width: 28px; height: 16px; border-radius: 999px; background: var(--active); transition: background 0.15s; }
  .track::after { content: ''; position: absolute; top: 2px; left: 2px; width: 12px; height: 12px; border-radius: 50%; background: #fff; transition: transform 0.15s; }
  .ro.on .track { background: var(--warn); }
  .ro.on .track::after { transform: translateX(12px); }
</style>
