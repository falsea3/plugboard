<script lang="ts">
  import { app, type Workspace } from '../stores/app.svelte';
  import Sidebar from './Sidebar.svelte';
  import TableView from './TableView.svelte';
  import QueryView from './QueryView.svelte';
  import DiagramView from './DiagramView.svelte';
  import Icon from './Icon.svelte';
  import { startDrag } from '../drag';

  let { ws, visible }: { ws: Workspace; visible: boolean } = $props();

  let sidebarWidth = $state(240);

  function startResize(e: PointerEvent) {
    const w0 = sidebarWidth;
    startDrag(e, 'col-resize', dx => (sidebarWidth = Math.max(180, Math.min(480, w0 + dx))));
  }
</script>

<div class="workspace" class:hidden={!visible}>
  <div class="side" style:width="{sidebarWidth}px">
    <Sidebar {ws} active={visible} />
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="side-resizer" onpointerdown={startResize}></div>

  <section class="center">
    <nav class="tabs" aria-label="Open tabs">
      {#each ws.tabs as tab (tab.id)}
        <div
          class="tab"
          class:active={tab.id === ws.activeTabId}
          role="tab"
          tabindex="-1"
          aria-selected={tab.id === ws.activeTabId}
          onmousedown={e => e.button === 1 && e.preventDefault()}
          onauxclick={e => e.button === 1 && app.requestCloseTab(ws, tab.id)}
        >
          <button
            class="tab-main"
            onclick={() => (ws.activeTabId = tab.id)}
            title={tab.kind === 'table' ? `${tab.schema}.${tab.table}` : tab.title}
          >
            <Icon name={tab.kind === 'query' ? 'code' : tab.kind === 'diagram' ? 'diagram' : tab.tableKind === 'view' ? 'view' : 'table'} size={12} />
            <span>{tab.kind === 'table' ? tab.table : tab.title}</span>
            {#if tab.kind === 'table' && tab.dirty}<span class="dirty-dot" title="Uncommitted changes"></span>{/if}
          </button>
          <button class="tab-close" onclick={() => app.requestCloseTab(ws, tab.id)} title="Close (⌘W or middle-click)" aria-label="Close tab"><Icon name="x" size={12} /></button>
        </div>
      {/each}
      <button class="tab-new" onclick={() => ws.newQuery()} title="New query (⌘T)"><Icon name="plus" size={13} /></button>
    </nav>

    <div class="panes">
      {#each ws.tabs as tab (tab.id)}
        {@const shown = tab.id === ws.activeTabId}
        <div class="pane" class:hidden={!shown}>
          {#if tab.kind === 'table'}
            <TableView {ws} {tab} active={visible && shown} ondirty={d => (tab.dirty = d)} />
          {:else if tab.kind === 'diagram'}
            <DiagramView {ws} {tab} />
          {:else}
            <QueryView {ws} {tab} />
          {/if}
        </div>
      {:else}
        <div class="welcome">
          <p class="muted">Pick a table on the left, or</p>
          <button class="btn" onclick={() => ws.newQuery()}><Icon name="code" size={13} />Open a SQL editor</button>
          <div class="shortcuts faint">
            <span><span class="kbd">⌘T</span> new query</span>
            <span><span class="kbd">⌘K</span> switch connection</span>
            <span><span class="kbd">⇧⌘F</span> filter tables</span>
            <span><span class="kbd">⌘,</span> settings</span>
          </div>
        </div>
      {/each}
    </div>
  </section>
</div>

<style>
  .workspace { position: absolute; inset: 0; display: flex; }
  .workspace.hidden, .pane.hidden { visibility: hidden; pointer-events: none; }
  .side { flex: none; min-width: 0; }
  .side-resizer { flex: none; width: 4px; margin-left: -2px; margin-right: -2px; z-index: 2; cursor: col-resize; }
  .side-resizer:hover { background: var(--accent); }
  .center { flex: 1; min-width: 0; display: flex; flex-direction: column; }

  .tabs {
    flex: none;
    display: flex;
    align-items: stretch;
    height: 34px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabs::-webkit-scrollbar { display: none; }
  .tab {
    position: relative;
    flex: none;
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 120px;
    max-width: 220px;
    padding: 0 6px 0 12px;
    border-right: 1px solid var(--border-subtle);
    color: var(--text-2);
  }
  .tab:hover { color: var(--text); }
  .tab.active { background: var(--bg); color: var(--text); }
  .tab.active::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: -1px;
    height: 1px;
    background: var(--bg);
  }
  .tab.active::before {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    top: 0;
    height: 2px;
    background: var(--accent);
  }
  .tab-main {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 7px;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    color: inherit;
    font-size: 12.5px;
    text-align: left;
  }
  .tab-main span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tab-main .dirty-dot { flex: none; width: 6px; height: 6px; border-radius: 50%; background: var(--warn); }
  .tab-close {
    flex: none;
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-3);
    opacity: 0;
  }
  .tab:hover .tab-close, .tab.active .tab-close { opacity: 1; }
  .tab-close:hover { background: var(--hover); color: var(--text); }
  .tab-new {
    flex: none;
    width: 34px;
    border: 0;
    background: transparent;
    color: var(--text-3);
    display: grid;
    place-items: center;
  }
  .tab-new:hover { color: var(--text); background: var(--hover); }

  .panes { flex: 1; min-height: 0; position: relative; }
  .pane { position: absolute; inset: 0; }

  .welcome {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
  }
  .welcome p { margin: 0; }
  .shortcuts { display: flex; gap: 16px; margin-top: 16px; font-size: 12px; }
</style>
