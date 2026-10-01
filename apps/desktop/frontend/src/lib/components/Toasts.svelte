<script lang="ts">
  import { app } from '../stores/app.svelte';
  import Icon from './Icon.svelte';
</script>

<div class="toasts" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}" role={t.kind === 'error' ? 'alert' : 'status'}>
      <Icon name={t.kind === 'error' ? 'alert' : 'check'} size={14} />
      <span class="text">{t.text}</span>
      <button class="close" onclick={() => app.dismiss(t.id)} aria-label="Dismiss"><Icon name="x" size={12} /></button>
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    right: 16px;
    bottom: 16px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 420px;
  }
  .toast {
    display: flex;
    align-items: flex-start;
    gap: 9px;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    font-size: 12.5px;
    animation: slide 0.16s ease-out;
  }
  .toast.error :global(.icon:first-child) { color: var(--danger); margin-top: 1px; }
  .toast.info :global(.icon:first-child) { color: var(--ok); margin-top: 1px; }
  .text { flex: 1; user-select: text; -webkit-user-select: text; word-break: break-word; }
  .close { border: 0; background: transparent; color: var(--text-3); padding: 2px; }
  .close:hover { color: var(--text); }
  @keyframes slide { from { opacity: 0; transform: translateY(8px); } }
</style>
