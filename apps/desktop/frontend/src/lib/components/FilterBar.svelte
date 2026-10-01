<script lang="ts" module>
  import type { FilterOp } from '../wire';

  type Col = { name: string };

  export type FilterRow = { id: number; on: boolean; column: string; op: FilterOp; value: string };

  export const OPS: { value: FilterOp; label: string }[] = [
    { value: '=', label: '=' },
    { value: '!=', label: '≠' },
    { value: '<', label: '<' },
    { value: '>', label: '>' },
    { value: '<=', label: '≤' },
    { value: '>=', label: '≥' },
    { value: 'contains', label: 'contains' },
    { value: 'not_contains', label: 'not contains' },
    { value: 'starts', label: 'starts with' },
    { value: 'ends', label: 'ends with' },
    { value: 'in', label: 'in list' },
    { value: 'not_in', label: 'not in list' },
    { value: 'empty', label: 'is empty' },
    { value: 'not_empty', label: 'is not empty' },
    { value: 'null', label: 'is NULL' },
    { value: 'not_null', label: 'is not NULL' },
  ];

  export const noValue = (op: FilterOp) => op === 'null' || op === 'not_null' || op === 'empty' || op === 'not_empty';

  let nextId = 0;
  export function newFilter(columns: Col[], column?: string, op: FilterOp = '=', value = ''): FilterRow {
    return { id: ++nextId, on: true, column: column ?? columns[0]?.name ?? '', op, value };
  }
</script>

<script lang="ts">
  import { tick } from 'svelte';
  import Icon from './Icon.svelte';

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
      <span class="join">{i === 0 ? 'Where' : 'and'}</span>
      <input type="checkbox" bind:checked={row.on} onchange={onapply} title={row.on ? 'Turn this condition off' : 'Turn this condition on'} />
      <select class="select col" bind:value={row.column} aria-label="Column">
        {#each columns as c (c.name)}<option value={c.name}>{c.name}</option>{/each}
      </select>
      <select class="select op" bind:value={row.op} aria-label="Operator" onchange={() => noValue(row.op) && onapply()}>
        {#each OPS as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
      </select>
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
    <span class="hint faint"><span class="kbd">↵</span> apply · <span class="kbd">esc</span> close</span>
    <button class="btn sm ghost" onclick={() => { rows = []; onapply(); onclose(); }}>Clear</button>
    <button class="btn sm primary" onclick={onapply}>Apply</button>
  </div>
</div>

<style>
  .filters {
    flex: none;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 10px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
  }
  .row { display: flex; align-items: center; gap: 6px; }
  .row.off .col, .row.off .op, .row.off .value { opacity: 0.45; }
  .join { width: 42px; flex: none; text-align: right; color: var(--text-3); font-size: 12px; }
  .col { width: 180px; flex: none; height: 26px; }
  .op { width: 120px; flex: none; height: 26px; }
  .value { flex: 1; min-width: 120px; height: 26px; }
  .no-value { flex: 1; }
  .actions { display: flex; align-items: center; gap: 6px; padding-left: 48px; }
  .hint { font-size: 11.5px; margin-right: 6px; }
</style>
