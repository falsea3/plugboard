<script lang="ts">
  import { untrack } from 'svelte';
  import type { Column, NewIndex, TableInfo } from '../api/wire';
  import Modal from '../ui/Modal.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import { indexName } from './tree';

  let {
    table,
    columns,
    first,
    onsubmit,
    onclose,
  }: { table: TableInfo; columns: Column[] | 'loading' | Error | undefined; first?: string; onsubmit: (idx: NewIndex) => void; onclose: () => void } = $props();

  let picked = $state<string[]>(untrack(() => (first ? [first] : [])));
  let unique = $state(false);
  let name = $state('');
  let named = $state(false);
  const shown = $derived(named ? name : indexName(table.name, picked, unique));

  function toggle(col: string) {
    picked = picked.includes(col) ? picked.filter(c => c !== col) : [...picked, col];
  }

  function submit(e: SubmitEvent) {
    e.preventDefault();
    if (picked.length > 0 && shown.trim()) onsubmit({ name: shown.trim(), columns: picked, unique });
  }
</script>

<Modal title="Create an index on {table.name}" width={440} {onclose}>
  <form id="index-form" onsubmit={submit}>
    <label class="label" for="index-name">Name</label>
    <input id="index-name" class="input mono" value={shown} oninput={e => ((name = e.currentTarget.value), (named = true))} spellcheck="false" autocomplete="off" />
    <span class="label">Columns <span class="faint">— in the order you pick them</span></span>
    <div class="cols" role="group" aria-label="Index columns">
      {#if Array.isArray(columns)}
        {#each columns as c (c.name)}
          {@const at = picked.indexOf(c.name)}
          <label class="col">
            <input type="checkbox" checked={at >= 0} onchange={() => toggle(c.name)} />
            <span class="order">{at >= 0 ? at + 1 : ''}</span>
            <span class="mono">{c.name}</span>
            <span class="type faint">{c.type}</span>
          </label>
        {/each}
      {:else if columns instanceof Error}
        <p class="error">{columns.message}</p>
      {:else}
        <p class="faint"><Spinner size={11} /> Loading columns…</p>
      {/if}
    </div>
    <label class="unique"><input type="checkbox" bind:checked={unique} />Unique — refuse two rows with the same values</label>
    <p class="hint faint">Opens the statement in a new query tab — look it over and run it there.</p>
  </form>
  {#snippet footer()}
    <span style="flex:1"></span>
    <button class="btn" type="button" onclick={onclose}>Cancel</button>
    <button class="btn primary" type="submit" form="index-form" disabled={picked.length === 0 || !shown.trim()}>Open SQL</button>
  {/snippet}
</Modal>

<style>
  form { display: flex; flex-direction: column; gap: 6px; }
  .label { margin-top: 6px; color: var(--text-2); font-size: 12px; }
  .cols { max-height: 220px; overflow-y: auto; padding: 4px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); }
  .col { display: flex; align-items: center; gap: 8px; height: 26px; padding: 0 6px; border-radius: 5px; font-size: 12.5px; }
  .col:hover { background: var(--hover); }
  .order { width: 12px; color: var(--accent); font-size: 11px; font-weight: 600; text-align: center; }
  .type { margin-left: auto; font-size: 11px; }
  .unique { display: flex; align-items: center; gap: 8px; margin-top: 6px; font-size: 12.5px; }
  .hint { margin: 4px 0 0; font-size: 11.5px; }
  .error { color: var(--danger); font-size: 12px; }
</style>
