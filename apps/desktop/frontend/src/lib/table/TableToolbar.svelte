<script lang="ts">
  import { keys } from '../ui/keys';
  import Icon from '../ui/Icon.svelte';
  import SegmentedControl from '../ui/SegmentedControl.svelte';

  let {
    mode = $bindable(),
    schema,
    table,
    filterCount,
    filtersOpen,
    lockReason,
    canAdd,
    onfilter,
    onadd,
    onrefresh,
    oncopy,
    onquery,
    rowsLabel = '',
    dirty = false,
    onsave,
  }: {
    mode: 'data' | 'structure' | 'ddl';
    schema: string;
    table: string;
    filterCount: number;
    filtersOpen: boolean;
    lockReason: string;
    canAdd: boolean;
    onfilter: () => void;
    onadd: () => void;
    onrefresh: () => void;
    oncopy: () => void;
    onquery: () => void;
    rowsLabel?: string;
    dirty?: boolean;
    onsave?: () => void;
  } = $props();
</script>

<div class="toolbar">
  <SegmentedControl label="Table view" bind:value={mode} options={[{ value: 'data', label: 'Data' }, { value: 'structure', label: 'Structure' }, { value: 'ddl', label: 'DDL' }]} />
  <span class="title" title={`${schema}.${table}`}>{schema}.<span class="name">{table}</span>{#if rowsLabel}{' · '}{rowsLabel}{/if}</span>
  {#if mode === 'ddl'}
  <button class="btn outline" onclick={oncopy}><Icon name="copy" size={14} />Copy</button>
  <button class="btn outline" onclick={onquery}><Icon name="code" size={14} />Open in query</button>
  {:else}
  <button class="btn outline filter-btn" class:on={filtersOpen || filterCount > 0} onclick={onfilter} title={keys('Filter rows (⌘F)')}>
    <Icon name="funnel" size={14} />Filter{#if filterCount > 0}<span class="count">{filterCount}</span>{/if}
  </button>
  {#if lockReason}
    <span class="ro-reason" title={mode === 'structure' ? "The structure can't be changed here" : 'Editing is off for this table'}><Icon name="lock" size={11} />{lockReason}</span>
  {:else if mode === 'structure'}
    <button class="btn outline" onclick={onadd} title={keys('Add column (⌘I)')}><Icon name="plus" size={14} />Column</button>
  {:else}
    <button class="btn outline" onclick={onadd} disabled={!canAdd} title={keys('Add row (⌘I)')}><Icon name="plus" size={14} />Row</button>
  {/if}
  {/if}
  <button class="btn outline icon" title={keys('Refresh (⌘R)')} aria-label="Refresh" onclick={onrefresh}><Icon name="refresh" size={14} /></button>
  {#if onsave && mode !== 'ddl'}
    <button class="btn primary" onclick={onsave} disabled={!dirty} title={keys('Save changes (⌘S)')}>Save<span class="kbd">{keys('⌘S')}</span></button>
  {/if}
</div>

<style>
  .toolbar { flex: none; display: flex; align-items: center; gap: 10px; height: 44px; margin-top: 6px; min-width: 0; overflow: hidden; }
  .title { flex: 1 1 auto; min-width: 0; overflow: hidden; color: var(--text-4); text-overflow: ellipsis; white-space: nowrap; }
  .name { color: var(--text); }
  .primary .kbd { background: transparent; color: inherit; opacity: 0.8; padding: 0; }
  .filter-btn.on { color: var(--accent-text); background: var(--accent-bg); border-color: transparent; }
  .count { min-width: 18px; height: 18px; padding: 0 5px; border-radius: 9px; background: var(--accent); color: var(--on-accent); font-size: 11px; line-height: 18px; text-align: center; }
  .ro-reason { display: inline-flex; align-items: center; gap: 5px; flex: 0 1 auto; min-width: 12px; max-width: 200px; overflow: hidden; font-size: 12px; color: var(--text-3); white-space: nowrap; text-overflow: ellipsis; }

  @container (max-width: 560px) {
    .toolbar { gap: 4px; }
    .ro-reason { max-width: 18px; font-size: 0; }
  }
</style>
