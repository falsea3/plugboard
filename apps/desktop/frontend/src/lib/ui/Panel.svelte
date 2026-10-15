<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    actions,
    children,
    tone = 'elevated',
    label,
    class: className = '',
  }: { title?: string; actions?: Snippet; children: Snippet; tone?: 'elevated' | 'card'; label?: string; class?: string } = $props();
</script>

<section class="panel {tone} {className}" aria-label={label ?? title}>
  {#if title || actions}
    <header>
      {#if title}<h3>{title}</h3>{/if}
      <span style="flex:1"></span>
      {#if actions}{@render actions()}{/if}
    </header>
  {/if}
  {@render children()}
</section>

<style>
  .panel { display: flex; flex-direction: column; min-width: 0; min-height: 0; border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); overflow: hidden; }
  .elevated { background: var(--elevated); }
  .card { background: var(--card); }
  header { display: flex; align-items: center; gap: 8px; min-height: 42px; padding: 0 8px 0 14px; border-bottom: 1px solid var(--border-subtle); }
  h3 { margin: 0; font-size: 13px; font-weight: 500; }
</style>
