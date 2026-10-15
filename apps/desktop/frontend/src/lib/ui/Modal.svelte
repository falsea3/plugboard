<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { dismiss } from './dismiss';
  import type { Snippet } from 'svelte';

  const returnFocus =
    typeof document !== 'undefined' &&
    document.activeElement instanceof HTMLElement &&
    document.activeElement !== document.body
      ? document.activeElement
      : null;

  let {
    title,
    width = 480,
    onclose,
    children,
    header,
    footer,
  }: {
    title: string;
    width?: number;
    onclose: () => void;
    children: Snippet;
    header?: Snippet;
    footer?: Snippet;
  } = $props();

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation();
      onclose();
    }
  }

  onDestroy(() => {
    if (typeof document === 'undefined') return;
    tick().then(() => {
      const active = document.activeElement;
      if (returnFocus?.isConnected && (active === document.body || !active?.isConnected)) {
        returnFocus.focus({ preventScroll: true });
      }
    });
  });
</script>

<svelte:window {onkeydown} />

<div class="backdrop" use:dismiss={onclose}>
  <div class="modal" role="dialog" aria-modal="true" aria-label={title} style:width="{width}px">
    <header>{#if header}{@render header()}{:else}{title}{/if}</header>
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
    border-radius: calc(var(--radius-xl) + 2px);
    background: var(--elevated);
    box-shadow: var(--shadow-modal);
    animation: pop 0.14s ease-out;
  }
  header {
    padding: 18px 20px 4px;
    font-size: 15px;
    font-weight: 600;
  }
  .body {
    padding: 12px 20px 18px;
    overflow: auto;
  }
  footer {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 14px 20px;
    border-top: 1px solid var(--border);
  }
  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(6px) scale(0.985); } }
</style>
