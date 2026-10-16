<script lang="ts">
  import { keys } from '../ui/keys';
  import { formatCount, formatDuration } from '../ui/format';
  import { MAX_ROWS, type TableRows } from './rows.svelte';
  import Icon from '../ui/Icon.svelte';
  import { journal } from '../app/journal.svelte';
  import Spinner from '../ui/Spinner.svelte';

  type Pending = { count: number; error: string; saving: boolean };

  let {
    rows,
    filtered,
    cell,
    pending,
    valueShown,
    ontogglevalue,
    ondiscard,
    onpreview,
  }: {
    rows: TableRows;
    filtered: boolean;
    cell: string;
    pending: Pending | null;
    valueShown: boolean;
    ontogglevalue: () => void;
    ondiscard: () => void;
    onpreview: () => void;
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

<footer class="status">
  {#if rangeLabel}<span class="chip">{rangeLabel}</span>{/if}
  {#if rows.loadingMore}
    <span class="chip"><Spinner size={10} />Loading more…</span>
  {:else if page?.hasMore && rows.atLimit}
    <span class="chip" title="Sort or filter to reach other rows">first {fmt(MAX_ROWS)} rows only</span>
  {/if}
  {#if page && !count?.exact && page.hasMore}
    <button class="btn sm outline act" onclick={() => rows.countExactly()} disabled={rows.counting} title="Run COUNT(*) for the exact number — can take a while on big tables">{#if rows.counting}<Spinner size={10} />Counting…{:else}<Icon name="sequence" size={13} />Count rows{/if}</button>
  {/if}
  {#if filtered}<span class="chip accent">filtered</span>{/if}
  {#if page}<span class="chip" title="How long the last fetch took">{formatDuration(page.result.durationMs)}</span>{/if}
  {#if pending}
    {#if pending.error}
      <button class="chip danger" title="Show the whole error in the journal (⇧⌘J)" onclick={() => journal.show()}><Icon name="alert" size={11} />{pending.error}</button>
    {:else}
      <span class="chip warn">{#if pending.saving}<Spinner size={10} />Saving…{:else}{formatCount(pending.count, 'change')} · {keys('⌘S')} to save{/if}</span>
    {/if}
    <button class="btn sm outline act" onclick={onpreview} disabled={pending.saving}><Icon name="code" size={13} />Preview SQL</button>
    <button class="btn sm outline act" onclick={ondiscard} disabled={pending.saving}><Icon name="restore" size={13} />Discard</button>
  {/if}
  <span class="space"></span>
  <button class="cell" class:on={valueShown} aria-pressed={valueShown} onclick={ontogglevalue} title={valueShown ? 'Hide the Value panel' : 'Show the Value panel'}>{cell || 'Value'}</button>
</footer>

<style>
  .status { flex: none; display: flex; align-items: center; gap: 8px; height: 30px; min-width: 0; margin-top: 8px; font-size: 12px; color: var(--text-3); }
  .chip { flex: none; display: inline-flex; align-items: center; gap: 5px; max-width: 360px; overflow: hidden; padding: 2px 8px; border: 0; border-radius: 6px; background: var(--surface); color: var(--text-3); font-size: 12px; white-space: nowrap; text-overflow: ellipsis; }
  .act { flex: none; height: 26px; gap: 6px; padding: 0 10px; border-radius: 8px; font-size: 12px; }
  .accent { background: var(--accent-bg); color: var(--accent-text); }
  .warn { background: var(--cell-edited); color: var(--warn); }
  .danger { flex-shrink: 1; min-width: 0; cursor: pointer; background: color-mix(in srgb, var(--danger) 14%, transparent); color: var(--danger); user-select: text; -webkit-user-select: text; }
  .space { flex: 1; }
  .cell { flex: 0 1 auto; min-width: 0; overflow: hidden; padding: 2px 6px; border: 0; border-radius: 6px; background: transparent; color: var(--text-3); font-size: 12px; white-space: nowrap; text-overflow: ellipsis; }
  .cell:hover, .cell.on { background: var(--surface); color: var(--text); }
</style>
