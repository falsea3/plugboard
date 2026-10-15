<script lang="ts">
  import { app } from './app.svelte';
  import type { Tab, Workspace } from './workspace.svelte';
  import Icon, { type IconName } from '../ui/Icon.svelte';
  import { DragOrder } from '../ui/dragOrder.svelte';
  import { moveItem } from '../ui/reorder';
  import { iconOf } from '../objects/kinds';
  import { isDirty } from '../scripts/names';

  let { ws }: { ws: Workspace } = $props();

  let list = $state<HTMLElement>();

  function tabIcon(tab: Tab): IconName {
    if (tab.kind === 'query') return ws.session.engine.keyValue ? 'terminal' : 'code';
    if (tab.kind === 'diagram') return 'diagram';
    if (tab.kind === 'ddl') return iconOf(tab.object);
    if (tab.kind === 'key') return 'key';
    if (tab.kind === 'config') return 'config';
    if (tab.kind === 'roles') return 'roles';
    return tab.tableKind === 'view' ? 'view' : 'table';
  }
  const order = new DragOrder({
    list: () => list,
    attr: 'tab',
    ids: () => ws.tabs.map(t => t.id),
    move: (id, drop) => {
      const kind = ws.tabs.find(t => t.id === id)?.kind;
      if (moveItem(ws.tabs, id, drop) && kind === 'query') ws.scripts.persist();
    },
  });
</script>

<svelte:window onpointermove={order.pointermove} onpointerup={order.pointerup} onpointercancel={order.pointercancel} />

<div class="tabbar drag">
  <nav class="tabs no-drag" aria-label="Open tabs" bind:this={list}>
    {#each ws.tabs as tab (tab.id)}
      <div
        class="tab"
        class:active={tab.id === ws.activeTabId}
        class:dragging={order.dragging === tab.id}
        data-tab={tab.id}
        role="tab"
        tabindex="-1"
        aria-selected={tab.id === ws.activeTabId}
        onmousedown={e => e.button === 1 && e.preventDefault()}
        onauxclick={e => e.button === 1 && app.requestCloseTab(ws, tab.id)}
      >
        <button
          class="tab-main"
          onpointerdown={e => order.down(e, tab.id)}
          onclick={e => order.clicked(e, tab.id) && (ws.activeTabId = tab.id)}
          title={tab.kind === 'table' ? `${tab.schema}.${tab.table}` : tab.title}
        >
          <Icon name={tabIcon(tab)} size={14} />
          <span>{tab.kind === 'table' ? tab.table : tab.title}</span>
          {#if (tab.kind === 'table' || tab.kind === 'config') && tab.dirty}<span class="dirty-dot" title="Uncommitted changes"></span>{/if}
          {#if tab.kind === 'query' && isDirty(tab)}<span class="dirty-dot" title="Unsaved changes"></span>{/if}
        </button>
        <button class="tab-close" onclick={() => app.requestCloseTab(ws, tab.id)} title="Close (⌘W or middle-click)" aria-label="Close tab"><Icon name="x" size={12} /></button>
      </div>
    {/each}
  </nav>
  <span class="space"></span>
  <button class="btn ghost icon no-drag" onclick={() => (app.settingsOpen = 'general')} title="Settings (⌘,)" aria-label="Settings"><Icon name="settings" size={16} /></button>
</div>
{#if order.indicator}
  {@const d = order.indicator}
  <span class="drop-indicator" style:left="{d.x}px" style:top="{d.top}px" style:height="{d.height}px" aria-hidden="true"></span>
{/if}

<style>
  .tabbar { flex: none; display: flex; align-items: center; gap: 4px; height: 48px; min-width: 0; padding: 0 0 0 2px; }
  .tabs { display: flex; align-items: center; gap: 4px; min-width: 0; overflow-x: auto; scrollbar-width: none; }
  .tabs::-webkit-scrollbar { display: none; }
  .space { flex: 1; align-self: stretch; }
  .drop-indicator { position: fixed; z-index: 100; width: 2px; border-radius: 2px; background: var(--accent); pointer-events: none; }
  .tab {
    position: relative;
    flex: none;
    display: flex;
    align-items: center;
    gap: 2px;
    height: 34px;
    max-width: 240px;
    padding: 0 14px;
    border-radius: var(--radius);
    color: var(--text-soft);
    transition: background 0.12s, color 0.12s;
  }
  .tab:hover { background: var(--elevated); color: var(--text); }
  .tab.dragging { opacity: 0.45; }
  .tab.active { background: var(--accent-bg); color: var(--tab-active); }
  .tab-main { flex: 1; display: flex; align-items: center; gap: 8px; min-width: 0; height: 100%; padding: 0; border: 0; background: transparent; color: inherit; font-size: 13.5px; text-align: left; user-select: none; }
  .tab-main span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tab-main .dirty-dot { flex: none; width: 6px; height: 6px; border-radius: 50%; background: var(--warn); }
  .tab-close { position: absolute; right: 4px; top: 50%; transform: translateY(-50%); display: grid; place-items: center; width: 20px; height: 20px; padding: 0; border: 0; border-radius: 6px; background: inherit; color: inherit; opacity: 0; }
  .tab:hover .tab-close { opacity: 0.8; }
  .tab-close:hover { opacity: 1; background: var(--active); }
</style>
