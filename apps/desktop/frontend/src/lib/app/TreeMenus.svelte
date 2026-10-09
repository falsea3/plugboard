<script lang="ts">
  import { api, type DBObject, type NewIndex, type StructureChange } from '../api/backend';
  import { app } from './app.svelte';
  import type { Workspace } from './workspace.svelte';
  import GridMenu from '../grid/GridMenu.svelte';
  import PromptModal from '../ui/PromptModal.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import { objectMenu } from '../objects/kinds';
  import IndexModal from './IndexModal.svelte';
  import { columnChange, treeMenu, treePrompt, treeSQL, typeSuggestions, type TreePrompt, type TreeTarget } from './tree';

  let { ws }: { ws: Workspace } = $props();

  let menu = $state<{ x: number; y: number; target: TreeTarget } | null>(null);
  let objMenu = $state<{ x: number; y: number; object: DBObject } | null>(null);
  let prompt = $state<(TreePrompt & { id: string; target: TreeTarget }) | null>(null);
  let newIndex = $state<TreeTarget | null>(null);

  const keyOf = (t: TreeTarget) => `${t.table.schema}.${t.table.name}`;

  export function openTree(e: MouseEvent, target: TreeTarget) {
    e.preventDefault();
    menu = { x: e.clientX, y: e.clientY, target };
  }

  export function openObject(e: MouseEvent, object: DBObject) {
    e.preventDefault();
    objMenu = { x: e.clientX, y: e.clientY, object };
  }

  async function structureSQL(sc: StructureChange) {
    return (await api.previewStructure(ws.session.sessionId, sc)).map(s => s + ';').join('\n');
  }

  async function sql(make: () => Promise<string>) {
    try {
      ws.newQuery(await make());
    } catch (err) {
      app.notify(err);
    }
  }

  function pick(id: string, target: TreeTarget) {
    menu = null;
    const { table, column, index } = target;
    const text = treeSQL(id, ws.session.engine, target);
    const known = ws.columns.get(keyOf(target));
    const ask = treePrompt(id, target, typeSuggestions(Array.isArray(known) ? known : [], ws.objects, ws.session.engine.columnTypes));
    if (ask) prompt = { ...ask, id, target };
    else if (id === 'tree-index') {
      ws.loadColumns(table);
      newIndex = target;
    } else if (id === 'tree-drop-index' && index) sql(() => api.dropIndexSQL(ws.session.sessionId, table.schema, table.name, index.name));
    else if (id === 'tree-truncate') sql(() => api.truncateSQL(ws.session.sessionId, table.schema, table.name));
    else if (id === 'tree-drop' && column) sql(() => structureSQL({ schema: table.schema, table: table.name, changes: [{ kind: 'delete', column: column.name, defaultSet: false, default: null }] }));
    else if (id === 'tree-open') ws.openTable(table);
    else if (id === 'tree-structure') ws.openTable(table, undefined, 'structure');
    else if (id === 'tree-ddl') ws.openTable(table, undefined, 'ddl');
    else if (id === 'tree-diagram') ws.openDiagram();
    else if (id === 'tree-copy-name') copyToClipboard(index?.name ?? column?.name ?? table.name);
    else if (id === 'tree-copy-select' && text) copyToClipboard(text);
    else if (text) ws.newQuery(text);
  }

  function createIndex(idx: NewIndex) {
    const table = newIndex?.table;
    newIndex = null;
    if (table) sql(() => api.createIndexSQL(ws.session.sessionId, table.schema, table.name, idx));
  }

  function pickObject(id: string, o: DBObject) {
    objMenu = null;
    if (id === 'obj-ddl') ws.openDDL(o);
    else if (id === 'obj-copy-name') copyToClipboard(o.name);
    else if (id === 'obj-query') sql(() => api.objectDDL(ws.session.sessionId, o));
  }

  function answer(p: TreePrompt & { id: string; target: TreeTarget }, value: string) {
    prompt = null;
    const { table } = p.target;
    const change = columnChange(p.id, p.target, value);
    sql(async () =>
      change
        ? structureSQL(change)
        : api.renameTableSQL(ws.session.sessionId, table.schema, table.name, value),
    );
  }
</script>

{#if menu}
  {@const m = menu}
  <GridMenu items={treeMenu(m.target, ws.readOnly, ws.session.engine.canAlterColumns, ws.session.engine.canRenameTables)} x={m.x} y={m.y} onpick={id => pick(id, m.target)} onclose={() => (menu = null)} />
{/if}

{#if objMenu}
  {@const m = objMenu}
  <GridMenu items={objectMenu()} x={m.x} y={m.y} onpick={id => pickObject(id, m.object)} onclose={() => (objMenu = null)} />
{/if}

{#if prompt}
  {@const p = prompt}
  <PromptModal title={p.title} label={p.label} value={p.value} suggestions={p.suggestions} action="Open SQL" onsubmit={v => answer(p, v)} onclose={() => (prompt = null)} />
{/if}

{#if newIndex}
  {@const n = newIndex}
  <IndexModal
    table={n.table}
    columns={ws.columns.get(keyOf(n))}
    first={n.column?.name}
    onsubmit={createIndex}
    onclose={() => (newIndex = null)}
  />
{/if}
