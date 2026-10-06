<script lang="ts">
  import type { Workspace } from '../app/workspace.svelte';
  import Modal from '../ui/Modal.svelte';
  import NameModal from './NameModal.svelte';
  import { suggestName } from './names';

  let { ws }: { ws: Workspace } = $props();

  const scripts = $derived(ws.scripts);

  function save() {
    const tab = scripts.closing;
    scripts.closing = null;
    if (tab) scripts.save(tab, true);
  }

  function discard() {
    const tab = scripts.closing;
    scripts.closing = null;
    if (tab) ws.closeTab(tab.id);
  }
</script>

{#if scripts.closing}
  {@const tab = scripts.closing}
  <Modal title="Save changes to “{tab.title}”?" width={420} onclose={() => (scripts.closing = null)}>
    <p class="text">
      {tab.script ? 'The script has changes that aren’t saved yet.' : 'This query hasn’t been saved as a script.'}
      Closing the tab without saving loses them.
    </p>
    {#snippet footer()}
      <button class="btn danger" onclick={discard}>Don’t save</button>
      <span style="flex:1"></span>
      <button class="btn" onclick={() => (scripts.closing = null)}>Cancel</button>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn primary" autofocus onclick={save}>{tab.script ? 'Save' : 'Save as…'}</button>
    {/snippet}
  </Modal>
{/if}

{#if scripts.naming}
  {@const n = scripts.naming}
  <NameModal
    title="Save as script"
    value={suggestName(n.tab.title, scripts.names)}
    taken={scripts.names}
    action={n.close ? 'Save and close' : 'Save'}
    onsubmit={async name => {
      await scripts.saveAs(n.tab, name, n.close);
      scripts.naming = null;
    }}
    onclose={() => (scripts.naming = null)}
  />
{/if}

<style>
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
