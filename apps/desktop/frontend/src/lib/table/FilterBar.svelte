<script lang="ts">
  import { keys } from '../ui/keys';
  import { tick } from 'svelte';
  import Icon from '../ui/Icon.svelte';
  import Select from '../ui/Select.svelte';
  import { OPS, newFilter, noValue, ready, type FilterRow } from './filters';

  type Col = { name: string };

  let {
    columns,
    rows = $bindable(),
    onapply,
    onclose,
  }: {
    columns: Col[];
    rows: FilterRow[];
    onapply: () => void;
    onclose: () => void;
  } = $props();

  let bar = $state<HTMLDivElement>();

  export async function focus(id?: number) {
    await tick();
    const all = bar?.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-focus]') ?? [];
    const target = id === undefined ? all[all.length - 1] : bar?.querySelector<HTMLElement>(`[data-row="${id}"] [data-focus]`);
    (target as HTMLElement | undefined)?.focus();
  }

  function add() {
    const last = rows[rows.length - 1];
    const row = newFilter(columns, last?.column);
    rows = [...rows, row];
    focus(row.id);
  }

  function remove(id: number) {
    rows = rows.filter(r => r.id !== id);
    if (rows.length === 0) onclose();
    onapply();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      onapply();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      onclose();
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="filters" bind:this={bar} {onkeydown}>
  {#each rows as row, i (row.id)}
    <div class="row" data-row={row.id} class:off={!row.on}>
      <span class="join">{i === 0 ? 'where' : 'and'}</span>
      <input type="checkbox" bind:checked={row.on} onchange={onapply} title={row.on ? 'Turn this condition off' : 'Turn this condition on'} />
      <div class="col"><Select bind:value={row.column} options={columns.map(c => ({ value: c.name, label: c.name }))} onchange={() => ready(row) && onapply()} aria-label="Column" /></div>
      <div class="op"><Select bind:value={row.op} options={OPS} onchange={() => ready(row) && onapply()} aria-label="Operator" /></div>
      {#if noValue(row.op)}
        <span class="no-value" data-focus tabindex="-1"></span>
      {:else}
        <input
          class="input value mono"
          bind:value={row.value}
          data-focus
          placeholder={row.op === 'in' || row.op === 'not_in' ? 'a, b, c' : 'value'}
          spellcheck="false"
          autocomplete="off"
        />
      {/if}
      <button class="btn icon sm ghost" onclick={() => remove(row.id)} title="Remove condition" aria-label="Remove condition"><Icon name="x" size={12} /></button>
    </div>
  {/each}
  <div class="actions">
    <button class="btn sm ghost" onclick={add}><Icon name="plus" size={12} />Condition</button>
    <span style="flex:1"></span>
    <span class="hint faint"><span class="kbd">{keys('↵')}</span> apply · <span class="kbd">esc</span> close</span>
    <button class="btn sm ghost" onclick={() => { rows = []; onapply(); onclose(); }}>Clear</button>
    <button class="btn sm primary" onclick={onapply}>Apply</button>
  </div>
</div>

<style>
  .filters { flex: none; display: flex; flex-direction: column; gap: 8px; margin-top: 8px; padding: 10px 12px; border: 1px solid var(--hover); border-radius: var(--radius); background: var(--surface); }
  .row { display: flex; align-items: center; gap: 8px; }
  .row.off .col, .row.off .op, .row.off .value { opacity: 0.45; }
  .join { width: 40px; flex: none; color: var(--text-4); font-family: var(--font-mono); font-size: 12.5px; }
  input[type='checkbox'] { flex: none; margin: 0; accent-color: var(--accent); }
  .col { width: 200px; flex: none; }
  .op { width: 130px; flex: none; }
  .col :global(.select-button), .op :global(.select-button) { height: 32px; border-radius: 9px; }
  .value { flex: 1; min-width: 120px; height: 32px; border-radius: 9px; font-size: 12.5px; }
  .no-value { flex: 1; }
  .actions { display: flex; align-items: center; gap: 6px; padding-left: 48px; }
  .actions .btn.sm { height: 30px; border-radius: 9px; }
  .hint { font-size: 11.5px; margin-right: 6px; }
</style>
