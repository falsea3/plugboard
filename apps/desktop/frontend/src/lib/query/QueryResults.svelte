<script lang="ts">
  import { api, type CellValue, type QueryRun, type ResultColumn } from '../api/backend';
  import { app } from '../app/app.svelte';
  import { formatCount, formatDuration } from '../ui/format';
  import DataGrid from '../grid/DataGrid.svelte';
  import ValueBar from '../grid/ValueBar.svelte';
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

  let selected = $state<{ value: CellValue; column: ResultColumn } | null>(null);
  let grid = $state<DataGrid>();
  let viewing = $state<ValueEdit | null>(null);

  function view(r: number, c: number) {
    const rs = current;
    if (!rs || c < 0 || rs.columns[c]?.kind === 'binary') return;
    const value = rs.rows[r]?.[c] ?? null;
    viewing = { r, c, title: rs.columns[c].name, value, json: rs.columns[c].kind === 'json' || looksLikeJSON(value), readOnly: true };
  }
  const current = $derived(run?.results[resultIndex] ?? null);

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

  function summary(): string {
    if (!run) return '';
    const total = run.results.reduce((a, r) => a + r.durationMs, 0);
    if (current?.hasRows) {
      const n = current.rows.length.toLocaleString('en-US');
      const rows = current.hasMore ? `${n}+ rows` : current.truncated ? `first ${n} rows` : formatCount(current.rows.length, 'row');
      return `${rows} · ${formatDuration(total)}`;
    }
    if (current) return `${formatCount(current.rowsAffected, 'row')} affected · ${formatDuration(total)}`;
    return '';
  }
</script>

  <div class="results">
    {#if loadingMore}<LoadBar label="Loading more rows" />{/if}
    {#if running}
      <div class="placeholder faint" role="status"><Spinner />Running…{#if elapsed >= 1000}<span class="elapsed">{(elapsed / 1000).toFixed(1)} s</span>{/if}</div>
    {:else if !run}
      <div class="placeholder faint">Results appear here.</div>
    {:else}
      {#if run.results.length > 1}
        <div class="result-tabs">
          {#each run.results as r, i (i)}
            <button class:on={i === resultIndex} onclick={() => (resultIndex = i)}>
              {r.hasRows ? `Result ${i + 1}` : `Statement ${i + 1}`}
            </button>
          {/each}
        </div>
      {/if}
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
              rows={current.rows}
              onselect={(value, column) => (selected = column && value !== undefined ? { value, column } : null)}
            />
          {:else if !run.error}
            <div class="placeholder"><Icon name="check" /> {formatCount(current.rowsAffected, 'row')} affected</div>
          {/if}
        </div>
      {/if}
    {/if}
  </div>

  <div class="footer">
    <span class="small muted">{summary()}</span>
    {#if current?.hasMore}
      <button
        class="btn sm"
        onclick={loadMore}
        disabled={loadingMore || running}
        title="Runs the query again from row {(current.offset + current.rows.length + 1).toLocaleString('en-US')} and appends the next rows"
      >{#if loadingMore}<Spinner size={11} />Loading…{:else}Load 1,000 more{/if}</button>
    {:else if current?.truncated}
      <span class="small faint" title="Only the first rows of this statement were kept">— rows beyond this weren’t loaded</span>
    {/if}
    <span style="flex:1"></span>
    {#if selected && current?.hasRows}
      <ValueBar value={selected.value} column={selected.column} onopen={() => { const at = grid?.cursor(); if (at) view(at.row, at.col); }} />
    {/if}
  </div>

{#if viewing}
  <ValueEditor edit={viewing} onsave={() => (viewing = null)} onclose={() => (viewing = null)} />
{/if}

<style>
  .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 37px;
    padding: 0 10px;
    border-top: 1px solid var(--border);
    background: var(--surface);
  }
  .small { font-size: 12px; margin-left: 6px; white-space: nowrap; }
  .small-text { font-size: 11.5px; margin-top: 3px; }
  .results { position: relative; flex: 1; min-height: 0; display: flex; flex-direction: column; border-top: 1px solid var(--border); }
  .grid-wrap { flex: 1; min-height: 0; }
  .placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
    color: var(--text-2);
  }
  .placeholder.faint { color: var(--text-3); }
  .elapsed { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
  .result-tabs {
    flex: none;
    display: flex;
    gap: 2px;
    padding: 4px 8px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface);
    overflow-x: auto;
  }
  .result-tabs button {
    height: 22px;
    padding: 0 10px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    font-size: 12px;
    white-space: nowrap;
  }
  .result-tabs button.on { background: var(--accent-dim); color: var(--text); }
  .error {
    flex: none;
    display: flex;
    gap: 8px;
    margin: 10px;
    padding: 10px 12px;
    border-radius: var(--radius);
    color: var(--danger);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
    font-family: var(--font-mono);
    font-size: 12px;
    user-select: text;
    -webkit-user-select: text;
  }
  .error .message { white-space: pre-wrap; }
  .error :global(.icon) { flex: none; margin-top: 1px; }
</style>
