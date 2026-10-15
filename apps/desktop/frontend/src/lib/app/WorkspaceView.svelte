<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity';
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
  import KeyView from '../keys/KeyView.svelte';
  import ConsoleView from '../keys/ConsoleView.svelte';
  import ConfigView from '../admin/ConfigView.svelte';
  import RolesView from '../admin/RolesView.svelte';

  let { ws, visible }: { ws: Workspace; visible: boolean } = $props();

  let sidebarWidth = $state(264);
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
  <div class="side" style:width="{sidebarWidth}px">
    <Sidebar {ws} active={visible} />
  </div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="side-resizer" onpointerdown={startResize}></div>

  <section class="center">
    <TabBar {ws} />

    <div class="panes">
      {#each ws.tabs as tab (tab.id)}
        {@const shown = tab.id === ws.activeTabId}
        {#if shown || opened.has(tab.id)}
          <div class="pane" class:hidden={!shown}>
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
          </div>
        {/if}
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

{#if visible}<ScriptDialogs {ws} />{/if}

<style>
  .workspace { position: absolute; inset: 0; display: flex; background: var(--bg); }
  .workspace.hidden, .pane.hidden { visibility: hidden; pointer-events: none; }
  .side { flex: none; min-width: 0; padding: 8px 0 8px 8px; }
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
