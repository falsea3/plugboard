<script lang="ts">
  import { untrack } from 'svelte';
  import { api, type DBObject } from '../api/backend';
  import type { Workspace } from '../app/workspace.svelte';
  import { engine } from '../engines';
  import { copyToClipboard } from '../ui/clipboard';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import SqlView from '../query/SqlView.svelte';
  import { iconOf } from './kinds';

  let { ws, object, toolbar = true }: { ws: Workspace; object: DBObject; toolbar?: boolean } = $props();

  const dialect = untrack(() => engine(ws.session.connection.driver).sqlDialect);
  let text = $state('');
  let loading = $state(false);
  let error = $state('');

  export function copy() {
    if (text) copyToClipboard(text);
  }

  export function openInQuery() {
    if (text) ws.newQuery(text);
  }

  export async function load() {
    loading = true;
    error = '';
    try {
      text = await api.objectDDL(ws.session.sessionId, object);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  untrack(load);
</script>

<div class="ddl">
  {#if toolbar}
    <div class="bar">
      <Icon name={iconOf(object)} size={13} />
      <span class="title mono faint">{object.schema ? object.schema + '.' : ''}<span class="muted">{object.name}</span>{object.kind === 'function' || object.kind === 'procedure' ? `(${object.detail ?? ''})` : ''}</span>
      <span style="flex:1"></span>
      <button class="btn sm ghost" onclick={copy} disabled={!text}><Icon name="copy" size={13} />Copy</button>
      <button class="btn sm ghost" onclick={openInQuery} disabled={!text}><Icon name="code" size={13} />Open in query</button>
      <button class="btn icon sm ghost" onclick={load} disabled={loading} title="Reload" aria-label="Reload">{#if loading}<Spinner size={12} />{:else}<Icon name="refresh" size={13} />{/if}</button>
    </div>
  {/if}
  <div class="body">
    {#if loading}<LoadBar label="Loading the DDL" />{/if}
    {#if error}
      <div class="error" role="alert"><Icon name="alert" />{error}</div>
    {:else if text}
      <SqlView {text} {dialect} />
    {/if}
  </div>
</div>

<style>
  .ddl { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .bar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 10px 0 14px;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-2);
  }
  .title { font-size: 12px; }
  .body { position: relative; flex: 1; min-height: 0; }
  .error {
    display: flex;
    gap: 8px;
    margin: 16px;
    padding: 10px 12px;
    border-radius: var(--radius);
    color: var(--danger);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
    font-size: 12px;
    user-select: text;
    -webkit-user-select: text;
  }
</style>
