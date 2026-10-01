<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    width = 480,
    onclose,
    children,
    footer,
  }: {
    title: string;
    width?: number;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
  } = $props();

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      onclose();
    }
  }
</script>

<svelte:window {onkeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="backdrop" onclick={e => e.target === e.currentTarget && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-label={title} style:width="{width}px">
    <header>{title}</header>
    <div class="body">{@render children()}</div>
    {#if footer}
      <footer>{@render footer()}</footer>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: grid;
    place-items: center;
    background: rgba(0, 0, 0, 0.42);
    animation: fade 0.12s ease-out;
  }
  .modal {
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 64px);
    display: flex;
    flex-direction: column;
    border-radius: var(--radius-lg);
    background: var(--bg);
    box-shadow: var(--shadow-modal);
    animation: pop 0.14s ease-out;
  }
  header {
    padding: 14px 18px 4px;
    font-size: 14px;
    font-weight: 600;
  }
  .body {
    padding: 10px 18px 16px;
    overflow: auto;
  }
  footer {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 18px;
    border-top: 1px solid var(--border-subtle);
    background: var(--surface);
    border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  }
  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(6px) scale(0.985); } }
</style>
