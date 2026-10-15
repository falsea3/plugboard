<script lang="ts">
  import { api, type QueryRun } from '../api/backend';
  import { appendCut, cutCells, cutKey, cutNote, markCut } from '../table/cut';
  import { app } from '../app/app.svelte';
  import { formatCount, formatDuration, toTSV } from '../ui/format';
  import { copyToClipboard } from '../ui/clipboard';
  import { textBase64 } from '../ui/encode';
  import { toCSV } from './csv';
  import DataGrid from '../grid/DataGrid.svelte';
  import ValueEditor, { type ValueEdit } from '../table/ValueEditor.svelte';
  import { looksLikeJSON } from '../json/text';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import LoadBar from '../ui/LoadBar.svelte';

  let {
    sessionId,
    tabId,
    run = $bindable(),
    running,
    elapsed,
    resultIndex = $bindable(0),
  }: { sessionId: string; tabId: string; run: QueryRun | null; running: boolean; elapsed: number; resultIndex?: number } = $props();

  let grid = $state<DataGrid>();
  let viewing = $state<ValueEdit | null>(null);

  function view(r: number, c: number) {
    const rs = current;
    if (!rs || c < 0 || rs.columns[c]?.kind === 'binary') return;
    const value = rs.rows[r]?.[c] ?? null;
    const size = cut.get(cutKey(r, c));
    if (size !== undefined) {
      viewing = { r, c, title: `${rs.columns[c].name} — ${cutNote(size, String(value).length).toLowerCase()}`, value, json: false, readOnly: true };
      return;
    }
    viewing = { r, c, title: rs.columns[c].name, value, json: rs.columns[c].kind === 'json' || looksLikeJSON(value), readOnly: true };
  }
  const current = $derived(run?.results[resultIndex] ?? null);
  const cut = $derived(cutCells(current?.cut));
  const shownRows = $derived(current ? markCut(current.rows, cut) : []);

  let loadingMore = $state(false);

  async function loadMore() {
    const rs = current;
    if (!run || !rs?.hasMore || loadingMore) return;
    const index = resultIndex;
    loadingMore = true;
    try {
      const next = await api.runMore(sessionId, `${tabId}-more-${Date.now()}`, rs.statement, rs.offset + rs.rows.length);
      run.results[index] = {
        ...rs,
        rows: [...rs.rows, ...next.rows],
        cut: appendCut(rs.cut, next.cut, rs.rows.length),
        hasMore: next.hasMore,
        durationMs: rs.durationMs + next.durationMs,
      };
    } catch (err) {
      app.notify(err);
    } finally {
      loadingMore = false;
    }
  }

  function errorText(r: QueryRun): string {
    if (!r.cancelled) return r.error ?? '';
    return r.rolledBack ? 'Query cancelled. The connection was reset, so the open transaction was rolled back.' : 'Query cancelled.';
  }

  function rowsChip(): string {
    if (!current) return '';
    if (!current.hasRows) return `${formatCount(current.rowsAffected, 'row')} affected`;
    const n = current.rows.length.toLocaleString('en-US');
    return current.hasMore ? `${n}+ rows` : current.truncated ? `first ${n} rows` : formatCount(current.rows.length, 'row');
  }

  const total = $derived(run ? run.results.reduce((a, r) => a + r.durationMs, 0) : 0);

  function copyAll() {
    if (current?.hasRows) void copyToClipboard(toTSV(current.columns.map(c => c.name), current.rows));
  }

  async function exportCsv() {
    if (!current?.hasRows) return;
    try {
      const path = await api.saveExport('result.csv', textBase64(toCSV(current.columns.map(c => c.name), current.rows)));
      if (path) app.notify(`Saved ${path}`, 'info');
    } catch (err) {
      app.notify(err);
    }
  }
</script>

