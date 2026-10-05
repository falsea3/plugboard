<script lang="ts">
  import { untrack } from 'svelte';
  import type { Column, Index, TableInfo } from '../api/wire';
  import type { Workspace } from './workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    ws,
    t,
    active,
    picked,
    onclick,
    onmenu,
  }: {
    ws: Workspace;
    t: TableInfo;
    active: boolean;
    picked: boolean;
    onclick: (e: MouseEvent) => void;
    onmenu: (e: MouseEvent, column?: Column, index?: Index) => void;
  } = $props();

  let open = $state(false);
  const columns = $derived(open ? ws.columns.get(`${t.schema}.${t.name}`) : undefined);
  const indexes = $derived(open ? ws.indexes.get(`${t.schema}.${t.name}`) : undefined);

  function toggle(on = !open) {
    open = on;
  }

  $effect(() => {
    if (open && ws.columns.get(`${t.schema}.${t.name}`) === undefined) untrack(() => ws.loadColumns(t));
    if (open && t.kind === 'table' && ws.indexes.get(`${t.schema}.${t.name}`) === undefined) untrack(() => ws.loadIndexes(t));
  });

  function onitemdblclick(e: MouseEvent) {
    if (e.target instanceof Element && e.target.closest('.twist')) return;
    toggle();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowRight' && !open) toggle(true);
    else if (e.key === 'ArrowLeft' && open) toggle(false);
    else return;
    e.preventDefault();
  }
</script>

<button class="item" class:active class:picked {onclick} {onkeydown} ondblclick={onitemdblclick} oncontextmenu={e => onmenu(e)} title={t.name} aria-expanded={open}>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <span
    class="twist"
    class:open
    onclick={e => {
      e.stopPropagation();
      toggle();
    }}
  ><Icon name="chevron-right" size={10} /></span>
  <Icon name={t.kind === 'view' ? 'view' : 'table'} size={13} />
  <span class="label">{t.name}</span>
</button>

{#if open}
  <div class="columns" role="group" aria-label="Columns of {t.name}">
    {#if columns === 'loading' || columns === undefined}
      <div class="col faint"><Spinner size={10} />Loading columns…</div>
    {:else if columns instanceof Error}
      <div class="col error" title={columns.message}>{columns.message}</div>
    {:else}
      {#each columns as c (c.name)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="col" oncontextmenu={e => onmenu(e, c)} ondblclick={() => ws.openTable(t, undefined, 'structure')} title="{c.name} {c.type}">
          <span class="key">{#if c.primaryKey}<Icon name="key" size={10} />{/if}</span>
          <span class="name">{c.name}</span>
          <span class="type">{c.type}</span>
        </div>
      {/each}
    {/if}
    {#if Array.isArray(indexes) && indexes.length > 0}
      <div class="sub">Indexes</div>
      {#each indexes as ix (ix.name)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="col" oncontextmenu={e => onmenu(e, undefined, ix)} title={ix.definition ?? ix.name}>
          <span class="key">{#if ix.primary}<Icon name="key" size={10} />{/if}</span>
          <span class="name">{ix.name}</span>
          <span class="type">{ix.unique && !ix.primary ? 'unique · ' : ''}{ix.columns.join(', ')}</span>
        </div>
      {/each}
    {/if}
  </div>
{/if}

<style>
  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 8px 0 2px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    text-align: left;
  }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.active { background: var(--accent-dim); color: var(--text); }
  .item :global(.icon) { color: var(--text-3); }
  .item.active > :global(.icon) { color: var(--accent); }
  .item.picked { background: var(--grid-selected); color: var(--text); }
  .twist { flex: none; display: flex; align-items: center; justify-content: center; width: 16px; height: 18px; border-radius: 4px; transition: transform 0.12s; }
  .twist:hover { background: var(--active); }
  .twist.open { transform: rotate(90deg); }
  .label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; }
  .columns { padding: 1px 0 4px; }
  .sub { padding: 6px 8px 2px 24px; color: var(--text-3); font-size: 10.5px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; }
  .col {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 22px;
    padding: 0 8px 0 24px;
    border-radius: 5px;
    color: var(--text-2);
    font-size: 12px;
  }
  .col:hover { background: var(--hover); color: var(--text); }
  .col.error { color: var(--danger); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .key { flex: none; display: flex; width: 12px; color: var(--text-3); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .type { flex: none; max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-3); font-family: var(--font-mono); font-size: 10.5px; text-transform: uppercase; }
</style>
