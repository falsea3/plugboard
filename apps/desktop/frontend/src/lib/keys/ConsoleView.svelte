<script lang="ts">
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
  <div class="toolbar">
    {#if running}
      <button class="btn sm" onclick={() => queryId && api.cancelQuery(queryId)}><Icon name="stop" size={11} />Stop</button>
    {:else}
      <button class="btn sm primary" onclick={() => execute(false)} title="Run the line under the cursor (⌘↵)"><Icon name="play" size={11} />{hasSelection ? 'Run selection' : 'Run line'}</button>
      {#if !hasSelection}<button class="btn sm" onclick={() => execute(true)} title="Run every line (⇧⌘↵)">Run all</button>{/if}
    {/if}
    <span class="faint small"><span class="kbd">⌘↵</span> line · <span class="kbd">⇧⌘↵</span> all</span>
    {#if ws.readOnly}
      <span style="flex:1"></span>
      <span class="ro-chip" title="Commands that change data are refused. Switch it in the sidebar."><Icon name="lock" size={11} />Read-only</span>
    {/if}
  </div>

  <div class="editor" style:height="{editorHeight}px">
    <CommandEditor value={tab.sql} onchange={text => ws.saveQuery(tab.id, text)} bind:editor bind:hasSelection onrun={execute} />
  </div>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="splitter" onpointerdown={startResize}></div>

  <QueryResults sessionId={session.sessionId} tabId={tab.id} bind:run {running} {elapsed} bind:resultIndex />
</div>

{#if pending}
  {@const p = pending}
  <ProdConfirm name={session.connection.name} writes={p.writes} onrun={() => ((pending = null), runScript(p.script))} onclose={() => (pending = null)} />
{/if}

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .toolbar { flex: none; display: flex; align-items: center; gap: 6px; height: 36px; padding: 0 10px; background: var(--surface); border-bottom: 1px solid var(--border); }
  .small { font-size: 12px; margin-left: 6px; white-space: nowrap; }
  .ro-chip { display: inline-flex; align-items: center; gap: 5px; height: 20px; padding: 0 8px; border-radius: 5px; color: var(--text-2); background: var(--elevated); font-size: 11.5px; font-weight: 550; }
  .editor { flex: none; min-height: 80px; }
  .splitter { flex: none; height: 5px; margin-top: -2px; margin-bottom: -3px; position: relative; z-index: 3; cursor: row-resize; border-top: 1px solid var(--border); }
  .splitter:hover { border-top: 2px solid var(--accent); }
</style>
