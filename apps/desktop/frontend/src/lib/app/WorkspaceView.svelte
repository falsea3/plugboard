<script lang="ts">
  import { keys } from '../ui/keys';
  import { SvelteSet } from 'svelte/reactivity';
  import type { Workspace } from './workspace.svelte';
  import Sidebar from './Sidebar.svelte';
  import Pane from './Pane.svelte';
  import TabBar from './TabBar.svelte';
  import TableView from '../table/TableView.svelte';
  import QueryView from '../query/QueryView.svelte';
  import DiagramView from '../diagram/DiagramView.svelte';
  import Icon from '../ui/Icon.svelte';
  import { startDrag } from '../ui/drag';
  import { layout } from './layout.svelte';
  import DdlView from '../objects/DdlView.svelte';
  import ScriptDialogs from '../scripts/Dialogs.svelte';
  import KeyView from '../keys/KeyView.svelte';
  import ConsoleView from '../keys/ConsoleView.svelte';
  import ConfigView from '../admin/ConfigView.svelte';
  import RolesView from '../admin/RolesView.svelte';

  let { ws, visible }: { ws: Workspace; visible: boolean } = $props();

  let sidebarWidth = $state(264);
  let peek = $state(false);
  const opened = new SvelteSet<string>();
  $effect(() => {
    if (ws.activeTabId) opened.add(ws.activeTabId);
  });

  function startResize(e: PointerEvent) {
    const w0 = sidebarWidth;
    startDrag(e, 'col-resize', dx => (sidebarWidth = Math.max(220, Math.min(480, w0 + dx))));
  }

</script>

<div class="workspace" class:hidden={!visible}>
  {#if layout.sidebarHidden}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="edge" onmouseenter={() => (peek = true)}></div>
  {/if}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="side" class:floating={layout.sidebarHidden} class:peek={peek && layout.sidebarHidden} style:width="{sidebarWidth}px" onmouseleave={() => (peek = false)}>
    <Sidebar {ws} active={visible} />
  </div>
  {#if !layout.sidebarHidden}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="side-resizer" onpointerdown={startResize}></div>
  {/if}

  <section class="center" class:full={layout.sidebarHidden} class:no-lights={!layout.windowButtons}>
    <TabBar {ws} />

    <div class="panes">
      {#each ws.tabs as tab (tab.id)}
        {@const shown = tab.id === ws.activeTabId}
        {#if shown || opened.has(tab.id)}
          <div class="pane" class:hidden={!shown}>
            <Pane visible={visible && shown}>
            {#if tab.kind === 'table'}
              <TableView {ws} {tab} active={visible && shown} ondirty={d => (tab.dirty = d)} />
            {:else if tab.kind === 'diagram'}
              <DiagramView {ws} {tab} />
            {:else if tab.kind === 'ddl'}
              <DdlView {ws} object={tab.object} />
            {:else if tab.kind === 'config'}
              <ConfigView {ws} ondirty={d => (tab.dirty = d)} />
            {:else if tab.kind === 'roles'}
              <RolesView {ws} />
            {:else if tab.kind === 'key'}
              <KeyView {ws} {tab} active={visible && shown} />
            {:else if ws.session.engine.keyValue}
              <ConsoleView {ws} {tab} />
            {:else}
              <QueryView {ws} {tab} />
            {/if}
            </Pane>
          </div>
        {/if}
      {:else}
        <div class="welcome">
          <p class="muted">Pick a table on the left, or</p>
          <button class="btn" onclick={() => ws.newQuery()}><Icon name="code" size={13} />Open a SQL editor</button>
          <div class="shortcuts faint">
            <span><span class="kbd">{keys('⌘T')}</span> new query</span>
            <span><span class="kbd">{keys('⌘K')}</span> switch connection</span>
            <span><span class="kbd">{keys('⇧⌘F')}</span> filter tables</span>
            <span><span class="kbd">{keys('⌘,')}</span> settings</span>
          </div>
        </div>
      {/each}
    </div>
  </section>

</div>

{#if visible}<ScriptDialogs {ws} />{/if}

<style>
  .workspace { position: absolute; inset: 0; display: flex; background: var(--bg); }
  .workspace.hidden, .pane.hidden { visibility: hidden; pointer-events: none; content-visibility: hidden; }
  .side { flex: none; min-width: 0; padding: 8px 0 8px 8px; }
  .side.floating { position: absolute; top: 0; bottom: 0; left: 0; z-index: 40; padding-right: 8px; transform: translateX(-100%); transition: transform 0.18s ease; }
  .side.floating.peek { transform: none; }
  .side.floating :global(.sidebar) { box-shadow: var(--shadow-pop); }
  .edge { position: absolute; top: 0; bottom: 0; left: 0; z-index: 39; width: 8px; }
  .center.full { padding-left: 12px; }
  .center.full:not(.no-lights) :global(.tabbar) { padding-left: 76px; }
  .side-resizer { flex: none; width: 8px; z-index: 2; cursor: col-resize; }
  .side-resizer:hover { background: linear-gradient(90deg, transparent 3px, var(--accent) 3px, var(--accent) 5px, transparent 5px); }
  .center { flex: 1; min-width: 0; display: flex; flex-direction: column; padding: 0 12px 10px 4px; container-type: inline-size; }
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
