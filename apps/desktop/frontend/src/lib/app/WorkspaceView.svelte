<script lang="ts">
  import { app } from './app.svelte';
  import type { Workspace } from './workspace.svelte';
  import type { CellValue, ResultColumn } from '../api/backend';
  import Sidebar from './Sidebar.svelte';
  import TableView from '../table/TableView.svelte';
  import QueryView from '../query/QueryView.svelte';
  import DiagramView from '../diagram/DiagramView.svelte';
  import Icon from '../ui/Icon.svelte';
  import { startDrag } from '../ui/drag';
  import DdlView from '../objects/DdlView.svelte';
  import { iconOf } from '../objects/kinds';
  import ScriptDialogs from '../scripts/Dialogs.svelte';
  import { isDirty } from '../scripts/names';
  import { createDragGhost, type DragGhost } from '../ui/dragGhost';
  import ValuePanel from '../table/ValuePanel.svelte';

  let { ws, visible }: { ws: Workspace; visible: boolean } = $props();

  let sidebarWidth = $state(240);
  let valueWidth = $state(320);
  let valueVisible = $state(true);
  let selectedCells = $state<Record<string, { value: CellValue; column: ResultColumn } | null>>({});
  let tabsEl = $state<HTMLElement>();
  let draggingTabId = $state('');
  let tabDropTarget = $state<{ id: string; position: 'before' | 'after'; x: number; top: number; height: number } | null>(null);
  let pointerDrag: { id: string; pointerId: number; x: number; y: number } | null = null;
  let dragGhost: DragGhost | null = null;
  let suppressClickTabId = '';
  let suppressClickTimer: ReturnType<typeof setTimeout> | undefined;

  const selectedValue = $derived.by(() => {
    const tab = ws.activeTab;
    return tab?.kind === 'table' ? selectedCells[tab.id] ?? null : null;
  });

  function startResize(e: PointerEvent) {
    const w0 = sidebarWidth;
    startDrag(e, 'col-resize', dx => (sidebarWidth = Math.max(180, Math.min(480, w0 + dx))));
  }

  function startValueResize(e: PointerEvent) {
    const w0 = valueWidth;
    const max = Math.max(220, Math.min(600, window.innerWidth - sidebarWidth - 360));
    startDrag(e, 'col-resize', dx => (valueWidth = Math.max(220, Math.min(max, w0 - dx))));
  }

  function tabDropTargetAt(x: number) {
    if (!tabsEl) return null;
    const items = ws.tabs.filter(tab => tab.id !== draggingTabId);
    if (items.length === 0) return null;
    const bounds = tabsEl.getBoundingClientRect();

    if (x <= bounds.left) {
      return { id: items[0].id, position: 'before' as const, x: bounds.left, top: bounds.top + 2, height: bounds.height - 4 };
    }
    if (x >= bounds.right) {
      return { id: items[items.length - 1].id, position: 'after' as const, x: bounds.right - 2, top: bounds.top + 2, height: bounds.height - 4 };
    }

    const nodes = new Map<string, HTMLElement>();
    for (const node of tabsEl.querySelectorAll<HTMLElement>('[data-tab]')) {
      if (node.dataset.tab) nodes.set(node.dataset.tab, node);
    }
    let lastVisible: (typeof items)[number] | undefined;
    for (const tab of items) {
      const node = nodes.get(tab.id);
      if (!node) continue;
      const rect = node.getBoundingClientRect();
      if (rect.right <= bounds.left || rect.left >= bounds.right) continue;
      if (x < rect.left + rect.width / 2) {
        return { id: tab.id, position: 'before' as const, x: Math.max(bounds.left, rect.left - 1), top: bounds.top + 2, height: bounds.height - 4 };
      }
      lastVisible = tab;
    }
    if (!lastVisible) return null;
    const rect = nodes.get(lastVisible.id)?.getBoundingClientRect();
    return rect ? { id: lastVisible.id, position: 'after' as const, x: Math.min(bounds.right - 2, rect.right - 1), top: bounds.top + 2, height: bounds.height - 4 } : null;
  }

  function reorderTab(id: string, target: NonNullable<typeof tabDropTarget>) {
    const from = ws.tabs.findIndex(tab => tab.id === id);
    if (from < 0) return;
    const [dragged] = ws.tabs.splice(from, 1);
    if (!dragged) return;
    let to = ws.tabs.findIndex(tab => tab.id === target.id);
    if (to < 0) {
      ws.tabs.splice(Math.min(from, ws.tabs.length), 0, dragged);
      return;
    }
    if (target.position === 'after') to++;
    ws.tabs.splice(to, 0, dragged);
  }

  function onTabPointerDown(e: PointerEvent, id: string) {
    if (!e.isPrimary || e.button !== 0) return;
    if (suppressClickTimer) clearTimeout(suppressClickTimer);
    suppressClickTimer = undefined;
    suppressClickTabId = '';
    pointerDrag = { id, pointerId: e.pointerId, x: e.clientX, y: e.clientY };
  }

  function onTabPointerMove(e: PointerEvent) {
    const drag = pointerDrag;
    if (!drag || drag.pointerId !== e.pointerId) return;
    if (!draggingTabId && Math.hypot(e.clientX - drag.x, e.clientY - drag.y) < 5) return;
    if (!draggingTabId) {
      const source = tabsEl?.querySelector<HTMLElement>(`[data-tab="${drag.id}"]`);
      if (source) dragGhost = createDragGhost(source, drag.x, drag.y);
    }
    draggingTabId = drag.id;
    dragGhost?.move(e.clientX, e.clientY);
    tabDropTarget = tabDropTargetAt(e.clientX);
  }

  function onTabPointerUp(e: PointerEvent) {
    const drag = pointerDrag;
    if (!drag || drag.pointerId !== e.pointerId) return;
    if (draggingTabId === drag.id) {
      const target = tabDropTargetAt(e.clientX);
      if (target) reorderTab(drag.id, target);
      suppressClickTabId = drag.id;
      suppressClickTimer = setTimeout(() => {
        suppressClickTabId = '';
        suppressClickTimer = undefined;
      });
    }
    dragGhost?.destroy();
    dragGhost = null;
    pointerDrag = null;
    draggingTabId = '';
    tabDropTarget = null;
  }

  function onTabPointerCancel(e: PointerEvent) {
    if (pointerDrag?.pointerId !== e.pointerId) return;
    dragGhost?.destroy();
    dragGhost = null;
    pointerDrag = null;
    draggingTabId = '';
    tabDropTarget = null;
  }

  function onTabClick(e: MouseEvent, id: string) {
    if (suppressClickTabId === id) {
      e.preventDefault();
      suppressClickTabId = '';
      if (suppressClickTimer) clearTimeout(suppressClickTimer);
      suppressClickTimer = undefined;
      return;
    }
    ws.activeTabId = id;
  }
