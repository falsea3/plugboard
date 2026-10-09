<script lang="ts">
  import { api } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { KeyTab, Workspace } from '../app/workspace.svelte';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import PromptModal from '../ui/PromptModal.svelte';
  import type { Change } from './items';

  let { ws, tab, ttl, onchange }: { ws: Workspace; tab: KeyTab; ttl: number; onchange: (ch: Change) => Promise<boolean> } = $props();

  let asking = $state<'ttl' | 'rename' | 'delete' | null>(null);

  async function setTTL(text: string) {
    const n = Number(text.trim() || '0');
    if (!Number.isInteger(n)) {
      app.notify('TTL is a whole number of seconds.', 'info');
      return;
    }
    asking = null;
    await onchange({ op: 'expire', index: n });
  }

  async function rename(to: string) {
    asking = null;
    try {
      await api.editKey(ws.session.sessionId, tab.db, { key: tab.key.key, op: 'rename', value: to });
      ws.renameKey(tab, { ...tab.key, name: to, key: to });
    } catch (err) {
      app.notify(err);
    }
  }

  async function remove() {
    asking = null;
    try {
      await api.editKey(ws.session.sessionId, tab.db, { key: tab.key.key, op: 'delete' });
      ws.keys.removed(tab.key.key);
      ws.closeKey(tab.key.key);
    } catch (err) {
      app.notify(err);
    }
  }
</script>

{#if !ws.readOnly}
  <button class="btn sm ghost" onclick={() => (asking = 'ttl')} title="Set when the key expires"><Icon name="timer" size={12} />TTL</button>
  <button class="btn sm ghost" onclick={() => (asking = 'rename')} disabled={tab.key.key !== tab.key.name}><Icon name="pencil" size={12} />Rename</button>
  <button class="btn sm ghost danger" onclick={() => (asking = 'delete')}><Icon name="trash" size={12} />Delete</button>
{/if}

{#if asking === 'ttl'}
  <PromptModal
    title="Expire “{tab.key.name}”"
    label="Seconds until it expires"
    value={ttl > 0 ? String(ttl) : ''}
    action="Set TTL"
    hint="Leave empty or 0 to keep the key forever."
    onsubmit={setTTL}
    onclose={() => (asking = null)}
  />
{:else if asking === 'rename'}
  <PromptModal title="Rename key" label="New name" value={tab.key.name} action="Rename" hint="Fails if a key with that name exists." onsubmit={rename} onclose={() => (asking = null)} />
{:else if asking === 'delete'}
  <Modal title="Delete “{tab.key.name}”?" width={400} onclose={() => (asking = null)}>
    <p class="text">The key and its value are deleted from database {tab.db}. This can’t be undone.</p>
    {#snippet footer()}
      <span style="flex:1"></span>
      <!-- svelte-ignore a11y_autofocus -->
      <button class="btn" autofocus onclick={() => (asking = null)}>Cancel</button>
      <button class="btn primary danger-fill" onclick={remove}>Delete</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .text { margin: 0; color: var(--text-2); line-height: 1.5; }
</style>
