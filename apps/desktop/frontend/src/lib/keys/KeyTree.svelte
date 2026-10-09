<script lang="ts">
  import { api, type KeyInfo } from '../api/backend';
  import { app } from '../app/app.svelte';
  import { FILTER_TABLES_EVENT } from '../app/commands';
  import type { Workspace } from '../app/workspace.svelte';
  import GridMenu from '../grid/GridMenu.svelte';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import NewKeyModal from './NewKeyModal.svelte';
  import ScriptList from '../scripts/ScriptList.svelte';
  import type { Change } from './items';
  import { keyMenu, keyRows } from './folders';
  import { TYPE_LABEL } from './items';

  let { ws, active }: { ws: Workspace; active: boolean } = $props();

  const keys = $derived(ws.keys);
  const rows = $derived(keyRows(keys.keys, keys.open));
  const activeKey = $derived(ws.activeTab?.kind === 'key' ? ws.activeTab.key.key : '');
  let filter = $state('');
  let input = $state<HTMLInputElement>();
  let timer: ReturnType<typeof setTimeout> | undefined;
  let menu = $state<{ x: number; y: number; key: KeyInfo } | null>(null);
  let deleting = $state<KeyInfo | null>(null);
  let creating = $state(false);

  function onFilter() {
    clearTimeout(timer);
    timer = setTimeout(() => keys.search(filter), 250);
  }

  function toggle(path: string) {
    if (keys.open.has(path)) keys.open.delete(path);
    else keys.open.add(path);
  }

  function pick(id: string, k: KeyInfo) {
    menu = null;
    if (id === 'key-open') ws.openKey(k);
    else if (id === 'key-copy-name') copyToClipboard(k.name);
    else if (id === 'key-delete') deleting = k;
  }

  async function remove() {
    const k = deleting;
    deleting = null;
    if (!k) return;
    try {
      await api.editKey(ws.session.sessionId, ws.schema, { key: k.key, op: 'delete' });
      keys.removed(k.key);
      ws.closeKey(k.key);
    } catch (err) {
      app.notify(err);
    }
  }

  async function create(name: string, ch: Change, ttl: number): Promise<boolean> {
    try {
      await api.editKey(ws.session.sessionId, ws.schema, { ...ch, key: name });
      if (ttl > 0) await api.editKey(ws.session.sessionId, ws.schema, { key: name, op: 'expire', index: ttl });
    } catch (err) {
      app.notify(err);
      return false;
    }
    creating = false;
    const k = { name, key: name, type: ch.op.slice('create:'.length) };
    keys.added(k);
    ws.openKey(k);
    return true;
  }

  $effect(() => {
    const focus = () => active && (input?.focus(), input?.select());
    window.addEventListener(FILTER_TABLES_EVENT, focus);
    return () => window.removeEventListener(FILTER_TABLES_EVENT, focus);
  });
</script>