</script>

<svelte:window onpointermove={onTabPointerMove} onpointerup={onTabPointerUp} onpointercancel={onTabPointerCancel} />

<div class="workspace" class:hidden={!visible}>
  <div class="side" style:width="{sidebarWidth}px">
    <Sidebar {ws} active={visible} />
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="side-resizer" onpointerdown={startResize}></div>

  <section class="center">
    <div class="tabbar">
      <nav class="tabs" aria-label="Open tabs" bind:this={tabsEl}>
        {#each ws.tabs as tab (tab.id)}
          <div
            class="tab"
            class:active={tab.id === ws.activeTabId}
            class:dragging={draggingTabId === tab.id}
            data-tab={tab.id}
            role="tab"
            tabindex="-1"
            aria-selected={tab.id === ws.activeTabId}
            onmousedown={e => e.button === 1 && e.preventDefault()}
            onauxclick={e => e.button === 1 && app.requestCloseTab(ws, tab.id)}
          >
            <button
              class="tab-main"
              onpointerdown={e => onTabPointerDown(e, tab.id)}
              onclick={e => onTabClick(e, tab.id)}
              title={tab.kind === 'table' ? `${tab.schema}.${tab.table}` : tab.title}
            >
              <Icon name={tab.kind === 'query' ? 'code' : tab.kind === 'diagram' ? 'diagram' : tab.kind === 'ddl' ? iconOf(tab.object) : tab.tableKind === 'view' ? 'view' : 'table'} size={12} />
              <span>{tab.kind === 'table' ? tab.table : tab.title}</span>
              {#if tab.kind === 'table' && tab.dirty}<span class="dirty-dot" title="Uncommitted changes"></span>{/if}
              {#if tab.kind === 'query' && isDirty(tab)}<span class="dirty-dot" title="Unsaved changes"></span>{/if}
            </button>
            <button class="tab-close" onclick={() => app.requestCloseTab(ws, tab.id)} title="Close (⌘W or middle-click)" aria-label="Close tab"><Icon name="x" size={12} /></button>
          </div>
        {/each}
        <button class="tab-new" onclick={() => ws.newQuery()} title="New query (⌘T)"><Icon name="plus" size={13} /></button>
      </nav>
      <button class="value-toggle" class:active={valueVisible} aria-pressed={valueVisible} aria-expanded={valueVisible} onclick={() => (valueVisible = !valueVisible)} title={valueVisible ? 'Hide Value panel' : 'Show Value panel'}>
        <Icon name="columns" size={13} /><span>Value</span>
      </button>
    </div>
    {#if tabDropTarget}
      <span class="tab-drop-indicator" style:left="{tabDropTarget.x}px" style:top="{tabDropTarget.top}px" style:height="{tabDropTarget.height}px" aria-hidden="true"></span>
    {/if}

    <div class="panes">
      {#each ws.tabs as tab (tab.id)}
        {@const shown = tab.id === ws.activeTabId}
        <div class="pane" class:hidden={!shown}>
          {#if tab.kind === 'table'}
            <TableView
              {ws}
              {tab}
              active={visible && shown}
              ondirty={d => (tab.dirty = d)}
              onselect={selected => (selectedCells[tab.id] = selected)}
            />
          {:else if tab.kind === 'diagram'}
            <DiagramView {ws} {tab} />
          {:else if tab.kind === 'ddl'}
            <DdlView {ws} object={tab.object} />
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

  {#if valueVisible}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="side-resizer" onpointerdown={startValueResize}></div>
    <aside class="value-side" style:width="{valueWidth}px">
      <ValuePanel selected={selectedValue} />
    </aside>
  {/if}
</div>

{#if visible}<ScriptDialogs {ws} />{/if}

<style>
  .workspace { position: absolute; inset: 0; display: flex; }
  .workspace.hidden, .pane.hidden { visibility: hidden; pointer-events: none; }
  .side { flex: none; min-width: 0; }
  .side-resizer { flex: none; width: 4px; margin-left: -2px; margin-right: -2px; z-index: 2; cursor: col-resize; }
  .side-resizer:hover { background: var(--accent); }
  .center { flex: 1; min-width: 0; display: flex; flex-direction: column; container-type: inline-size; }

  .tabbar {
    flex: none;
    display: flex;
    align-items: stretch;
    height: 34px;
    min-width: 0;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
  }
  .tabs {
    flex: 1;
    display: flex;
    align-items: stretch;
    min-width: 0;
    height: 34px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabs::-webkit-scrollbar { display: none; }
  .value-toggle {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px;
    border: 0;
    border-left: 1px solid var(--border-subtle);
    background: transparent;
    color: var(--text-3);
    font-size: 12px;
  }
  .value-toggle:hover { background: var(--hover); color: var(--text); }
  .value-toggle.active { color: var(--accent); background: var(--accent-dim); }
  .value-side {
    flex: none;
    min-width: 0;
    height: 100%;
    border-left: 1px solid var(--border);
    background: var(--bg);
  }
  .tab-drop-indicator {
    position: fixed;
    z-index: 100;
    width: 2px;
    border-radius: 2px;
    background: var(--accent);
    pointer-events: none;
  }
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
  .tab.dragging { opacity: 0.45; }
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
    user-select: none;
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
