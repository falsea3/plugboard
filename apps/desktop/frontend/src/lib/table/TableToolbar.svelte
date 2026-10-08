<script lang="ts">
  import Icon from '../ui/Icon.svelte';

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
  } = $props();
</script>

<div class="toolbar">
  <div class="segmented" role="tablist">
    <button role="tab" aria-selected={mode === 'data'} class:on={mode === 'data'} onclick={() => (mode = 'data')}><Icon name="rows" size={13} />Data</button>
    <button role="tab" aria-selected={mode === 'structure'} class:on={mode === 'structure'} onclick={() => (mode = 'structure')}><Icon name="columns" size={13} />Structure</button>
    <button role="tab" aria-selected={mode === 'ddl'} class:on={mode === 'ddl'} onclick={() => (mode = 'ddl')}><Icon name="ddl" size={13} />DDL</button>
  </div>
  <span class="title mono faint" title={`${schema}.${table}`}>{schema}.<span class="muted">{table}</span></span>
  {#if mode === 'ddl'}
  <button class="btn sm ghost" onclick={oncopy}><Icon name="copy" size={13} />Copy</button>
  <button class="btn sm ghost" onclick={onquery}><Icon name="code" size={13} />Open in query</button>
  {:else}
  <button class="btn sm ghost filter-btn" class:on={filtersOpen || filterCount > 0} onclick={onfilter} title="Filter rows (⌘F)">
    <Icon name="funnel" size={13} />Filter{#if filterCount > 0}<span class="count">{filterCount}</span>{/if}
  </button>
  {#if lockReason}
    <span class="ro-reason" title={mode === 'structure' ? "The structure can't be changed here" : 'Editing is off for this table'}><Icon name="lock" size={11} />{lockReason}</span>
  {:else if mode === 'structure'}
    <button class="btn sm ghost" onclick={onadd} title="Add column (⌘I)"><Icon name="plus" size={13} />Column</button>
  {:else}
    <button class="btn sm ghost" onclick={onadd} disabled={!canAdd} title="Add row (⌘I)"><Icon name="plus" size={13} />Row</button>
  {/if}
  {/if}
  <button class="btn icon sm ghost" title="Refresh (⌘R)" onclick={onrefresh}><Icon name="refresh" size={13} /></button>
</div>

<style>
  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    min-width: 0;
    padding: 0 8px;
    overflow: hidden;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
  }
  .title { flex: 1 1 auto; min-width: 0; margin-left: 2px; overflow: hidden; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
  .filter-btn.on { color: var(--accent); background: var(--accent-dim); }
  .count {
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--on-accent);
    font-size: 10.5px;
    line-height: 16px;
    text-align: center;
  }
  .ro-reason {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    flex: 0 1 auto;
    min-width: 12px;
    max-width: 150px;
    overflow: hidden;
    font-size: 11.5px;
    color: var(--text-3);
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 7px;
    background: var(--elevated);
    border: 1px solid var(--border-subtle);
  }
  .segmented button {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 22px;
    padding: 0 8px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    font-size: 12px;
    font-weight: 500;
  }
  .segmented button.on { background: var(--bg); color: var(--text); box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2); }

  @container (max-width: 500px) {
    .toolbar { gap: 4px; padding-inline: 6px; }
    .segmented button { gap: 4px; padding-inline: 5px; }
    .ro-reason { max-width: 18px; font-size: 0; }
  }
</style>