<div class="bar">
  <div class="filter">
    <Icon name="search" size={13} />
    <input class="input" bind:this={input} bind:value={filter} oninput={onFilter} onkeydown={e => e.key === 'Escape' && ((filter = ''), onFilter())} placeholder="Filter keys (user:* or text)" spellcheck="false" aria-label="Filter keys" />
  </div>
  {#if !ws.readOnly}<button class="btn icon sm ghost" onclick={() => (creating = true)} title="New key" aria-label="New key"><Icon name="plus" size={13} /></button>{/if}
  <button class="btn icon sm ghost" onclick={() => keys.load()} title="Reload keys" aria-label="Reload keys"><Icon name="refresh" size={13} /></button>
</div>

<div class="list" role="tree" aria-label="Keys">
  {#each rows as row (row.kind === 'folder' ? 'f:' + row.path : 'k:' + row.key.key)}
    {#if row.kind === 'folder'}
      <button class="row" style:padding-left="{4 + row.depth * 14}px" role="treeitem" aria-expanded={row.open} aria-selected="false" onclick={() => toggle(row.path)}>
        <span class="twist" class:open={row.open}><Icon name="chevron-right" size={9} /></span>
        <Icon name="folder" size={13} />
        <span class="label">{row.name}</span>
        <span class="count">{row.count}</span>
      </button>
    {:else}
      <button
        class="row"
        class:active={activeKey === row.key.key}
        style:padding-left="{4 + row.depth * 14}px"
        role="treeitem"
        aria-selected={activeKey === row.key.key}
        title={row.key.name}
        onclick={() => ws.openKey(row.key)}
        oncontextmenu={e => (e.preventDefault(), (menu = { x: e.clientX, y: e.clientY, key: row.key }))}
      >
        <span class="twist"></span>
        <span class="type t-{row.key.type}">{TYPE_LABEL[row.key.type] ?? row.key.type}</span>
        <span class="label mono">{row.name}</span>
      </button>
    {/if}
  {/each}
  {#if keys.loading}
    <div class="note faint"><Spinner size={11} />Scanning keys…</div>
  {:else if keys.error}
    <div class="note error">{keys.error}</div>
  {:else if keys.keys.length === 0}
    <div class="note faint">{filter ? `No keys match “${filter}”.` : 'This database is empty.'}</div>
  {/if}
  {#if !keys.done && !keys.loading}
    <button class="btn sm ghost more" onclick={() => keys.more()}>Load more keys</button>
  {/if}
  <ScriptList {ws} filter="" />
</div>
<div class="status faint">{keys.keys.length.toLocaleString('en-US')} keys{keys.done ? '' : ' loaded'}</div>

{#if menu}
  {@const m = menu}
  <GridMenu items={keyMenu(ws.readOnly)} x={m.x} y={m.y} onpick={id => pick(id, m.key)} onclose={() => (menu = null)} />
{/if}

{#if deleting}
  <Modal title="Delete “{deleting.name}”?" width={400} onclose={() => (deleting = null)}>
    <p class="text">The key and its value are deleted from database {ws.schema}. This can’t be undone.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (deleting = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={remove}>Delete</button>
    {/snippet}
  </Modal>
{/if}

{#if creating}
  <NewKeyModal db={ws.schema} onsubmit={create} onclose={() => (creating = false)} />
{/if}

<style>
  .bar { flex: none; display: flex; align-items: center; gap: 2px; padding: 6px 8px; }
  .filter { position: relative; flex: 1; min-width: 0; display: flex; align-items: center; }
  .filter :global(.icon) { position: absolute; left: 8px; color: var(--text-3); pointer-events: none; }
  .filter .input { width: 100%; padding-left: 26px; }
  .list { flex: 1; min-height: 0; overflow: auto; padding: 0 6px 8px; }
  .row { width: 100%; display: flex; align-items: center; gap: 6px; height: 26px; padding-right: 8px; border: 0; border-radius: 5px; background: transparent; color: var(--text-2); text-align: left; }
  .row:hover { background: var(--hover); color: var(--text); }
  .row.active { background: var(--accent-dim); color: var(--text); }
  .row :global(.icon) { color: var(--text-3); }
  .twist { flex: none; display: flex; align-items: center; justify-content: center; width: 12px; transition: transform 0.12s; }
  .twist.open { transform: rotate(90deg); }
  .label { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; }
  .label.mono { font-family: var(--font-mono); font-size: 12px; }
  .count { color: var(--text-3); font-size: 11px; }
  .type { flex: none; width: 42px; color: var(--accent); font-size: 9.5px; font-weight: 650; letter-spacing: 0.03em; }
  .note { display: flex; align-items: center; gap: 6px; padding: 8px; font-size: 12px; }
  .note.error { color: var(--danger); }
  .more { margin: 4px 8px; }
  .status { flex: none; padding: 4px 12px 6px; font-size: 11px; border-top: 1px solid var(--border-subtle); }
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
