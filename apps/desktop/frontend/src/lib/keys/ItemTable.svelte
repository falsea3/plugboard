<script lang="ts">
  import { tick } from 'svelte';
  import type { KeyItem } from '../api/wire';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import { addChange, addFields, canEdit, itemColumns, removeChange, type Change } from './items';

  let {
    type,
    items,
    readOnly,
    done,
    loading,
    onmore,
    onchange,
  }: {
    type: string;
    items: KeyItem[];
    readOnly: boolean;
    done: boolean;
    loading: boolean;
    onmore: () => void;
    onchange: (ch: Change) => Promise<boolean>;
  } = $props();

  const columns = $derived(itemColumns(type));
  const fields = $derived(addFields(type));
  let editing = $state<{ row: number; col: number; text: string } | null>(null);
  let adding = $state<Record<string, string> | null>(null);
  let busy = $state(false);
  let input = $state<HTMLInputElement>();

  async function startEdit(row: number, col: number) {
    if (readOnly || !canEdit(items[row], columns[col])) return;
    editing = { row, col, text: columns[col].get(items[row]) };
    await tick();
    input?.focus();
    input?.select();
  }

  async function commit() {
    const e = editing;
    if (!e || busy) return;
    const it = items[e.row];
    const col = columns[e.col];
    if (e.text === col.get(it)) {
      editing = null;
      return;
    }
    busy = true;
    if (await onchange(col.edit!(it, e.text))) editing = null;
    busy = false;
  }

  function onEditKey(ev: KeyboardEvent) {
    if (ev.key === 'Enter') {
      ev.preventDefault();
      commit();
    }
    if (ev.key === 'Escape') {
      ev.stopPropagation();
      editing = null;
    }
  }

  async function remove(it: KeyItem) {
    const ch = removeChange(type, it);
    if (ch && !busy) {
      busy = true;
      await onchange(ch);
      busy = false;
    }
  }

  async function add(ev: SubmitEvent) {
    ev.preventDefault();
    if (!adding || busy) return;
    busy = true;
    if (await onchange(addChange(type, adding))) adding = null;
    busy = false;
  }

  function onscroll(ev: Event) {
    const el = ev.currentTarget as HTMLElement;
    if (!done && !loading && el.scrollTop + el.clientHeight > el.scrollHeight - 200) onmore();
  }
</script>

<div class="items" {onscroll}>
  <table>
    <thead>
      <tr>
        {#each columns as c (c.name)}<th>{c.name}</th>{/each}
        <th class="act"></th>
      </tr>
    </thead>
    <tbody>
      {#each items as it, r (it.field + ':' + r)}
        <tr>
          {#each columns as c, ci (c.name)}
            {@const text = c.get(it)}
            <td class:mono={c.mono} class:binary={it.binary || (ci === 0 && it.label)} ondblclick={() => startEdit(r, ci)} title={text}>
              {#if editing?.row === r && editing.col === ci}
                <input class="cell-input" bind:this={input} bind:value={editing.text} onkeydown={onEditKey} onblur={commit} disabled={busy} spellcheck="false" />
              {:else}
                {text}
              {/if}
            </td>
          {/each}
          <td class="act">
            {#if !readOnly && removeChange(type, it)}
              <button class="btn icon sm ghost" onclick={() => remove(it)} title="Remove" aria-label="Remove {columns[0].get(it)}"><Icon name="trash" size={12} /></button>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  {#if loading}
    <div class="more faint"><Spinner size={11} />Loading…</div>
  {:else if !done}
    <button class="btn sm ghost more" onclick={onmore}>Load more</button>
  {/if}
</div>

{#if !readOnly && fields.length > 0}
  {#if adding}
    <form class="add" onsubmit={add}>
      {#each fields as f, i (f.name)}
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input" aria-label={f.label} placeholder={f.placeholder} bind:value={adding[f.name]} autofocus={i === 0} spellcheck="false" />
      {/each}
      <button class="btn sm primary" type="submit" disabled={busy}>Add</button>
      <button class="btn sm" type="button" onclick={() => (adding = null)}>Cancel</button>
    </form>
  {:else}
    <div class="add">
      <button class="btn sm" onclick={() => (adding = Object.fromEntries(fields.map(f => [f.name, ''])))}><Icon name="plus" size={12} />Add {fields[0].label.toLowerCase()}</button>
      <span class="faint hint">Double-click a cell to edit it</span>
    </div>
  {/if}
{/if}

<style>
  .items { flex: 1; min-height: 0; overflow: auto; }
  table { width: 100%; border-collapse: collapse; table-layout: fixed; font-size: 12.5px; }
  th { position: sticky; top: 0; z-index: 1; height: 28px; padding: 0 10px; background: var(--surface); border-bottom: 1px solid var(--border); color: var(--text-3); font-weight: 600; font-size: 11.5px; text-align: left; }
  td { height: 28px; padding: 0 10px; border-bottom: 1px solid var(--border-subtle); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; user-select: text; -webkit-user-select: text; }
  tr:hover td { background: var(--hover); }
  .mono { font-family: var(--font-mono); font-size: 12px; }
  .binary { color: var(--text-3); font-family: var(--font-mono); }
  th.act, td.act { width: 36px; padding: 0 4px; text-align: center; }
  td.act :global(.btn) { opacity: 0; }
  tr:hover td.act :global(.btn) { opacity: 1; }
  .cell-input { width: 100%; height: 22px; padding: 0 6px; border: 1px solid var(--accent); border-radius: 4px; background: var(--bg); color: var(--text); font: inherit; }
  .more { display: flex; align-items: center; gap: 6px; margin: 8px auto; font-size: 12px; justify-content: center; }
  .add { flex: none; display: flex; align-items: center; gap: 6px; padding: 8px 10px; border-top: 1px solid var(--border); background: var(--surface); }
  .add .input { flex: 1; min-width: 0; }
  .hint { font-size: 11.5px; margin-left: 4px; }
</style>
