<script lang="ts">
  import type { Workspace } from '../app/workspace.svelte';
  import GridMenu from '../grid/GridMenu.svelte';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import NameModal from './NameModal.svelte';
  import { isDirty, scriptMenu } from './names';

  let { ws, filter }: { ws: Workspace; filter: string } = $props();

  let open = $state(true);
  let menu = $state<{ x: number; y: number; name: string } | null>(null);
  let renaming = $state<string | null>(null);
  let deleting = $state<string | null>(null);

  const names = $derived(ws.scripts.names.filter(n => n.toLowerCase().includes(filter.trim().toLowerCase())));
  const active = $derived(ws.activeTab?.kind === 'query' ? ws.activeTab.script : undefined);
  const shown = $derived(open || !!filter.trim());

  function isDirtyName(name: string) {
    return ws.tabs.some(t => t.kind === 'query' && t.script === name && isDirty(t));
  }

  function pick(id: string, name: string) {
    menu = null;
    if (id === 'script-open') ws.scripts.open(name);
    else if (id === 'script-rename') renaming = name;
    else if (id === 'script-copy-name') copyToClipboard(name);
    else if (id === 'script-delete') deleting = name;
  }

  function remove() {
    const name = deleting;
    deleting = null;
    if (name) ws.scripts.remove(name);
  }
</script>

{#if names.length > 0}
  <button class="group" aria-expanded={shown} onclick={() => (open = !open)} title={shown ? 'Hide scripts' : 'Show scripts'}>
    <span>Scripts</span>
    <span class="count">{names.length}</span>
  </button>
  {#if shown}
    {#each names as name (name)}
      <button
        class="item"
        class:active={active === name}
        onclick={() => ws.scripts.open(name)}
        oncontextmenu={e => (e.preventDefault(), (menu = { x: e.clientX, y: e.clientY, name }))}
        title={name}
      >
        <span class="label">{name}</span>
        {#if isDirtyName(name)}<span class="dirty"></span>{/if}
      </button>
    {/each}
  {/if}
{/if}

{#if menu}
  {@const m = menu}
  <GridMenu items={scriptMenu()} x={m.x} y={m.y} onpick={id => pick(id, m.name)} onclose={() => (menu = null)} />
{/if}

{#if renaming !== null}
  {@const from = renaming}
  <NameModal
    title="Rename script"
    value={from}
    current={from}
    taken={ws.scripts.names}
    action="Rename"
    onsubmit={async to => {
      await ws.scripts.rename(from, to);
      renaming = null;
    }}
    onclose={() => (renaming = null)}
  />
{/if}

{#if deleting !== null}
  <Modal title="Delete “{deleting}”?" width={400} onclose={() => (deleting = null)}>
    <p class="text">The script file is deleted. A tab that has it open keeps its text as an unsaved query.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (deleting = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={remove}>Delete</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .dirty { flex: none; width: 6px; height: 6px; border-radius: 50%; background: var(--warn); }
  .item { padding-left: 26px !important; }
  .group {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 4px;
    padding: 18px 8px 8px;
    border: 0;
    background: transparent;
    color: var(--text-3);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    text-align: left;
  }
  .group:hover { color: var(--text-2); }
  .count { margin-left: auto; font-weight: 500; }
  .item { width: 100%; display: flex; align-items: center; gap: 6px; height: 30px; padding: 0 8px; border: 0; border-radius: 8px; background: transparent; color: var(--text-2); font-size: 13px; text-align: left; }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.active { background: var(--tree-selected); color: var(--accent-text); font-weight: 500; }
  .item.active :global(.icon) { color: var(--accent-text); }
  .item :global(.icon) { color: var(--text-3); }
  .label { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
