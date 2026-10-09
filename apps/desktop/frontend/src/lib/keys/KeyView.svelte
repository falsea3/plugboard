<script lang="ts">
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

<div class="view">
  {#if loading}<LoadBar />{/if}
  <header>
    {#if value}<span class="type t-{value.type}">{TYPE_LABEL[value.type] ?? value.type.toUpperCase()}</span>{/if}
    <span class="name mono" title={tab.key.name}>{tab.key.name}</span>
    <button class="btn icon sm ghost" onclick={() => copyToClipboard(tab.key.name)} title="Copy key name" aria-label="Copy key name"><Icon name="copy" size={12} /></button>
    <span style="flex:1"></span>
    {#if value}
      <span class="meta faint">{sizeLabel(value.type, value.size)}</span>
      <span class="meta faint" title="Time to live">TTL {formatTTL(value.ttl)}</span>
      <KeyActions {ws} {tab} ttl={value.ttl} onchange={change} />
    {/if}
    <button class="btn icon sm ghost" onclick={() => load()} title="Refresh (⌘R)" aria-label="Refresh"><Icon name="refresh" size={13} /></button>
  </header>

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

<style>
  .view { position: relative; height: 100%; display: flex; flex-direction: column; min-width: 0; }
  header { flex: none; display: flex; align-items: center; gap: 8px; height: 36px; padding: 0 8px 0 12px; background: var(--surface); border-bottom: 1px solid var(--border); min-width: 0; }
  .type { flex: none; padding: 0 6px; border: 1px solid currentColor; border-radius: 4px; font-size: 10.5px; font-weight: 650; letter-spacing: 0.03em; color: var(--accent); }
  .name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; user-select: text; -webkit-user-select: text; }
  .meta { flex: none; font-size: 11.5px; white-space: nowrap; }
  .error { padding: 16px; color: var(--danger); font-size: 12.5px; }
  .error.faint { color: var(--text-3); }
</style>
