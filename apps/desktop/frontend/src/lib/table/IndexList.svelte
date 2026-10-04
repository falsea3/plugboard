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
    border-top: 1px solid var(--border);
    background: var(--surface);
  }
  header {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 30px;
    padding: 0 12px;
    font-size: 12px;
  }
  .label { font-weight: 600; color: var(--text); }
  .note { display: flex; align-items: center; gap: 6px; margin: 0; padding: 0 12px 12px; font-size: 12px; }
  .note.error { color: var(--danger); user-select: text; -webkit-user-select: text; }
  .scroll { min-height: 0; overflow: auto; padding: 0 0 6px; }
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  th {
    position: sticky;
    top: 0;
    padding: 4px 12px;
    background: var(--surface);
    color: var(--text-3);
    font-weight: 500;
    text-align: left;
    white-space: nowrap;
  }
  td { padding: 4px 12px; border-top: 1px solid var(--border-subtle); white-space: nowrap; color: var(--text); }
  td.name { display: flex; align-items: center; gap: 6px; }
  td.name :global(.icon) { color: var(--text-2); }
  .mono { font-family: var(--font-mono); }
  .kind { padding: 0 6px; border: 1px solid var(--border); border-radius: 4px; font-size: 11px; color: var(--text-2); }
  .kind.strong { color: var(--text); }
</style>
