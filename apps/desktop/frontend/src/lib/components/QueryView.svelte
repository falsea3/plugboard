<script lang="ts">
  import { untrack } from 'svelte';
  import type { EditorView } from '@codemirror/view';
  import { api, type CellValue, type QueryRun, type ResultColumn } from '../backend';
  import { app, type QueryTab, type Workspace } from '../stores/app.svelte';
  import { statementAt } from '../sqlStatements';
  import { engine } from '../engines';
  import { formatCount, formatDuration } from '../format';
  import SqlEditor, { showProblem, showRan } from './SqlEditor.svelte';
  import { serverProblems } from '../sqlProblems';
  import DataGrid from './DataGrid.svelte';
  import ValueBar from './ValueBar.svelte';
  import Icon from './Icon.svelte';
  import Spinner from './Spinner.svelte';
  import Modal from './Modal.svelte';
  import { startDrag } from '../drag';

  let { ws, tab }: { ws: Workspace; tab: QueryTab } = $props();

  const session = untrack(() => ws.session);
  const dialect = engine(session.connection.driver).sqlDialect;
  const syntax = session.engine.syntax;
  const isProd = session.connection.env === 'prod';

  let pending = $state<{ script: string; base: number; writes: string[] } | null>(null);

  let editor = $state<EditorView>();
  let hasSelection = $state(false);
  let run = $state<QueryRun | null>(null);
  let running = $state(false);
  let elapsed = $state(0);
  let queryId = '';
  let resultIndex = $state(0);
  let editorHeight = $state(260);
  let selected = $state<{ value: CellValue; column: ResultColumn } | null>(null);

  const tableNames = $derived(ws.tables.map(t => t.name));

  $effect(() => {
    if (!running) return;
    const started = performance.now();
    elapsed = 0;
    const timer = setInterval(() => (elapsed = performance.now() - started), 100);
    return () => clearInterval(timer);
  });
  const current = $derived(run?.results[resultIndex] ?? null);

  async function execute(all: boolean) {
    if (running || !editor) return;
    const state = editor.state;
    const sel = state.selection.main;
    let script: string;
    let base: number;
    if (!sel.empty) {
      script = state.sliceDoc(sel.from, sel.to);
      base = sel.from;
    } else if (all) {
      script = state.doc.toString();
      base = 0;
    } else {
      const stmt = statementAt(state.doc.toString(), sel.head, syntax);
      if (!stmt) return;
      script = stmt.text;
      base = stmt.from;
    }
    if (!script.trim()) return;
    showRan(editor, base, base + script.length);

    if (isProd && app.settings.confirmProdWrites && !ws.readOnly) {
      let writes: string[];
      try {
        writes = await api.writeStatements(session.sessionId, script);
      } catch (err) {
        app.notify(err);
        return;
      }
      if (writes.length > 0) {
        pending = { script, base, writes };
        return;
      }
    }
    await runScript(script, base);
  }

  async function runScript(script: string, base: number) {
    const docAtRun = editor?.state.doc;
    running = true;
    queryId = `${tab.id}-${Date.now()}`;
    try {
      const r = await api.runQuery(session.sessionId, queryId, script);
      run = r;
      if (r.error && r.errorIndex >= 0 && r.errorPosition >= 0 && editor && editor.state.doc === docAtRun) {
        const [p] = serverProblems(script, base, syntax, [{ index: r.errorIndex, position: r.errorPosition, message: r.error }]);
        if (p) showProblem(editor, p);
      }
      const withRows = r.results.map((x, i) => (x.hasRows ? i : -1)).filter(i => i >= 0);
      resultIndex = withRows.length ? withRows[withRows.length - 1] : Math.max(0, r.results.length - 1);
      if (/\b(create|drop|alter|rename)\s/i.test(script)) ws.loadTables();
    } catch (err) {
      run = { results: [], error: err instanceof Error ? err.message : String(err), errorIndex: -1, errorPosition: -1, cancelled: false, rolledBack: false };
    } finally {
      running = false;
    }
  }

  async function checkSyntax(doc: string) {
    return serverProblems(doc, 0, syntax, await api.checkSyntax(session.sessionId, doc));
  }

  function cancel() {
    if (queryId) api.cancelQuery(queryId);
  }

  function startResize(e: PointerEvent) {
    const h0 = editorHeight;
    startDrag(e, 'row-resize', (_, dy) => (editorHeight = Math.max(80, Math.min(window.innerHeight - 220, h0 + dy))));
  }

  let loadingMore = $state(false);

  async function loadMore() {
    const rs = current;
    if (!run || !rs?.hasMore || loadingMore) return;
    const index = resultIndex;
    loadingMore = true;
    queryId = `${tab.id}-more-${Date.now()}`;
    try {
      const next = await api.runMore(session.sessionId, queryId, rs.statement, rs.offset + rs.rows.length);
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

<div class="view">
  <div class="toolbar">
    {#if running}
      <button class="btn sm" onclick={cancel}><Icon name="stop" size={11} />Stop</button>
    {:else}
      <button class="btn sm primary" onclick={() => execute(false)} title={hasSelection ? 'Run the selected text (⌘↵)' : 'Run the statement under the cursor (⌘↵)'}><Icon name="play" size={11} />{hasSelection ? 'Run selection' : 'Run'}</button>
      {#if !hasSelection}<button class="btn sm" onclick={() => execute(true)} title="Run all (⇧⌘↵)">Run all</button>{/if}
    {/if}
    <span class="faint small"><span class="kbd">⌘↵</span> statement · <span class="kbd">⇧⌘↵</span> all</span>
    {#if ws.readOnly}
      <span style="flex:1"></span>
      <span class="ro-chip" title="Writes are refused in this session. Switch it in the sidebar."><Icon name="lock" size={11} />Read-only</span>
    {/if}
  </div>

  <div class="editor" style:height="{editorHeight}px">
    <SqlEditor bind:value={tab.sql} bind:editor bind:hasSelection {dialect} {syntax} check={checkSyntax} tables={tableNames} defaultSchema={ws.schema} onrun={execute} />
  </div>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="splitter" onpointerdown={startResize}></div>

  <div class="results">
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
      <ValueBar value={selected.value} column={selected.column} />
    {/if}
  </div>
</div>

{#if pending}
  {@const p = pending}
  <Modal title="Run on Production?" width={500} onclose={() => (pending = null)}>
    <p class="confirm-text">
      <strong>{session.connection.name}</strong> is tagged Production. {p.writes.length === 1 ? 'This statement can change' : `These ${p.writes.length} statements can change`} data, schema or settings:
    </p>
    <div class="confirm-list">
      {#each p.writes.slice(0, 4) as w, i (i)}
        <pre>{w.length > 400 ? w.slice(0, 400) + '…' : w}</pre>
      {/each}
      {#if p.writes.length > 4}<div class="faint">…and {p.writes.length - 4} more</div>{/if}
    </div>
    {#snippet footer()}
      <span class="faint confirm-hint">Turn this off in Settings ▸ General.</span>
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (pending = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={() => { const { script, base } = p; pending = null; runScript(script, base); }}>Run on Production</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .confirm-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .confirm-text strong { color: var(--text); }
  .confirm-list { display: flex; flex-direction: column; gap: 6px; max-height: 240px; overflow: auto; }
  .confirm-list pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--env-prod);
    font-family: var(--font-mono);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    -webkit-user-select: text;
  }
  .confirm-hint { font-size: 11.5px; }
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .toolbar, .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 10px;
    background: var(--surface);
  }
  .toolbar { border-bottom: 1px solid var(--border); }
  .footer { height: 37px; border-top: 1px solid var(--border); }
  .small { font-size: 12px; margin-left: 6px; white-space: nowrap; }
  .ro-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 20px;
    padding: 0 8px;
    border-radius: 5px;
    color: var(--text-2);
    background: var(--elevated);
    font-size: 11.5px;
    font-weight: 550;
  }
  .small-text { font-size: 11.5px; margin-top: 3px; }
  .editor { flex: none; min-height: 80px; }
  .splitter {
    flex: none;
    height: 5px;
    margin-top: -2px;
    margin-bottom: -3px;
    position: relative;
    z-index: 3;
    cursor: row-resize;
    border-top: 1px solid var(--border);
  }
  .splitter:hover { border-top: 2px solid var(--accent); }
  .results { flex: 1; min-height: 0; display: flex; flex-direction: column; border-top: 1px solid var(--border); }
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
