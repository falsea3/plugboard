<script lang="ts">
  import { untrack } from 'svelte';
  import { api, type Index } from '../api/backend';
  import { formatCount } from '../ui/format';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let { sessionId, schema, table }: { sessionId: string; schema: string; table: string } = $props();

  let indexes = $state<Index[] | null>(null);
  let loading = $state(false);
  let error = $state('');

  export async function reload() {
    loading = true;
    error = '';
    try {
      indexes = await api.indexes(sessionId, schema, table);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    untrack(reload);
  });

  const kind = (ix: Index) => (ix.primary ? 'Primary key' : ix.unique ? 'Unique' : 'Index');
</script>

<section class="indexes" aria-label="Indexes">
  <header>
    <span class="label">Indexes</span>
    {#if indexes}<span class="faint">{formatCount(indexes.length, 'index', 'indexes')}</span>{/if}
    {#if loading}<Spinner size={10} />{/if}
  </header>
  {#if error}
    <p class="note error"><Icon name="alert" size={12} />{error}</p>
  {:else if indexes && indexes.length === 0}
    <p class="note faint">This table has no indexes.</p>
  {:else if indexes}
    <div class="scroll">
      <table>
        <thead>
          <tr><th>Name</th><th>Columns</th><th>Kind</th><th>Method</th><th>Where</th></tr>
        </thead>
        <tbody>
          {#each indexes as ix (ix.name)}
            <tr title={ix.definition || undefined}>
              <td class="mono name">{#if ix.primary}<Icon name="key" size={11} />{/if}{ix.name}</td>
              <td class="mono">{ix.columns.join(', ')}</td>
              <td><span class="kind" class:strong={ix.primary || ix.unique}>{kind(ix)}</span></td>
              <td class="mono faint">{ix.method}</td>
              <td class="mono faint">{ix.where}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>

<style>
  .indexes {
    flex: none;
    max-height: 38%;
    display: flex;
    flex-direction: column;
    margin-top: 10px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    background: var(--card);
    overflow: hidden;
  }
  header {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 40px;
    padding: 0 14px;
    border-bottom: 1px solid var(--grid-line);
    font-size: 12px;
  }
  .label { font-weight: 500; font-size: 13px; color: var(--text); }
  .note { display: flex; align-items: center; gap: 6px; margin: 0; padding: 0 12px 12px; font-size: 12px; }
  .note.error { color: var(--danger); user-select: text; -webkit-user-select: text; }
  .scroll { min-height: 0; overflow: auto; padding: 0 0 6px; }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th {
    position: sticky;
    top: 0;
    height: 32px;
    padding: 0 14px;
    background: var(--card);
    color: var(--text-3);
    font-size: 11.5px;
    font-weight: 500;
    text-align: left;
    white-space: nowrap;
  }
  td { height: 32px; padding: 0 14px; border-top: 1px solid var(--grid-line); white-space: nowrap; color: var(--text); }
  td.name { display: flex; align-items: center; gap: 6px; }
  td.name :global(.icon) { color: var(--text-3); }
  .mono { font-family: var(--font-mono); font-size: 12px; }
  .kind { padding: 1px 7px; border-radius: 6px; background: var(--hover); font-size: 11.5px; color: var(--text-2); }
  .kind.strong { color: var(--text); }
</style>
