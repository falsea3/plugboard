<script lang="ts">
  import type { Workspace } from './workspace.svelte';
  import { FILTER_TABLES_EVENT } from './commands';
  import SessionHeader from './SessionHeader.svelte';
  import Icon from '../ui/Icon.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import Select from '../ui/Select.svelte';
  import type { TableInfo } from '../api/wire';
  import TableNode from './TableNode.svelte';
  import TreeMenus from './TreeMenus.svelte';
  import ObjectGroups from '../objects/ObjectGroups.svelte';
  import ScriptList from '../scripts/ScriptList.svelte';

  let { ws, active }: { ws: Workspace; active: boolean } = $props();

  let filter = $state('');
  let input = $state<HTMLInputElement>();

  const visible = $derived.by(() => {
    const q = filter.trim().toLowerCase();
    return q ? ws.tables.filter(t => t.name.toLowerCase().includes(q)) : ws.tables;
  });
  const tables = $derived(visible.filter(t => t.kind === 'table'));
  const views = $derived(visible.filter(t => t.kind === 'view'));
  const activeTable = $derived(ws.activeTab?.kind === 'table' ? ws.activeTab : null);
  const shown = $derived([...tables, ...views]);

  const keyOf = (t: TableInfo) => `${t.schema}.${t.name}`;
  let picked = $state<string[]>([]);
  let anchor = $state('');
  const pickedShown = $derived(shown.filter(t => picked.includes(keyOf(t))));

  function onItemClick(e: MouseEvent, t: TableInfo) {
    const key = keyOf(t);
    if (e.metaKey || e.ctrlKey) {
      picked = picked.includes(key) ? picked.filter(k => k !== key) : [...picked, key];
      anchor = key;
      return;
    }
    if (e.shiftKey) {
      const from = shown.findIndex(x => keyOf(x) === (anchor || activeKey()));
      const to = shown.indexOf(t);
      if (from >= 0) {
        picked = shown.slice(Math.min(from, to), Math.max(from, to) + 1).map(keyOf);
        return;
      }
    }
    picked = [];
    anchor = key;
    ws.openTable(t);
  }

  const activeKey = () => (activeTable ? `${activeTable.schema}.${activeTable.table}` : '');

  function openPicked() {
    const list = pickedShown;
    for (const t of list) ws.openTable(t);
    if (list.length > 0) ws.openTable(list[0]);
    picked = [];
  }

  function onListKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && picked.length > 0) picked = [];
    if (e.key === 'Enter' && pickedShown.length > 0) {
      e.preventDefault();
      openPicked();
    }
  }

  $effect(() => {
    const focus = () => {
      if (!active) return;
      input?.focus();
      input?.select();
    };
    window.addEventListener(FILTER_TABLES_EVENT, focus);
    return () => window.removeEventListener(FILTER_TABLES_EVENT, focus);
  });

  let menus = $state<TreeMenus>();

  function onFilterKey(e: KeyboardEvent) {
    if (e.key === 'Enter' && visible.length > 0) ws.openTable(visible[0]);
    if (e.key === 'Escape') filter = '';
  }
</script>

<aside class="sidebar">
  <SessionHeader {ws} />

  {#if ws.session.schemas.length > 1}
    <div class="schema">
      <Select value={ws.schema} options={ws.session.schemas.map(s => ({ value: s, label: s }))} onchange={s => ws.setSchema(s)} aria-label="Schema" />
    </div>
  {/if}

  <div class="filter">
    <Icon name="search" size={13} />
    <input class="input" bind:this={input} bind:value={filter} onkeydown={onFilterKey} placeholder="Filter tables" spellcheck="false" />
  </div>

  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="list" onkeydown={onListKey}>
    {#if ws.tablesLoading && ws.tables.length === 0}
      <div class="note faint loading"><Spinner size={11} />Loading tables…</div>
    {:else if ws.tablesError}
      <div class="note error">{ws.tablesError}</div>
    {:else}
      {#each [{ label: 'Tables', items: tables, icon: 'table' as const }, { label: 'Views', items: views, icon: 'view' as const }] as group (group.label)}
        {#if group.items.length > 0 || (group.label === 'Tables' && !filter)}
          <div class="group">
            <span>{group.label}</span>
            <span class="count">{group.items.length}</span>
          </div>
          {#each group.items as t (t.name)}
            <TableNode
              {ws}
              {t}
              active={activeTable?.table === t.name && activeTable?.schema === t.schema}
              picked={picked.includes(keyOf(t))}
              onclick={e => onItemClick(e, t)}
              onmenu={(e, column, index) => menus?.openTree(e, { table: t, column, index })}
            />
          {/each}
        {/if}
      {/each}
      <ObjectGroups {ws} {filter} onmenu={(e, o) => menus?.openObject(e, o)} />
      <ScriptList {ws} {filter} />
      {#if visible.length === 0 && filter && !ws.objects.some(o => o.name.toLowerCase().includes(filter.trim().toLowerCase())) && !ws.scripts.names.some(n => n.toLowerCase().includes(filter.trim().toLowerCase()))}
        <div class="note faint">Nothing matches “{filter}”.</div>
      {/if}
    {/if}
  </div>

  {#if pickedShown.length > 0}
    <div class="picked-bar">
      <span class="small">{pickedShown.length} selected</span>
      <span style="flex:1"></span>
      <button class="btn sm ghost" onclick={() => (picked = [])}>Clear</button>
      <button class="btn sm primary" onclick={openPicked}>Open {pickedShown.length}</button>
    </div>
  {/if}

  <TreeMenus bind:this={menus} {ws} />

  <div class="bottom">
    <button class="btn sm ghost" onclick={() => ws.newQuery()} title="New query (⌘T)"><Icon name="code" size={13} />New query</button>
    <span style="flex:1"></span>
    <button class="btn icon sm ghost" onclick={() => ws.openDiagram()} title="Schema diagram" aria-label="Schema diagram"><Icon name="diagram" size={13} /></button>
    <button class="btn icon sm ghost" onclick={() => ws.loadTables()} disabled={ws.tablesLoading} title="Reload tables">{#if ws.tablesLoading}<Spinner size={12} label="Loading tables" />{:else}<Icon name="refresh" size={13} />{/if}</button>
  </div>
</aside>

<style>
  .sidebar {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border-right: 1px solid var(--border);
    min-width: 0;
  }
  .schema { padding: 10px 10px 0; }
  .filter { position: relative; padding: 10px; color: var(--text-3); }
  .filter :global(.icon) { position: absolute; left: 19px; top: 18px; }
  .filter .input { padding-left: 27px; height: 26px; }

  .list { flex: 1; overflow-y: auto; padding: 0 6px 8px; }
  .group {
    display: flex;
    justify-content: space-between;
    padding: 10px 8px 4px;
    color: var(--text-3);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .count { font-weight: 500; }
  .picked-bar { display: flex; align-items: center; gap: 6px; padding: 6px 8px; border-top: 1px solid var(--border-subtle); color: var(--text-2); }
  .note { padding: 12px 8px; font-size: 12px; }
  .note.loading { display: flex; align-items: center; gap: 7px; }
  .note.error { color: var(--danger); user-select: text; -webkit-user-select: text; }

  .bottom {
    display: flex;
    align-items: center;
    padding: 6px 8px;
    border-top: 1px solid var(--border-subtle);
  }
</style>
