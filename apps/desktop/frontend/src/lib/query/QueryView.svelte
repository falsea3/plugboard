<script lang="ts">
  import { untrack } from 'svelte';
  import type { EditorView } from '@codemirror/view';
  import { api, type QueryRun } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { QueryTab, Workspace } from '../app/workspace.svelte';
  import { statementAt } from '../sql/statements';
  import { engine } from '../engines';
  import SqlEditor, { showProblem, showRan } from './SqlEditor.svelte';
  import { serverProblems } from '../sql/problems';
  import QueryResults from './QueryResults.svelte';
  import ProdConfirm from './ProdConfirm.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import { startDrag } from '../ui/drag';

  let { ws, tab }: { ws: Workspace; tab: QueryTab } = $props();

  const session = untrack(() => ws.session);
  const dialect = engine(session.connection.driver).sqlDialect!;
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

  $effect(() => untrack(() => ws.scripts.track(tab.id, () => editor?.state.doc.toString() ?? tab.sql)));

  const tableNames = $derived(ws.tables.map(t => t.name));

  $effect(() => {
    if (!running) return;
    const started = performance.now();
    elapsed = 0;
    const timer = setInterval(() => (elapsed = performance.now() - started), 100);
    return () => clearInterval(timer);
  });

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
    <SqlEditor value={tab.sql} onchange={sql => ws.saveQuery(tab.id, sql)} bind:editor bind:hasSelection {dialect} {syntax} check={checkSyntax} tables={tableNames} defaultSchema={ws.schema} onrun={execute} />
  </div>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="splitter" onpointerdown={startResize}></div>

  <QueryResults sessionId={session.sessionId} tabId={tab.id} bind:run {running} {elapsed} bind:resultIndex />
</div>

{#if pending}
  {@const p = pending}
  <ProdConfirm
    name={session.connection.name}
    writes={p.writes}
    onrun={() => ((pending = null), runScript(p.script, p.base))}
    onclose={() => (pending = null)}
  />
{/if}

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 10px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
  }
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
</style>
