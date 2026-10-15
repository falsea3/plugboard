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
  import { isDirty } from '../scripts/names';
  import { formatSql } from './format';
  import type { CursorInfo } from './statementMark';

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
  let editorHeight = $state(330);
  let cursor = $state<CursorInfo | null>(null);
  const formatAs = engine(session.connection.driver).formatAs;

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

  function format() {
    if (!editor || !formatAs) return;
    const state = editor.state;
    const sel = state.selection.main;
    const from = sel.empty ? 0 : sel.from;
    const to = sel.empty ? state.doc.length : sel.to;
    try {
      const text = formatSql(state.sliceDoc(from, to), formatAs);
      editor.dispatch({ changes: { from, to, insert: text } });
    } catch (err) {
      app.notify(err);
    }
    editor.focus();
  }

  function startResize(e: PointerEvent) {
    const h0 = editorHeight;
    startDrag(e, 'row-resize', (_, dy) => (editorHeight = Math.max(80, Math.min(window.innerHeight - 220, h0 + dy))));
  }

</script>

<div class="view">
  <section class="editor-card" aria-label="Editor" style:height="{editorHeight}px">
    <div class="toolbar">
      {#if running}
        <button class="btn sm stop" onclick={cancel}><Icon name="stop" size={11} />Stop</button>
      {:else}
        <button class="btn sm primary run" onclick={() => execute(false)} title={hasSelection ? 'Run the selected text' : 'Run the statement under the cursor'}><Icon name="play" size={11} />{hasSelection ? 'Run selection' : 'Run'}<span class="kbd">⌘↵</span></button>
        {#if !hasSelection}<button class="btn sm all" onclick={() => execute(true)}>Run all<span class="kbd">⇧⌘↵</span></button>{/if}
      {/if}
      {#if formatAs}<button class="btn sm ghost" onclick={format} title={hasSelection ? 'Format the selection' : 'Format the script'}>Format</button>{/if}
      <span class="space"></span>
      {#if ws.readOnly}<span class="ro-chip" title="Writes are refused in this session. Switch it in the sidebar."><Icon name="lock" size={11} />Read only</span>{/if}
      <span class="saved">{#if tab.script}saved as <span class="name">{tab.script}</span>{isDirty(tab) ? ' · edited' : ''}{:else}not saved{/if}</span>
    </div>
    <div class="editor">
      <SqlEditor value={tab.sql} onchange={sql => ws.saveQuery(tab.id, sql)} bind:editor bind:hasSelection {dialect} {syntax} check={checkSyntax} tables={tableNames} defaultSchema={ws.schema} onrun={execute} oncursor={c => (cursor = c)} />
    </div>
  </section>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="splitter" onpointerdown={startResize}></div>

  <QueryResults sessionId={session.sessionId} tabId={tab.id} bind:run {running} {elapsed} bind:resultIndex />

  <footer class="status">
    {#if cursor}<span class="chip">Ln {cursor.line}, Col {cursor.col}</span>{/if}
    {#if cursor && cursor.statements > 1 && cursor.statement > 0}<span class="chip">statement {cursor.statement} of {cursor.statements}</span>{/if}
    <span class="space"></span>
    <span>⌘S save script</span>
  </footer>
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
  .editor-card { flex: none; min-height: 120px; margin-top: 6px; display: flex; flex-direction: column; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); overflow: hidden; }
  .toolbar { flex: none; display: flex; align-items: center; gap: 8px; height: 46px; padding: 0 10px; border-bottom: 1px solid var(--grid-line); }
  .toolbar .btn.sm { height: 30px; border-radius: 9px; padding: 0 12px; gap: 7px; }
  .run .kbd { padding: 0; background: transparent; color: inherit; opacity: 0.8; }
  .all { background: var(--border-subtle); color: var(--text-soft); }
  .all .kbd { padding: 0; background: transparent; color: var(--text-4); }
  .space { flex: 1; }
  .saved { overflow: hidden; color: var(--text-4); font-size: 12px; white-space: nowrap; text-overflow: ellipsis; }
  .saved .name { color: var(--text-2); }
  .ro-chip { display: inline-flex; align-items: center; gap: 5px; padding: 1px 8px; border-radius: 6px; background: var(--hover); color: var(--text-2); font-size: 12px; }
  .editor { flex: 1; min-height: 0; }
  .splitter { flex: none; height: 10px; cursor: row-resize; position: relative; }
  .splitter:hover::after { content: ''; position: absolute; left: 40%; right: 40%; top: 4px; height: 2px; border-radius: 1px; background: var(--accent); }
  .status { flex: none; display: flex; align-items: center; gap: 8px; height: 30px; margin-top: 8px; font-size: 12px; color: var(--text-3); }
  .chip { padding: 2px 8px; border-radius: 6px; background: var(--surface); }
</style>
