<script lang="ts">
  import type { Workspace } from '../stores/app.svelte';
  import { FILTER_TABLES_EVENT } from '../commands';
  import { connectionTarget } from '../format';
  import ConnAvatar from './ConnAvatar.svelte';
  import EnvBadge from './EnvBadge.svelte';
  import Icon from './Icon.svelte';
  import Select from './Select.svelte';
  import { engine } from '../engines';
  import Modal from './Modal.svelte';

  let { ws, active }: { ws: Workspace; active: boolean } = $props();

  let filter = $state('');
  let input = $state<HTMLInputElement>();

  const conn = $derived(ws.connection);
  const visible = $derived.by(() => {
    const q = filter.trim().toLowerCase();
    return q ? ws.tables.filter(t => t.name.toLowerCase().includes(q)) : ws.tables;
  });
  const tables = $derived(visible.filter(t => t.kind === 'table'));
  const views = $derived(visible.filter(t => t.kind === 'view'));
  const activeTable = $derived(ws.activeTab?.kind === 'table' ? ws.activeTab : null);

  $effect(() => {
    const focus = () => {
      if (!active) return;
      input?.focus();
      input?.select();
    };
    window.addEventListener(FILTER_TABLES_EVENT, focus);
    return () => window.removeEventListener(FILTER_TABLES_EVENT, focus);
  });

  let confirmWrites = $state(false);

  function toggleReadOnly() {
    if (ws.readOnly && conn.env === 'prod') {
      confirmWrites = true;
      return;
    }
    ws.setReadOnly(!ws.readOnly);
  }

  function onFilterKey(e: KeyboardEvent) {
    if (e.key === 'Enter' && visible.length > 0) ws.openTable(visible[0]);
    if (e.key === 'Escape') filter = '';
  }
</script>

<aside class="sidebar">
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
    <Icon name={ws.readOnly ? 'lock' : 'lockOpen'} size={13} />
    <span>{ws.switchingReadOnly ? 'Reconnecting…' : 'Read-only'}</span>
    <span class="ro-switch" aria-hidden="true"></span>
  </button>

  {#if ws.session.schemas.length > 1}
    <div class="schema">
      <Select value={ws.schema} options={ws.session.schemas.map(s => ({ value: s, label: s }))} onchange={s => ws.setSchema(s)} aria-label="Schema" />
    </div>
  {/if}

  <div class="filter">
    <Icon name="search" size={13} />
    <input class="input" bind:this={input} bind:value={filter} onkeydown={onFilterKey} placeholder="Filter tables" spellcheck="false" />
  </div>

  <div class="list">
    {#if ws.tablesLoading && ws.tables.length === 0}
      <div class="note faint">Loading…</div>
    {:else if ws.tablesError}
      <div class="note error">{ws.tablesError}</div>
    {:else}
      {#each [{ label: 'Tables', items: tables, icon: 'table' as const }, { label: 'Views', items: views, icon: 'view' as const }] as group (group.label)}
        {#if group.items.length > 0 || (group.label === 'Tables' && !filter)}
          <div class="group">
            <span>{group.label}</span>
            <span class="count">{group.items.length}</span>
          </div>
          {#each group.items as t (t.name)}
            <button
              class="item"
              class:active={activeTable?.table === t.name && activeTable?.schema === t.schema}
              onclick={() => ws.openTable(t)}
              title={t.name}
            >
              <Icon name={group.icon} size={13} />
              <span class="label">{t.name}</span>
            </button>
          {/each}
        {/if}
      {/each}
      {#if visible.length === 0 && filter}
        <div class="note faint">Nothing matches “{filter}”.</div>
      {/if}
    {/if}
  </div>

  <div class="bottom">
    <button class="btn sm ghost" onclick={() => ws.newQuery()} title="New query (⌘T)"><Icon name="code" size={13} />New query</button>
    <span style="flex:1"></span>
    <button class="btn icon sm ghost" onclick={() => ws.loadTables()} title="Reload tables"><Icon name="refresh" size={13} /></button>
  </div>
</aside>

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
  .sidebar {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border-right: 1px solid var(--border);
    min-width: 0;
  }
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
  .schema { padding: 10px 10px 0; }
  .filter { position: relative; padding: 10px; color: var(--text-3); }
  .filter :global(.icon) { position: absolute; left: 19px; top: 18px; }
  .filter .input { padding-left: 27px; height: 26px; }

  .list { flex: 1; overflow-y: auto; padding: 0 6px 8px; }
  .group {
    display: flex;
    justify-content: space-between;
    padding: 10px 8px 4px;
    color: var(--text-3);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .count { font-weight: 500; }
  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 7px;
    height: 26px;
    padding: 0 8px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    text-align: left;
  }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.active { background: var(--accent-dim); color: var(--text); }
  .item :global(.icon) { color: var(--text-3); }
  .item.active :global(.icon) { color: var(--accent); }
  .label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; }
  .note { padding: 12px 8px; font-size: 12px; }
  .note.error { color: var(--danger); user-select: text; -webkit-user-select: text; }

  .bottom {
    display: flex;
    align-items: center;
    padding: 6px 8px;
    border-top: 1px solid var(--border-subtle);
  }
</style>
