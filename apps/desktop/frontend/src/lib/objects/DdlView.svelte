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

  const dialect = untrack(() => engine(ws.session.connection.driver).sqlDialect!);
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
    <section class="pane-card boxed">
      <div class="pane-head">
        <span class="pane-title">{object.schema ? object.schema + '.' : ''}<strong>{object.name}</strong>{object.kind === 'function' || object.kind === 'procedure' ? `(${object.detail ?? ''})` : ''} · {object.kind}</span>
        <button class="btn ghost" onclick={copy} disabled={!text}><Icon name="copy" size={13} />Copy</button>
        <button class="btn ghost" onclick={openInQuery} disabled={!text}><Icon name="code" size={13} />Open in query</button>
        <button class="btn ghost icon" onclick={load} disabled={loading} title="Reload" aria-label="Reload">{#if loading}<Spinner size={12} />{:else}<Icon name="refresh" size={13} />{/if}</button>
      </div>
      {@render content()}
    </section>
  {:else}
    {@render content()}
  {/if}
</div>

{#snippet content()}
  <div class="body">
    {#if loading}<LoadBar label="Loading the DDL" />{/if}
    {#if error}
      <div class="error" role="alert"><Icon name="alert" />{error}</div>
    {:else if text}
      <SqlView {text} {dialect} />
    {/if}
  </div>
{/snippet}

<style>
  .ddl { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .body { position: relative; flex: 1; min-height: 0; }
  .error { display: flex; gap: 8px; margin: 16px; padding: 10px 12px; border-radius: var(--radius); color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); font-size: 12px; user-select: text; -webkit-user-select: text; }
</style>
