<script lang="ts">
  import { keys } from '../ui/keys';
  import { untrack } from 'svelte';
  import { api, type KeyItem, type KeyValue } from '../api/backend';
  import { app } from '../app/app.svelte';
  import { REFRESH_EVENT } from '../app/commands';
  import type { KeyTab, Workspace } from '../app/workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import ItemTable from './ItemTable.svelte';
  import KeyActions from './KeyActions.svelte';
  import StringValue from './StringValue.svelte';
  import { formatTTL, patchItems, sizeLabel, TYPE_LABEL, type Change } from './items';

  let { ws, tab, active }: { ws: Workspace; tab: KeyTab; active: boolean } = $props();

  const sessionId = untrack(() => ws.session.sessionId);
  let value = $state<KeyValue | null>(null);
  let items = $state<KeyItem[]>([]);
  let loading = $state(false);
  let error = $state('');
  let seq = 0;

  async function load(cursor = '') {
    const n = cursor ? seq : ++seq;
    loading = true;
    if (!cursor) error = '';
    try {
      const v = await api.readKey(sessionId, tab.db, tab.key.key, cursor);
      if (n !== seq) return;
      value = v;
      items = cursor ? [...items, ...v.items] : v.items;
    } catch (err) {
      if (n !== seq) return;
      error = err instanceof Error ? err.message : String(err);
      if (!cursor) value = null;
    } finally {
      if (n === seq) loading = false;
    }
  }

  async function change(ch: Change): Promise<boolean> {
    try {
      await api.editKey(sessionId, tab.db, { key: tab.key.key, ...ch });
    } catch (err) {
      app.notify(err);
      return false;
    }
    const patched = value && patchItems(items, ch);
    if (patched && value) {
      value.size += patched.length - items.length;
      items = patched;
    } else {
      await load();
    }
    return true;
  }

  async function saveText(text: string) {
    const ok = await change({ op: 'set', value: text });
    if (ok && value) value.text = text;
    return ok;
  }

  $effect(() => {
    untrack(() => load());
  });

  $effect(() => {
    const reload = () => active && load();
    window.addEventListener(REFRESH_EVENT, reload);
    return () => window.removeEventListener(REFRESH_EVENT, reload);
  });

  const note = $derived(
    !value ? '' : value.binary ? 'Binary value — shown escaped, read-only.' : value.truncated ? 'Showing the first 1 MB — read-only.' : '',
  );
</script>

<div class="view pane-card boxed">
  {#if loading}<LoadBar />{/if}
  <div class="pane-head">
    {#if value}<span class="type t-{value.type}">{TYPE_LABEL[value.type] ?? value.type.toUpperCase()}</span>{/if}
    <span class="name mono" title={tab.key.name}>{tab.key.name}</span>
    <button class="btn icon ghost" onclick={() => copyToClipboard(tab.key.name)} title="Copy key name" aria-label="Copy key name"><Icon name="copy" size={13} /></button>
    <span class="space"></span>
    {#if value}
      <span class="meta faint">{sizeLabel(value.type, value.size)}</span>
      <span class="meta faint" title="Time to live">TTL {formatTTL(value.ttl)}</span>
      <KeyActions {ws} {tab} ttl={value.ttl} onchange={change} />
    {/if}
    <button class="btn ghost icon" onclick={() => load()} title={keys('Refresh (⌘R)')} aria-label="Refresh"><Icon name="refresh" size={14} /></button>
  </div>

  <div class="body">
  {#if error}
    <div class="error">{error}</div>
  {:else if value?.type === 'string'}
    <StringValue text={value.text} readOnly={ws.readOnly || !!value.binary || !!value.truncated} {note} onsave={saveText} />
  {:else if value && ['hash', 'list', 'set', 'zset', 'stream'].includes(value.type)}
    <ItemTable type={value.type} {items} readOnly={ws.readOnly} done={value.done} {loading} onmore={() => value && load(value.cursor)} onchange={change} />
  {:else if value}
    <div class="error faint">{value.text}</div>
  {/if}
  </div>
</div>

<style>
  .view { position: relative; height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .space { flex: 1; }
  .body { display: flex; flex-direction: column; }
  .type { flex: none; padding: 1px 7px; border-radius: 6px; background: var(--accent-bg); color: var(--accent-text); font-size: 11px; font-weight: 600; letter-spacing: 0.03em; }
  .name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--font-mono); font-size: 13px; user-select: text; -webkit-user-select: text; }
  .meta { flex: none; padding: 2px 8px; border-radius: 6px; background: var(--surface); color: var(--text-3); font-size: 12px; white-space: nowrap; }
  .error { padding: 16px; color: var(--danger); font-size: 12.5px; }
  .error.faint { color: var(--text-3); }
</style>
