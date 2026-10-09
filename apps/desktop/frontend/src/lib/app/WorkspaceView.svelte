<script lang="ts">
  import type { Workspace } from './workspace.svelte';
  import Sidebar from './Sidebar.svelte';
  import TabBar from './TabBar.svelte';
  import TableView from '../table/TableView.svelte';
  import QueryView from '../query/QueryView.svelte';
  import DiagramView from '../diagram/DiagramView.svelte';
  import Icon from '../ui/Icon.svelte';
  import { startDrag } from '../ui/drag';
  import DdlView from '../objects/DdlView.svelte';
  import ScriptDialogs from '../scripts/Dialogs.svelte';
  import ValuePanel from '../table/ValuePanel.svelte';

  let { ws, visible }: { ws: Workspace; visible: boolean } = $props();

  let sidebarWidth = $state(240);
  let valueWidth = $state(320);
  let valueShown = $state(true);

  const tableTab = $derived(ws.activeTab?.kind === 'table' ? ws.activeTab : null);

  function startResize(e: PointerEvent) {
    const w0 = sidebarWidth;
    startDrag(e, 'col-resize', dx => (sidebarWidth = Math.max(180, Math.min(480, w0 + dx))));
  }

  function startValueResize(e: PointerEvent) {
    const w0 = valueWidth;
    const max = Math.max(220, Math.min(600, window.innerWidth - sidebarWidth - 360));
    startDrag(e, 'col-resize', dx => (valueWidth = Math.max(220, Math.min(max, w0 - dx))));
  }
</script>

<div class="workspace" class:hidden={!visible}>
  <div class="side" style:width="{sidebarWidth}px">
    <Sidebar {ws} active={visible} />
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="side-resizer" onpointerdown={startResize}></div>

  <section class="center">
    <TabBar {ws} bind:valueShown valueToggle={!!tableTab} />

    <div class="panes">
      {#each ws.tabs as tab (tab.id)}
        {@const shown = tab.id === ws.activeTabId}
        <div class="pane" class:hidden={!shown}>
          {#if tab.kind === 'table'}
            <TableView {ws} {tab} active={visible && shown} ondirty={d => (tab.dirty = d)} />
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

  {#if valueShown && tableTab}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="side-resizer" onpointerdown={startValueResize}></div>
    <aside class="value-side" style:width="{valueWidth}px">
      <ValuePanel selected={tableTab.selected ?? null} />
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
  .value-side {
    flex: none;
    min-width: 0;
    height: 100%;
    border-left: 1px solid var(--border);
    background: var(--bg);
  }

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
