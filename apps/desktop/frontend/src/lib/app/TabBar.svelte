<script lang="ts">
  import { app } from './app.svelte';
  import type { Workspace } from './workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import { DragOrder } from '../ui/dragOrder.svelte';
  import { moveItem } from '../ui/reorder';
  import { iconOf } from '../objects/kinds';
  import { isDirty } from '../scripts/names';

  let { ws, valueShown = $bindable(), valueToggle }: { ws: Workspace; valueShown: boolean; valueToggle: boolean } = $props();

  let list = $state<HTMLElement>();
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

<div class="tabbar">
  <nav class="tabs" aria-label="Open tabs" bind:this={list}>
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
  {#if valueToggle}
    <button class="value-toggle" class:active={valueShown} aria-pressed={valueShown} onclick={() => (valueShown = !valueShown)} title={valueShown ? 'Hide Value panel' : 'Show Value panel'}>
      <Icon name="columns" size={13} /><span>Value</span>
    </button>
  {/if}
</div>
{#if order.indicator}
  {@const d = order.indicator}
  <span class="drop-indicator" style:left="{d.x}px" style:top="{d.top}px" style:height="{d.height}px" aria-hidden="true"></span>
{/if}

<style>

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
  .drop-indicator {
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
</style>
