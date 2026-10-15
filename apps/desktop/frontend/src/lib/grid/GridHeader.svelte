<script lang="ts">
  import type { ResultColumn } from '../api/wire';
  import Icon from '../ui/Icon.svelte';

  let {
    columns,
    widths,
    shown,
    leftPad,
    numberWidth,
    height,
    sort,
    keys,
    numeric,
    onsort,
    onmenu,
    onresize,
    onfit,
  }: {
    columns: ResultColumn[];
    widths: number[];
    shown: number[];
    leftPad: number;
    numberWidth: number;
    height: number;
    sort: { column: string; desc: boolean } | null;
    keys: Set<string>;
    numeric: boolean[];
    onsort?: (column: string) => void;
    onmenu: (e: MouseEvent, c: number) => void;
    onresize: (e: PointerEvent, c: number) => void;
    onfit: (c: number) => void;
  } = $props();
</script>

<div class="header" role="row" style:height="{height}px">
  <div class="corner" style:width="{numberWidth}px"></div>
  <div class="pad" style:width="{leftPad}px"></div>
  {#each shown as i (i)}
    {@const col = columns[i]}
    {@const sorted = sort?.column === col.name}
    <div class="th" class:sortable={!!onsort} class:num={numeric[i]} role="columnheader" style:width="{widths[i]}px" title="{col.name} · {col.type || 'unknown'}">
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="th-label" onclick={() => onsort?.(col.name)} oncontextmenu={e => onmenu(e, i)}>
        {#if keys.has(col.name)}<span class="pk"><Icon name="key" size={11} /></span>{/if}
        <span class="th-name">{col.name}</span>
        {#if col.type}<span class="th-type">{col.type}</span>{/if}
        {#if sorted}<span class="sorted"><Icon name={sort?.desc ? 'arrowDown' : 'arrowUp'} size={11} /></span>{/if}
      </div>
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="resizer" onpointerdown={e => onresize(e, i)} ondblclick={() => onfit(i)}></div>
    </div>
  {/each}
</div>

<style>
  .header {
    position: sticky;
    top: 0;
    z-index: 2;
    display: flex;
    background: var(--card);
    border-bottom: 1px solid var(--border-subtle);
    font-family: var(--font-ui);
    font-size: 13px;
  }
  .corner {
    position: sticky;
    left: 0;
    z-index: 3;
    flex: none;
    background: var(--card);
  }
  .pad { flex: none; }
  .th {
    position: relative;
    flex: none;
    display: flex;
    align-items: center;
    color: var(--text);
    font-weight: 500;
  }
  .th-label {
    flex: 1;
    min-width: 0;
    height: 100%;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px;
    overflow: hidden;
  }
  .th.num .th-label { justify-content: flex-end; }
  .th.sortable .th-label:hover { background: var(--surface); }
  .th:hover .resizer { background: linear-gradient(to right, transparent 3px, var(--border) 3px, var(--border) 5px, transparent 5px); }
  .th-name, .th-type { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .th-type { flex-shrink: 100; color: var(--text-4); font-size: 10.5px; font-weight: 400; letter-spacing: 0.03em; text-transform: uppercase; }
  .pk { color: var(--text-3); display: flex; }
  .sorted { display: flex; flex: none; color: var(--accent-text); }
  .resizer {
    position: absolute;
    top: 0;
    right: -4px;
    bottom: 0;
    width: 8px;
    z-index: 1;
    cursor: col-resize;
  }
  .resizer:hover { background: linear-gradient(to right, transparent 3px, var(--accent) 3px, var(--accent) 5px, transparent 5px); }
</style>
