<script lang="ts">
  import { app } from '../stores/app.svelte';
  import Icon from './Icon.svelte';
  import Spinner from './Spinner.svelte';
  import Modal from './Modal.svelte';
  import ReleaseNotes from './ReleaseNotes.svelte';

  let showNotes = $state(false);
  const u = $derived(app.update);

  function download() {
    if (u) window.runtime?.BrowserOpenURL?.(u.info.releaseUrl);
  }
</script>

{#if u && !app.updateDismissed}
  <div class="banner" role="status">
    {#if u.status === 'installing'}<Spinner size={15} />{:else}<Icon name={u.status === 'failed' ? 'alert' : 'update'} size={16} />{/if}
    <div class="body">
      {#if u.status === 'available'}
        <div class="title">Relay DB {u.info.version} is available</div>
      {:else if u.status === 'installing'}
        <div class="title">Installing Relay DB {u.info.version}…</div>
      {:else if u.status === 'installed'}
        <div class="title">Relay DB {u.info.version} is installed</div>
        <div class="sub">It starts with the next launch.</div>
      {:else if u.status === 'failed'}
        <div class="title">Relay DB {u.info.version} couldn’t be installed</div>
        <div class="sub" title={u.error}>{u.error}</div>
      {/if}
      <div class="actions">
        <button class="btn sm ghost" onclick={() => (showNotes = true)}>What’s new</button>
        {#if u.status === 'available'}
          <button class="btn sm primary" onclick={() => app.installUpdate()}><Icon name="download" size={12} />Install</button>
        {:else if u.status === 'installed'}
          <button class="btn sm primary" onclick={() => app.restartToUpdate()}>Restart now</button>
        {:else if u.status === 'failed'}
          <button class="btn sm" onclick={download}>Download</button>
          <button class="btn sm primary" onclick={() => app.installUpdate()}>Try again</button>
        {/if}
      </div>
    </div>
    {#if u.status !== 'installing'}
      <button class="close" onclick={() => (app.updateDismissed = true)} aria-label="Hide until next launch"><Icon name="x" size={12} /></button>
    {/if}
  </div>
{/if}

{#if showNotes && u}
  <Modal title="What’s new in {u.info.version}" width={520} onclose={() => (showNotes = false)}>
    <ReleaseNotes markdown={u.info.notes} />
    {#snippet footer()}
      <button class="btn ghost" onclick={download}>Release page</button>
      <span style="flex:1"></span>
      <button class="btn" onclick={() => (showNotes = false)}>Close</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .banner {
    position: fixed;
    left: 16px;
    bottom: 16px;
    z-index: 90;
    display: flex;
    align-items: flex-start;
    gap: 10px;
    width: 340px;
    padding: 12px 12px 10px;
    border-radius: 10px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    font-size: 12.5px;
    animation: slide 0.18s ease-out;
  }
  .banner > :global(.icon), .banner > :global(.spinner) { color: var(--accent); margin-top: 1px; }
  .body { flex: 1; min-width: 0; }
  .title { font-weight: 600; color: var(--text); }
  .sub { margin-top: 2px; color: var(--text-3); overflow: hidden; text-overflow: ellipsis; display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; }
  .actions { display: flex; gap: 6px; margin-top: 8px; }
  .close { border: 0; background: transparent; color: var(--text-3); padding: 2px; }
  .close:hover { color: var(--text); }
  @keyframes slide { from { opacity: 0; transform: translateY(8px); } }
</style>