<section class="card" aria-label="Result">
  <div class="head">
    {#if run && run.results.length > 1}
      <div class="result-tabs" role="tablist" aria-label="Results">
        {#each run.results as r, i (i)}
          <button role="tab" aria-selected={i === resultIndex} class:on={i === resultIndex} onclick={() => (resultIndex = i)}>{r.hasRows ? `Result ${i + 1}` : `Statement ${i + 1}`}</button>
        {/each}
      </div>
    {:else}
      <span class="title">Result</span>
    {/if}
    {#if run && current && !running}
      <span class="chip">{rowsChip()}</span>
      <span class="chip">{formatDuration(total)}</span>
    {/if}
    {#if current?.hasMore && !running}
      <button class="btn sm ghost" onclick={loadMore} disabled={loadingMore} title="Runs the query again from row {(current.offset + current.rows.length + 1).toLocaleString('en-US')} and appends the next rows">{#if loadingMore}<Spinner size={11} />Loading…{:else}Load 1,000 more{/if}</button>
    {/if}
    <span class="space"></span>
    {#if current?.hasRows && !running}
      <button class="btn sm outline" onclick={copyAll} title="Copy all rows as tab-separated text">Copy</button>
      <button class="btn sm outline" onclick={exportCsv} title="Save the rows as CSV">Export</button>
    {/if}
  </div>

  <div class="results">
    {#if loadingMore}<LoadBar label="Loading more rows" />{/if}
    {#if running}
      <div class="placeholder faint" role="status"><Spinner />Running…{#if elapsed >= 1000}<span class="elapsed">{(elapsed / 1000).toFixed(1)} s</span>{/if}</div>
    {:else if !run}
      <div class="placeholder faint">Results appear here.</div>
    {:else}
      {#if run.error}
        <div class="error" role="alert">
          <Icon name="alert" />
          <div>
            <div class="message">{errorText(run)}</div>
            {#if run.results.length > 0}
              <div class="faint small-text">{formatCount(run.results.length, 'statement')} ran before the error.</div>
            {/if}
          </div>
        </div>
      {/if}
      {#if current}
        <div class="grid-wrap">
          {#if current.hasRows}
            <DataGrid
              bind:this={grid}
              cellMenu={(r, c) => (c >= 0 ? [{ id: 'value', label: 'View in editor', kbd: '⇧↵' }] : [])}
              onmenu={(id, at) => (id === 'value' ? (view(at.r, at.c), true) : false)}
              columns={current.columns}
              rows={shownRows}
            />
          {:else if !run.error}
            <div class="placeholder"><Icon name="check" /> {formatCount(current.rowsAffected, 'row')} affected</div>
          {/if}
        </div>
      {/if}
    {/if}
  </div>

</section>

{#if viewing}
  <ValueEditor edit={viewing} onsave={() => (viewing = null)} onclose={() => (viewing = null)} />
{/if}

<style>
  .card { flex: 1; min-height: 0; display: flex; flex-direction: column; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); overflow: hidden; }
  .head { flex: none; display: flex; align-items: center; gap: 8px; height: 42px; padding: 0 12px; border-bottom: 1px solid var(--grid-line); }
  .title { font-weight: 500; }
  .chip { padding: 1px 7px; border-radius: 6px; background: var(--border-subtle); color: var(--text-3); font-size: 12px; white-space: nowrap; }
  .space { flex: 1; }
  .head .btn.sm { height: 28px; border-radius: 9px; }
  .result-tabs { display: flex; gap: 2px; min-width: 0; overflow-x: auto; }
  .result-tabs button { height: 26px; padding: 0 10px; border: 0; border-radius: 7px; background: transparent; color: var(--text-3); font-size: 12.5px; white-space: nowrap; }
  .result-tabs button.on { background: var(--active); color: var(--text); font-weight: 500; }
  .small-text { font-size: 11.5px; margin-top: 3px; }
  .results { position: relative; flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .grid-wrap { flex: 1; min-height: 0; }
  .placeholder { display: flex; align-items: center; justify-content: center; gap: 8px; height: 100%; color: var(--text-2); }
  .placeholder.faint { color: var(--text-3); }
  .elapsed { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
  .error { flex: none; display: flex; gap: 8px; margin: 10px; padding: 10px 12px; border-radius: var(--radius); color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); font-family: var(--font-mono); font-size: 12px; user-select: text; -webkit-user-select: text; }
  .error .message { white-space: pre-wrap; }
  .error :global(.icon) { flex: none; margin-top: 1px; }
</style>
