<script lang="ts">
  import type { Workspace } from '../app/workspace.svelte';
  import GridMenu from '../grid/GridMenu.svelte';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import { copyToClipboard } from '../ui/clipboard';
  import NameModal from './NameModal.svelte';
  import { scriptMenu } from './names';

  let { ws, filter }: { ws: Workspace; filter: string } = $props();

  let open = $state(true);
  let menu = $state<{ x: number; y: number; name: string } | null>(null);
  let renaming = $state<string | null>(null);
  let deleting = $state<string | null>(null);

  const names = $derived(ws.scripts.names.filter(n => n.toLowerCase().includes(filter.trim().toLowerCase())));
  const active = $derived(ws.activeTab?.kind === 'query' ? ws.activeTab.script : undefined);
  const shown = $derived(open || !!filter.trim());

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
  <button class="group" aria-expanded={shown} onclick={() => (open = !open)}>
    <span class="twist" class:open={shown}><Icon name="chevron-right" size={9} /></span>
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
        <span class="twist"></span>
        <Icon name="code" size={13} />
        <span class="label">{name}</span>
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
  .group {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 10px 8px 4px 2px;
    border: 0;
    background: transparent;
    color: var(--text-3);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    text-align: left;
  }
  .group:hover { color: var(--text-2); }
  .count { margin-left: auto; font-weight: 500; }
  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 8px 0 2px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    text-align: left;
  }
  .item:hover { background: var(--hover); color: var(--text); }
  .item.active { background: var(--accent-dim); color: var(--text); }
  .item :global(.icon) { color: var(--text-3); }
  .twist { flex: none; display: flex; align-items: center; justify-content: center; width: 16px; height: 18px; transition: transform 0.12s; }
  .twist.open { transform: rotate(90deg); }
  .label { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12.5px; }
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
