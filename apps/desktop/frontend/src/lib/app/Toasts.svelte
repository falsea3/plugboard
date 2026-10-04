<script lang="ts">
  import { app } from './app.svelte';
  import Icon from '../ui/Icon.svelte';
</script>

<div class="toasts" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}" role={t.kind === 'error' ? 'alert' : 'status'}>
      <Icon name={t.kind === 'error' ? 'alert' : 'check'} size={14} />
      <div class="body">
        <span class="text">{t.text}</span>
        {#if t.detail}<span class="detail">{t.detail}</span>{/if}
        {#if t.action}
          {@const action = t.action}
          <button class="btn sm action" onclick={() => { action.run(); app.dismiss(t.id); }}>{action.label}</button>
        {/if}
      </div>
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
  .body { flex: 1; min-width: 0; display: flex; flex-direction: column; align-items: flex-start; gap: 4px; }
  .text { user-select: text; -webkit-user-select: text; word-break: break-word; }
  .detail { font-family: var(--font-mono); font-size: 11px; color: var(--text-3); user-select: text; -webkit-user-select: text; word-break: break-word; }
  .action { margin-top: 4px; }
  .close { border: 0; background: transparent; color: var(--text-3); padding: 2px; }
  .close:hover { color: var(--text); }
  @keyframes slide { from { opacity: 0; transform: translateY(8px); } }
</style>
