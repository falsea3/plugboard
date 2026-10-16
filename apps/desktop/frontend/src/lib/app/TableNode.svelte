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

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowRight' && !open) toggle(true);
    else if (e.key === 'ArrowLeft' && open) toggle(false);
    else return;
    e.preventDefault();
  }
</script>

<button class="item" class:active class:picked {onclick} {onkeydown} oncontextmenu={e => onmenu(e)} title={t.name} aria-expanded={open}>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <span
    class="twist"
    class:open
    onclick={e => {
      e.stopPropagation();
      toggle();
    }}
  ><Icon name="chevron-right" size={12} /></span>
  <Icon name={t.kind === 'view' ? 'view' : 'table'} size={14} />
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
          <span class="key">{#if c.primaryKey}<Icon name="key" size={12} />{/if}</span>
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
          <span class="key">{#if ix.primary}<Icon name="key" size={12} />{/if}</span>
          <span class="name">{ix.name}</span>
          <span class="type">{ix.unique && !ix.primary ? 'unique · ' : ''}{ix.columns.join(', ')}</span>
        </div>
      {/each}
    {/if}
  </div>
{/if}

<style>
  .item { width: 100%; display: flex; align-items: center; gap: 6px; height: 30px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text); font-size: 13px; text-align: left; }
  .item:hover { background: var(--hover); }
  .item.active { background: var(--tree-selected); color: var(--accent-text); font-weight: 500; }
  .item :global(.icon) { color: var(--text-3); }
  .item .twist :global(.icon) { color: var(--text-4); }
  .item.active > :global(.icon) { color: var(--accent-text); }
  .item.picked { background: var(--grid-selected); }
  .twist { flex: none; display: flex; align-items: center; justify-content: center; width: 12px; height: 18px; border-radius: 4px; transition: transform 0.12s; }
  .twist:hover :global(.icon) { color: var(--text); }
  .twist.open { transform: rotate(90deg); }
  .label { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sub { padding: 6px 8px 2px 40px; color: var(--text-4); font-size: 10.5px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; }
  .col { display: flex; align-items: center; gap: 6px; height: 26px; padding: 0 8px 0 40px; border-radius: 8px; color: var(--text-soft); font-size: 12.5px; }
  .col:hover { background: var(--hover); color: var(--text); }
  .col.error { color: var(--danger); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .key { flex: none; display: flex; width: 14px; margin-left: -20px; margin-right: 6px; color: var(--text-3); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .type { flex: none; max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-4); font-size: 10.5px; letter-spacing: 0.03em; text-transform: uppercase; }
</style>
