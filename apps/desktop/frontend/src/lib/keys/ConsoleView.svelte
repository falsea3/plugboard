<script lang="ts">
  import { keys } from '../ui/keys';
  import { untrack } from 'svelte';
  import type { EditorView } from '@codemirror/view';
  import { api, type QueryRun } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { QueryTab, Workspace } from '../app/workspace.svelte';
  import QueryResults from '../query/QueryResults.svelte';
  import ProdConfirm from '../query/ProdConfirm.svelte';
  import Icon from '../ui/Icon.svelte';
  import { startDrag } from '../ui/drag';
  import CommandEditor from './CommandEditor.svelte';
  import { lineAt } from './commands';

  let { ws, tab }: { ws: Workspace; tab: QueryTab } = $props();

  const session = untrack(() => ws.session);
  const isProd = session.connection.env === 'prod';

  let editor = $state<EditorView>();
  let hasSelection = $state(false);
  let run = $state<QueryRun | null>(null);
  let running = $state(false);
  let elapsed = $state(0);
  let resultIndex = $state(0);
  let editorHeight = $state(200);
  let pending = $state<{ script: string; writes: string[] } | null>(null);
  let queryId = '';

  $effect(() => untrack(() => ws.scripts.track(tab.id, () => editor?.state.doc.toString() ?? tab.sql)));

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
    let script = '';
    if (!sel.empty) script = state.sliceDoc(sel.from, sel.to);
    else if (all) script = state.doc.toString();
    else script = lineAt(state.doc.toString(), sel.head)?.text ?? '';
    if (!script.trim()) return;
    if (isProd && app.settings.confirmProdWrites && !ws.readOnly) {
      try {
        const writes = await api.writeStatements(session.sessionId, script);
        if (writes.length > 0) {
          pending = { script, writes };
          return;
        }
      } catch (err) {
        app.notify(err);
        return;
      }
    }
    await runScript(script);
  }

  async function runScript(script: string) {
    running = true;
    queryId = `${tab.id}-${Date.now()}`;
    try {
      const r = await api.runQuery(session.sessionId, queryId, script);
      run = r;
      resultIndex = Math.max(0, r.results.length - 1);
      if (r.results.length > 0) api.writeStatements(session.sessionId, script).then(w => void (w.length > 0 && ws.keys.load()), () => {});
    } catch (err) {
      run = { results: [], error: err instanceof Error ? err.message : String(err), errorIndex: -1, errorPosition: -1, cancelled: false, rolledBack: false };
    } finally {
      running = false;
    }
  }

  function startResize(e: PointerEvent) {
    const h0 = editorHeight;
    startDrag(e, 'row-resize', (_, dy) => (editorHeight = Math.max(80, Math.min(window.innerHeight - 220, h0 + dy))));
  }
</script>

<div class="view">
  <section class="editor-card" aria-label="Console" style:height="{editorHeight}px">
    <div class="toolbar">
      {#if running}
        <button class="btn sm stop" onclick={() => queryId && api.cancelQuery(queryId)}><Icon name="stop" size={11} />Stop</button>
      {:else}
        <button class="btn sm primary run" onclick={() => execute(false)} title="Run the line under the cursor"><Icon name="play" size={11} />{hasSelection ? 'Run selection' : 'Run line'}<span class="kbd">{keys('⌘↵')}</span></button>
        {#if !hasSelection}<button class="btn sm all" onclick={() => execute(true)}>Run all<span class="kbd">{keys('⇧⌘↵')}</span></button>{/if}
      {/if}
      <span class="space"></span>
      {#if ws.readOnly}<span class="ro-chip" title="Commands that change data are refused. Switch it in the sidebar."><Icon name="lock" size={11} />Read only</span>{/if}
    </div>
    <div class="editor">
      <CommandEditor value={tab.sql} onchange={text => ws.saveQuery(tab.id, text)} bind:editor bind:hasSelection onrun={execute} />
    </div>
  </section>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="splitter" onpointerdown={startResize}></div>

  <QueryResults sessionId={session.sessionId} tabId={tab.id} bind:run {running} {elapsed} bind:resultIndex />
</div>

{#if pending}
  {@const p = pending}
  <ProdConfirm name={session.connection.name} writes={p.writes} onrun={() => ((pending = null), runScript(p.script))} onclose={() => (pending = null)} />
{/if}

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; padding-bottom: 2px; }
  .editor-card { flex: none; min-height: 120px; margin-top: 6px; display: flex; flex-direction: column; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); background: var(--card); overflow: hidden; }
  .toolbar { flex: none; display: flex; align-items: center; gap: 8px; height: 46px; padding: 0 10px; border-bottom: 1px solid var(--grid-line); }
  .toolbar .btn.sm { height: 30px; border-radius: 9px; padding: 0 12px; gap: 7px; }
  .run .kbd { padding: 0; background: transparent; color: inherit; opacity: 0.8; }
  .all { background: var(--border-subtle); color: var(--text-soft); }
  .all .kbd { padding: 0; background: transparent; color: var(--text-4); }
  .space { flex: 1; }
  .ro-chip { display: inline-flex; align-items: center; gap: 5px; padding: 1px 8px; border-radius: 6px; background: var(--hover); color: var(--text-2); font-size: 12px; }
  .editor { flex: 1; min-height: 0; }
  .splitter { flex: none; height: 10px; cursor: row-resize; position: relative; }
  .splitter:hover::after { content: ''; position: absolute; left: 40%; right: 40%; top: 4px; height: 2px; border-radius: 1px; background: var(--accent); }
</style>
