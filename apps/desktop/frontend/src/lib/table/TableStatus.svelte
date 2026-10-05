<script lang="ts">
  import type { CellValue, ResultColumn } from '../api/backend';
  import { formatCount, formatDuration } from '../ui/format';
  import { MAX_ROWS, type TableRows } from './rows.svelte';
  import ValueBar from '../grid/ValueBar.svelte';
  import Spinner from '../ui/Spinner.svelte';

  let {
    rows,
    filtered,
    selected,
    onopen,
  }: {
    rows: TableRows;
    filtered: boolean;
    selected: { value: CellValue; column: ResultColumn } | null;
    onopen: () => void;
  } = $props();

  const fmt = (n: number) => n.toLocaleString('en-US');
  const page = $derived(rows.page);
  const count = $derived(rows.count);
  const rangeLabel = $derived.by(() => {
    if (!page) return '';
    const n = page.result.rows.length;
    if (n === 0) return 'No rows';
    if (!page.hasMore) return formatCount(n, 'row');
    if (count?.known) return `${fmt(n)} of ${count.exact ? '' : '~'}${fmt(count.count)} rows loaded`;
    return `${fmt(n)} rows loaded`;
  });
</script>

<div class="footer">
  <span class="small muted">{rangeLabel}</span>
  {#if rows.loadingMore}
    <span class="small faint loading-more"><Spinner size={10} />Loading more…</span>
  {:else if page?.hasMore && rows.atLimit}
    <span class="small faint" title="Sort or filter to reach other rows">first {fmt(MAX_ROWS)} rows only</span>
  {/if}
  {#if page && !count?.exact && page.hasMore}
    <button class="link-btn" onclick={() => rows.countExactly()} disabled={rows.counting} title="Run COUNT(*) — can take a while on big tables">{#if rows.counting}<Spinner size={10} />Counting…{:else}Count{/if}</button>
  {/if}
  {#if filtered}<span class="filtered">filtered</span>{/if}
  {#if page}<span class="small faint" title="How long the last fetch took">· {formatDuration(page.result.durationMs)}</span>{/if}
  <span style="flex:1"></span>
  {#if selected}
    <ValueBar value={selected.value} column={selected.column} {onopen} />
  {/if}
</div>

<style>
  .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    height: 37px;
    min-width: 0;
    padding: 0 10px;
    border-top: 1px solid var(--border);
    background: var(--surface);
  }
  .small { font-size: 12px; padding: 0 4px; white-space: nowrap; }
  .loading-more { display: inline-flex; align-items: center; gap: 6px; }
  .link-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    border: 0;
    padding: 0 4px;
    background: transparent;
    color: var(--accent);
    font-size: 12px;
  }
  .link-btn:hover:not(:disabled) { text-decoration: underline; }
  .link-btn:disabled { color: var(--text-3); }
  .filtered {
    padding: 0 6px;
    border-radius: 4px;
    background: var(--accent-dim);
    color: var(--accent);
    font-size: 11px;
    line-height: 17px;
  }
</style>
