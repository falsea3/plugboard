<script lang="ts">
  import type { DiagramTable, Relation } from '../api/backend';
  import type { Card, Point } from './layout';
  import Icon from '../ui/Icon.svelte';

  let {
    table,
    card,
    at,
    on,
    dim,
    picked,
    relation,
    ondrag,
    onpick,
    onopen,
  }: {
    table: DiagramTable;
    card: Card;
    at: Point;
    on: boolean;
    dim: boolean;
    picked: boolean;
    relation: Relation | null;
    ondrag: (e: PointerEvent) => void;
    onpick: () => void;
    onopen: () => void;
  } = $props();

  const marked = (column: string) =>
    !!relation &&
    ((relation.table === table.name && relation.columns.includes(column)) || (relation.refTable === table.name && relation.refColumns.includes(column)));
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div
  class="card"
  class:on
  class:dim
  class:picked
  data-table={table.name}
  style:left="{at.x}px"
  style:top="{at.y}px"
  style:width="{card.w}px"
  style:height="{card.h}px"
  title="Double-click to open {table.name}"
  onpointerdown={ondrag}
  onclick={e => {
    e.stopPropagation();
    onpick();
  }}
  ondblclick={onopen}
>
  <div class="head">
    <Icon name={table.kind === 'view' ? 'view' : 'table'} size={12} />
    <span class="name">{table.name}</span>
  </div>
  {#each card.columns as col (col.name)}
    <div class="row" class:marked={marked(col.name)}>
      <span class="key" class:pk={col.primaryKey}>
        {#if col.primaryKey}<Icon name="key" size={10} />{:else if card.foreign.has(col.name)}<Icon name="link" size={10} />{/if}
      </span>
      <span class="col">{col.name}</span>
      <span class="type">{col.type}</span>
    </div>
  {/each}
  {#if card.hidden > 0}<div class="row more">+{card.hidden} more</div>{/if}
</div>

<style>
  .card {
    position: absolute;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface);
    overflow: hidden;
    cursor: default;
    transition: opacity 0.12s;
  }
  .card.on { border-color: color-mix(in srgb, var(--accent) 55%, var(--border)); }
  .card.picked { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
  .card.dim { opacity: 0.35; }
  .head {
    flex: none;
    display: flex;
    align-items: center;
    gap: 7px;
    height: 32px;
    padding: 0 10px;
    border-bottom: 1px solid var(--border);
    background: var(--elevated);
    color: var(--text);
    font-family: var(--font-ui);
    font-size: 12.5px;
    font-weight: 600;
    cursor: grab;
  }
  .head :global(.icon) { color: var(--text-3); }
  .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 22px;
    padding: 0 10px 0 6px;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--text);
  }
  .row.marked { background: var(--accent-dim); }
  .row.more { padding-left: 26px; color: var(--text-3); font-style: italic; }
  .key { flex: none; width: 14px; display: flex; justify-content: center; color: var(--text-3); }
  .key.pk { color: var(--text-2); }
  .col { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .type { flex: none; max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-3); font-size: 11px; }
</style>
